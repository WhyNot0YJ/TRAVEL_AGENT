package eino

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"travel-agent/internal/domain"
)

func TestDeepSeekStrictToolSchemaPayload(t *testing.T) {
	req := buildChatCompletionRequest(LLMConfig{
		Provider: "deepseek",
		Model:    "deepseek-v4-flash",
	}, []chatMessage{{Role: "user", Content: "plan"}}, travelPlanJSONSchema(), submitTravelPlanToolName, "Submit the final structured travel plan.", "quick")

	if req.Model != "deepseek-v4-flash" {
		t.Fatalf("unexpected model: %s", req.Model)
	}
	if len(req.Tools) != 1 {
		t.Fatalf("expected one tool, got %d", len(req.Tools))
	}
	tool := req.Tools[0]
	if tool.Type != "function" {
		t.Fatalf("expected function tool, got %s", tool.Type)
	}
	if tool.Function.Name != submitTravelPlanToolName {
		t.Fatalf("unexpected tool name: %s", tool.Function.Name)
	}
	if !tool.Function.Strict {
		t.Fatal("expected strict tool schema")
	}
	if req.Thinking == nil || req.Thinking.Type != "disabled" {
		t.Fatalf("expected DeepSeek tool call request to disable thinking mode, got %#v", req.Thinking)
	}
	if additional, ok := tool.Function.Parameters["additionalProperties"].(bool); !ok || additional {
		t.Fatalf("expected root additionalProperties=false, got %#v", tool.Function.Parameters["additionalProperties"])
	}
	required, ok := tool.Function.Parameters["required"].([]string)
	if !ok {
		t.Fatalf("expected []string required, got %#v", tool.Function.Parameters["required"])
	}
	if !containsAll(required, "title", "summary", "days", "budget", "warnings") {
		t.Fatalf("root required fields missing: %#v", required)
	}
	choice, ok := req.ToolChoice.(map[string]any)
	if !ok {
		t.Fatalf("expected tool choice map, got %#v", req.ToolChoice)
	}
	fn, ok := choice["function"].(map[string]any)
	if !ok || fn["name"] != submitTravelPlanToolName {
		t.Fatalf("expected forced submit_travel_plan tool choice, got %#v", req.ToolChoice)
	}
}

func TestStructuredResponseFormatPayloadForCompatibleProvider(t *testing.T) {
	req := buildChatCompletionRequest(LLMConfig{
		Provider: "compatible",
		Model:    "model",
	}, []chatMessage{{Role: "user", Content: "plan"}}, travelPlanJSONSchema(), "", "", "quick")

	if len(req.Tools) != 0 {
		t.Fatalf("compatible provider should not use tools by default")
	}
	format, ok := req.ResponseFormat.(map[string]any)
	if !ok {
		t.Fatalf("expected response_format map, got %#v", req.ResponseFormat)
	}
	if format["type"] != "json_schema" {
		t.Fatalf("expected json_schema response format, got %#v", format["type"])
	}
}

func TestLLMConfigDefaultsDoNotContainAPIKey(t *testing.T) {
	t.Setenv("TRAVEL_AGENT_LLM_ENABLED", "")
	t.Setenv("TRAVEL_AGENT_LLM_PROVIDER", "")
	t.Setenv("TRAVEL_AGENT_LLM_API_KEY", "")
	t.Setenv("TRAVEL_AGENT_LLM_BASE_URL", "")
	t.Setenv("TRAVEL_AGENT_LLM_MODEL", "")
	t.Setenv("TRAVEL_AGENT_LLM_MODEL_QUICK", "")
	t.Setenv("TRAVEL_AGENT_LLM_MODEL_EXPERT", "")
	t.Setenv("TRAVEL_AGENT_LLM_STREAM_ENABLED", "")
	t.Setenv("DEEPSEEK_API_KEY", "")

	cfg := loadLLMConfigFromEnv()
	if cfg.Enabled {
		t.Fatal("LLM should be disabled by default")
	}
	if cfg.Provider != defaultLLMProvider {
		t.Fatalf("expected default provider %q, got %q", defaultLLMProvider, cfg.Provider)
	}
	if cfg.BaseURL != defaultLLMBaseURL {
		t.Fatalf("expected default base URL %q, got %q", defaultLLMBaseURL, cfg.BaseURL)
	}
	if cfg.Model != defaultLLMModel {
		t.Fatalf("expected default model %q, got %q", defaultLLMModel, cfg.Model)
	}
	if cfg.QuickModel != defaultLLMQuickModel {
		t.Fatalf("expected default quick model %q, got %q", defaultLLMQuickModel, cfg.QuickModel)
	}
	if cfg.ExpertModel != defaultLLMExpertModel {
		t.Fatalf("expected default expert model %q, got %q", defaultLLMExpertModel, cfg.ExpertModel)
	}
	if !cfg.StreamEnabled {
		t.Fatal("StreamEnabled should default to true")
	}
	if cfg.APIKey != "" {
		t.Fatal("default config must not contain an API key")
	}
}

func TestModelForModeSelectsByAgentMode(t *testing.T) {
	cfg := LLMConfig{
		Model:       "fallback",
		QuickModel:  "deepseek-v4-flash",
		ExpertModel: "deepseek-v4-pro",
	}
	if got := modelForMode(cfg, "quick"); got != "deepseek-v4-flash" {
		t.Fatalf("quick mode: got %q", got)
	}
	if got := modelForMode(cfg, "expert"); got != "deepseek-v4-pro" {
		t.Fatalf("expert mode: got %q", got)
	}
	if got := modelForMode(cfg, ""); got != "deepseek-v4-flash" {
		t.Fatalf("empty mode should fall back to quick: got %q", got)
	}
	// When per-mode models are unset, fall back to legacy Model field.
	bare := LLMConfig{Model: "legacy"}
	if got := modelForMode(bare, "expert"); got != "legacy" {
		t.Fatalf("expected legacy fallback, got %q", got)
	}
}

func TestStreamEnabledRespectsExplicitFalse(t *testing.T) {
	t.Setenv("TRAVEL_AGENT_LLM_STREAM_ENABLED", "false")
	cfg := loadLLMConfigFromEnv()
	if cfg.StreamEnabled {
		t.Fatal("StreamEnabled should be false when env set to false")
	}
}

func TestLegacyModelOverrideAppliesToQuickAndExpert(t *testing.T) {
	t.Setenv("TRAVEL_AGENT_LLM_MODEL", "custom-model")
	t.Setenv("TRAVEL_AGENT_LLM_MODEL_QUICK", "")
	t.Setenv("TRAVEL_AGENT_LLM_MODEL_EXPERT", "")
	cfg := loadLLMConfigFromEnv()
	if cfg.QuickModel != "custom-model" {
		t.Fatalf("legacy MODEL should propagate to QuickModel: got %q", cfg.QuickModel)
	}
	if cfg.ExpertModel != "custom-model" {
		t.Fatalf("legacy MODEL should propagate to ExpertModel: got %q", cfg.ExpertModel)
	}
}

func TestLLMGeneratorDisabledFallback(t *testing.T) {
	t.Setenv("TRAVEL_AGENT_LLM_ENABLED", "")
	t.Setenv("TRAVEL_AGENT_LLM_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")

	state := llmTestState()
	plan, err := newDefaultPlanGenerator().Generate(context.Background(), state)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if plan == nil {
		t.Fatal("Generate returned nil plan")
	}
	if !containsWarning(plan.Warnings, "deterministic mock tools") {
		t.Fatalf("expected deterministic fallback warning, got %#v", plan.Warnings)
	}
	if !containsWarning(plan.Warnings, "LLM fallback:") || !containsWarning(plan.Warnings, "category=disabled") {
		t.Fatalf("expected disabled LLM fallback warning, got %#v", plan.Warnings)
	}
}

func TestLLMGeneratorNoToolCallFallback(t *testing.T) {
	state := llmTestState()
	generator := llmPlanGenerator{
		client:   fakeLLMClient{err: errors.New("LLM response did not call submit_travel_plan")},
		fallback: deterministicPlanGenerator{},
	}
	plan, err := generator.Generate(context.Background(), state)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if !containsWarning(plan.Warnings, "LLM fallback:") || !containsWarning(plan.Warnings, "category=provider_error") {
		t.Fatalf("expected LLM fallback warning, got %#v", plan.Warnings)
	}
}

func TestParseTravelPlanArgumentsValid(t *testing.T) {
	req := llmTestRequest()
	raw := mustMarshal(t, llmValidPlan(req))
	plan, err := parseTravelPlanArguments(raw, req)
	if err != nil {
		t.Fatalf("parseTravelPlanArguments returned error: %v", err)
	}
	if plan.Title == "" || len(plan.Days) != req.Days {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestParseTravelPlanArgumentsRejectsInvalidShapes(t *testing.T) {
	req := llmTestRequest()
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "missing title",
			raw:  `{"summary":"杭州计划","days":[],"budget":{"transport":1,"food":1,"hotel":1,"ticket":1,"total":4},"warnings":[]}`,
		},
		{
			name: "extra field",
			raw:  strings.TrimSuffix(mustMarshal(t, llmValidPlan(req)), "}") + `,"extra":true}`,
		},
		{
			name: "negative budget",
			raw:  mustMarshal(t, planWithBudget(req, -1)),
		},
		{
			name: "day mismatch",
			raw:  mustMarshal(t, planWithDays(req, 1)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseTravelPlanArguments(tt.raw, req); err == nil {
				t.Fatal("expected parseTravelPlanArguments to reject invalid plan")
			}
		})
	}
}

func TestExtractTravelPlanPayloadRequiresDeepSeekToolCall(t *testing.T) {
	data := []byte(`{"choices":[{"message":{"content":"plain text"}}]}`)
	if _, _, err := extractTravelPlanPayload(LLMConfig{Provider: "deepseek"}, data); err == nil {
		t.Fatal("expected missing tool call error")
	}
}

func TestOpenAICompatibleClientFakeServerSuccessRecordsUsage(t *testing.T) {
	req := llmTestRequest()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization header not set correctly: %q", got)
		}
		assertStreamRequest(t, r)
		writeLLMStreamResponse(t, w, submitTravelPlanToolName, mustMarshal(t, llmValidPlan(req)), tokenUsage{PromptTokens: 11, CompletionTokens: 22, TotalTokens: 33})
	}))
	defer server.Close()

	client := newOpenAICompatibleClient(LLMConfig{
		Provider:      "deepseek",
		APIKey:        "test-key",
		BaseURL:       server.URL,
		Model:         "test-model",
		StreamEnabled: true,
	})
	result, err := client.GenerateTravelPlan(context.Background(), llmTestState())
	if err != nil {
		t.Fatalf("GenerateTravelPlan returned error: %v", err)
	}
	if result.Plan == nil || len(result.Plan.Days) != req.Days {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.PromptVersion != travelPlanPromptVersion {
		t.Fatalf("prompt version mismatch: %q", result.PromptVersion)
	}
	if !result.TokenUsage.Known || result.TokenUsage.TotalTokens != 33 {
		t.Fatalf("token usage not recorded: %#v", result.TokenUsage)
	}
}

func TestLLMGeneratorRetriesThenSucceedsWithFakeServer(t *testing.T) {
	var calls int32
	req := llmTestRequest()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		assertStreamRequest(t, r)
		if call == 1 {
			writeLLMStreamContent(t, w, "plain text")
			return
		}
		writeLLMStreamResponse(t, w, submitTravelPlanToolName, mustMarshal(t, llmValidPlan(req)), tokenUsage{})
	}))
	defer server.Close()

	generator := llmPlanGenerator{
		client: newOpenAICompatibleClient(LLMConfig{
			Provider:      "deepseek",
			APIKey:        "test-key",
			BaseURL:       server.URL,
			Model:         "test-model",
			StreamEnabled: true,
		}),
		fallback:   deterministicPlanGenerator{},
		maxRetries: 1,
	}
	plan, err := generator.Generate(context.Background(), llmTestState())
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected two calls, got %d", got)
	}
	if containsWarning(plan.Warnings, "LLM fallback:") {
		t.Fatalf("did not expect fallback warning after retry success: %#v", plan.Warnings)
	}
	if !containsWarning(plan.Warnings, "LLM trace:") || !containsWarning(plan.Warnings, "prompt_tokens=unknown") {
		t.Fatalf("expected LLM trace warning with unknown usage, got %#v", plan.Warnings)
	}
}

func TestLLMGeneratorFakeServerFallbackCategories(t *testing.T) {
	req := llmTestRequest()
	tests := []struct {
		name             string
		response         func(t *testing.T, w http.ResponseWriter)
		maxRetries       int
		expectedCategory string
	}{
		{
			name: "no tool call",
			response: func(t *testing.T, w http.ResponseWriter) {
				writeLLMStreamContent(t, w, "plain text")
			},
			expectedCategory: "provider_error",
		},
		{
			name: "bad plan json",
			response: func(t *testing.T, w http.ResponseWriter) {
				writeLLMStreamResponse(t, w, submitTravelPlanToolName, `{`, tokenUsage{})
			},
			expectedCategory: "invalid_json",
		},
		{
			name: "business validation failed",
			response: func(t *testing.T, w http.ResponseWriter) {
				writeLLMStreamResponse(t, w, submitTravelPlanToolName, mustMarshal(t, planWithBudget(req, 999999)), tokenUsage{})
			},
			expectedCategory: "business_validation_failed",
		},
		{
			name: "retry exhausted",
			response: func(t *testing.T, w http.ResponseWriter) {
				writeLLMStreamContent(t, w, "plain text")
			},
			maxRetries:       1,
			expectedCategory: "retry_exhausted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assertStreamRequest(t, r)
				tt.response(t, w)
			}))
			defer server.Close()

			generator := llmPlanGenerator{
				client: newOpenAICompatibleClient(LLMConfig{
					Provider:      "deepseek",
					APIKey:        "test-key",
					BaseURL:       server.URL,
					Model:         "test-model",
					StreamEnabled: true,
				}),
				fallback:   deterministicPlanGenerator{},
				maxRetries: tt.maxRetries,
			}
			plan, err := generator.Generate(context.Background(), llmTestState())
			if err != nil {
				t.Fatalf("Generate returned error: %v", err)
			}
			if !containsWarning(plan.Warnings, "LLM fallback:") || !containsWarning(plan.Warnings, "category="+tt.expectedCategory) {
				t.Fatalf("expected %s fallback warning, got %#v", tt.expectedCategory, plan.Warnings)
			}
		})
	}
}

type fakeLLMClient struct {
	plan *domain.TravelPlan
	err  error
}

func (c fakeLLMClient) GenerateTravelPlan(context.Context, TravelPlanningState) (*llmPlanResult, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &llmPlanResult{Plan: c.plan, PromptVersion: travelPlanPromptVersion}, nil
}

func llmTestState() TravelPlanningState {
	req := llmTestRequest()
	plan := llmValidPlan(req)
	return TravelPlanningState{
		Request:               req,
		NormalizedDestination: req.DestinationCity,
		NormalizedDays:        req.Days,
		NormalizedBudget:      req.Budget,
		Interests:             req.Interests,
		TransportMode:         req.TransportMode,
		Pace:                  req.Pace,
		Budget:                plan.Budget,
		Itinerary:             plan.Days,
	}
}

func llmTestRequest() domain.TravelRequest {
	return domain.TravelRequest{
		ID:              "case_llm",
		DepartureCity:   "上海",
		DestinationCity: "杭州",
		Days:            2,
		Budget:          3000,
		Interests:       []string{"自然风光", "美食"},
		TransportMode:   "train_taxi",
		Pace:            "balanced",
	}
}

func llmValidPlan(req domain.TravelRequest) domain.TravelPlan {
	return planWithDays(req, req.Days)
}

func planWithDays(req domain.TravelRequest, days int) domain.TravelPlan {
	planDays := make([]domain.TravelDay, 0, days)
	for day := 1; day <= days; day++ {
		planDays = append(planDays, domain.TravelDay{
			Day:   day,
			Theme: "杭州城市体验",
			Items: []domain.TravelItem{
				{
					Time:            "09:30",
					Type:            "sightseeing",
					Name:            "西湖",
					Address:         "杭州西湖景区",
					Reason:          "匹配杭州自然风光偏好",
					EstimatedCost:   80,
					Cost:            domain.AvailableCost(80, "per_person", "amap.poi.biz_ext.cost", true),
					DurationMinutes: 120,
				},
			},
		})
	}
	return domain.TravelPlan{
		Title:   "杭州2日旅行计划",
		Summary: "围绕杭州安排2天路线。",
		Days:    planDays,
		Budget: domain.TravelBudget{
			Transport:  500,
			Food:       600,
			Hotel:      1000,
			Ticket:     200,
			Total:      2300,
			KnownTotal: 2300,
			Complete:   true,
			Currency:   "CNY",
			Items: []domain.BudgetLine{
				availableBudgetLineForTest("transport", "市内交通", 500),
				availableBudgetLineForTest("food", "餐饮", 600),
				availableBudgetLineForTest("hotel", "住宿", 1000),
				availableBudgetLineForTest("ticket", "门票", 200),
			},
			Missing: []string{},
		},
		Warnings: []string{},
	}
}

func planWithBudget(req domain.TravelRequest, total float64) domain.TravelPlan {
	plan := llmValidPlan(req)
	plan.Budget.Total = total
	plan.Budget.KnownTotal = total
	return plan
}

func availableBudgetLineForTest(key, label string, amount float64) domain.BudgetLine {
	value := amount
	return domain.BudgetLine{
		Key:      key,
		Label:    label,
		Amount:   &value,
		Currency: "CNY",
		Status:   domain.CostAvailable,
		Source:   "test",
		Included: true,
	}
}

func mustMarshal(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	return string(data)
}

func containsAll(values []string, expected ...string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range expected {
		if _, ok := seen[value]; !ok {
			return false
		}
	}
	return true
}

func containsWarning(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}

func writeLLMResponse(t *testing.T, w http.ResponseWriter, arguments string, usage tokenUsage) {
	t.Helper()
	payload := map[string]any{
		"choices": []map[string]any{
			{
				"message": map[string]any{
					"tool_calls": []map[string]any{
						{
							"type": "function",
							"function": map[string]any{
								"name":      submitTravelPlanToolName,
								"arguments": arguments,
							},
						},
					},
				},
			},
		},
	}
	if usage.PromptTokens > 0 || usage.CompletionTokens > 0 || usage.TotalTokens > 0 {
		payload["usage"] = usage
	}
	writeRawJSON(t, w, mustMarshal(t, payload))
}

func writeLLMStreamResponse(t *testing.T, w http.ResponseWriter, toolName, arguments string, usage tokenUsage) {
	t.Helper()
	payload := map[string]any{
		"choices": []map[string]any{
			{
				"index": 0,
				"delta": map[string]any{
					"tool_calls": []map[string]any{
						{
							"index": 0,
							"type":  "function",
							"function": map[string]any{
								"name":      toolName,
								"arguments": arguments,
							},
						},
					},
				},
				"finish_reason": "tool_calls",
			},
		},
	}
	if usage.PromptTokens > 0 || usage.CompletionTokens > 0 || usage.TotalTokens > 0 {
		payload["usage"] = usage
	}
	writeRawSSE(t, w, mustMarshal(t, payload))
}

func writeLLMStreamContent(t *testing.T, w http.ResponseWriter, content string) {
	t.Helper()
	payload := map[string]any{
		"choices": []map[string]any{
			{
				"index": 0,
				"delta": map[string]any{
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
	}
	writeRawSSE(t, w, mustMarshal(t, payload))
}

func writeRawSSE(t *testing.T, w http.ResponseWriter, data string) {
	t.Helper()
	w.Header().Set("Content-Type", "text/event-stream")
	if _, err := w.Write([]byte("data: " + data + "\n\n")); err != nil {
		t.Fatalf("write sse data failed: %v", err)
	}
	if _, err := w.Write([]byte("data: [DONE]\n\n")); err != nil {
		t.Fatalf("write sse done failed: %v", err)
	}
}

func writeRawJSON(t *testing.T, w http.ResponseWriter, data string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(data)); err != nil {
		t.Fatalf("write raw json failed: %v", err)
	}
}

func assertStreamRequest(t *testing.T, r *http.Request) {
	t.Helper()
	var payload struct {
		Stream bool `json:"stream"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if !payload.Stream {
		t.Fatal("expected DeepSeek request stream=true")
	}
}
