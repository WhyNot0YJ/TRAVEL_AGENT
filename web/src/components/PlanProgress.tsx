import type { TaskEvent } from "../api/types";

interface PlanProgressProps {
  taskId: string | null;
  status: string;
  events: TaskEvent[];
  connected: boolean;
  polling: boolean;
  error: string;
  creating: boolean;
}

const stepLabels = ["提交需求", "连接进度", "生成路线", "完成"];

function eventTitle(event: TaskEvent): string {
  if (event.type === "node") {
    return event.node_name || "planner node";
  }
  if (event.type === "assistant_delta" || event.type === "assistant_done") {
    return "助手输出";
  }
  return event.type;
}

function eventMessage(event: TaskEvent): string {
  if (event.type === "node") {
    const duration = typeof event.duration_ms === "number" ? ` · ${event.duration_ms}ms` : "";
    return `${event.node_status || "running"}${duration}`;
  }
  return event.message || event.status || "收到事件";
}

export default function PlanProgress({ taskId, status, events, connected, polling, error, creating }: PlanProgressProps) {
  if (!taskId && !creating) {
    return (
      <section className="state-panel" data-testid="progress-panel">
        <p className="muted">确认需求后，行程生成进度会在这里实时更新。</p>
      </section>
    );
  }

  const activeIndex = status === "succeeded" ? 3 : connected || polling || status === "running" ? 2 : taskId ? 1 : 0;
  const statusText = creating ? "creating" : status;
  const connectionText = creating ? "正在提交" : connected ? "SSE 已连接" : polling ? "轮询中" : "等待连接";

  return (
    <section className="progress-panel" data-testid="progress-panel">
      <div className="status-row">
        <span className={`status-pill ${statusText}`}>{statusText}</span>
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
      <div className="event-list">
        {events.length === 0 ? (
          <p className="muted">{creating ? "正在把需求交给规划服务。" : "任务已创建，等待第一条实时事件。"}</p>
        ) : (
          events.map((event, index) => (
            <div className="event-item" key={`${event.type}-${event.created_at ?? index}`}>
              <span>{eventTitle(event)}</span>
              <p>{eventMessage(event)}</p>
            </div>
          ))
        )}
      </div>
    </section>
  );
}
