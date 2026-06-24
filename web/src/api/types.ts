export type TaskStatus = "pending" | "running" | "succeeded" | "failed";
export type StreamEventType = "progress" | "node" | "warning" | "error" | "done" | "heartbeat";

export interface TravelPlanRequest {
  departure_city: string;
  destination_city: string;
  days: number;
  budget: number;
  interests: string[];
  transport_mode: string;
  pace: string;
}

export interface CreateTaskResponse {
  task_id: string;
  request_hash: string;
  status: TaskStatus;
  cached: boolean;
}

export interface TaskResponse {
  task_id: string;
  request_hash: string;
  status: TaskStatus;
  plan?: TravelPlan;
  error?: string;
  created_at: string;
  updated_at: string;
}

export interface ErrorResponse {
  request_id: string;
  code: string;
  message: string;
}

export interface TaskEvent {
  type: StreamEventType;
  request_id?: string;
  task_id?: string;
  status?: TaskStatus;
  message?: string;
  plan?: TravelPlan;
  node_name?: string;
  node_status?: string;
  duration_ms?: number;
  created_at?: string;
  time?: string;
}

export interface TravelPlan {
  title: string;
  summary: string;
  days: TravelDay[];
  budget: TravelBudget;
  warnings: string[];
}

export interface TravelDay {
  day: number;
  theme: string;
  items: TravelItem[];
}

export interface TravelItem {
  time: string;
  type: string;
  name: string;
  address: string;
  reason: string;
  estimated_cost: number;
  duration_minutes: number;
}

export interface TravelBudget {
  transport: number;
  food: number;
  hotel: number;
  ticket: number;
  total: number;
}
