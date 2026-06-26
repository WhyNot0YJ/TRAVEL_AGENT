import type { ChatRequest, ChatResponse, CreateTaskResponse, ErrorResponse, TaskResponse, TravelPlanRequest } from "./types";

const rawBaseUrl = import.meta.env.VITE_API_BASE_URL ?? "";
export const apiBaseUrl = rawBaseUrl.replace(/\/$/, "");

async function parseJson<T>(response: Response): Promise<T> {
  const data = (await response.json()) as T | ErrorResponse;
  if (!response.ok) {
    const error = data as ErrorResponse;
    throw new Error(error.message || `Request failed with ${response.status}`);
  }
  return data as T;
}

export async function createTravelPlanTask(payload: TravelPlanRequest): Promise<CreateTaskResponse> {
  const response = await fetch(`${apiBaseUrl}/api/v1/travel/plans`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });
  return parseJson<CreateTaskResponse>(response);
}

export async function getTravelPlanTask(taskId: string): Promise<TaskResponse> {
  const response = await fetch(`${apiBaseUrl}/api/v1/travel/plans/${encodeURIComponent(taskId)}`);
  return parseJson<TaskResponse>(response);
}

export function createTravelPlanEventSource(taskId: string): EventSource {
  return new EventSource(`${apiBaseUrl}/api/v1/travel/plans/${encodeURIComponent(taskId)}/stream`);
}

export async function chatTravelInfo(payload: ChatRequest): Promise<ChatResponse> {
  const response = await fetch(`${apiBaseUrl}/api/v1/travel/chat`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });
  return parseJson<ChatResponse>(response);
}

export async function streamChatTravelInfo(
  payload: ChatRequest,
  onDelta: (chunk: string) => void,
): Promise<ChatResponse> {
  const response = await fetch(`${apiBaseUrl}/api/v1/travel/chat/stream`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: "text/event-stream",
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok || !response.body) {
    const data = (await response.json().catch(() => undefined)) as ErrorResponse | undefined;
    throw new Error(data?.message || `Request failed with ${response.status}`);
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let finalResponse: ChatResponse | undefined;

  const consumeBlock = (block: string) => {
    const lines = block.split(/\r?\n/);
    const eventLine = lines.find((line) => line.startsWith("event:"));
    const dataLines = lines.filter((line) => line.startsWith("data:"));
    if (!eventLine || dataLines.length === 0) {
      return;
    }
    const eventName = eventLine.slice("event:".length).trim();
    const dataText = dataLines.map((line) => line.slice("data:".length).trimStart()).join("\n");
    if (eventName === "assistant_delta") {
      const event = JSON.parse(dataText) as { message?: string };
      if (event.message) {
        onDelta(event.message);
      }
      return;
    }
    if (eventName === "done") {
      finalResponse = JSON.parse(dataText) as ChatResponse;
    }
    if (eventName === "error") {
      const event = JSON.parse(dataText) as { message?: string };
      throw new Error(event.message || "聊天流处理失败");
    }
  };

  while (true) {
    const { done, value } = await reader.read();
    buffer += decoder.decode(value ?? new Uint8Array(), { stream: !done });
    let splitIndex = buffer.indexOf("\n\n");
    while (splitIndex >= 0) {
      const block = buffer.slice(0, splitIndex).trim();
      buffer = buffer.slice(splitIndex + 2);
      if (block) {
        consumeBlock(block);
      }
      splitIndex = buffer.indexOf("\n\n");
    }
    if (done) {
      const block = buffer.trim();
      if (block) {
        consumeBlock(block);
      }
      break;
    }
  }

  if (!finalResponse) {
    throw new Error("聊天流没有返回最终结果");
  }
  return finalResponse;
}
