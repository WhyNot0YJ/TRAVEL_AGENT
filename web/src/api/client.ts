import type { CreateTaskResponse, ErrorResponse, TaskResponse, TravelPlanRequest } from "./types";

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
