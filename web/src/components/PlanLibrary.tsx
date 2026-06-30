import { useState } from "react";
import { Link } from "react-router-dom";
import { deletePlan, publishPlan, unpublishPlan, usePlanLibrary } from "../hooks/usePlanLibrary";
import ConfirmDialog from "./ConfirmDialog";
import EmptyState from "./EmptyState";
import { useToast } from "./Toast";

export default function PlanLibrary() {
  const [query, setQuery] = useState("");
  const [visibility, setVisibility] = useState("");
  const { items, loading, error, reload } = usePlanLibrary({ query: query.trim() || undefined, visibility: visibility || undefined });
  const toast = useToast();

  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [unpublishingId, setUnpublishingId] = useState<string | null>(null);
  const [actionPlanId, setActionPlanId] = useState<string | null>(null);

  const onDelete = async () => {
    if (!deletingId) return;
    setActionPlanId(deletingId);
    try {
      await deletePlan(deletingId);
      toast.show("计划已删除");
      await reload();
    } catch (err) {
      toast.show(err instanceof Error ? err.message : "删除失败");
    } finally {
      setActionPlanId(null);
      setDeletingId(null);
    }
  };

  const onUnpublish = async () => {
    if (!unpublishingId) return;
    setActionPlanId(unpublishingId);
    try {
      await unpublishPlan(unpublishingId);
      toast.show("已取消发布");
      await reload();
    } catch (err) {
      toast.show(err instanceof Error ? err.message : "取消发布失败");
    } finally {
      setActionPlanId(null);
      setUnpublishingId(null);
    }
  };

  const onPublish = async (planId: string) => {
    setActionPlanId(planId);
    try {
      await publishPlan(planId);
      toast.show("计划已发布");
      await reload();
    } catch (err) {
      toast.show(err instanceof Error ? err.message : "发布失败");
    } finally {
      setActionPlanId(null);
    }
  };

  return (
    <section data-testid="plan-library">
      <div className="ta-library-toolbar">
        <input
          type="search"
          placeholder="搜索我的计划"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          aria-label="搜索我的计划"
          data-testid="library-search"
        />
        <select
          value={visibility}
          onChange={(event) => setVisibility(event.target.value)}
          aria-label="可见性筛选"
          data-testid="library-visibility"
        >
          <option value="">全部</option>
          <option value="private">私有</option>
          <option value="public">已公开</option>
        </select>
      </div>
      {error ? <p style={{ color: "var(--ta-sunset-coral)" }}>{error}</p> : null}
      {loading ? (
        <p style={{ color: "var(--ta-slate)" }}>正在加载…</p>
      ) : items.length === 0 ? (
        <EmptyState
          title="还没有保存的计划"
          description="生成完成后，点击保存会出现在这里。"
          testId="library-empty"
        />
      ) : (
        <div className="ta-library-list">
          {items.map((plan) => (
            <article key={plan.plan_id} className="ta-library-row" data-testid="library-row">
              <div>
                <strong>{plan.title}</strong>
                <span>
                  {plan.destination_city} · {plan.days} 日 · 更新于 {plan.updated_at.slice(0, 10)} ·{" "}
                  {plan.publish_status === "published" ? "已发布" : plan.publish_status === "unpublished" ? "已取消发布" : "私有"}
                </span>
              </div>
              <div className="ta-row-actions">
                <Link to={`/me/plans/${plan.plan_id}`} data-testid="library-view">
                  查看
                </Link>
                {plan.publish_status === "published" ? (
                  <button
                    type="button"
                    className="ta-row-published"
                    onClick={() => setUnpublishingId(plan.plan_id)}
                    disabled={actionPlanId === plan.plan_id}
                    data-testid="library-unpublish"
                  >
                    取消发布
                  </button>
                ) : (
                  <button
                    type="button"
                    className="ta-row-publish"
                    onClick={() => onPublish(plan.plan_id)}
                    disabled={actionPlanId === plan.plan_id}
                    data-testid="library-publish"
                  >
                    发布
                  </button>
                )}
                <button
                  type="button"
                  className="ta-row-delete"
                  onClick={() => setDeletingId(plan.plan_id)}
                  disabled={actionPlanId === plan.plan_id}
                  data-testid="library-delete"
                >
                  删除
                </button>
              </div>
            </article>
          ))}
        </div>
      )}

      <ConfirmDialog
        open={Boolean(deletingId)}
        title="确认删除"
        description="计划会从我的历史中移除。已发布的计划公开页面也会下架。"
        confirmLabel="删除"
        danger
        onCancel={() => setDeletingId(null)}
        onConfirm={onDelete}
      />
      <ConfirmDialog
        open={Boolean(unpublishingId)}
        title="取消发布？"
        description="公开页面不再展示该计划，但你仍然保留私有版本。"
        confirmLabel="取消发布"
        onCancel={() => setUnpublishingId(null)}
        onConfirm={onUnpublish}
      />
    </section>
  );
}
