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
	now         func() time.Time
}

func NewTravelPlanService(planner agent.TravelPlanner, store TaskStore, limiter RateLimiter) *TravelPlanService {
	if limiter == nil {
		limiter = NewMemoryRateLimiter(60)
	}
	return &TravelPlanService{
		planner:     planner,
		store:       store,
		rateLimiter: limiter,
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
		RequestHash: requestHash,
		Status:      TaskPending,
		Request:     domainReq,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.Create(ctx, task); err != nil {
		return CreateTaskResponse{}, err
	}

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

func (s *TravelPlanService) runTask(task Task) {
	ctx := context.Background()
	defer func() {
		if recovered := recover(); recovered != nil {
			task.Status = TaskFailed
			task.Error = fmt.Sprintf("panic: %v", recovered)
			task.UpdatedAt = s.now().UTC()
			_ = s.store.Update(ctx, task)
		}
	}()

	task.Status = TaskRunning
	task.UpdatedAt = s.now().UTC()
	if err := s.store.Update(ctx, task); err != nil {
		log.Printf("update task running failed task_id=%s: %v", task.ID, err)
		return
	}

	plan, err := s.planner.Plan(ctx, task.Request)
	task.UpdatedAt = s.now().UTC()
	if err != nil {
		task.Status = TaskFailed
		task.Error = err.Error()
	} else {
		task.Status = TaskSucceeded
		task.Plan = plan
		task.Error = ""
	}
	if err := s.store.Update(ctx, task); err != nil {
		log.Printf("update task finished failed task_id=%s: %v", task.ID, err)
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
