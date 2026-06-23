package eino

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"travel-agent/internal/domain"
)

func TestDeepSeekStrictToolSchemaPayload(t *testing.T) {
	req := buildChatCompletionRequest(LLMConfig{
		Provider: "deepseek",
		Model:    "deepseek-v4-flash",
	}, []chatMessage{{Role: "user", Content: "plan"}}, travelPlanJSONSchema())

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
	}, []chatMessage{{Role: "user", Content: "plan"}}, travelPlanJSONSchema())

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
	if cfg.APIKey != "" {
		t.Fatal("default config must not contain an API key")
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
	if !containsWarning(plan.Warnings, "LLM fallback used") {
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
	if _, err := extractTravelPlanPayload(LLMConfig{Provider: "deepseek"}, data); err == nil {
		t.Fatal("expected missing tool call error")
	}
}

type fakeLLMClient struct {
	plan *domain.TravelPlan
	err  error
}

func (c fakeLLMClient) GenerateTravelPlan(context.Context, TravelPlanningState) (*domain.TravelPlan, error) {
	return c.plan, c.err
}

func llmTestState() TravelPlanningState {
	req := llmTestRequest()
	return TravelPlanningState{
		Request:               req,
		NormalizedDestination: req.DestinationCity,
		NormalizedDays:        req.Days,
		NormalizedBudget:      req.Budget,
		Interests:             req.Interests,
		TransportMode:         req.TransportMode,
		Pace:                  req.Pace,
		Budget: domain.TravelBudget{
			Transport: 500,
			Food:      600,
			Hotel:     1000,
			Ticket:    200,
			Total:     2300,
		},
		Itinerary: llmValidPlan(req).Days,
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
			Transport: 500,
			Food:      600,
			Hotel:     1000,
			Ticket:    200,
			Total:     2300,
		},
		Warnings: []string{},
	}
}

func planWithBudget(req domain.TravelRequest, total float64) domain.TravelPlan {
	plan := llmValidPlan(req)
	plan.Budget.Total = total
	return plan
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
