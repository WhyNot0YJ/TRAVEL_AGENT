package travel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"travel-agent/internal/agent"
	"travel-agent/internal/domain"
)

var ErrRateLimited = errors.New("rate limit exceeded")
var ErrInvalidRequest = errors.New("invalid request")

const (
	AgentModeQuick  = "quick"
	AgentModeExpert = "expert"
)

type TravelPlanService struct {
	planner     agent.TravelPlanner
	extractor   agent.TravelInfoExtractor
	store       TaskStore
	rateLimiter RateLimiter
	events      *EventBus
	now         func() time.Time
}

func NewTravelPlanService(planner agent.TravelPlanner, store TaskStore, limiter RateLimiter, extractor agent.TravelInfoExtractor, buses ...*EventBus) *TravelPlanService {
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
		extractor:   extractor,
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
	if err := validateCreatePlanRequest(req); err != nil {
		return CreateTaskResponse{}, err
	}
	allowed, err := s.rateLimiter.Allow(ctx, clientRateKey(clientKey))
	if err != nil {
		log.Printf("rate limiter failed, allowing request: %v", err)
	} else if !allowed {
		return CreateTaskResponse{}, ErrRateLimited
	}

	taskID := newTaskID()
	domainReq := req.ToDomain(taskID)
	agentMode := normalizeAgentMode(req.AgentMode)
	requestHash, err := RequestHashWithOptions(domainReq, req.TestMode, agentMode)
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
		TestMode:    req.TestMode,
		AgentMode:   agentMode,
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

func (s *TravelPlanService) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if s == nil || s.extractor == nil {
		return ChatResponse{}, fmt.Errorf("chat extractor is not configured")
	}
	agentMode := normalizeAgentMode(req.AgentMode)
	ctx = agent.WithPlannerOptions(ctx, agent.PlannerOptions{TestMode: req.TestMode, AgentMode: agentMode})
	current := domain.NormalizeTravelBrief(domain.TravelRequest{
		DepartureCity:    req.DepartureCity,
		DestinationCity:  req.DestinationCity,
		Days:             req.Days,
		Budget:           req.Budget,
		Interests:        req.Interests,
		Travelers:        req.Travelers,
		DateRange:        req.DateRange,
		TransportMode:    req.TransportMode,
		Pace:             req.Pace,
		WalkingTolerance: req.WalkingTolerance,
		HotelArea:        req.HotelArea,
		MustVisit:        req.MustVisit,
		Avoid:            req.Avoid,
		TravelerType:     req.TravelerType,
		BudgetType:       req.BudgetType,
		BudgetIncludes:   req.BudgetIncludes,
	})
	result, err := s.extractor.Extract(ctx, req.Message, current)
	if err != nil {
		return ChatResponse{}, err
	}
	return ChatResponse{
		DepartureCity:    result.DepartureCity,
		DestinationCity:  result.DestinationCity,
		Days:             result.Days,
		Budget:           result.Budget,
		Interests:        result.Interests,
		Travelers:        result.Travelers,
		DateRange:        result.DateRange,
		TransportMode:    result.TransportMode,
		Pace:             result.Pace,
		WalkingTolerance: result.WalkingTolerance,
		HotelArea:        result.HotelArea,
		MustVisit:        result.MustVisit,
		Avoid:            result.Avoid,
		TravelerType:     result.TravelerType,
		BudgetType:       result.BudgetType,
		BudgetIncludes:   result.BudgetIncludes,
		Reply:            result.Reply,
		Missing:          result.Missing,
		IsComplete:       result.IsComplete,
		AgentMode:        agentMode,
	}, nil
}

func (s *TravelPlanService) ChatStream(ctx context.Context, req ChatRequest, emit func(TaskEvent) bool) (ChatResponse, error) {
	if emit == nil {
		resp, err := s.Chat(ctx, req)
		return resp, err
	}
	reporter := newChatDeltaReporter(s, RequestIDFromContext(ctx), emit)
	ctx = agent.WithLLMDeltaReporter(ctx, reporter)
	resp, err := s.Chat(ctx, req)
	if err != nil {
		return ChatResponse{}, err
	}
	now := s.now().UTC()
	if !reporter.SawAnyDelta() {
		// No streaming happened — emit the assembled reply via the legacy
		// chunkText fallback so the user still sees a typewriter effect.
		// Deprecated: kept for the LLM-disabled / stream-disabled path.
		for _, chunk := range chunkText(resp.Reply, 18) {
			if !emit(TaskEvent{Type: EventAssistantDelta, Message: chunk, CreatedAt: now}) {
				return resp, ctx.Err()
			}
		}
	}
	emit(TaskEvent{Type: EventAssistantDone, Message: resp.Reply, CreatedAt: s.now().UTC()})
	return resp, nil
}

func (s *TravelPlanService) Subscribe(taskID string) (<-chan TaskEvent, func()) {
	return s.events.Subscribe(taskID)
}

func (s *TravelPlanService) EventHistory(taskID string) []TaskEvent {
	if s == nil || s.events == nil {
		return nil
	}
	return s.events.History(taskID)
}

func (s *TravelPlanService) runTask(task Task) {
	ctx := WithRequestID(context.Background(), task.RequestID)
	ctx = agent.WithPlannerOptions(ctx, agent.PlannerOptions{TestMode: task.TestMode, AgentMode: normalizeAgentMode(task.AgentMode)})
	ctx = agent.WithPlannerEventReporter(ctx, plannerEventReporter{service: s, taskID: task.ID, requestID: task.RequestID})
	ctx = agent.WithPlannerBusinessEventReporter(ctx, plannerBusinessEventReporter{service: s, taskID: task.ID, requestID: task.RequestID})
	plannerDelta := newPlannerDeltaReporter(s, task.RequestID, task.ID)
	ctx = agent.WithLLMDeltaReporter(ctx, plannerDelta)
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
		s.publishPlanSummary(task, plan, plannerDelta)
		s.publish(TaskEvent{Type: EventDone, RequestID: task.RequestID, TaskID: task.ID, Status: task.Status, Message: "planner finished", Plan: plan, CreatedAt: task.UpdatedAt})
	}
	if err := s.store.Update(ctx, task); err != nil {
		log.Printf("update task finished failed task_id=%s: %v", task.ID, err)
	}
}

func (s *TravelPlanService) publishPlanSummary(task Task, plan *domain.TravelPlan, reporter *plannerDeltaReporter) {
	if plan == nil {
		return
	}
	text := strings.TrimSpace(plan.Summary)
	if text == "" {
		text = strings.TrimSpace(plan.Title)
	}
	if text == "" {
		return
	}
	// If link B narration already streamed text via the LLMDeltaReporter, skip
	// the legacy 24-char chunkText slicing so we don't double-emit. The narration
	// already produced organic-looking deltas; here we only finalize.
	if reporter == nil || !reporter.SawAnyDelta() {
		// Deprecated: 仅用于 LLM 流式不可用时的兜底。
		for _, chunk := range chunkText(text, 24) {
			s.publish(TaskEvent{
				Type:      EventAssistantDelta,
				RequestID: task.RequestID,
				TaskID:    task.ID,
				Status:    TaskRunning,
				Message:   chunk,
				CreatedAt: s.now().UTC(),
			})
		}
	}
	s.publish(TaskEvent{
		Type:      EventAssistantDone,
		RequestID: task.RequestID,
		TaskID:    task.ID,
		Status:    TaskSucceeded,
		Message:   text,
		CreatedAt: s.now().UTC(),
	})
}

type plannerEventReporter struct {
	service   *TravelPlanService
	taskID    string
	requestID string
}

type plannerBusinessEventReporter struct {
	service   *TravelPlanService
	taskID    string
	requestID string
}

// chatDeltaReporter forwards LLM token deltas from the chat info link onto the
// SSE writer. It scopes to a single in-flight ChatStream call, so it carries an
// emit closure rather than going through the EventBus.
type chatDeltaReporter struct {
	service   *TravelPlanService
	requestID string
	emit      func(TaskEvent) bool
	saw       atomic.Bool
}

func newChatDeltaReporter(s *TravelPlanService, requestID string, emit func(TaskEvent) bool) *chatDeltaReporter {
	return &chatDeltaReporter{service: s, requestID: requestID, emit: emit}
}

func (r *chatDeltaReporter) ReportLLMDelta(_ context.Context, delta string) {
	if r == nil || delta == "" {
		return
	}
	r.saw.Store(true)
	if r.emit == nil {
		return
	}
	r.emit(TaskEvent{
		Type:      EventAssistantDelta,
		RequestID: r.requestID,
		Message:   delta,
		CreatedAt: r.now(),
	})
}

func (r *chatDeltaReporter) ReportLLMDone(_ context.Context, _ string) {
	// Done is emitted by ChatStream after the structured fields are returned;
	// nothing extra to do here.
}

func (r *chatDeltaReporter) SawAnyDelta() bool {
	if r == nil {
		return false
	}
	return r.saw.Load()
}

func (r *chatDeltaReporter) now() time.Time {
	if r == nil || r.service == nil || r.service.now == nil {
		return time.Now().UTC()
	}
	return r.service.now().UTC()
}

// plannerDeltaReporter forwards narration deltas from the plan-generation link
// onto the EventBus so all subscribers of a task see the same SSE feed.
type plannerDeltaReporter struct {
	service   *TravelPlanService
	requestID string
	taskID    string
	saw       atomic.Bool
}

func newPlannerDeltaReporter(s *TravelPlanService, requestID, taskID string) *plannerDeltaReporter {
	return &plannerDeltaReporter{service: s, requestID: requestID, taskID: taskID}
}

func (r *plannerDeltaReporter) ReportLLMDelta(_ context.Context, delta string) {
	if r == nil || r.service == nil || delta == "" {
		return
	}
	r.saw.Store(true)
	r.service.publish(TaskEvent{
		Type:      EventAssistantDelta,
		RequestID: r.requestID,
		TaskID:    r.taskID,
		Status:    TaskRunning,
		Message:   delta,
		CreatedAt: time.Now().UTC(),
	})
}

func (r *plannerDeltaReporter) ReportLLMDone(_ context.Context, _ string) {
	// Plan-link "done" is signaled by the EventDone event after structured plan
	// is persisted; nothing extra to do here.
}

func (r *plannerDeltaReporter) SawAnyDelta() bool {
	if r == nil {
		return false
	}
	return r.saw.Load()
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

func (r plannerBusinessEventReporter) ReportPlannerBusinessEvent(_ context.Context, event agent.PlannerBusinessEvent) {
	if r.service == nil {
		return
	}
	eventType := EventType(event.Type)
	if eventType == "" {
		return
	}
	r.service.publish(TaskEvent{
		Type:      eventType,
		RequestID: r.requestID,
		TaskID:    r.taskID,
		Status:    TaskRunning,
		Message:   event.Message,
		Brief:     event.Brief,
		Day:       event.Day,
		POIs:      event.POIs,
		Weather:   event.Weather,
		Routes:    event.Routes,
		Budget:    event.Budget,
		NodeName:  event.NodeName,
		Draft:     event.Draft,
		CreatedAt: time.Now().UTC(),
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

func normalizeAgentMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AgentModeExpert:
		return AgentModeExpert
	default:
		return AgentModeQuick
	}
}

func validateCreatePlanRequest(req CreatePlanRequest) error {
	if strings.TrimSpace(req.DepartureCity) == "" {
		return fmt.Errorf("%w: departure_city is required", ErrInvalidRequest)
	}
	if strings.TrimSpace(req.DestinationCity) == "" {
		return fmt.Errorf("%w: destination_city is required", ErrInvalidRequest)
	}
	if req.Days <= 0 {
		return fmt.Errorf("%w: days must be positive", ErrInvalidRequest)
	}
	if req.Budget <= 0 {
		return fmt.Errorf("%w: budget must be positive", ErrInvalidRequest)
	}
	if len(req.Interests) == 0 {
		return fmt.Errorf("%w: interests is required", ErrInvalidRequest)
	}
	if req.Travelers <= 0 {
		return fmt.Errorf("%w: travelers is required", ErrInvalidRequest)
	}
	return nil
}

func chunkText(text string, size int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if size <= 0 {
		size = 16
	}
	runes := []rune(text)
	chunks := make([]string, 0, len(runes)/size+1)
	for start := 0; start < len(runes); start += size {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}
