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
	EventBriefDelta     EventType = "brief_delta"
	EventPOIBatch       EventType = "poi_batch"
	EventWeatherDelta   EventType = "weather_delta"
	EventRouteDelta     EventType = "route_delta"
	EventBudgetDelta    EventType = "budget_delta"
	EventDayDelta       EventType = "day_delta"
	EventPlanDraft      EventType = "plan_draft"
)

type TaskEvent struct {
	Type       EventType             `json:"type"`
	RequestID  string                `json:"request_id,omitempty"`
	TaskID     string                `json:"task_id"`
	Status     TaskStatus            `json:"status,omitempty"`
	Message    string                `json:"message,omitempty"`
	Plan       *domain.TravelPlan    `json:"plan,omitempty"`
	Brief      *domain.TravelRequest `json:"brief,omitempty"`
	Day        *domain.TravelDay     `json:"day,omitempty"`
	POIs       []domain.POIInfo      `json:"pois,omitempty"`
	Weather    []domain.WeatherInfo  `json:"weather,omitempty"`
	Routes     []domain.RouteInfo    `json:"routes,omitempty"`
	Budget     *domain.TravelBudget  `json:"budget,omitempty"`
	NodeName   string                `json:"node_name,omitempty"`
	NodeStatus string                `json:"node_status,omitempty"`
	DurationMs int64                 `json:"duration_ms,omitempty"`
	Draft      bool                  `json:"draft,omitempty"`
	Sequence   int64                 `json:"sequence,omitempty"`
	CreatedAt  time.Time             `json:"created_at"`
}
