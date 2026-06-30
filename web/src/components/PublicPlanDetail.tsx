import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";
import { savePublicCopy, usePublicPlanDetail } from "../hooks/usePublicPlans";
import PlanDetail from "./PlanDetail";
import EmptyState from "./EmptyState";
import { useToast } from "./Toast";

export default function PublicPlanDetail() {
  const { publicPlanId } = useParams<{ publicPlanId: string }>();
  const { plan, loading, error } = usePublicPlanDetail(publicPlanId);
  const auth = useAuth();
  const navigate = useNavigate();
  const toast = useToast();
  const [saving, setSaving] = useState(false);

  if (loading) {
    return <p style={{ color: "var(--ta-slate)" }}>正在加载公开计划…</p>;
  }
  if (error || !plan) {
    return (
      <EmptyState
        title="计划不存在或已不可见"
        description={error || "作者可能已经下架，或者你没有权限。"}
        action={{ label: "返回公开列表", onClick: () => navigate("/public") }}
        testId="public-detail-error"
      />
    );
  }

  const handleSaveCopy = async () => {
    if (auth.status !== "authenticated") {
      navigate(`/login?return_to=${encodeURIComponent(`/public/${plan.public_plan_id}`)}`);
      return;
    }
    setSaving(true);
    try {
      const copied = await savePublicCopy(plan.public_plan_id);
      toast.show("已保存到我的计划");
      navigate(`/me/plans/${copied.plan_id}`);
    } catch (err) {
      toast.show(err instanceof Error ? err.message : "保存失败，请稍后重试");
    } finally {
      setSaving(false);
    }
  };

  return (
    <section className="ta-plan-page" data-testid="public-detail-page">
      <header className="ta-plan-head">
        <h1>{plan.title}</h1>
        <div className="ta-plan-meta">
          {plan.author.display_name || "匿名作者"} · {plan.destination_city} · {plan.days} 日 · 浏览 {plan.view_count} ·
          收藏 {plan.save_count}
        </div>
        {plan.summary ? <p style={{ margin: 0, color: "var(--ta-slate)" }}>{plan.summary}</p> : null}
        <div className="ta-tags">
          {plan.tags.map((tag) => (
            <span key={tag} className="ta-tag">
              {tag}
            </span>
          ))}
        </div>
        <div className="ta-plan-actions">
          <button
            type="button"
            className="ta-action-button"
            disabled={saving}
            onClick={handleSaveCopy}
            data-testid="public-save-copy"
          >
            {saving ? "保存中…" : "保存到我的计划"}
          </button>
        </div>
      </header>
      <PlanDetail plan={plan.plan} status="succeeded" />
    </section>
  );
}
