package eino

import "travel-agent/internal/domain"

// TravelPlanningState is the internal state passed between Eino graph nodes.
type TravelPlanningState struct {
	Request               domain.TravelRequest `json:"request"`
	NormalizedDestination string               `json:"normalized_destination"`
	NormalizedDays        int                  `json:"normalized_days"`
	NormalizedBudget      float64              `json:"normalized_budget"`
	Interests             []string             `json:"interests"`
	TransportMode         string               `json:"transport_mode"`
	Pace                  string               `json:"pace"`
	POIs                  []MockPOI            `json:"pois"`
	Weather               []MockWeather        `json:"weather"`
	Routes                []MockRoute          `json:"routes"`
	Budget                domain.TravelBudget  `json:"budget"`
	Itinerary             []domain.TravelDay   `json:"itinerary"`
	Warnings              []string             `json:"warnings"`
	Trace                 []TraceEvent         `json:"trace"`
}

// MockPOI is a deterministic POI returned by the local POI tool.
type MockPOI struct {
	Name                     string  `json:"name"`
	City                     string  `json:"city"`
	Category                 string  `json:"category"`
	Address                  string  `json:"address"`
	SuggestedDurationMinutes int     `json:"suggested_duration_minutes"`
	EstimatedCost            float64 `json:"estimated_cost"`
}

// MockWeather is deterministic weather returned by the local weather tool.
type MockWeather struct {
	Day         int    `json:"day"`
	Condition   string `json:"condition"`
	Temperature string `json:"temperature"`
	Suggestion  string `json:"suggestion"`
}

// MockRoute is a deterministic route segment returned by the local route tool.
type MockRoute struct {
	From            string `json:"from"`
	To              string `json:"to"`
	DurationMinutes int    `json:"duration_minutes"`
	DistanceMeters  int    `json:"distance_meters"`
	Mode            string `json:"mode"`
}

// TraceEvent records an internal graph step for future diagnostics.
type TraceEvent struct {
	Step       string `json:"step"`
	Message    string `json:"message"`
	DurationMs int64  `json:"duration_ms"`
	Success    bool   `json:"success"`
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
	Request domain.TravelRequest `json:"request"`
	Days    int                  `json:"days"`
	POIs    []MockPOI            `json:"pois"`
	Routes  []MockRoute          `json:"routes"`
}
