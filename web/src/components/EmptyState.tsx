interface EmptyStateProps {
  title: string;
  description?: string;
  action?: { label: string; onClick: () => void };
  testId?: string;
}

export default function EmptyState({ title, description, action, testId }: EmptyStateProps) {
  return (
    <div className="ta-empty" data-testid={testId ?? "empty-state"}>
      <strong>{title}</strong>
      {description ? <p style={{ marginTop: 8 }}>{description}</p> : null}
      {action ? (
        <div style={{ marginTop: 14 }}>
          <button type="button" className="ta-action-button" onClick={action.onClick}>
            {action.label}
          </button>
        </div>
      ) : null}
    </div>
  );
}
