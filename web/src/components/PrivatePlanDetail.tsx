import { useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import { deletePlan, patchPlan, publishPlan, unpublishPlan, usePlanDetail } from "../hooks/usePlanLibrary";
import ConfirmDialog from "./ConfirmDialog";
import EmptyState from "./EmptyState";
import PlanDetail from "./PlanDetail";
import PlanEditor from "./PlanEditor";
import { useToast } from "./Toast";

export default function PrivatePlanDetail() {
  const { planId } = useParams<{ planId: string }>();
  const navigate = useNavigate();
  const toast = useToast();
  const { plan, loading, error, reload } = usePlanDetail(planId);
  const [editing, setEditing] = useState(false);
  const [savingEdit, setSavingEdit] = useState(false);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [confirmingUnpublish, setConfirmingUnpublish] = useState(false);
  const [actionBusy, setActionBusy] = useState(false);

  if (!planId) {
    return <Navigate to="/me" replace />;
  }
  if (loading && !plan) {
    return <p style={{ color: "var(--ta-slate)" }}>正在加载计划…</p>;
  }
  if (error && !plan) {
    return (
      <EmptyState
        title="计划不存在或已不可见"
        description={error}
        action={{ label: "返回我的计划", onClick: () => navigate("/me") }}
        testId="private-detail-error"
      />
    );
  }
  if (!plan) {
    return null;
  }

  const onSaveEdit = async (payload: { title: string; note: string; summary: string; tags: string[] }) => {
    setSavingEdit(true);
    try {
      await patchPlan(plan.plan_id, payload);
      toast.show("修改已保存");
      setEditing(false);
      await reload();
    } catch (err) {
      toast.show(err instanceof Error ? err.message : "保存失败");
    } finally {
      setSavingEdit(false);
    }
  };

  const onPublish = async () => {
    setActionBusy(true);
    try {
      await publishPlan(plan.plan_id);
      toast.show("计划已发布到首页");
      await reload();
    } catch (err) {
      toast.show(err instanceof Error ? err.message : "发布失败");
    } finally {
      setActionBusy(false);
    }
  };

  const onUnpublishConfirmed = async () => {
    setActionBusy(true);
    try {
      await unpublishPlan(plan.plan_id);
      toast.show("已取消发布");
      setConfirmingUnpublish(false);
      await reload();
    } catch (err) {
      toast.show(err instanceof Error ? err.message : "取消发布失败");
    } finally {
      setActionBusy(false);
    }
  };

  const onDeleteConfirmed = async () => {
    setActionBusy(true);
    try {
      await deletePlan(plan.plan_id);
      toast.show("计划已删除");
      navigate("/me");
    } catch (err) {
      toast.show(err instanceof Error ? err.message : "删除失败");
    } finally {
      setActionBusy(false);
      setConfirmingDelete(false);
    }
  };

  return (
    <section className="ta-plan-page" data-testid="private-detail-page">
      <header className="ta-plan-head">
        <h1 data-testid="private-detail-title">{plan.title}</h1>
        <div className="ta-plan-meta">
          {plan.destination_city} · {plan.days} 日 · 状态：
          {plan.publish_status === "published" ? "已发布" : plan.publish_status === "unpublished" ? "已取消发布" : "私有"} · 更新于{" "}
          {plan.updated_at.slice(0, 16).replace("T", " ")}
        </div>
        {plan.summary ? <p style={{ margin: 0, color: "var(--ta-slate)" }}>{plan.summary}</p> : null}
        <div className="ta-tags">
          {plan.tags.map((tag) => (
            <span key={tag} className="ta-tag">
              {tag}
            </span>
          ))}
        </div>
        {plan.note ? (
          <p style={{ margin: 0, color: "var(--ta-slate)", fontSize: 13 }} data-testid="private-detail-note">
            备注：{plan.note}
          </p>
        ) : null}
        <div className="ta-plan-actions">
          {!editing ? (
            <>
              <button type="button" className="ta-action-button secondary" onClick={() => setEditing(true)} data-testid="detail-edit">
                编辑信息
              </button>
              {plan.publish_status === "published" ? (
                <button
                  type="button"
                  className="ta-action-button secondary"
                  onClick={() => setConfirmingUnpublish(true)}
                  disabled={actionBusy}
                  data-testid="detail-unpublish"
                >
                  取消发布
                </button>
              ) : (
                <button
                  type="button"
                  className="ta-action-button"
                  onClick={onPublish}
                  disabled={actionBusy}
                  data-testid="detail-publish"
                >
                  发布到首页
                </button>
              )}
              <button
                type="button"
                className="ta-action-button danger"
                onClick={() => setConfirmingDelete(true)}
                disabled={actionBusy}
                data-testid="detail-delete"
              >
                删除
              </button>
            </>
          ) : null}
        </div>
        {editing ? (
          <div className="ta-card" style={{ marginTop: 12 }}>
            <PlanEditor plan={plan} saving={savingEdit} onSave={onSaveEdit} onCancel={() => setEditing(false)} />
          </div>
        ) : null}
      </header>

      <PlanDetail plan={plan.plan} status="succeeded" />

      <ConfirmDialog
        open={confirmingDelete}
        title="确认删除"
        description={
          plan.publish_status === "published"
            ? "计划会从我的历史中移除,公开页面也会下架。"
            : "计划会从我的历史中移除。"
        }
        confirmLabel="删除"
        danger
        onCancel={() => setConfirmingDelete(false)}
        onConfirm={onDeleteConfirmed}
      />
      <ConfirmDialog
        open={confirmingUnpublish}
        title="取消发布?"
        description="公开页面不再展示该计划,但你仍然保留私有版本。"
        confirmLabel="取消发布"
        onCancel={() => setConfirmingUnpublish(false)}
        onConfirm={onUnpublishConfirmed}
      />
    </section>
  );
}
