import type { TravelPlan } from "../api/types";

interface PlanDetailProps {
  plan?: TravelPlan;
  status?: string;
}

function money(value: number): string {
  return `¥${Math.round(value).toLocaleString("zh-CN")}`;
}

export default function PlanDetail({ plan, status = "empty" }: PlanDetailProps) {
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

      <div className="budget-grid">
        <span>交通 {money(plan.budget.transport)}</span>
        <span>餐饮 {money(plan.budget.food)}</span>
        <span>住宿 {money(plan.budget.hotel)}</span>
        <span>门票 {money(plan.budget.ticket)}</span>
      </div>

      {plan.warnings.length > 0 ? (
        <div className="warnings">
          {plan.warnings.map((warning) => (
            <p key={warning}>{warning}</p>
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
              {day.items.map((item) => (
                <div className="timeline-item" key={`${day.day}-${item.time}-${item.name}`}>
                  <time>{item.time}</time>
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
          </details>
        ))}
      </div>
    </section>
  );
}
