import type { TravelPlan } from "../api/types";

interface PlanDetailProps {
  plan?: TravelPlan;
  status?: string;
  onRefine?: (text: string) => void;
}

function money(value: number): string {
  return `¥${Math.round(value).toLocaleString("zh-CN")}`;
}

function budgetItems(plan: TravelPlan) {
  return [
    { label: "交通", value: plan.budget.transport },
    { label: "餐饮", value: plan.budget.food },
    { label: "住宿", value: plan.budget.hotel },
    { label: "门票", value: plan.budget.ticket },
  ];
}

function warningLabel(warning: string): string {
  if (warning.startsWith("route feasibility:")) {
    return "路线可信度";
  }
  if (warning.startsWith("LLM fallback:") || warning.startsWith("LLM trace:")) {
    return "生成链路";
  }
  if (warning.startsWith("tool fallback:")) {
    return "外部数据";
  }
  if (warning.includes("weather") || warning.includes("rainy")) {
    return "天气";
  }
  return "提醒";
}

function warningText(warning: string): string {
  return warning
    .replace("route feasibility:", "路线校验：")
    .replace("LLM fallback:", "生成降级：")
    .replace("LLM trace:", "生成记录：")
    .replace("tool fallback:", "外部数据降级：");
}

export default function PlanDetail({ plan, status = "empty", onRefine }: PlanDetailProps) {
  if (!plan) {
    const message =
      status === "running" || status === "pending"
        ? "路线正在生成，完成后会在这里展开每天安排、预算和提醒。"
        : "生成完成后会展示每天安排、预算和提醒。";
    return (
      <section className="empty-detail" data-testid="plan-empty">
        <h2>路线详情</h2>
        <p className="muted">{message}</p>
      </section>
    );
  }

  return (
    <section className="plan-detail has-plan" data-testid="plan-detail">
      <p className="answer-kicker">已生成路线</p>
      <div className="plan-header">
        <div>
          <h2>{plan.title}</h2>
          <p>{plan.summary}</p>
        </div>
        <strong>{money(plan.budget.total)}</strong>
      </div>

      <div className="refine-bar" aria-label="路线调整入口">
        <button type="button" onClick={() => onRefine?.("预算调低 15%，保持核心景点")}>
          调整预算
        </button>
        <button type="button" onClick={() => onRefine?.("节奏放轻松一点，每天少安排一个点")}>
          放慢节奏
        </button>
        <button type="button" onClick={() => onRefine?.("请重新生成第 1 天，增加更多室内备选")}>
          重做第 1 天
        </button>
      </div>

      <div className="budget-panel" aria-label="预算拆分">
        <p className="section-label">预算拆分</p>
        {budgetItems(plan).map((item) => {
          const pct = plan.budget.total > 0 ? Math.max(4, Math.round((item.value / plan.budget.total) * 100)) : 0;
          return (
            <div className="budget-row" key={item.label}>
              <span>{item.label}</span>
              <div className="budget-track" aria-hidden="true">
                <i style={{ width: `${pct}%` }} />
              </div>
              <strong>{money(item.value)}</strong>
            </div>
          );
        })}
      </div>

      {plan.warnings.length > 0 ? (
        <div className="warnings">
          {plan.warnings.map((warning) => (
            <article key={warning}>
              <span>{warningLabel(warning)}</span>
              <p>{warningText(warning)}</p>
            </article>
          ))}
        </div>
      ) : null}

      <div className="days">
        {plan.days.map((day, index) => (
          <details className="day-card" key={day.day} open={index === 0}>
            <summary>
              <span>Day {day.day}</span>
              <h3>{day.theme}</h3>
            </summary>
            <div className="timeline">
              {day.items.map((item, itemIndex) => (
                <div className="timeline-item" key={`${day.day}-${item.time}-${item.name}`}>
                  <div className="time-mark">
                    <time>{item.time}</time>
                    <span>{itemIndex + 1}</span>
                  </div>
                  <div>
                    <div className="item-title">
                      <strong>{item.name}</strong>
                      <span>{item.type}</span>
                    </div>
                    <p>{item.reason}</p>
                    <small>
                      {item.address} · {item.duration_minutes} 分钟 · {money(item.estimated_cost)}
                    </small>
                  </div>
                </div>
              ))}
            </div>
            <div className="day-actions">
              <button type="button" onClick={() => onRefine?.(`请重新生成第 ${day.day} 天，减少排队和回头路`)}>
                重做这一天
              </button>
            </div>
          </details>
        ))}
      </div>
    </section>
  );
}
