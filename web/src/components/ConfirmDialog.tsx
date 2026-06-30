import type { ReactNode } from "react";

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  description?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
  children?: ReactNode;
  onConfirm: () => void;
  onCancel: () => void;
}

export default function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel = "确认",
  cancelLabel = "取消",
  danger = false,
  children,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  if (!open) {
    return null;
  }
  return (
    <div className="ta-modal-backdrop" role="dialog" aria-modal="true" data-testid="confirm-dialog">
      <div className="ta-modal">
        <h2>{title}</h2>
        {description ? <p>{description}</p> : null}
        {children}
        <div className="ta-modal-actions">
          <button type="button" className="ta-action-button secondary" onClick={onCancel} data-testid="confirm-cancel">
            {cancelLabel}
          </button>
          <button
            type="button"
            className={`ta-action-button${danger ? " danger" : ""}`}
            onClick={onConfirm}
            data-testid="confirm-ok"
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
