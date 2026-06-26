package travel

import (
	"context"
	"errors"
	"testing"
	"time"

	"travel-agent/internal/agent"
	"travel-agent/internal/domain"
)

func TestTravelPlanServiceCreateTaskAndGet(t *testing.T) {
	service := NewTravelPlanService(stubPlanner{}, NewMemoryTaskStore(), NewMemoryRateLimiter(60), nil)
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
	service := NewTravelPlanService(stubPlanner{}, NewMemoryTaskStore(), NewMemoryRateLimiter(60), nil)
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
	service := NewTravelPlanService(stubPlanner{}, NewMemoryTaskStore(), NewMemoryRateLimiter(1), nil)
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

func TestTravelPlanServicePublishesRequestIDAndNodeEvents(t *testing.T) {
	planner := &eventPlanner{entered: make(chan struct{}), release: make(chan struct{})}
	service := NewTravelPlanService(planner, NewMemoryTaskStore(), NewMemoryRateLimiter(60), nil)
	ctx := WithRequestID(context.Background(), "req-test")
	resp, err := service.CreateTask(ctx, validCreateRequest(), "127.0.0.1")
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	<-planner.entered

	events, unsubscribe := service.Subscribe(resp.TaskID)
	defer unsubscribe()
	close(planner.release)

	var sawNode, sawDone bool
	deadline := time.After(2 * time.Second)
	for !sawDone {
		select {
		case event := <-events:
			if event.RequestID != "req-test" {
				t.Fatalf("expected request id on event, got %#v", event)
			}
			if event.Type == EventNode {
				sawNode = true
				if event.NodeName != "UnitTestNode" || event.NodeStatus != "success" || event.DurationMs != 12 {
					t.Fatalf("unexpected node event: %#v", event)
				}
			}
			if event.Type == EventDone {
				sawDone = true
			}
		case <-deadline:
			t.Fatal("timed out waiting for node/done events")
		}
	}
	if !sawNode {
		t.Fatal("expected node event")
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

func TestRequestHashIncludesTestMode(t *testing.T) {
	req := validCreateRequest().ToDomain("task")
	testModeHash, err := RequestHashWithOptions(req, true)
	if err != nil {
		t.Fatalf("RequestHashWithOptions returned error: %v", err)
	}
	realModeHash, err := RequestHashWithOptions(req, false)
	if err != nil {
		t.Fatalf("RequestHashWithOptions returned error: %v", err)
	}
	if testModeHash == realModeHash {
		t.Fatal("test mode and real mode should use different request hashes")
	}
}

func TestRequestHashIncludesAgentMode(t *testing.T) {
	req := validCreateRequest().ToDomain("task")
	quickHash, err := RequestHashWithOptions(req, false, AgentModeQuick)
	if err != nil {
		t.Fatalf("RequestHashWithOptions returned error: %v", err)
	}
	expertHash, err := RequestHashWithOptions(req, false, AgentModeExpert)
	if err != nil {
		t.Fatalf("RequestHashWithOptions returned error: %v", err)
	}
	if quickHash == expertHash {
		t.Fatal("quick mode and expert mode should use different request hashes")
	}
}

func TestTravelPlanServiceChatStreamEmitsDeltas(t *testing.T) {
	service := NewTravelPlanService(stubPlanner{}, NewMemoryTaskStore(), NewMemoryRateLimiter(60), simpleExtractor{})
	var events []TaskEvent
	resp, err := service.ChatStream(context.Background(), ChatRequest{Message: "上海出发，杭州 2 天，预算 1000", TestMode: true}, func(event TaskEvent) bool {
		events = append(events, event)
		return true
	})
	if err != nil {
		t.Fatalf("ChatStream returned error: %v", err)
	}
	if resp.Reply == "" {
		t.Fatal("expected chat response reply")
	}
	var sawDelta, sawDone bool
	for _, event := range events {
		if event.Type == EventAssistantDelta {
			sawDelta = true
		}
		if event.Type == EventAssistantDone {
			sawDone = true
		}
	}
	if !sawDelta || !sawDone {
		t.Fatalf("expected assistant delta and done events, got %#v", events)
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

type eventPlanner struct {
	entered chan struct{}
	release chan struct{}
}

type simpleExtractor struct{}

func (simpleExtractor) Extract(ctx context.Context, message string, current domain.TravelRequest) (*agent.TravelInfoResult, error) {
	return &agent.TravelInfoResult{
		DepartureCity:   "上海",
		DestinationCity: "杭州",
		Days:            2,
		Budget:          1000,
		Interests:       []string{"美食"},
		Reply:           "信息已经整理好，可以生成行程。",
		IsComplete:      true,
	}, nil
}

func (p *eventPlanner) Plan(ctx context.Context, req domain.TravelRequest) (*domain.TravelPlan, error) {
	close(p.entered)
	<-p.release
	agent.ReportPlannerEvent(ctx, agent.PlannerTraceEvent{
		Name:           "UnitTestNode",
		Kind:           "node",
		Status:         "success",
		Duration:       12 * time.Millisecond,
		FallbackReason: "unit test node",
	})
	return stubPlanner{}.Plan(ctx, req)
}
