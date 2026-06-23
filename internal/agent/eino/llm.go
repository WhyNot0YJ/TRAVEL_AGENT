package eino

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	GenerateTravelPlan(ctx context.Context, state TravelPlanningState) (*domain.TravelPlan, error)
}

type llmPlanGenerator struct {
	client     travelPlanLLMClient
	fallback   TravelPlanGenerator
	maxRetries int
}

func newDefaultPlanGenerator() TravelPlanGenerator {
	cfg := loadLLMConfigFromEnv()
	fallback := deterministicPlanGenerator{}
	if !cfg.Enabled {
		return fallback
	}
	if cfg.APIKey == "" || cfg.BaseURL == "" || cfg.Model == "" {
		return llmPlanGenerator{
			fallback: fallback,
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
	if g.client == nil {
		lastErr = fmt.Errorf("LLM is enabled but not configured")
	} else {
		attempts := g.maxRetries + 1
		if attempts <= 0 {
			attempts = 1
		}
		for attempt := 0; attempt < attempts; attempt++ {
			plan, err := g.client.GenerateTravelPlan(ctx, state)
			if err == nil {
				return plan, nil
			}
			lastErr = err
		}
	}

	plan, err := g.fallback.Generate(ctx, state)
	if err != nil {
		return nil, err
	}
	plan.Warnings = append(plan.Warnings, fmt.Sprintf("LLM fallback used: %v", lastErr))
	return plan, nil
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

func (c *openAICompatibleClient) GenerateTravelPlan(ctx context.Context, state TravelPlanningState) (*domain.TravelPlan, error) {
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

	rawPlan, err := extractTravelPlanPayload(c.config, respBody)
	if err != nil {
		return nil, err
	}
	return parseTravelPlanArguments(rawPlan, state.Request)
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

func extractTravelPlanPayload(cfg LLMConfig, data []byte) (string, error) {
	var resp chatCompletionResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("decode LLM response: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM response has no choices")
	}

	message := resp.Choices[0].Message
	if strings.EqualFold(cfg.Provider, "deepseek") {
		for _, toolCall := range message.ToolCalls {
			if toolCall.Type == "function" && toolCall.Function.Name == submitTravelPlanToolName {
				if strings.TrimSpace(toolCall.Function.Arguments) == "" {
					return "", fmt.Errorf("submit_travel_plan tool call has empty arguments")
				}
				return toolCall.Function.Arguments, nil
			}
		}
		return "", fmt.Errorf("LLM response did not call %s", submitTravelPlanToolName)
	}
	if strings.TrimSpace(message.Content) == "" {
		return "", fmt.Errorf("LLM response content is empty")
	}
	return message.Content, nil
}

func chatCompletionsEndpoint(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(baseURL, "/chat/completions") {
		return baseURL
	}
	return baseURL + "/chat/completions"
}
