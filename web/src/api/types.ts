export type TaskStatus = "pending" | "running" | "succeeded" | "failed";
export type StreamEventType =
  | "progress"
  | "node"
  | "warning"
  | "error"
  | "done"
  | "heartbeat"
  | "assistant_delta"
  | "assistant_done"
  | "brief_delta"
  | "poi_batch"
  | "weather_delta"
  | "route_delta"
  | "budget_delta"
  | "day_delta"
  | "plan_draft";
export type AgentMode = "quick" | "expert";

export interface TravelPlanRequest {
  departure_city: string;
  destination_city: string;
  days: number;
  budget: number;
  interests: string[];
  travelers: number;
  date_range: string;
  transport_mode: string;
  pace: string;
  walking_tolerance: string;
  hotel_area: string;
  must_visit: string[];
  avoid: string[];
  traveler_type: string;
  budget_type: string;
  budget_includes: string[];
  test_mode?: boolean;
  agent_mode?: AgentMode;
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
  brief?: TravelPlanRequest;
  day?: TravelDay;
  pois?: POIInfo[];
  weather?: WeatherInfo[];
  routes?: RouteInfo[];
  budget?: TravelBudget;
  node_name?: string;
  node_status?: string;
  duration_ms?: number;
  draft?: boolean;
  sequence?: number;
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
  cost?: CostInfo;
  duration_minutes: number;
  poi?: POIMetadata;
}

export interface TravelBudget {
  transport: number;
  food: number;
  hotel: number;
  ticket: number;
  total: number;
  known_total?: number;
  complete?: boolean;
  currency?: string;
  items?: BudgetLine[];
  missing?: string[];
}

export type CostStatus = "available" | "unavailable" | "not_applicable";

export interface CostInfo {
  amount: number | null;
  currency: string;
  unit: string;
  status: CostStatus;
  source?: string;
  display?: string;
  included: boolean;
}

export interface BudgetLine {
  key: string;
  label: string;
  amount: number | null;
  currency: string;
  status: CostStatus;
  source?: string;
  display?: string;
  included: boolean;
}

export interface POIPhoto {
  title?: string;
  url?: string;
}

export interface POIMetadata {
  provider?: string;
  id?: string;
  parent?: string;
  typecode?: string;
  biz_type?: string;
  tel?: string;
  business_area?: string;
  tag?: string;
  rating?: number;
  photos?: POIPhoto[];
  cost: CostInfo;
}

export interface POIInfo {
  name: string;
  city: string;
  category: string;
  address: string;
  location?: string;
  suggested_duration_minutes: number;
  estimated_cost: number;
  cost: CostInfo;
  metadata?: POIMetadata;
}

export interface WeatherInfo {
  day: number;
  condition: string;
  temperature: string;
  suggestion: string;
}

export interface RouteInfo {
  from: string;
  to: string;
  duration_minutes: number;
  distance_meters: number;
  mode: string;
  cost: CostInfo;
}

export interface ChatRequest {
  message: string;
  departure_city?: string;
  destination_city?: string;
  days?: number;
  budget?: number;
  interests?: string[];
  travelers?: number;
  date_range?: string;
  transport_mode?: string;
  pace?: string;
  walking_tolerance?: string;
  hotel_area?: string;
  must_visit?: string[];
  avoid?: string[];
  traveler_type?: string;
  budget_type?: string;
  budget_includes?: string[];
  test_mode?: boolean;
  agent_mode?: AgentMode;
}

export interface ChatResponse {
  departure_city: string;
  destination_city: string;
  days: number;
  budget: number;
  interests: string[];
  travelers: number;
  date_range: string;
  transport_mode: string;
  pace: string;
  walking_tolerance: string;
  hotel_area: string;
  must_visit: string[];
  avoid: string[];
  traveler_type: string;
  budget_type: string;
  budget_includes: string[];
  reply: string;
  missing: string[];
  is_complete: boolean;
  agent_mode: AgentMode;
}
