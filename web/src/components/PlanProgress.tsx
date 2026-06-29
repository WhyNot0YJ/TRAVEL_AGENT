import type { POIInfo, RouteInfo, TaskEvent, TravelBudget, TravelDay, WeatherInfo } from "../api/types";

interface PlanProgressProps {
  taskId: string | null;
  status: string;
  events: TaskEvent[];
  connected: boolean;
  polling: boolean;
  error: string;
  creating: boolean;
  pois?: POIInfo[];
  weather?: WeatherInfo[];
  routes?: RouteInfo[];
  budget?: TravelBudget;
  draftDays?: TravelDay[];
}

const stepLabels = ["提交需求", "连接进度", "生成路线", "完成"];

const nodeLabels: Record<string, string> = {
  ParseTravelRequestNode: "理解旅行需求",
  SearchPOIsToolNode: "筛选候选地点",
  GetWeatherToolNode: "查询天气影响",
  ComputeRouteToolNode: "估算路程时间",
  EstimateBudgetToolNode: "拆分预算",
  OptimizeItineraryNode: "优化每日顺序",
  ValidateRouteFeasibilityNode: "校验路线可行性",
  GenerateTravelPlanNode: "生成行程文本",
  ValidatePlanNode: "校验最终结果",
};

const statusLabels: Record<string, string> = {
  pending: "等待中",
  running: "进行中",
  success: "已完成",
  succeeded: "已完成",
  failed: "失败",
  error: "失败",
  creating: "提交中",
  empty: "未开始",
};

const progressMessages: Record<string, string> = {
  "stream connected": "实时连接已建立",
  "planner started": "规划任务已开始",
  "task finished": "规划任务已完成",
};

function eventTitle(event: TaskEvent): string {
  if (event.type === "node") {
    return event.node_name ? (nodeLabels[event.node_name] ?? "规划步骤") : "规划步骤";
  }
  if (event.type === "assistant_delta" || event.type === "assistant_done") {
    return "助手输出";
  }
  if (event.type === "brief_delta") {
    return "需求确认";
  }
  if (event.type === "poi_batch") {
    return "候选地点";
  }
  if (event.type === "weather_delta") {
    return "天气";
  }
  if (event.type === "route_delta") {
    return "路线";
  }
  if (event.type === "budget_delta") {
    return "预算";
  }
  if (event.type === "day_delta") {
    return "路线草稿";
  }
  if (event.type === "progress") {
    return "进度";
  }
  if (event.type === "warning") {
    return "提醒";
  }
  if (event.type === "error") {
    return "错误";
  }
  if (event.type === "done") {
    return "完成";
  }
  return "事件";
}

function eventMessage(event: TaskEvent): string {
  if (event.type === "node") {
    const duration = typeof event.duration_ms === "number" ? ` · ${event.duration_ms}ms` : "";
    const status = event.node_status ? (statusLabels[event.node_status] ?? event.node_status) : "进行中";
    return `${status}${duration}`;
  }
  if (event.message && progressMessages[event.message]) {
    return progressMessages[event.message];
  }
  return event.message || (event.status ? (statusLabels[event.status] ?? event.status) : "收到事件");
}

function statusText(status: string): string {
  return statusLabels[status] ?? status;
}

function eventKey(event: TaskEvent, index: number): string {
  const stableParts = [
    event.type,
    event.created_at ?? event.time ?? "",
    event.node_name ?? "",
    event.node_status ?? "",
    event.sequence ?? "",
    event.message ?? "",
    event.duration_ms ?? "",
    index,
  ];
  return stableParts.join("|");
}

function money(value: number): string {
  return `¥${Math.round(value).toLocaleString("zh-CN")}`;
}

function stageSummary({
  pois = [],
  weather = [],
  routes = [],
  budget,
  draftDays = [],
}: Pick<PlanProgressProps, "pois" | "weather" | "routes" | "budget" | "draftDays">) {
  return [
    { label: "地点", value: pois.length > 0 ? `${pois.length} 个` : "等待中" },
    { label: "天气", value: weather.length > 0 ? `${weather.length} 天` : "等待中" },
    { label: "路线", value: routes.length > 0 ? `${routes.length} 段` : "等待中" },
    { label: "预算", value: budget ? money(budget.known_total ?? budget.total) : "等待中" },
    { label: "草稿", value: draftDays.length > 0 ? `${draftDays.length} 天` : "等待中" },
  ];
}

export default function PlanProgress({
  taskId,
  status,
  events,
  connected,
  polling,
  error,
  creating,
  pois,
  weather,
  routes,
  budget,
  draftDays,
}: PlanProgressProps) {
  if (!taskId && !creating) {
    return (
      <section className="state-panel" data-testid="progress-panel">
        <p className="muted">确认需求后，行程生成进度会在这里实时更新。</p>
      </section>
    );
  }

  const activeIndex = status === "succeeded" ? 3 : connected || polling || status === "running" ? 2 : taskId ? 1 : 0;
  const rawStatusText = creating ? "creating" : status;
  const connectionText = creating ? "正在提交" : connected ? "SSE 已连接" : polling ? "轮询中" : "等待连接";

  return (
    <section className="progress-panel" data-testid="progress-panel">
      <div className="status-row">
        <span className={`status-pill ${rawStatusText}`}>{statusText(rawStatusText)}</span>
        <span className="connection-state">{connectionText}</span>
      </div>
      {error ? <p className="inline-error">{error}</p> : null}
      <div className="planning-steps" aria-label="规划进度">
        {stepLabels.map((label, index) => (
          <div className={index <= activeIndex ? "active" : ""} key={label}>
            <span>{index + 1}</span>
            <p>{label}</p>
          </div>
        ))}
      </div>
      <div className="stage-strip" aria-label="阶段数据">
        {stageSummary({ pois, weather, routes, budget, draftDays }).map((item) => (
          <div key={item.label}>
            <span>{item.label}</span>
            <strong>{item.value}</strong>
          </div>
        ))}
      </div>
      <div className="event-list">
        {events.length === 0 ? (
          <p className="muted">{creating ? "正在把需求交给规划服务。" : "任务已创建，等待第一条实时事件。"}</p>
        ) : (
          events.map((event, index) => (
            <div className="event-item" key={eventKey(event, index)}>
              <span>{eventTitle(event)}</span>
              <p>{eventMessage(event)}</p>
            </div>
          ))
        )}
      </div>
    </section>
  );
}
