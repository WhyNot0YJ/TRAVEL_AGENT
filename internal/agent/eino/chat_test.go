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
	if !containsString(result.Missing, "兴趣偏好") || !containsString(result.Missing, "出行人数") {
		t.Fatalf("expected interests and travelers to remain missing, got %#v", result.Missing)
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
		Travelers:       2,
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
	if result.Pace != "轻松" {
		t.Fatalf("expected 轻松 pace, got %q", result.Pace)
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
		Travelers:       2,
	}
	result, err := simpleFallbackExtractor{}.Extract(context.Background(), "高铁优先，少走回头路", current)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if result.TransportMode != "高铁 + 打车" {
		t.Fatalf("expected 高铁 + 打车 transport mode, got %q", result.TransportMode)
	}
	if !result.IsComplete {
		t.Fatalf("expected complete result, missing=%#v", result.Missing)
	}
}

func TestSimpleFallbackExtractorRequiresTravelers(t *testing.T) {
	result, err := simpleFallbackExtractor{}.Extract(context.Background(), "上海出发，杭州 3 天，预算 3000，喜欢美食", domain.TravelRequest{})
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if result.IsComplete {
		t.Fatalf("expected incomplete result without travelers, got %#v", result)
	}
	if !containsString(result.Missing, "出行人数") {
		t.Fatalf("expected travelers missing, got %#v", result.Missing)
	}
	if result.Pace != "适中" || result.TransportMode != "任意" || result.WalkingTolerance != "任意" || result.BudgetType != "总预算" {
		t.Fatalf("expected product defaults, got pace=%q transport=%q walking=%q budget_type=%q", result.Pace, result.TransportMode, result.WalkingTolerance, result.BudgetType)
	}
}

func TestSimpleFallbackExtractorDefaultsOptionalFields(t *testing.T) {
	result, err := simpleFallbackExtractor{}.Extract(context.Background(), "上海出发，杭州 3 天，2 人，预算 3000，喜欢美食", domain.TravelRequest{})
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !result.IsComplete || len(result.Missing) != 0 {
		t.Fatalf("expected complete result, got complete=%v missing=%#v", result.IsComplete, result.Missing)
	}
	if result.DateRange != "任意" || result.Pace != "适中" || result.TransportMode != "任意" || result.WalkingTolerance != "任意" {
		t.Fatalf("expected stable defaults, got %#v", result)
	}
	if len(result.MustVisit) != 0 || len(result.Avoid) != 0 {
		t.Fatalf("expected empty optional lists, got must=%#v avoid=%#v", result.MustVisit, result.Avoid)
	}
	if got := strings.Join(result.BudgetIncludes, "、"); got != "住宿、餐饮、门票、市内交通" {
		t.Fatalf("unexpected budget includes: %q", got)
	}
}

func TestSimpleFallbackExtractorExtractsMustAvoidAndBudgetType(t *testing.T) {
	result, err := simpleFallbackExtractor{}.Extract(context.Background(), "上海出发去杭州3天，2人，人均2000，喜欢美食，必去西湖和灵隐寺，避开网红店，少走路", domain.TravelRequest{})
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	if !result.IsComplete {
		t.Fatalf("expected complete result, missing=%#v", result.Missing)
	}
	if result.BudgetType != "人均预算" {
		t.Fatalf("expected 人均预算, got %q", result.BudgetType)
	}
	if result.WalkingTolerance != "低" {
		t.Fatalf("expected low walking tolerance, got %q", result.WalkingTolerance)
	}
	if !containsString(result.MustVisit, "西湖") || !containsString(result.MustVisit, "灵隐寺") {
		t.Fatalf("expected must visit extraction, got %#v", result.MustVisit)
	}
	if !containsString(result.Avoid, "网红店") {
		t.Fatalf("expected avoid extraction, got %#v", result.Avoid)
	}
}

func TestChatInfoExtractorUsesStreamingToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertStreamRequest(t, r)
		writeLLMStreamResponse(t, w, extractTravelInfoToolName, `{"departure_city":"上海","destination_city":"杭州","days":3,"budget":3000,"interests":["美食"],"travelers":2,"transport_mode":"train_taxi","pace":"balanced","reply":"信息已齐全，可以生成行程。","missing":[],"is_complete":true}`, tokenUsage{PromptTokens: 3, CompletionTokens: 4, TotalTokens: 7})
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
