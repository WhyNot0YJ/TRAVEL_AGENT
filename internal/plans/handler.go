package plans

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"travel-agent/internal/auth"
)

// SimpleCurrentTaskLookup is the wiring-layer callback that GET /me/current
// uses to surface the user's in-flight generation task. The adapter lives in
// the server package so plans does not depend on internal/travel.
type SimpleCurrentTaskLookup func(userID string) *RunningTask

type Handler struct {
	service     *Service
	currentTask SimpleCurrentTaskLookup
}

func NewHandler(service *Service, currentTask SimpleCurrentTaskLookup) *Handler {
	return &Handler{service: service, currentTask: currentTask}
}

// SaveBody is POST /me/plans body.
type SaveBody struct {
	TaskID string `json:"task_id" binding:"required"`
	Title  string `json:"title"`
	Note   string `json:"note"`
}

func (h *Handler) Save(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	var body SaveBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	plan, err := h.service.Save(c.Request.Context(), user.ID, SaveInput{TaskID: body.TaskID, Title: body.Title, Note: body.Note})
	if status, code, ok := mapDomainError(err); ok {
		respondError(c, status, code, err.Error())
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "save_failed", err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"plan": ToUserPlanDTO(plan, true)})
}

func (h *Handler) List(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	filter := ListFilter{
		Query:           strings.TrimSpace(c.Query("q")),
		Visibility:      strings.TrimSpace(c.Query("visibility")),
		PublishStatus:   strings.TrimSpace(c.Query("publish_status")),
		DestinationCity: strings.TrimSpace(c.Query("destination_city")),
		Page:            atoiOr(c.Query("page"), 1),
		PageSize:        atoiOr(c.Query("page_size"), defaultPageSize),
	}
	plans, total, err := h.service.List(c.Request.Context(), user.ID, filter)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	items := make([]UserPlanDTO, 0, len(plans))
	for _, plan := range plans {
		items = append(items, ToUserPlanDTO(plan, false))
	}
	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"page":      filter.Page,
		"page_size": filter.PageSize,
		"total":     total,
	})
}

func (h *Handler) Get(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	plan, err := h.service.Get(c.Request.Context(), user.ID, c.Param("plan_id"))
	if status, code, ok := mapDomainError(err); ok {
		respondError(c, status, code, "")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"plan": ToUserPlanDTO(plan, true)})
}

// PatchBody is PATCH /me/plans/:plan_id body. All fields optional; pointer
// rules let us distinguish "field not present" from "field set to empty".
type PatchBody struct {
	Title      *string   `json:"title"`
	Note       *string   `json:"note"`
	Summary    *string   `json:"summary"`
	Tags       *[]string `json:"tags"`
	Visibility *string   `json:"visibility"`
}

func (h *Handler) Patch(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	var body PatchBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	plan, err := h.service.Edit(c.Request.Context(), user.ID, c.Param("plan_id"), EditInput{
		Title:      body.Title,
		Note:       body.Note,
		Summary:    body.Summary,
		Tags:       body.Tags,
		Visibility: body.Visibility,
	})
	if status, code, ok := mapDomainError(err); ok {
		respondError(c, status, code, "")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "patch_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"plan": ToUserPlanDTO(plan, true)})
}

func (h *Handler) Delete(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	err := h.service.Delete(c.Request.Context(), user.ID, c.Param("plan_id"))
	if status, code, ok := mapDomainError(err); ok {
		respondError(c, status, code, "")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetConversation(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	archive, err := h.service.GetArchive(c.Request.Context(), user.ID, c.Param("plan_id"))
	if status, code, ok := mapDomainError(err); ok {
		respondError(c, status, code, "")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "archive_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversation": ToArchiveDTO(archive)})
}

// PublishBody is POST /me/plans/:plan_id/publish body.
type PublishBody struct {
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Tags    []string `json:"tags"`
}

func (h *Handler) Publish(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	var body PublishBody
	if err := c.ShouldBindJSON(&body); err != nil {
		// allow empty body — defaults take over.
		body = PublishBody{}
	}
	pub, err := h.service.Publish(c.Request.Context(), user.ID, c.Param("plan_id"), PublishInput{
		Title:   body.Title,
		Summary: body.Summary,
		Tags:    body.Tags,
	})
	if status, code, ok := mapDomainError(err); ok {
		respondError(c, status, code, "")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "publish_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"public_plan": ToPublicPlanDTO(pub, true)})
}

func (h *Handler) Unpublish(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	err := h.service.Unpublish(c.Request.Context(), user.ID, c.Param("plan_id"))
	if status, code, ok := mapDomainError(err); ok {
		respondError(c, status, code, "")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "unpublish_failed", err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Current(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	var running *RunningTask
	if h.currentTask != nil {
		running = h.currentTask(user.ID)
	}
	view, err := h.service.Current(c.Request.Context(), user.ID, running)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "current_failed", err.Error())
		return
	}
	resp := gin.H{}
	if view.RunningTask != nil {
		resp["running_task"] = gin.H{
			"task_id":          view.RunningTask.TaskID,
			"status":           view.RunningTask.Status,
			"destination_city": view.RunningTask.DestinationCity,
			"updated_at":       view.RunningTask.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	if view.LatestPlan != nil {
		latest := ToUserPlanDTO(*view.LatestPlan, false)
		resp["latest_plan"] = latest
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListPublic(c *gin.Context) {
	filter := PublicListFilter{
		Query:           strings.TrimSpace(c.Query("q")),
		DestinationCity: strings.TrimSpace(c.Query("destination_city")),
		Days:            atoiOr(c.Query("days"), 0),
		Interest:        strings.TrimSpace(c.Query("interest")),
		Sort:            strings.ToLower(strings.TrimSpace(c.Query("sort"))),
		Page:            atoiOr(c.Query("page"), 1),
		PageSize:        atoiOr(c.Query("page_size"), defaultPageSize),
	}
	plans, total, err := h.service.ListPublic(c.Request.Context(), filter)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	items := make([]PublicPlanDTO, 0, len(plans))
	for _, p := range plans {
		items = append(items, ToPublicPlanDTO(p, false))
	}
	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"page":      filter.Page,
		"page_size": filter.PageSize,
		"total":     total,
	})
}

func (h *Handler) GetPublic(c *gin.Context) {
	viewerID := ""
	if user, ok := auth.UserFromGin(c); ok {
		viewerID = user.ID
	}
	pub, err := h.service.GetPublic(c.Request.Context(), c.Param("public_plan_id"), viewerID)
	if errors.Is(err, ErrPublicPlanNotFound) {
		respondError(c, http.StatusNotFound, "not_found", "")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"public_plan": ToPublicPlanDTO(pub, true)})
}

func (h *Handler) SavePublicAsCopy(c *gin.Context) {
	user, ok := auth.UserFromGin(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return
	}
	plan, err := h.service.SavePublicAsCopy(c.Request.Context(), user.ID, c.Param("public_plan_id"))
	if status, code, ok := mapDomainError(err); ok {
		respondError(c, status, code, "")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "save_failed", err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"plan": ToUserPlanDTO(plan, true)})
}

func mapDomainError(err error) (int, string, bool) {
	if err == nil {
		return 0, "", false
	}
	switch {
	case errors.Is(err, ErrPlanNotFound):
		return http.StatusNotFound, "not_found", true
	case errors.Is(err, ErrPublicPlanNotFound):
		return http.StatusNotFound, "not_found", true
	case errors.Is(err, ErrTaskNotFound):
		return http.StatusBadRequest, "task_not_found", true
	case errors.Is(err, ErrTaskNotOwned):
		return http.StatusForbidden, "forbidden", true
	case errors.Is(err, ErrTaskNotSucceeded):
		return http.StatusConflict, "task_not_ready", true
	case errors.Is(err, ErrAlreadyPublished):
		return http.StatusConflict, "already_published", true
	case errors.Is(err, ErrNotPublished):
		return http.StatusConflict, "not_published", true
	case errors.Is(err, ErrInvalidTitle):
		return http.StatusBadRequest, "invalid_title", true
	case errors.Is(err, ErrInvalidVisibility):
		return http.StatusBadRequest, "invalid_visibility", true
	case errors.Is(err, ErrSourcePlanForbidden):
		return http.StatusConflict, "public_plan_unavailable", true
	}
	return 0, "", false
}

func atoiOr(value string, fallback int) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func respondError(c *gin.Context, status int, code, message string) {
	requestID := c.Writer.Header().Get("X-Request-ID")
	if message == "" {
		switch code {
		case "not_found":
			message = "plan not found"
		case "forbidden":
			message = "you do not have permission for this plan"
		case "unauthenticated":
			message = "sign in required"
		default:
			message = code
		}
	}
	c.JSON(status, gin.H{
		"request_id": requestID,
		"code":       code,
		"message":    message,
	})
}
