package agent

import (
	"context"

	"travel-agent/internal/domain"
)

// TravelPlanner is the only contract the harness needs from a route planner.
type TravelPlanner interface {
	Plan(ctx context.Context, req domain.TravelRequest) (*domain.TravelPlan, error)
}

// TravelInfoExtractor extracts structured travel information from a user's free-text message,
// optionally merging with already-known fields.
type TravelInfoExtractor interface {
	Extract(ctx context.Context, message string, current domain.TravelRequest) (*TravelInfoResult, error)
}

// TravelInfoResult carries extracted fields and assistant reply after processing one message.
type TravelInfoResult struct {
	DepartureCity    string
	DestinationCity  string
	Days             int
	Budget           float64
	Interests        []string
	Travelers        int
	DateRange        string
	TransportMode    string
	Pace             string
	WalkingTolerance string
	HotelArea        string
	MustVisit        []string
	Avoid            []string
	TravelerType     string
	BudgetType       string
	BudgetIncludes   []string
	Reply            string
	Missing          []string
	IsComplete       bool
}
