import type { TaskEvent } from "../api/types";

interface PlanProgressProps {
  taskId: string | null;
  status: string;
  events: TaskEvent[];
  connected: boolean;
  polling: boolean;
  error: string;
}

export default function PlanProgress({ taskId, status, events, connected, polling, error }: PlanProgressProps) {
  if (!taskId) {
    return (
      <section className="state-panel">
        <p className="muted">填写需求后提交，路线会在这里实时更新。</p>
      </section>
    );
  }

  return (
    <section className="progress-panel">
      <div className="status-row">
        <span className={`status-pill ${status}`}>{status}</span>
        <span className="connection-state">{connected ? "SSE 已连接" : polling ? "轮询中" : "等待连接"}</span>
      </div>
      {error ? <p className="inline-error">{error}</p> : null}
      <div className="event-list">
        {events.length === 0 ? (
          <p className="muted">任务已创建，等待后端执行。</p>
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
