package agent

import (
	"context"
	"testing"

	"travel-agent/internal/domain"
)

func TestMockPlannerPlan(t *testing.T) {
	planner := NewMockPlanner()
	req := domain.TravelRequest{
		ID:              "case_mock",
		DepartureCity:   "\u4e0a\u6d77",
		DestinationCity: "\u676d\u5dde",
		Days:            3,
		Budget:          3000,
		Interests:       []string{"\u81ea\u7136\u98ce\u5149", "\u7f8e\u98df"},
		TransportMode:   "train_taxi",
		Pace:            "relaxed",
	}

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
	for _, day := range plan.Days {
		if len(day.Items) < 2 {
			t.Fatalf("day %d has fewer than 2 items", day.Day)
		}
	}
}
