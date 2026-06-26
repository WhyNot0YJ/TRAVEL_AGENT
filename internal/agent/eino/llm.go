package eino

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"travel-agent/internal/agent"
	"travel-agent/internal/domain"
)

// TravelPlanGenerator generates the final plan from graph state.
type TravelPlanGenerator interface {
	Generate(ctx context.Context, state TravelPlanningState) (*domain.TravelPlan, error)
}

type deterministicPlanGenerator struct{}

func (g deterministicPlanGenerator) Generate(ctx context.Context, state TravelPlanningState) (*domain.TravelPlan, error) {
	return generateDeterministicTravelPlan(ctx, state)
}

type travelPlanLLMClient interface {
	GenerateTravelPlan(ctx context.Context, state TravelPlanningState) (*llmPlanResult, error)
}

type llmPlanGenerator struct {
	client         travelPlanLLMClient
	fallback       TravelPlanGenerator
	maxRetries     int
	disabledReason string
}

func newDefaultPlanGenerator() TravelPlanGenerator {
	cfg := loadLLMConfigFromEnv()
	fallback := deterministicPlanGenerator{}
	if !cfg.Enabled {
		return llmPlanGenerator{
			fallback:       fallback,
			disabledReason: "disabled",
		}
	}
	if cfg.APIKey == "" {
		return llmPlanGenerator{
			fallback:       fallback,
			disabledReason: "missing_api_key",
		}
	}
	if cfg.BaseURL == "" || cfg.Model == "" {
		return llmPlanGenerator{
			fallback:       fallback,
			disabledReason: "provider_error",
		}
	}
	return llmPlanGenerator{
		client:     newOpenAICompatibleClient(cfg),
		fallback:   fallback,
		maxRetries: cfg.MaxRetries,
	}
}

func (g llmPlanGenerator) Generate(ctx context.Context, state TravelPlanningState) (*domain.TravelPlan, error) {
	if g.fallback == nil {
		g.fallback = deterministicPlanGenerator{}
	}
	if agent.PlannerOptionsFromContext(ctx).TestMode {
		plan, err := g.fallback.Generate(ctx, state)
		if err != nil {
			return nil, err
		}
		plan.Warnings = append(plan.Warnings, llmFallbackWarning(fmt.Errorf("test_mode"), 0, 0))
		return plan, nil
	}

	var lastErr error
	attemptsMade := 0
	started := time.Now()
	if g.disabledReason != "" {
		lastErr = fmt.Errorf("%s", g.disabledReason)
	} else if g.client == nil {
		lastErr = fmt.Errorf("missing_api_key")
	} else {
		attempts := g.maxRetries + 1
		if attempts <= 0 {
			attempts = 1
		}
		for attempt := 0; attempt < attempts; attempt++ {
			attemptsMade++
			result, err := g.client.GenerateTravelPlan(ctx, state)
			if err == nil {
				if result == nil || result.Plan == nil {
					lastErr = fmt.Errorf("LLM returned empty plan")
					break
				}
				appendLLMTraceWarning(result.Plan, result)
				return result.Plan, nil
			}
			lastErr = err
		}
	}

	plan, err := g.fallback.Generate(ctx, state)
	if err != nil {
		return nil, err
	}
	plan.Warnings = append(plan.Warnings, llmFallbackWarning(lastErr, attemptsMade, time.Since(started)))
	return plan, nil
}

type llmPlanResult struct {
	Plan          *domain.TravelPlan
	PromptVersion string
	Duration      time.Duration
	TokenUsage    LLMTokenUsage
}

type LLMTokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	Known            bool
}

type openAICompatibleClient struct {
	config     LLMConfig
	httpClient *http.Client
}

func newOpenAICompatibleClient(cfg LLMConfig) *openAICompatibleClient {
	return &openAICompatibleClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *openAICompatibleClient) GenerateTravelPlan(ctx context.Context, state TravelPlanningState) (*llmPlanResult, error) {
	options := agent.PlannerOptionsFromContext(ctx)
	reporter := agent.LLMDeltaReporterFromContext(ctx)
	if c.config.StreamEnabled {
		return c.generateTravelPlanStreaming(ctx, state, options.AgentMode, reporter)
	}
	return c.generateTravelPlanBuffered(ctx, state, options.AgentMode)
}

// generateTravelPlanBuffered is the original non-streaming code path. We keep
// it for the harness, for reporter-less tests, and for the rollback flag.
func (c *openAICompatibleClient) generateTravelPlanBuffered(ctx context.Context, state TravelPlanningState, agentMode string) (*llmPlanResult, error) {
	started := time.Now()

	messages, err := buildTravelPlanMessages(state, agentMode)
	if err != nil {
		return nil, err
	}
	payload := buildChatCompletionRequest(c.config, messages, travelPlanJSONSchema(), submitTravelPlanToolName, "Submit the final structured travel plan.", agentMode)

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal chat completion request: %w", err)
	}
	endpoint := chatCompletionsEndpoint(c.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build chat completion request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[%s] call purpose=generate_travel_plan provider=%s model=%s endpoint=%s stream=false", llmAPILogLabel(c.config), c.config.Provider, payload.Model, endpoint)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[%s] return purpose=generate_travel_plan error=%v duration_ms=%d", llmAPILogLabel(c.config), err, time.Since(started).Milliseconds())
		return nil, fmt.Errorf("call LLM provider: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		log.Printf("[%s] return purpose=generate_travel_plan status=%d error=%v duration_ms=%d", llmAPILogLabel(c.config), resp.StatusCode, err, time.Since(started).Milliseconds())
		return nil, fmt.Errorf("read LLM response: %w", err)
	}
	log.Printf("[%s] return purpose=generate_travel_plan status=%d bytes=%d duration_ms=%d body=%s", llmAPILogLabel(c.config), resp.StatusCode, len(respBody), time.Since(started).Milliseconds(), string(respBody))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("LLM provider returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	rawPlan, usage, err := extractTravelPlanPayload(c.config, respBody)
	if err != nil {
		return nil, err
	}
	plan, err := parseTravelPlanArguments(rawPlan, state.Request)
	if err != nil {
		return nil, err
	}
	log.Printf("[%s] value purpose=generate_travel_plan title=%q days=%d prompt_tokens=%d completion_tokens=%d total_tokens=%d", llmAPILogLabel(c.config), plan.Title, len(plan.Days), usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	return &llmPlanResult{
		Plan:          plan,
		PromptVersion: travelPlanPromptVersion,
		Duration:      time.Since(started),
		TokenUsage:    usage,
	}, nil
}

// generateTravelPlanStreaming runs a single strict tool-call request with
// stream=true, progressively scans the partial JSON for the "summary" field,
// and emits each newly produced character of that field through the reporter.
// On completion the accumulated tool-call arguments are parsed exactly the
// same way the buffered path parses them, so business validation is identical.
func (c *openAICompatibleClient) generateTravelPlanStreaming(ctx context.Context, state TravelPlanningState, agentMode string, reporter agent.LLMDeltaReporter) (*llmPlanResult, error) {
	started := time.Now()

	messages, err := buildTravelPlanMessages(state, agentMode)
	if err != nil {
		return nil, err
	}
	payload := buildChatCompletionRequest(c.config, messages, travelPlanJSONSchema(), submitTravelPlanToolName, "Submit the final structured travel plan.", agentMode)

	// Track how much of the summary string the user has already seen so we
	// only emit the new tail on each frame.
	var emittedPrefix string
	onToolArgs := func(_ int, accumulated string) {
		current := extractSummarySoFar(accumulated)
		if len(current) <= len(emittedPrefix) {
			return
		}
		if !strings.HasPrefix(current, emittedPrefix) {
			// Defensive: extractor revised earlier output (shouldn't happen
			// with append-only token streaming, but worth handling). Replay.
			emittedPrefix = ""
		}
		delta := current[len(emittedPrefix):]
		emittedPrefix = current
		if delta != "" && reporter != nil {
			reporter.ReportLLMDelta(ctx, delta)
		}
	}

	result, err := c.chatCompletionStream(ctx, payload, nil, onToolArgs)
	if err != nil {
		return nil, err
	}

	rawArgs, err := extractStreamToolPayload(c.config.Provider, payload.Model, result, submitTravelPlanToolName)
	if err != nil {
		return nil, err
	}

	plan, err := parseTravelPlanArguments(rawArgs, state.Request)
	if err != nil {
		return nil, err
	}

	// Flush any remaining summary tail (handles the case where the closing
	// quote of summary arrived in the same frame as the rest of the JSON).
	if final := extractSummarySoFar(rawArgs); len(final) > len(emittedPrefix) {
		if strings.HasPrefix(final, emittedPrefix) {
			tail := final[len(emittedPrefix):]
			if tail != "" && reporter != nil {
				reporter.ReportLLMDelta(ctx, tail)
			}
			emittedPrefix = final
		}
	}
	if emittedPrefix != "" && reporter != nil {
		reporter.ReportLLMDone(ctx, emittedPrefix)
	}

	usage := result.Usage
	log.Printf("[%s] value purpose=generate_travel_plan_stream title=%q days=%d prompt_tokens=%d completion_tokens=%d total_tokens=%d duration_ms=%d", llmAPILogLabel(c.config), plan.Title, len(plan.Days), usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, result.Duration.Milliseconds())
	return &llmPlanResult{
		Plan:          plan,
		PromptVersion: travelPlanPromptVersion,
		Duration:      time.Since(started),
		TokenUsage:    usage,
	}, nil
}

type chatCompletionRequest struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	Tools          []chatTool    `json:"tools,omitempty"`
	ToolChoice     any           `json:"tool_choice,omitempty"`
	Thinking       *chatThinking `json:"thinking,omitempty"`
	ResponseFormat any           `json:"response_format,omitempty"`
	Temperature    float64       `json:"temperature"`
	Stream         bool          `json:"stream"`
}

type chatThinking struct {
	Type string `json:"type"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatTool struct {
	Type     string           `json:"type"`
	Function chatToolFunction `json:"function"`
}

type chatToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
	Strict      bool           `json:"strict"`
}

// modelForMode returns the configured DeepSeek model for the given agent mode.
// Quick mode → flash, expert mode → pro. cfg.Model still serves as a global
// override when QuickModel/ExpertModel are unset, preserving backward compat.
func modelForMode(cfg LLMConfig, agentMode string) string {
	if strings.EqualFold(agentMode, "expert") && strings.TrimSpace(cfg.ExpertModel) != "" {
		return cfg.ExpertModel
	}
	if strings.TrimSpace(cfg.QuickModel) != "" {
		return cfg.QuickModel
	}
	return cfg.Model
}

// modelSupportsToolCall reports whether the chosen model is expected to honor
// strict OpenAI-style function/tool calls. Reasoner-style models do not.
func modelSupportsToolCall(model string) bool {
	lower := strings.ToLower(model)
	if strings.Contains(lower, "reasoner") {
		return false
	}
	return true
}

func buildChatCompletionRequest(cfg LLMConfig, messages []chatMessage, schema map[string]any, toolName, toolDesc, agentMode string) chatCompletionRequest {
	model := modelForMode(cfg, agentMode)
	req := chatCompletionRequest{
		Model:       model,
		Messages:    messages,
		Temperature: 0.2,
		Stream:      false,
	}
	supportsTools := modelSupportsToolCall(model)
	if strings.EqualFold(cfg.Provider, "deepseek") && supportsTools {
		name := toolName
		if name == "" {
			name = submitTravelPlanToolName
		}
		desc := toolDesc
		if desc == "" {
			desc = "Submit the final structured output."
		}
		req.Tools = []chatTool{
			{
				Type: "function",
				Function: chatToolFunction{
					Name:        name,
					Description: desc,
					Parameters:  schema,
					Strict:      true,
				},
			},
		}
		req.ToolChoice = map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": name,
			},
		}
		req.Thinking = &chatThinking{Type: "disabled"}
		return req
	}

	// OpenAI-compatible json_schema path; also used as the reasoner fallback
	// when the chosen model is known not to support tool calls.
	req.ResponseFormat = map[string]any{
		"type": "json_schema",
		"json_schema": map[string]any{
			"name":   "travel_plan",
			"strict": true,
			"schema": schema,
		},
	}
	return req
}

type chatCompletionResponse struct {
	Choices []chatChoice `json:"choices"`
	Usage   tokenUsage   `json:"usage"`
}

type tokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type chatChoice struct {
	Message chatResponseMessage `json:"message"`
}

type chatResponseMessage struct {
	Content   string         `json:"content"`
	ToolCalls []chatToolCall `json:"tool_calls"`
}

type chatToolCall struct {
	Type     string                   `json:"type"`
	Function chatToolCallFunctionBody `json:"function"`
}

type chatToolCallFunctionBody struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func extractTravelPlanPayload(cfg LLMConfig, data []byte) (string, LLMTokenUsage, error) {
	var resp chatCompletionResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", LLMTokenUsage{}, fmt.Errorf("decode LLM response: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", llmTokenUsage(resp.Usage), fmt.Errorf("LLM response has no choices")
	}

	message := resp.Choices[0].Message
	if strings.EqualFold(cfg.Provider, "deepseek") {
		for _, toolCall := range message.ToolCalls {
			if toolCall.Type == "function" && toolCall.Function.Name == submitTravelPlanToolName {
				if strings.TrimSpace(toolCall.Function.Arguments) == "" {
					return "", llmTokenUsage(resp.Usage), fmt.Errorf("submit_travel_plan tool call has empty arguments")
				}
				return toolCall.Function.Arguments, llmTokenUsage(resp.Usage), nil
			}
		}
		// Reasoner-style models cannot emit tool_calls; accept content as JSON.
		// Only the configured model name unlocks this fallback so providers that
		// silently drop tool_calls still surface as an error.
		if !modelSupportsToolCall(cfg.Model) && strings.TrimSpace(message.Content) != "" {
			return strings.TrimSpace(message.Content), llmTokenUsage(resp.Usage), nil
		}
		return "", llmTokenUsage(resp.Usage), fmt.Errorf("LLM response did not call %s", submitTravelPlanToolName)
	}
	if strings.TrimSpace(message.Content) == "" {
		return "", llmTokenUsage(resp.Usage), fmt.Errorf("LLM response content is empty")
	}
	return message.Content, llmTokenUsage(resp.Usage), nil
}

func chatCompletionsEndpoint(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(baseURL, "/chat/completions") {
		return baseURL
	}
	return baseURL + "/chat/completions"
}

func llmTokenUsage(usage tokenUsage) LLMTokenUsage {
	known := usage.PromptTokens > 0 || usage.CompletionTokens > 0 || usage.TotalTokens > 0
	return LLMTokenUsage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
		Known:            known,
	}
}

func llmAPILogLabel(cfg LLMConfig) string {
	if strings.EqualFold(cfg.Provider, "deepseek") {
		return "DeepSeek API"
	}
	return "LLM API"
}

func appendLLMTraceWarning(plan *domain.TravelPlan, result *llmPlanResult) {
	if plan == nil || result == nil {
		return
	}
	promptVersion := result.PromptVersion
	if promptVersion == "" {
		promptVersion = travelPlanPromptVersion
	}
	usage := "prompt_tokens=unknown completion_tokens=unknown total_tokens=unknown"
	if result.TokenUsage.Known {
		usage = fmt.Sprintf("prompt_tokens=%d completion_tokens=%d total_tokens=%d",
			result.TokenUsage.PromptTokens,
			result.TokenUsage.CompletionTokens,
			result.TokenUsage.TotalTokens,
		)
	}
	plan.Warnings = append(plan.Warnings, fmt.Sprintf("LLM trace: prompt_version=%s duration_ms=%d %s",
		promptVersion,
		result.Duration.Milliseconds(),
		usage,
	))
}

func llmFallbackWarning(err error, attempts int, duration time.Duration) string {
	reason := "unknown"
	if err != nil {
		reason = err.Error()
	}
	category := classifyLLMFallback(reason, attempts)
	return fmt.Sprintf("LLM fallback: prompt_version=%s category=%s attempts=%d duration_ms=%d reason=%s",
		travelPlanPromptVersion,
		category,
		attempts,
		duration.Milliseconds(),
		reason,
	)
}

func classifyLLMFallback(reason string, attempts int) string {
	lower := strings.ToLower(strings.TrimSpace(reason))
	switch {
	case lower == "disabled":
		return "disabled"
	case lower == "test_mode":
		return "test_mode"
	case strings.Contains(lower, "missing_api_key"), strings.Contains(lower, "api key"):
		return "missing_api_key"
	case strings.Contains(lower, "timeout"), strings.Contains(lower, "deadline exceeded"):
		if attempts > 1 {
			return "retry_exhausted"
		}
		return "timeout"
	case strings.Contains(lower, "did not call"), strings.Contains(lower, "no choices"), strings.Contains(lower, "content is empty"):
		if attempts > 1 {
			return "retry_exhausted"
		}
		return "provider_error"
	case strings.Contains(lower, "decode llm response"), strings.Contains(lower, "decode travel plan"), strings.Contains(lower, "invalid character"):
		if attempts > 1 {
			return "retry_exhausted"
		}
		return "invalid_json"
	case strings.Contains(lower, "validation"), strings.Contains(lower, "expected"), strings.Contains(lower, "exceeds"), strings.Contains(lower, "negative"), strings.Contains(lower, "empty"):
		if attempts > 1 {
			return "retry_exhausted"
		}
		return "business_validation_failed"
	case attempts > 1:
		return "retry_exhausted"
	default:
		return "provider_error"
	}
}
