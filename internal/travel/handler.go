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
	resp, err := h.service.CreatePlan(c.Request.Context(), req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "planner_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetPlan(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "plan id is required")
		return
	}
	resp, err := h.service.GetPlan(c.Request.Context(), id)
	if errors.Is(err, ErrPlanNotFound) {
		respondError(c, http.StatusNotFound, "not_found", "plan not found")
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
