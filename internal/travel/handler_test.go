package travel

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandlerCreateAndGetPlan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := NewTravelPlanService(stubPlanner{}, NewMemoryPlanStore())
	router := gin.New()
	handler := NewHandler(service)
	router.POST("/plans", handler.CreatePlan)
	router.GET("/plans/:id", handler.GetPlan)

	body := `{"departure_city":"上海","destination_city":"杭州","days":2,"budget":1000}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/plans", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var created CreatePlanResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.PlanID == "" {
		t.Fatal("expected plan id")
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/plans/"+created.PlanID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerValidationAndNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := NewTravelPlanService(stubPlanner{}, NewMemoryPlanStore())
	router := gin.New()
	handler := NewHandler(service)
	router.POST("/plans", handler.CreatePlan)
	router.GET("/plans/:id", handler.GetPlan)

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
