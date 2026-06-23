package eino

import (
	"context"
	"strings"
	"testing"

	"travel-agent/internal/domain"
	"travel-agent/internal/harness"
)

func TestEinoTravelPlannerPlanPassesEvaluator(t *testing.T) {
	planner, err := NewEinoTravelPlanner()
	if err != nil {
		t.Fatalf("NewEinoTravelPlanner returned error: %v", err)
	}

	req := testRequest("\u676d\u5dde", 3, 3000)
	plan, err := planner.Plan(context.Background(), req)
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if plan == nil {
		t.Fatal("Plan returned nil")
	}
	if len(plan.Days) != req.Days {
		t.Fatalf("expected %d days, got %d", req.Days, len(plan.Days))
	}
	if plan.Budget.Total > req.Budget*1.1 {
		t.Fatalf("budget total %.2f exceeds limit", plan.Budget.Total)
	}

	result := harness.NewEvaluator().Evaluate(harness.TravelCase{
		ID:    req.ID,
		Input: req,
		Expectation: harness.TravelExpectation{
			MinDays:          req.Days,
			MaxBudgetRatio:   1.1,
			RequiredKeywords: []string{req.DestinationCity},
		},
	}, plan, nil, 1)
	if !result.Success {
		t.Fatalf("expected evaluator success, got errors: %v", result.Errors)
	}
	if result.Score != 100 {
		t.Fatalf("expected score 100, got %.2f", result.Score)
	}
}

func TestEinoTravelPlannerUnknownCityFallback(t *testing.T) {
	planner, err := NewEinoTravelPlanner()
	if err != nil {
		t.Fatalf("NewEinoTravelPlanner returned error: %v", err)
	}

	req := testRequest("\u672a\u77e5\u57ce\u5e02", 2, 1000)
	plan, err := planner.Plan(context.Background(), req)
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if len(plan.Days) != req.Days {
		t.Fatalf("expected %d days, got %d", req.Days, len(plan.Days))
	}
	foundFallback := false
	for _, day := range plan.Days {
		for _, item := range day.Items {
			if strings.Contains(item.Name, "\u57ce\u5e02") || strings.Contains(item.Name, "\u7279\u8272") {
				foundFallback = true
			}
		}
	}
	if !foundFallback {
		t.Fatalf("expected unknown city fallback POIs, got plan: %#v", plan)
	}
}

func testRequest(destination string, days int, budget float64) domain.TravelRequest {
	return domain.TravelRequest{
		ID:              "case_eino",
		DepartureCity:   "\u4e0a\u6d77",
		DestinationCity: destination,
		Days:            days,
		Budget:          budget,
		Interests:       []string{"\u81ea\u7136\u98ce\u5149", "\u7f8e\u98df"},
		TransportMode:   "train_taxi",
		Pace:            "relaxed",
	}
}
