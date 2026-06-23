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
	resp, err := h.service.CreateTask(c.Request.Context(), req, c.ClientIP())
	if errors.Is(err, ErrRateLimited) {
		respondError(c, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "task_error", err.Error())
		return
	}
	c.JSON(http.StatusAccepted, resp)
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

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	if current.Status == TaskSucceeded {
		c.SSEvent(string(EventDone), TaskEvent{
			Type:      EventDone,
			TaskID:    current.TaskID,
			Status:    current.Status,
			Message:   "task already finished",
			Plan:      current.Plan,
			CreatedAt: time.Now().UTC(),
		})
		c.Writer.Flush()
		return
	}
	if current.Status == TaskFailed {
		c.SSEvent(string(EventError), TaskEvent{
			Type:      EventError,
			TaskID:    current.TaskID,
			Status:    current.Status,
			Message:   current.Error,
			CreatedAt: time.Now().UTC(),
		})
		c.Writer.Flush()
		return
	}

	events, unsubscribe := h.service.Subscribe(id)
	defer unsubscribe()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	c.SSEvent(string(EventProgress), TaskEvent{
		Type:      EventProgress,
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

func respondError(c *gin.Context, status int, code, message string) {
	requestID := c.Writer.Header().Get("X-Request-ID")
	c.JSON(status, ErrorResponse{
		RequestID: requestID,
		Code:      code,
		Message:   message,
	})
}
