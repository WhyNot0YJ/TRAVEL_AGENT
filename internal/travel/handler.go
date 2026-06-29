package travel

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *TravelPlanService
}

func NewHandler(service *TravelPlanService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreatePlan(c *gin.Context) {
	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	requestID := c.Writer.Header().Get("X-Request-ID")
	ctx := WithRequestID(c.Request.Context(), requestID)
	resp, err := h.service.CreateTask(ctx, req, c.ClientIP())
	if errors.Is(err, ErrRateLimited) {
		respondError(c, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded")
		return
	}
	if errors.Is(err, ErrInvalidRequest) {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "task_error", err.Error())
		return
	}
	c.JSON(http.StatusAccepted, resp)
}

func (h *Handler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	resp, err := h.service.Chat(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "chat_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ChatStream(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	resp, err := h.service.ChatStream(c.Request.Context(), req, func(event TaskEvent) bool {
		c.SSEvent(string(event.Type), event)
		c.Writer.Flush()
		return c.Request.Context().Err() == nil
	})
	if err != nil {
		c.SSEvent(string(EventError), TaskEvent{
			Type:      EventError,
			RequestID: c.Writer.Header().Get("X-Request-ID"),
			Message:   err.Error(),
			CreatedAt: time.Now().UTC(),
		})
		c.Writer.Flush()
		return
	}
	c.SSEvent(string(EventDone), resp)
	c.Writer.Flush()
}

func (h *Handler) GetPlan(c *gin.Context) {
	id := c.Param("task_id")
	if id == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "task id is required")
		return
	}
	resp, err := h.service.GetTask(c.Request.Context(), id)
	if errors.Is(err, ErrTaskNotFound) {
		respondError(c, http.StatusNotFound, "not_found", "task not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) StreamPlan(c *gin.Context) {
	id := c.Param("task_id")
	if id == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "task id is required")
		return
	}

	current, err := h.service.GetTask(c.Request.Context(), id)
	if errors.Is(err, ErrTaskNotFound) {
		respondError(c, http.StatusNotFound, "not_found", "task not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	if current.Status == TaskSucceeded {
		if !h.replayHistory(c, current.TaskID) {
			c.SSEvent(string(EventDone), TaskEvent{
				Type:      EventDone,
				RequestID: c.Writer.Header().Get("X-Request-ID"),
				TaskID:    current.TaskID,
				Status:    current.Status,
				Message:   "task already finished",
				Plan:      current.Plan,
				CreatedAt: time.Now().UTC(),
			})
		}
		c.Writer.Flush()
		return
	}
	if current.Status == TaskFailed {
		if !h.replayHistory(c, current.TaskID) {
			c.SSEvent(string(EventError), TaskEvent{
				Type:      EventError,
				RequestID: c.Writer.Header().Get("X-Request-ID"),
				TaskID:    current.TaskID,
				Status:    current.Status,
				Message:   current.Error,
				CreatedAt: time.Now().UTC(),
			})
		}
		c.Writer.Flush()
		return
	}

	events, unsubscribe := h.service.Subscribe(id)
	defer unsubscribe()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	c.SSEvent(string(EventProgress), TaskEvent{
		Type:      EventProgress,
		RequestID: c.Writer.Header().Get("X-Request-ID"),
		TaskID:    current.TaskID,
		Status:    current.Status,
		Message:   "stream connected",
		CreatedAt: time.Now().UTC(),
	})
	c.Writer.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-heartbeat.C:
			c.SSEvent("heartbeat", gin.H{"task_id": id, "time": time.Now().UTC()})
			c.Writer.Flush()
		case event, ok := <-events:
			if !ok {
				return
			}
			c.SSEvent(string(event.Type), event)
			c.Writer.Flush()
			if event.Type == EventDone || event.Type == EventError {
				return
			}
		}
	}
}

func (h *Handler) replayHistory(c *gin.Context, taskID string) bool {
	history := h.service.EventHistory(taskID)
	if len(history) == 0 {
		return false
	}
	terminal := false
	for _, event := range history {
		c.SSEvent(string(event.Type), event)
		if event.Type == EventDone || event.Type == EventError {
			terminal = true
		}
	}
	return terminal
}

func respondError(c *gin.Context, status int, code, message string) {
	requestID := c.Writer.Header().Get("X-Request-ID")
	c.JSON(status, ErrorResponse{
		RequestID: requestID,
		Code:      code,
		Message:   message,
	})
}
