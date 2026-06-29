package eino

import (
	"strings"
	"testing"

	"travel-agent/internal/domain"
)

func TestValidateRouteFeasibilityDetectsObviousIssues(t *testing.T) {
	state := TravelPlanningState{
		Pace: "relaxed",
		POIs: []MockPOI{
			{Name: "西湖"},
			{Name: "西湖"},
		},
		Weather: []MockWeather{{Day: 1, Condition: "rainy"}},
		Routes:  []MockRoute{{From: "西湖", To: "西湖", DurationMinutes: 140}},
		Budget:  domain.TravelBudget{Transport: 10, Food: 10, Hotel: 10, Ticket: 10, Total: 1000, KnownTotal: 1000, Currency: "CNY"},
		Itinerary: []domain.TravelDay{
			{
				Day:   1,
				Theme: "overloaded rainy day",
				Items: []domain.TravelItem{
					{Name: "西湖", Type: "nature", Reason: "outdoor", DurationMinutes: 120},
					{Name: "西湖", Type: "nature", Reason: "outdoor", DurationMinutes: 120},
					{Name: "湖滨", Type: "citywalk", Reason: "outdoor", DurationMinutes: 120},
				},
			},
		},
	}

	result := validateRouteFeasibility(state)
	if result.Score >= 100 {
		t.Fatalf("expected score penalty, got %#v", result)
	}
	for _, expected := range []string{"daily_pace", "route_duration", "poi_coordinates", "weather_backup", "budget_breakdown", "duplicate_poi"} {
		if !containsRouteWarning(result.Warnings, expected) {
			t.Fatalf("expected warning for %s, got %#v", expected, result.Warnings)
		}
	}
}

func TestValidateRouteFeasibilityPassesBalancedMockShapeWithCoordinates(t *testing.T) {
	state := TravelPlanningState{
		Pace: "balanced",
		POIs: []MockPOI{
			{Name: "西湖", Location: "120.1,30.2"},
			{Name: "灵隐寺", Location: "120.2,30.3"},
		},
		Weather: []MockWeather{{Day: 1, Condition: "sunny"}},
		Routes:  []MockRoute{{From: "西湖", To: "灵隐寺", DurationMinutes: 30}},
		Budget:  domain.TravelBudget{Transport: 200, Food: 300, Hotel: 400, Ticket: 100, Total: 1000, KnownTotal: 1000, Currency: "CNY"},
		Itinerary: []domain.TravelDay{
			{
				Day:   1,
				Theme: "balanced day",
				Items: []domain.TravelItem{
					{Name: "西湖", Type: "nature", Reason: "自然风光", DurationMinutes: 120},
					{Name: "灵隐寺", Type: "culture", Reason: "文化", DurationMinutes: 120},
				},
			},
		},
	}

	result := validateRouteFeasibility(state)
	if result.Score != 100 {
		t.Fatalf("expected perfect score, got %#v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", result.Warnings)
	}
}

func containsRouteWarning(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}
