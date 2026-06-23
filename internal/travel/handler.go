package travel

import (
	"errors"
	"net/http"

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

func respondError(c *gin.Context, status int, code, message string) {
	requestID := c.Writer.Header().Get("X-Request-ID")
	c.JSON(status, ErrorResponse{
		RequestID: requestID,
		Code:      code,
		Message:   message,
	})
}
