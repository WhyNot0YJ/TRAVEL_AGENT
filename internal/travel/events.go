package travel

import (
	"time"

	"travel-agent/internal/domain"
)

type EventType string

const (
	EventProgress EventType = "progress"
	EventWarning  EventType = "warning"
	EventError    EventType = "error"
	EventDone     EventType = "done"
)

type TaskEvent struct {
	Type      EventType          `json:"type"`
	TaskID    string             `json:"task_id"`
	Status    TaskStatus         `json:"status,omitempty"`
	Message   string             `json:"message,omitempty"`
	Plan      *domain.TravelPlan `json:"plan,omitempty"`
	CreatedAt time.Time          `json:"created_at"`
}
