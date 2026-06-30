package travel

import (
	"time"

	"travel-agent/internal/domain"
)

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskSucceeded TaskStatus = "succeeded"
	TaskFailed    TaskStatus = "failed"
)

type Task struct {
	ID          string               `json:"task_id"`
	RequestID   string               `json:"request_id,omitempty"`
	UserID      string               `json:"user_id,omitempty"`
	RequestHash string               `json:"request_hash"`
	Status      TaskStatus           `json:"status"`
	Request     domain.TravelRequest `json:"request"`
	TestMode    bool                 `json:"test_mode,omitempty"`
	AgentMode   string               `json:"agent_mode,omitempty"`
	Plan        *domain.TravelPlan   `json:"plan,omitempty"`
	Error       string               `json:"error,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}
