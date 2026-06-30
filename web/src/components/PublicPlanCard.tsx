import { Link } from "react-router-dom";
import type { PublicPlan } from "../api/types";

interface PublicPlanCardProps {
  plan: PublicPlan;
  rank?: number;
}

function budgetSummary(plan: PublicPlan): { text: string; partial: boolean } {
  const budget = plan.plan?.budget;
  if (!budget) {
    return { text: "暂无预算", partial: true };
  }
  const known = budget.known_total ?? budget.total ?? 0;
  if (known <= 0) {
    return { text: "暂无预算", partial: true };
  }
  const partial = budget.complete === false;
  return { text: `已知预算 ¥${Math.round(known)}${partial ? " (部分项暂无)" : ""}`, partial };
}

export default function PublicPlanCard({ plan, rank }: PublicPlanCardProps) {
  const budget = budgetSummary(plan);
  return (
    <Link
      to={`/public/${encodeURIComponent(plan.public_plan_id)}`}
      className="ta-public-card"
      data-testid="public-plan-card"
      aria-label={`查看公开计划 ${plan.title}`}
    >
      <div className="ta-public-card-meta">
        {rank ? <span className="ta-public-card-rank">{rank}</span> : null}
        <span>{plan.destination_city}</span>
        <span>·</span>
        <span>{plan.days} 日</span>
        <span>·</span>
        <span>{plan.author.display_name || "匿名作者"}</span>
      </div>
      <h3>{plan.title}</h3>
      {plan.summary ? <p>{plan.summary}</p> : null}
      <div className="ta-tags">
        {plan.tags.slice(0, 4).map((tag) => (
          <span key={tag} className="ta-tag">
            {tag}
          </span>
        ))}
      </div>
      <div className="ta-public-card-footer">
        <span className={`ta-budget-pill${budget.partial ? " partial" : ""}`}>{budget.text}</span>
        <span>
          热度 {plan.hot_score} · 浏览 {plan.view_count} · 收藏 {plan.save_count}
        </span>
      </div>
    </Link>
  );
}
