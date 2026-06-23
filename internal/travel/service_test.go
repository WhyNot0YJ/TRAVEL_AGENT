package travel

import (
	"context"
	"errors"
	"testing"
	"time"

	"travel-agent/internal/domain"
)

func TestTravelPlanServiceCreateTaskAndGet(t *testing.T) {
	service := NewTravelPlanService(stubPlanner{}, NewMemoryTaskStore(), NewMemoryRateLimiter(60))
	resp, err := service.CreateTask(context.Background(), validCreateRequest(), "127.0.0.1")
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if resp.TaskID == "" || resp.RequestHash == "" || resp.Status != TaskPending {
		t.Fatalf("unexpected create response: %#v", resp)
	}

	got := waitForTask(t, service, resp.TaskID)
	if got.Status != TaskSucceeded || got.Plan == nil {
		t.Fatalf("expected succeeded task with plan, got %#v", got)
	}
}

func TestTravelPlanServiceReusesRequestHash(t *testing.T) {
	service := NewTravelPlanService(stubPlanner{}, NewMemoryTaskStore(), NewMemoryRateLimiter(60))
	first, err := service.CreateTask(context.Background(), validCreateRequest(), "127.0.0.1")
	if err != nil {
		t.Fatalf("first CreateTask returned error: %v", err)
	}
	second, err := service.CreateTask(context.Background(), validCreateRequest(), "127.0.0.1")
	if err != nil {
		t.Fatalf("second CreateTask returned error: %v", err)
	}
	if first.TaskID != second.TaskID {
		t.Fatalf("expected duplicate request to reuse task id, got %s and %s", first.TaskID, second.TaskID)
	}
}

func TestTravelPlanServiceRateLimit(t *testing.T) {
	service := NewTravelPlanService(stubPlanner{}, NewMemoryTaskStore(), NewMemoryRateLimiter(1))
	if _, err := service.CreateTask(context.Background(), validCreateRequest(), "127.0.0.1"); err != nil {
		t.Fatalf("first CreateTask returned error: %v", err)
	}
	req := validCreateRequest()
	req.DestinationCity = "南京"
	_, err := service.CreateTask(context.Background(), req, "127.0.0.1")
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestRequestHashStable(t *testing.T) {
	req := validCreateRequest().ToDomain("one")
	first, err := RequestHash(req)
	if err != nil {
		t.Fatalf("RequestHash returned error: %v", err)
	}
	req.ID = "two"
	second, err := RequestHash(req)
	if err != nil {
		t.Fatalf("RequestHash returned error: %v", err)
	}
	if first != second {
		t.Fatalf("request hash should ignore generated ids")
	}
}

func waitForTask(t *testing.T, service *TravelPlanService, id string) GetTaskResponse {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, err := service.GetTask(context.Background(), id)
		if err != nil {
			t.Fatalf("GetTask returned error: %v", err)
		}
		if got.Status == TaskSucceeded || got.Status == TaskFailed {
			return got
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("task %s did not finish", id)
	return GetTaskResponse{}
}

func validCreateRequest() CreatePlanRequest {
	return CreatePlanRequest{
		DepartureCity:   "上海",
		DestinationCity: "杭州",
		Days:            2,
		Budget:          1000,
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
