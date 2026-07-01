import type {
  AuthLoginRequest,
  AuthMeResponse,
  AuthRegisterRequest,
  AuthUser,
  ChatRequest,
  ChatResponse,
  ConversationArchive,
  CreateTaskResponse,
  CurrentResponse,
  ErrorResponse,
  PatchPlanRequest,
  PublicPlan,
  PublicPlanListResponse,
  PublishPlanRequest,
  SavePlanRequest,
  TaskResponse,
  TravelPlanRequest,
  UserPlan,
  UserPlanListResponse,
} from "./types";

const rawBaseUrl = import.meta.env.VITE_API_BASE_URL ?? "";
export const apiBaseUrl = rawBaseUrl.replace(/\/$/, "");

// All authenticated routes need cookies; defaulting to "include" keeps a
// single fetch helper across anonymous and signed-in calls.
const credentialsMode: RequestCredentials = "include";
const publicPlanDetailRequests = new Map<string, Promise<PublicPlan>>();

async function parseJson<T>(response: Response): Promise<T> {
  const text = await response.text();
  let parsed: unknown;
  if (text.length > 0) {
    try {
      parsed = JSON.parse(text);
    } catch {
      parsed = undefined;
    }
  }
  if (!response.ok) {
    const error = (parsed as ErrorResponse | undefined) ?? undefined;
    const err = new Error(error?.message || `Request failed with ${response.status}`) as Error & {
      status?: number;
      code?: string;
    };
    err.status = response.status;
    err.code = error?.code;
    throw err;
  }
  return (parsed as T) ?? ({} as T);
}

function jsonRequest<T>(method: string, path: string, body?: unknown): Promise<T> {
  return fetch(`${apiBaseUrl}${path}`, {
    method,
    credentials: credentialsMode,
    headers: body !== undefined ? { "Content-Type": "application/json" } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  }).then((response) => parseJson<T>(response));
}

export async function createTravelPlanTask(payload: TravelPlanRequest): Promise<CreateTaskResponse> {
  return jsonRequest<CreateTaskResponse>("POST", "/api/v1/travel/plans", payload);
}

export async function getTravelPlanTask(taskId: string): Promise<TaskResponse> {
  return jsonRequest<TaskResponse>("GET", `/api/v1/travel/plans/${encodeURIComponent(taskId)}`);
}

export function createTravelPlanEventSource(taskId: string): EventSource {
  return new EventSource(`${apiBaseUrl}/api/v1/travel/plans/${encodeURIComponent(taskId)}/stream`, {
    withCredentials: true,
  });
}

export async function chatTravelInfo(payload: ChatRequest): Promise<ChatResponse> {
  return jsonRequest<ChatResponse>("POST", "/api/v1/travel/chat", payload);
}

export async function streamChatTravelInfo(
  payload: ChatRequest,
  onDelta: (chunk: string) => void,
): Promise<ChatResponse> {
  const response = await fetch(`${apiBaseUrl}/api/v1/travel/chat/stream`, {
    method: "POST",
    credentials: credentialsMode,
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

// Auth

export async function authRegister(payload: AuthRegisterRequest): Promise<AuthUser> {
  const data = await jsonRequest<{ user: AuthUser }>("POST", "/api/v1/auth/register", payload);
  return data.user;
}

export async function authLogin(payload: AuthLoginRequest): Promise<AuthUser> {
  const data = await jsonRequest<{ user: AuthUser }>("POST", "/api/v1/auth/login", payload);
  return data.user;
}

export async function authLogout(): Promise<void> {
  await fetch(`${apiBaseUrl}/api/v1/auth/logout`, {
    method: "POST",
    credentials: credentialsMode,
  });
}

export async function authMe(): Promise<AuthUser | null> {
  const response = await fetch(`${apiBaseUrl}/api/v1/auth/me`, { credentials: credentialsMode });
  if (response.status === 401) {
    return null;
  }
  const data = await parseJson<AuthMeResponse>(response);
  return data.user;
}

// My plans

export async function savePlan(payload: SavePlanRequest): Promise<UserPlan> {
  const data = await jsonRequest<{ plan: UserPlan }>("POST", "/api/v1/me/plans", payload);
  return data.plan;
}

export async function listMyPlans(params: {
  q?: string;
  visibility?: string;
  publish_status?: string;
  page?: number;
  page_size?: number;
}): Promise<UserPlanListResponse> {
  const search = new URLSearchParams();
  if (params.q) search.set("q", params.q);
  if (params.visibility) search.set("visibility", params.visibility);
  if (params.publish_status) search.set("publish_status", params.publish_status);
  if (params.page) search.set("page", String(params.page));
  if (params.page_size) search.set("page_size", String(params.page_size));
  const qs = search.toString();
  return jsonRequest<UserPlanListResponse>("GET", `/api/v1/me/plans${qs ? `?${qs}` : ""}`);
}

export async function getMyPlan(planId: string): Promise<UserPlan> {
  const data = await jsonRequest<{ plan: UserPlan }>("GET", `/api/v1/me/plans/${encodeURIComponent(planId)}`);
  return data.plan;
}

export async function patchMyPlan(planId: string, payload: PatchPlanRequest): Promise<UserPlan> {
  const data = await jsonRequest<{ plan: UserPlan }>(
    "PATCH",
    `/api/v1/me/plans/${encodeURIComponent(planId)}`,
    payload,
  );
  return data.plan;
}

export async function deleteMyPlan(planId: string): Promise<void> {
  await fetch(`${apiBaseUrl}/api/v1/me/plans/${encodeURIComponent(planId)}`, {
    method: "DELETE",
    credentials: credentialsMode,
  }).then((res) => {
    if (!res.ok && res.status !== 204) {
      throw new Error(`Delete failed: ${res.status}`);
    }
  });
}

export async function getPlanConversation(planId: string): Promise<ConversationArchive> {
  const data = await jsonRequest<{ conversation: ConversationArchive }>(
    "GET",
    `/api/v1/me/plans/${encodeURIComponent(planId)}/conversation`,
  );
  return data.conversation;
}

export async function publishMyPlan(planId: string, payload: PublishPlanRequest = {}): Promise<PublicPlan> {
  const data = await jsonRequest<{ public_plan: PublicPlan }>(
    "POST",
    `/api/v1/me/plans/${encodeURIComponent(planId)}/publish`,
    payload,
  );
  return data.public_plan;
}

export async function unpublishMyPlan(planId: string): Promise<void> {
  await fetch(`${apiBaseUrl}/api/v1/me/plans/${encodeURIComponent(planId)}/unpublish`, {
    method: "POST",
    credentials: credentialsMode,
  }).then((res) => {
    if (!res.ok && res.status !== 204) {
      throw new Error(`Unpublish failed: ${res.status}`);
    }
  });
}

export async function getCurrent(): Promise<CurrentResponse> {
  return jsonRequest<CurrentResponse>("GET", "/api/v1/me/current");
}

// Public plans

export async function listPublicPlans(params: {
  q?: string;
  destination_city?: string;
  days?: number;
  interest?: string;
  sort?: "hot" | "latest";
  page?: number;
  page_size?: number;
}): Promise<PublicPlanListResponse> {
  const search = new URLSearchParams();
  if (params.q) search.set("q", params.q);
  if (params.destination_city) search.set("destination_city", params.destination_city);
  if (params.days) search.set("days", String(params.days));
  if (params.interest) search.set("interest", params.interest);
  if (params.sort) search.set("sort", params.sort);
  if (params.page) search.set("page", String(params.page));
  if (params.page_size) search.set("page_size", String(params.page_size));
  const qs = search.toString();
  return jsonRequest<PublicPlanListResponse>("GET", `/api/v1/public/plans${qs ? `?${qs}` : ""}`);
}

export async function getPublicPlan(publicPlanId: string): Promise<PublicPlan> {
  const existing = publicPlanDetailRequests.get(publicPlanId);
  if (existing) {
    return existing;
  }
  const request = jsonRequest<{ public_plan: PublicPlan }>(
    "GET",
    `/api/v1/public/plans/${encodeURIComponent(publicPlanId)}`,
  )
    .then((data) => data.public_plan)
    .finally(() => {
      publicPlanDetailRequests.delete(publicPlanId);
    });
  publicPlanDetailRequests.set(publicPlanId, request);
  return request;
}

export async function savePublicPlanCopy(publicPlanId: string): Promise<UserPlan> {
  const data = await jsonRequest<{ plan: UserPlan }>(
    "POST",
    `/api/v1/public/plans/${encodeURIComponent(publicPlanId)}/save`,
  );
  return data.plan;
}
