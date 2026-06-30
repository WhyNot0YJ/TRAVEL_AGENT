package plans

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"travel-agent/internal/auth"
	"travel-agent/internal/domain"
)

func setupHandlerForUser(t *testing.T, user auth.User, snapshots ...TaskSnapshot) (*gin.Engine, *Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	plansStore := NewMemoryPlanStore()
	publicStore := NewMemoryPublicPlanStore()
	tasks := stubTaskLookup{tasks: map[string]TaskSnapshot{}}
	for _, snap := range snapshots {
		tasks.tasks[snap.TaskID] = snap
	}
	svc := NewService(plansStore, publicStore, tasks, stubAuthorLookup{name: user.DisplayName})
	frozen := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return frozen }

	handler := NewHandler(svc, nil)
	router := gin.New()
	authed := router.Group("/api/v1")
	authed.Use(func(c *gin.Context) {
		if user.ID != "" {
			c.Set("auth.user", user)
			ctx := auth.WithUserID(c.Request.Context(), user.ID)
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	})
	authed.POST("/me/plans", handler.Save)
	authed.GET("/me/plans", handler.List)
	authed.GET("/me/plans/:plan_id", handler.Get)
	authed.PATCH("/me/plans/:plan_id", handler.Patch)
	authed.DELETE("/me/plans/:plan_id", handler.Delete)
	authed.GET("/me/plans/:plan_id/conversation", handler.GetConversation)
	authed.POST("/me/plans/:plan_id/publish", handler.Publish)
	authed.POST("/me/plans/:plan_id/unpublish", handler.Unpublish)
	authed.GET("/me/current", handler.Current)
	router.GET("/api/v1/public/plans", handler.ListPublic)
	router.GET("/api/v1/public/plans/:public_plan_id", handler.GetPublic)
	router.POST("/api/v1/public/plans/:public_plan_id/save", func(c *gin.Context) {
		if user.ID != "" {
			c.Set("auth.user", user)
		}
		handler.SavePublicAsCopy(c)
	})
	return router, svc
}

func TestHandlerSavePlanReturnsCreated(t *testing.T) {
	user := auth.User{ID: "user_a", DisplayName: "Alice"}
	router, _ := setupHandlerForUser(t, user, sampleSnapshot("task_a", "user_a"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/plans", strings.NewReader(`{"task_id":"task_a","title":"我的杭州行程"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "我的杭州行程") {
		t.Fatalf("expected title in response, got %s", rec.Body.String())
	}
}

func TestHandlerSavePlanRejectsUnknownTask(t *testing.T) {
	user := auth.User{ID: "user_a", DisplayName: "Alice"}
	router, _ := setupHandlerForUser(t, user)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/plans", strings.NewReader(`{"task_id":"missing"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerListAndGet(t *testing.T) {
	user := auth.User{ID: "user_a", DisplayName: "Alice"}
	router, svc := setupHandlerForUser(t, user, sampleSnapshot("task_a", "user_a"))
	plan, err := svc.Save(context.Background(), user.ID, SaveInput{TaskID: "task_a"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/me/plans", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), plan.ID) {
		t.Fatalf("list missing plan id: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/me/plans/"+plan.ID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Plan UserPlanDTO `json:"plan"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if body.Plan.Plan == nil || body.Plan.Plan.Title == "" {
		t.Fatalf("expected full plan on detail response, got %+v", body)
	}
}

func TestHandlerForbidsCrossUserAccess(t *testing.T) {
	owner := auth.User{ID: "user_a", DisplayName: "Alice"}
	router, svc := setupHandlerForUser(t, owner, sampleSnapshot("task_a", "user_a"))
	plan, err := svc.Save(context.Background(), owner.ID, SaveInput{TaskID: "task_a"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Re-bind handler with a different user.
	other := auth.User{ID: "user_b", DisplayName: "Bob"}
	otherRouter, _ := setupHandlerForUser(t, other, sampleSnapshot("task_a", "user_a"))
	rec := httptest.NewRecorder()
	otherRouter.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/me/plans/"+plan.ID, nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 cross-user, got %d body=%s", rec.Code, rec.Body.String())
	}
	_ = router
}

func TestHandlerPublishAndPublicGet(t *testing.T) {
	user := auth.User{ID: "user_a", DisplayName: "Alice"}
	router, svc := setupHandlerForUser(t, user, sampleSnapshot("task_a", "user_a"))
	plan, err := svc.Save(context.Background(), user.ID, SaveInput{TaskID: "task_a"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/plans/"+plan.ID+"/publish", strings.NewReader(`{"title":"杭州亲子游","tags":["杭州","亲子"]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("publish: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var pubBody struct {
		PublicPlan PublicPlanDTO `json:"public_plan"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &pubBody); err != nil {
		t.Fatalf("decode publish response: %v", err)
	}
	if pubBody.PublicPlan.PublicPlanID == "" || pubBody.PublicPlan.Author.DisplayName == "" {
		t.Fatalf("unexpected publish response: %+v", pubBody)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/public/plans/"+pubBody.PublicPlan.PublicPlanID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("public get: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "杭州亲子游") {
		t.Fatalf("expected title in public detail, got %s", rec.Body.String())
	}
}

// TestSavePlanRejectsUnauthenticated proves the handler refuses anonymous
// callers even when no auth middleware is mounted in front of it.
func TestSavePlanRejectsUnauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	plansStore := NewMemoryPlanStore()
	publicStore := NewMemoryPublicPlanStore()
	svc := NewService(plansStore, publicStore, stubTaskLookup{tasks: map[string]TaskSnapshot{}}, stubAuthorLookup{})
	handler := NewHandler(svc, nil)
	router := gin.New()
	router.POST("/me/plans", handler.Save)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/me/plans", strings.NewReader(`{"task_id":"task_a"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// reuse imported domain to avoid unused warning
var _ = domain.TravelPlan{}
