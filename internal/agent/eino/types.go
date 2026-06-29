package eino

import (
	"context"

	"travel-agent/internal/domain"
)

// TravelPlanningState is the internal state passed between Eino graph nodes.
type TravelPlanningState struct {
	Request               domain.TravelRequest  `json:"request"`
	NormalizedDestination string                `json:"normalized_destination"`
	NormalizedDays        int                   `json:"normalized_days"`
	NormalizedBudget      float64               `json:"normalized_budget"`
	Interests             []string              `json:"interests"`
	TransportMode         string                `json:"transport_mode"`
	Pace                  string                `json:"pace"`
	POIs                  []MockPOI             `json:"pois"`
	Weather               []MockWeather         `json:"weather"`
	Routes                []MockRoute           `json:"routes"`
	Budget                domain.TravelBudget   `json:"budget"`
	Itinerary             []domain.TravelDay    `json:"itinerary"`
	RouteValidation       RouteValidationResult `json:"route_validation"`
	Warnings              []string              `json:"warnings"`
	Trace                 []TraceEvent          `json:"trace"`
}

type POITool interface {
	Run(ctx context.Context, input POIToolInput) ([]MockPOI, error)
}

type WeatherTool interface {
	Run(ctx context.Context, input WeatherToolInput) ([]MockWeather, error)
}

type RouteTool interface {
	Run(ctx context.Context, input RouteToolInput) ([]MockRoute, error)
}

type BudgetTool interface {
	Run(ctx context.Context, input BudgetToolInput) (domain.TravelBudget, error)
}

// MockPOI is a deterministic POI returned by the local POI tool.
type MockPOI = domain.POIInfo

// MockWeather is deterministic weather returned by the local weather tool.
type MockWeather = domain.WeatherInfo

// MockRoute is a deterministic route segment returned by the local route tool.
type MockRoute = domain.RouteInfo

// TraceEvent records an internal graph step for future diagnostics.
type TraceEvent struct {
	Step       string `json:"step"`
	Message    string `json:"message"`
	DurationMs int64  `json:"duration_ms"`
	Success    bool   `json:"success"`
}

type RouteValidationResult struct {
	Score    int                    `json:"score"`
	Checks   []RouteValidationCheck `json:"checks"`
	Warnings []string               `json:"warnings"`
}

type RouteValidationCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// POIToolInput is the input for MockPOITool.
type POIToolInput struct {
	City      string   `json:"city"`
	Interests []string `json:"interests"`
}

// WeatherToolInput is the input for MockWeatherTool.
type WeatherToolInput struct {
	City string `json:"city"`
	Days int    `json:"days"`
}

// RouteToolInput is the input for MockRouteTool.
type RouteToolInput struct {
	POIs []MockPOI `json:"pois"`
	Mode string    `json:"mode"`
}

// BudgetToolInput is the input for MockBudgetTool.
type BudgetToolInput struct {
	Request   domain.TravelRequest `json:"request"`
	Days      int                  `json:"days"`
	POIs      []MockPOI            `json:"pois"`
	Routes    []MockRoute          `json:"routes"`
	Itinerary []domain.TravelDay   `json:"itinerary"`
}
