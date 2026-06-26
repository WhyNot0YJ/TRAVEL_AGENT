package travel

import (
	"time"

	"travel-agent/internal/domain"
)

type EventType string

const (
	EventProgress       EventType = "progress"
	EventWarning        EventType = "warning"
	EventError          EventType = "error"
	EventDone           EventType = "done"
	EventNode           EventType = "node"
	EventAssistantDelta EventType = "assistant_delta"
	EventAssistantDone  EventType = "assistant_done"
)

type TaskEvent struct {
	Type       EventType          `json:"type"`
	RequestID  string             `json:"request_id,omitempty"`
	TaskID     string             `json:"task_id"`
	Status     TaskStatus         `json:"status,omitempty"`
	Message    string             `json:"message,omitempty"`
	Plan       *domain.TravelPlan `json:"plan,omitempty"`
	NodeName   string             `json:"node_name,omitempty"`
	NodeStatus string             `json:"node_status,omitempty"`
	DurationMs int64              `json:"duration_ms,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
}
