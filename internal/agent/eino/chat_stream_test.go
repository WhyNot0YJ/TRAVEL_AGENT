package eino

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"travel-agent/internal/agent"
	"travel-agent/internal/domain"
)

type captureReporter struct {
	deltas atomic.Int64
	parts  []string
	full   string
}

func (r *captureReporter) ReportLLMDelta(_ context.Context, delta string) {
	r.deltas.Add(1)
	r.parts = append(r.parts, delta)
}

func (r *captureReporter) ReportLLMDone(_ context.Context, full string) {
	r.full = full
}

func (r *captureReporter) SawAnyDelta() bool {
	return r.deltas.Load() > 0
}

// fakeStreamingChatServer returns a httptest.Server that responds with the given
// SSE frames. Each request body is also captured for assertions.
func fakeStreamingChatServer(t *testing.T, frames []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, frame := range frames {
			fmt.Fprintf(w, "data: %s\n\n", frame)
			if flusher != nil {
				flusher.Flush()
			}
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
}

func TestStreamChatReplyAccumulatesDeltas(t *testing.T) {
	frames := []string{
		`{"choices":[{"delta":{"content":"信息已经齐全，"}}]}`,
		`{"choices":[{"delta":{"content":"可以"}}]}`,
		`{"choices":[{"delta":{"content":"生成"}}]}`,
		`{"choices":[{"delta":{"content":"行程了。"},"finish_reason":"stop"}]}`,
	}
	server := fakeStreamingChatServer(t, frames)
	defer server.Close()

	client := &openAICompatibleClient{
		config: LLMConfig{
			Provider:      "deepseek",
			BaseURL:       server.URL,
			Model:         "deepseek-v4-flash",
			QuickModel:    "deepseek-v4-flash",
			ExpertModel:   "deepseek-v4-pro",
			APIKey:        "test-key",
			Timeout:       5 * time.Second,
			StreamEnabled: true,
		},
		httpClient: server.Client(),
	}
	rep := &captureReporter{}
	prior := &agent.TravelInfoResult{
		DepartureCity: "上海", DestinationCity: "杭州", Days: 3, Budget: 3000,
		Interests: []string{"美食"}, IsComplete: true, Reply: "fallback",
	}
	got := client.streamChatReply(context.Background(), prior, "继续", rep)
	if got != "信息已经齐全，可以生成行程了。" {
		t.Fatalf("streamed reply mismatch: %q", got)
	}
	if rep.deltas.Load() < 2 {
		t.Fatalf("expected multiple deltas, got %d", rep.deltas.Load())
	}
	if rep.full != got {
		t.Fatalf("ReportLLMDone payload mismatch: %q vs %q", rep.full, got)
	}
}

func TestStreamChatReplyFailureReturnsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &openAICompatibleClient{
		config: LLMConfig{
			Provider:      "deepseek",
			BaseURL:       server.URL,
			Model:         "deepseek-v4-flash",
			APIKey:        "test-key",
			StreamEnabled: true,
		},
		httpClient: server.Client(),
	}
	rep := &captureReporter{}
	got := client.streamChatReply(context.Background(), &agent.TravelInfoResult{}, "msg", rep)
	if got != "" {
		t.Fatalf("expected empty reply on stream failure, got %q", got)
	}
	if rep.deltas.Load() != 0 {
		t.Fatalf("expected no deltas on failure, got %d", rep.deltas.Load())
	}
}

func TestStreamChatReplyDisabledShortCircuits(t *testing.T) {
	// Server should never be hit when StreamEnabled=false.
	hit := atomic.Int64{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hit.Add(1)
	}))
	defer server.Close()

	client := &openAICompatibleClient{
		config: LLMConfig{
			Provider:      "deepseek",
			BaseURL:       server.URL,
			StreamEnabled: false,
		},
		httpClient: server.Client(),
	}
	rep := &captureReporter{}
	got := client.streamChatReply(context.Background(), &agent.TravelInfoResult{}, "msg", rep)
	if got != "" {
		t.Fatalf("expected empty reply when stream disabled, got %q", got)
	}
	if hit.Load() != 0 {
		t.Fatalf("expected zero requests when stream disabled, got %d", hit.Load())
	}
}

func TestStreamingTravelPlanReportsSummaryDeltas(t *testing.T) {
	// Stream a strict tool call where the "summary" field is split across
	// frames. Verify the reporter receives summary characters as they arrive
	// and the final structured plan parses correctly.
	rawPlan := mustMarshal(t, domain.TravelPlan{
		Title:   "杭州",
		Summary: "从上海出发的3天行程",
		Days: []domain.TravelDay{{
			Day:   1,
			Theme: "美食",
			Items: []domain.TravelItem{{
				Time:            "09:30",
				Type:            "餐厅",
				Name:            "店",
				Address:         "a",
				Reason:          "r",
				EstimatedCost:   1,
				Cost:            domain.AvailableCost(1, "per_person", "amap.poi.biz_ext.cost", true),
				DurationMinutes: 1,
			}},
		}},
		Budget: domain.TravelBudget{
			Transport:  1,
			Food:       1,
			Hotel:      1,
			Ticket:     1,
			Total:      4,
			KnownTotal: 4,
			Complete:   true,
			Currency:   "CNY",
			Items: []domain.BudgetLine{
				availableBudgetLineForTest("transport", "市内交通", 1),
				availableBudgetLineForTest("food", "餐饮", 1),
				availableBudgetLineForTest("hotel", "住宿", 1),
				availableBudgetLineForTest("ticket", "门票", 1),
			},
			Missing: []string{},
		},
		Warnings: []string{},
	})
	cut1 := strings.Index(rawPlan, "从上") + len("从上")
	cut2 := strings.Index(rawPlan, "3天行程")
	frames := []string{
		streamToolArgsFrame(rawPlan[:cut1], false, true),
		streamToolArgsFrame(rawPlan[cut1:cut2], false, false),
		streamToolArgsFrame(rawPlan[cut2:], true, false),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, f := range frames {
			fmt.Fprintf(w, "data: %s\n\n", f)
			if flusher != nil {
				flusher.Flush()
			}
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer server.Close()

	client := &openAICompatibleClient{
		config: LLMConfig{
			Provider:      "deepseek",
			BaseURL:       server.URL,
			Model:         "deepseek-v4-flash",
			QuickModel:    "deepseek-v4-flash",
			ExpertModel:   "deepseek-v4-pro",
			APIKey:        "k",
			Timeout:       5 * time.Second,
			StreamEnabled: true,
		},
		httpClient: server.Client(),
	}
	rep := &captureReporter{}
	state := TravelPlanningState{
		Request:               domain.TravelRequest{DepartureCity: "上海", DestinationCity: "杭州", Days: 1, Budget: 100, Interests: []string{"美食"}, Travelers: 2, Pace: "balanced", TransportMode: "train_taxi"},
		NormalizedDestination: "杭州",
		NormalizedDays:        1,
		NormalizedBudget:      100,
		Interests:             []string{"美食"},
		Itinerary:             []domain.TravelDay{{Day: 1}},
	}
	result, err := client.generateTravelPlanStreaming(context.Background(), state, "quick", rep)
	if err != nil {
		t.Fatalf("generateTravelPlanStreaming returned error: %v", err)
	}
	if result.Plan == nil {
		t.Fatal("expected plan to be parsed")
	}
	if result.Plan.Summary != "从上海出发的3天行程" {
		t.Fatalf("plan summary mismatch: %q", result.Plan.Summary)
	}
	if rep.deltas.Load() < 2 {
		t.Fatalf("expected reporter to receive multiple summary deltas, got %d", rep.deltas.Load())
	}
	if got := strings.Join(rep.parts, ""); got != "从上海出发的3天行程" {
		t.Fatalf("accumulated reporter deltas mismatch: %q", got)
	}
	if rep.full != "从上海出发的3天行程" {
		t.Fatalf("ReportLLMDone payload mismatch: %q", rep.full)
	}
}

func streamToolArgsFrame(args string, finish, includeName bool) string {
	name := ""
	if includeName {
		name = `"id":"call_1","type":"function",`
	}
	finishReason := ""
	if finish {
		finishReason = `,"finish_reason":"tool_calls"`
	}
	return `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,` + name + `"function":{"name":"submit_travel_plan","arguments":` + strconv.Quote(args) + `}}]}` + finishReason + `}]}`
}
