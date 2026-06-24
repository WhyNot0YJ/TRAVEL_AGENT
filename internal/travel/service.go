package travel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"travel-agent/internal/agent"
)

var ErrRateLimited = errors.New("rate limit exceeded")

type TravelPlanService struct {
	planner     agent.TravelPlanner
	store       TaskStore
	rateLimiter RateLimiter
	events      *EventBus
	now         func() time.Time
}

func NewTravelPlanService(planner agent.TravelPlanner, store TaskStore, limiter RateLimiter, buses ...*EventBus) *TravelPlanService {
	if limiter == nil {
		limiter = NewMemoryRateLimiter(60)
	}
	var bus *EventBus
	if len(buses) > 0 {
		bus = buses[0]
	}
	if bus == nil {
		bus = NewEventBus()
	}
	return &TravelPlanService{
		planner:     planner,
		store:       store,
		rateLimiter: limiter,
		events:      bus,
		now:         time.Now,
	}
}

func (s *TravelPlanService) CreateTask(ctx context.Context, req CreatePlanRequest, clientKey string) (CreateTaskResponse, error) {
	if s == nil || s.planner == nil || s.store == nil {
		return CreateTaskResponse{}, fmt.Errorf("travel plan service is not initialized")
	}
	allowed, err := s.rateLimiter.Allow(ctx, clientRateKey(clientKey))
	if err != nil {
		log.Printf("rate limiter failed, allowing request: %v", err)
	} else if !allowed {
		return CreateTaskResponse{}, ErrRateLimited
	}

	taskID := newTaskID()
	domainReq := req.ToDomain(taskID)
	requestHash, err := RequestHash(domainReq)
	if err != nil {
		return CreateTaskResponse{}, err
	}
	if existing, ok, err := s.store.FindByHash(ctx, requestHash); err == nil && ok {
		return CreateTaskResponse{
			TaskID:      existing.ID,
			RequestHash: existing.RequestHash,
			Status:      existing.Status,
			Cached:      existing.Status == TaskSucceeded,
		}, nil
	} else if err != nil {
		log.Printf("request hash lookup failed, creating new task: %v", err)
	}

	now := s.now().UTC()
	task := Task{
		ID:          taskID,
		RequestID:   RequestIDFromContext(ctx),
		RequestHash: requestHash,
		Status:      TaskPending,
		Request:     domainReq,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.Create(ctx, task); err != nil {
		return CreateTaskResponse{}, err
	}
	s.publish(TaskEvent{Type: EventProgress, RequestID: task.RequestID, TaskID: task.ID, Status: task.Status, Message: "task created", CreatedAt: now})

	go s.runTask(task)
	return CreateTaskResponse{
		TaskID:      task.ID,
		RequestHash: task.RequestHash,
		Status:      task.Status,
		Cached:      false,
	}, nil
}

func (s *TravelPlanService) GetTask(ctx context.Context, id string) (GetTaskResponse, error) {
	task, err := s.store.Get(ctx, id)
	if err != nil {
		return GetTaskResponse{}, err
	}
	return taskResponse(task), nil
}

func (s *TravelPlanService) Subscribe(taskID string) (<-chan TaskEvent, func()) {
	return s.events.Subscribe(taskID)
}

func (s *TravelPlanService) runTask(task Task) {
	ctx := WithRequestID(context.Background(), task.RequestID)
	ctx = agent.WithPlannerEventReporter(ctx, plannerEventReporter{service: s, taskID: task.ID, requestID: task.RequestID})
	defer func() {
		if recovered := recover(); recovered != nil {
			task.Status = TaskFailed
			task.Error = fmt.Sprintf("panic: %v", recovered)
			task.UpdatedAt = s.now().UTC()
			_ = s.store.Update(ctx, task)
			log.Printf("request_id=%s task_id=%s status=failed error=%q", task.RequestID, task.ID, task.Error)
			s.publish(TaskEvent{Type: EventError, RequestID: task.RequestID, TaskID: task.ID, Status: task.Status, Message: task.Error, CreatedAt: task.UpdatedAt})
		}
	}()

	task.Status = TaskRunning
	task.UpdatedAt = s.now().UTC()
	if err := s.store.Update(ctx, task); err != nil {
		log.Printf("update task running failed task_id=%s: %v", task.ID, err)
		return
	}
	log.Printf("request_id=%s task_id=%s status=running planner_started=true", task.RequestID, task.ID)
	s.publish(TaskEvent{Type: EventProgress, RequestID: task.RequestID, TaskID: task.ID, Status: task.Status, Message: "planner started", CreatedAt: task.UpdatedAt})

	plan, err := s.planner.Plan(ctx, task.Request)
	task.UpdatedAt = s.now().UTC()
	if err != nil {
		task.Status = TaskFailed
		task.Error = err.Error()
		log.Printf("request_id=%s task_id=%s status=failed error=%q", task.RequestID, task.ID, task.Error)
		s.publish(TaskEvent{Type: EventError, RequestID: task.RequestID, TaskID: task.ID, Status: task.Status, Message: task.Error, CreatedAt: task.UpdatedAt})
	} else {
		task.Status = TaskSucceeded
		task.Plan = plan
		task.Error = ""
		for _, warning := range plan.Warnings {
			s.publish(TaskEvent{Type: EventWarning, RequestID: task.RequestID, TaskID: task.ID, Status: task.Status, Message: warning, CreatedAt: task.UpdatedAt})
		}
		log.Printf("request_id=%s task_id=%s status=succeeded warning_count=%d", task.RequestID, task.ID, len(plan.Warnings))
		s.publish(TaskEvent{Type: EventDone, RequestID: task.RequestID, TaskID: task.ID, Status: task.Status, Message: "planner finished", Plan: plan, CreatedAt: task.UpdatedAt})
	}
	if err := s.store.Update(ctx, task); err != nil {
		log.Printf("update task finished failed task_id=%s: %v", task.ID, err)
	}
}

type plannerEventReporter struct {
	service   *TravelPlanService
	taskID    string
	requestID string
}

func (r plannerEventReporter) ReportPlannerEvent(ctx context.Context, event agent.PlannerTraceEvent) {
	if r.service == nil {
		return
	}
	durationMs := event.Duration.Milliseconds()
	log.Printf("request_id=%s task_id=%s node=%s duration_ms=%d status=%s message=%q",
		r.requestID,
		r.taskID,
		event.Name,
		durationMs,
		event.Status,
		event.FallbackReason,
	)
	r.service.publish(TaskEvent{
		Type:       EventNode,
		RequestID:  r.requestID,
		TaskID:     r.taskID,
		Status:     TaskRunning,
		Message:    event.FallbackReason,
		NodeName:   event.Name,
		NodeStatus: event.Status,
		DurationMs: durationMs,
		CreatedAt:  time.Now().UTC(),
	})
}

func (s *TravelPlanService) publish(event TaskEvent) {
	if s != nil && s.events != nil {
		s.events.Publish(event)
	}
}

func taskResponse(task Task) GetTaskResponse {
	return GetTaskResponse{
		TaskID:      task.ID,
		RequestHash: task.RequestHash,
		Status:      task.Status,
		Plan:        task.Plan,
		Error:       task.Error,
		CreatedAt:   task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   task.UpdatedAt.Format(time.RFC3339),
	}
}

func newTaskID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "task_" + hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}
