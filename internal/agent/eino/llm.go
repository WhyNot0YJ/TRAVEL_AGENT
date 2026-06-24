package eino

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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

	var lastErr error
	attemptsMade := 0
	started := time.Now()
	if g.disabledReason != "" {
		lastErr = fmt.Errorf(g.disabledReason)
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
	started := time.Now()
	messages, err := buildTravelPlanMessages(state)
	if err != nil {
		return nil, err
	}
	payload := buildChatCompletionRequest(c.config, messages, travelPlanJSONSchema())

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal chat completion request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatCompletionsEndpoint(c.config.BaseURL), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build chat completion request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call LLM provider: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read LLM response: %w", err)
	}
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
	ResponseFormat any           `json:"response_format,omitempty"`
	Temperature    float64       `json:"temperature"`
	Stream         bool          `json:"stream"`
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

func buildChatCompletionRequest(cfg LLMConfig, messages []chatMessage, schema map[string]any) chatCompletionRequest {
	req := chatCompletionRequest{
		Model:       cfg.Model,
		Messages:    messages,
		Temperature: 0.2,
		Stream:      false,
	}
	if strings.EqualFold(cfg.Provider, "deepseek") {
		req.Tools = []chatTool{
			{
				Type: "function",
				Function: chatToolFunction{
					Name:        submitTravelPlanToolName,
					Description: "Submit the final structured travel plan.",
					Parameters:  schema,
					Strict:      true,
				},
			},
		}
		req.ToolChoice = map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": submitTravelPlanToolName,
			},
		}
		return req
	}

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
