import { useEffect, useState } from "react";
import type { UserPlan } from "../api/types";

interface PlanEditorProps {
  plan: UserPlan;
  saving?: boolean;
  onSave: (payload: { title: string; note: string; summary: string; tags: string[] }) => Promise<void>;
  onCancel: () => void;
}

export default function PlanEditor({ plan, saving, onSave, onCancel }: PlanEditorProps) {
  const [title, setTitle] = useState(plan.title);
  const [note, setNote] = useState(plan.note ?? "");
  const [summary, setSummary] = useState(plan.summary ?? "");
  const [tagsInput, setTagsInput] = useState((plan.tags ?? []).join(", "));
  const [error, setError] = useState("");

  useEffect(() => {
    setTitle(plan.title);
    setNote(plan.note ?? "");
    setSummary(plan.summary ?? "");
    setTagsInput((plan.tags ?? []).join(", "));
    setError("");
  }, [plan.plan_id, plan.title, plan.note, plan.summary, plan.tags]);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (title.trim().length < 2 || title.trim().length > 80) {
      setError("标题长度需在 2-80 个字符之间");
      return;
    }
    const tags = tagsInput
      .split(/[,，]/)
      .map((tag) => tag.trim())
      .filter(Boolean);
    setError("");
    await onSave({ title: title.trim(), note: note.trim(), summary: summary.trim(), tags });
  };

  return (
    <form className="ta-form" onSubmit={submit} data-testid="plan-editor">
      <label>
        标题
        <input
          type="text"
          value={title}
          onChange={(event) => setTitle(event.target.value)}
          maxLength={80}
          required
          data-testid="editor-title"
        />
      </label>
      <label>
        摘要
        <input
          type="text"
          value={summary}
          onChange={(event) => setSummary(event.target.value)}
          data-testid="editor-summary"
        />
      </label>
      <label>
        标签 (用逗号分隔)
        <input
          type="text"
          value={tagsInput}
          onChange={(event) => setTagsInput(event.target.value)}
          data-testid="editor-tags"
        />
      </label>
      <label>
        私密备注
        <input
          type="text"
          value={note}
          onChange={(event) => setNote(event.target.value)}
          data-testid="editor-note"
        />
      </label>
      {error ? (
        <span className="ta-form-error" role="alert" data-testid="editor-error">
          {error}
        </span>
      ) : null}
      <div className="ta-modal-actions" style={{ marginTop: 6 }}>
        <button type="button" className="ta-action-button secondary" onClick={onCancel} data-testid="editor-cancel">
          取消
        </button>
        <button type="submit" className="ta-action-button" disabled={saving} data-testid="editor-save">
          {saving ? "保存中…" : "保存更改"}
        </button>
      </div>
    </form>
  );
}
