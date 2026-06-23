package travel

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestHandlerCreateAndGetTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := NewTravelPlanService(stubPlanner{}, NewMemoryTaskStore(), NewMemoryRateLimiter(60))
	router := gin.New()
	handler := NewHandler(service)
	router.POST("/plans", handler.CreatePlan)
	router.GET("/plans/:task_id", handler.GetPlan)

	body := `{"departure_city":"上海","destination_city":"杭州","days":2,"budget":1000}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plans", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", rec.Code, rec.Body.String())
	}
	var created CreateTaskResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.TaskID == "" || created.RequestHash == "" {
		t.Fatal("expected task id and request hash")
	}

	var got GetTaskResponse
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/plans/"+created.TaskID, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode get response: %v", err)
		}
		if got.Status == TaskSucceeded {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("task did not succeed, last response=%#v", got)
}

func TestHandlerValidationAndNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := NewTravelPlanService(stubPlanner{}, NewMemoryTaskStore(), NewMemoryRateLimiter(60))
	router := gin.New()
	handler := NewHandler(service)
	router.POST("/plans", handler.CreatePlan)
	router.GET("/plans/:task_id", handler.GetPlan)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plans", bytes.NewBufferString(`{"destination_city":"杭州"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/plans/missing", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandlerStreamCompletedTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := NewTravelPlanService(stubPlanner{}, NewMemoryTaskStore(), NewMemoryRateLimiter(60))
	router := gin.New()
	handler := NewHandler(service)
	router.GET("/plans/:task_id/stream", handler.StreamPlan)

	created, err := service.CreateTask(httptest.NewRequest(http.MethodPost, "/", nil).Context(), validCreateRequest(), "127.0.0.1")
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	waitForTask(t, service, created.TaskID)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/plans/"+created.TaskID+"/stream", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event:done") || !strings.Contains(body, created.TaskID) {
		t.Fatalf("expected done SSE event, got %s", body)
	}
}
