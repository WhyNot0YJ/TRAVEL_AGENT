package travel

import (
	"context"
	"errors"
	"testing"

	"travel-agent/internal/domain"
)

func TestTravelPlanServiceCreateAndGet(t *testing.T) {
	service := NewTravelPlanService(stubPlanner{}, NewMemoryPlanStore())
	resp, err := service.CreatePlan(context.Background(), CreatePlanRequest{
		DepartureCity:   "上海",
		DestinationCity: "杭州",
		Days:            2,
		Budget:          1000,
	})
	if err != nil {
		t.Fatalf("CreatePlan returned error: %v", err)
	}
	if resp.PlanID == "" || resp.Plan == nil {
		t.Fatalf("unexpected create response: %#v", resp)
	}
	got, err := service.GetPlan(context.Background(), resp.PlanID)
	if err != nil {
		t.Fatalf("GetPlan returned error: %v", err)
	}
	if got.Plan.Title != resp.Plan.Title {
		t.Fatalf("unexpected stored plan: %#v", got.Plan)
	}
}

func TestTravelPlanServiceGetNotFound(t *testing.T) {
	service := NewTravelPlanService(stubPlanner{}, NewMemoryPlanStore())
	_, err := service.GetPlan(context.Background(), "missing")
	if !errors.Is(err, ErrPlanNotFound) {
		t.Fatalf("expected ErrPlanNotFound, got %v", err)
	}
}

type stubPlanner struct{}

func (stubPlanner) Plan(ctx context.Context, req domain.TravelRequest) (*domain.TravelPlan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &domain.TravelPlan{
		Title:   req.DestinationCity + "计划",
		Summary: req.DestinationCity + "摘要",
		Days: []domain.TravelDay{{
			Day:   1,
			Theme: "城市探索",
			Items: []domain.TravelItem{{
				Time:            "09:00",
				Type:            "sightseeing",
				Name:            "景点",
				Address:         "地址",
				Reason:          "理由",
				EstimatedCost:   10,
				DurationMinutes: 60,
			}},
		}},
		Budget: domain.TravelBudget{Total: 100},
	}, nil
}
