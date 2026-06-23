package agent

import (
	"context"

	"travel-agent/internal/domain"
)

// TravelPlanner is the only contract the harness needs from a route planner.
type TravelPlanner interface {
	Plan(ctx context.Context, req domain.TravelRequest) (*domain.TravelPlan, error)
}
