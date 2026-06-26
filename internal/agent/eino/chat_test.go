package eino

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"travel-agent/internal/domain"
)

func TestSimpleFallbackExtractorExtractsInitialTravelInfo(t *testing.T) {
	result, err := simpleFallbackExtractor{}.Extract(context.Background(), "上海出发，杭州 3 天，预算 3000", domain.TravelRequest{})
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if result.DepartureCity != "上海" {
		t.Fatalf("expected departure city 上海, got %q", result.DepartureCity)
	}
	if result.DestinationCity != "杭州" {
		t.Fatalf("expected destination city 杭州, got %q", result.DestinationCity)
	}
	if result.Days != 3 {
		t.Fatalf("expected 3 days, got %d", result.Days)
	}
	if result.Budget != 3000 {
		t.Fatalf("expected budget 3000, got %.2f", result.Budget)
	}
	if !containsString(result.Missing, "兴趣偏好") {
		t.Fatalf("expected interests to remain missing, got %#v", result.Missing)
	}
	for _, field := range []string{"出发城市", "目的地", "天数", "预算"} {
		if strings.Contains(strings.Join(result.Missing, ","), field) {
			t.Fatalf("did not expect %s to be missing, got %#v", field, result.Missing)
		}
	}
}

func TestSimpleFallbackExtractorMergesFollowUpInfo(t *testing.T) {
	current := domain.TravelRequest{
		DepartureCity:   "上海",
		DestinationCity: "杭州",
		Days:            3,
		Budget:          3000,
		TransportMode:   "train_taxi",
		Pace:            "balanced",
	}
	result, err := simpleFallbackExtractor{}.Extract(context.Background(), "想轻松一点，喜欢美食和自然风光", current)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if result.DepartureCity != current.DepartureCity || result.DestinationCity != current.DestinationCity {
		t.Fatalf("expected existing cities to be preserved, got %#v", result)
	}
	if result.Pace != "relaxed" {
		t.Fatalf("expected relaxed pace, got %q", result.Pace)
	}
	if !containsString(result.Interests, "美食") || !containsString(result.Interests, "自然风光") {
		t.Fatalf("expected interests 美食 and 自然风光, got %#v", result.Interests)
	}
	if !result.IsComplete || len(result.Missing) != 0 {
		t.Fatalf("expected complete result, got complete=%v missing=%#v", result.IsComplete, result.Missing)
	}
}

func TestSimpleFallbackExtractorExtractsTransportPreference(t *testing.T) {
	current := domain.TravelRequest{
		DepartureCity:   "上海",
		DestinationCity: "杭州",
		Days:            3,
		Budget:          3000,
		Interests:       []string{"美食"},
	}
	result, err := simpleFallbackExtractor{}.Extract(context.Background(), "高铁优先，少走回头路", current)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if result.TransportMode != "train_taxi" {
		t.Fatalf("expected train_taxi transport mode, got %q", result.TransportMode)
	}
	if !result.IsComplete {
		t.Fatalf("expected complete result, missing=%#v", result.Missing)
	}
}

func TestChatInfoExtractorUsesStreamingToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertStreamRequest(t, r)
		writeLLMStreamResponse(t, w, extractTravelInfoToolName, `{"departure_city":"上海","destination_city":"杭州","days":3,"budget":3000,"interests":["美食"],"transport_mode":"train_taxi","pace":"balanced","reply":"信息已齐全，可以生成行程。","missing":[],"is_complete":true}`, tokenUsage{PromptTokens: 3, CompletionTokens: 4, TotalTokens: 7})
	}))
	defer server.Close()

	extractor := &chatInfoExtractor{
		client: newOpenAICompatibleClient(LLMConfig{
			Provider:      "deepseek",
			APIKey:        "test-key",
			BaseURL:       server.URL,
			Model:         "deepseek-v4-flash",
			StreamEnabled: true,
		}),
		fallback: simpleFallbackExtractor{},
	}
	extractor.client.httpClient = server.Client()

	result, err := extractor.callLLM(context.Background(), "上海出发去杭州三天，预算3000，喜欢美食", domain.TravelRequest{})
	if err != nil {
		t.Fatalf("callLLM returned error: %v", err)
	}
	if !result.IsComplete || result.DepartureCity != "上海" || result.DestinationCity != "杭州" || result.Days != 3 {
		t.Fatalf("unexpected extracted result: %#v", result)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
