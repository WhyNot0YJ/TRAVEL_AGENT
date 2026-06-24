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

const stepLabels = ["创建任务", "连接进度", "生成路线", "完成"];

export default function PlanProgress({ taskId, status, events, connected, polling, error, creating }: PlanProgressProps) {
  if (!taskId && !creating) {
    return (
      <section className="state-panel" data-testid="progress-panel">
        <p className="muted">和助手确认需求后，路线生成进度会在这里实时更新。</p>
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
          <p className="muted">{creating ? "正在把需求交给旅行规划服务。" : "任务已创建，等待第一条实时事件。"}</p>
        ) : (
          events.map((event, index) => (
            <div className="event-item" key={`${event.type}-${event.created_at ?? index}`}>
              <span>{event.type}</span>
              <p>{event.message || event.status || "收到事件"}</p>
            </div>
          ))
        )}
      </div>
    </section>
  );
}
