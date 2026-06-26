package eino

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// chatCompletionStreamChunk is one SSE frame in the OpenAI/DeepSeek streaming
// chat completion wire format.
type chatCompletionStreamChunk struct {
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content   string                   `json:"content"`
			ToolCalls []chatToolCallStreamPart `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *tokenUsage `json:"usage"`
}

type chatToolCallStreamPart struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// streamResult collects everything observed from a streaming chat completion.
type streamResult struct {
	Content           string
	ToolCallArguments map[int]string
	ToolCallNames     map[int]string
	Usage             LLMTokenUsage
	Duration          time.Duration
	FinishReason      string
	ChunksReceived    int
}

// chatCompletionStream issues a streaming chat completion request and invokes
// onDelta for each non-empty content chunk. Tool-call argument fragments are
// accumulated by index and (optionally) reported to onToolArgs after each
// frame so the caller can scan the partial JSON for fields it wants to surface
// progressively (e.g. the "summary" field of a strict tool call).
//
// The method does not honor c.httpClient.Timeout; pass a ctx with deadline
// instead so a slow stream does not get killed mid-frame.
func (c *openAICompatibleClient) chatCompletionStream(
	ctx context.Context,
	payload chatCompletionRequest,
	onDelta func(string),
	onToolArgs func(index int, accumulated string),
) (*streamResult, error) {
	started := time.Now()
	payload.Stream = true

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal stream chat completion request: %w", err)
	}
	endpoint := chatCompletionsEndpoint(c.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build stream chat completion request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	streamingClient := &http.Client{Timeout: 0, Transport: c.httpClient.Transport}
	log.Printf("[%s] call purpose=chat_completion_stream provider=%s model=%s endpoint=%s stream=true", llmAPILogLabel(c.config), c.config.Provider, payload.Model, endpoint)
	resp, err := streamingClient.Do(req)
	if err != nil {
		log.Printf("[%s] return purpose=chat_completion_stream error=%v duration_ms=%d", llmAPILogLabel(c.config), err, time.Since(started).Milliseconds())
		return nil, fmt.Errorf("call LLM stream provider: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		preview, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		log.Printf("[%s] return purpose=chat_completion_stream status=%d bytes=%d duration_ms=%d body=%s", llmAPILogLabel(c.config), resp.StatusCode, len(preview), time.Since(started).Milliseconds(), string(preview))
		return nil, fmt.Errorf("LLM stream provider returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(preview)))
	}
	log.Printf("[%s] return purpose=chat_completion_stream status=%d stream_open=true duration_ms=%d", llmAPILogLabel(c.config), resp.StatusCode, time.Since(started).Milliseconds())

	result := &streamResult{
		ToolCallArguments: map[int]string{},
		ToolCallNames:     map[int]string{},
	}
	var contentBuf strings.Builder
	reader := bufio.NewReader(resp.Body)

	// SSE framing: events are separated by a blank line. Within an event,
	// any line starting with "data: " contributes to the JSON payload (multiple
	// data: lines are joined by '\n' per the spec).
	var eventLines []string

	flushEvent := func() error {
		defer func() { eventLines = eventLines[:0] }()
		if len(eventLines) == 0 {
			return nil
		}
		var dataLines []string
		for _, line := range eventLines {
			if strings.HasPrefix(line, "data:") {
				dataLines = append(dataLines, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			}
		}
		if len(dataLines) == 0 {
			return nil
		}
		data := strings.Join(dataLines, "\n")
		if strings.TrimSpace(data) == "[DONE]" {
			return io.EOF
		}
		var chunk chatCompletionStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Be liberal: skip malformed frames rather than tearing down the stream.
			return nil
		}
		result.ChunksReceived++
		if chunk.Usage != nil {
			result.Usage = llmTokenUsage(*chunk.Usage)
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				contentBuf.WriteString(choice.Delta.Content)
				if onDelta != nil {
					onDelta(choice.Delta.Content)
				}
			}
			for _, tc := range choice.Delta.ToolCalls {
				if tc.Function.Name != "" {
					result.ToolCallNames[tc.Index] = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					result.ToolCallArguments[tc.Index] += tc.Function.Arguments
					if onToolArgs != nil {
						onToolArgs(tc.Index, result.ToolCallArguments[tc.Index])
					}
				}
			}
			if choice.FinishReason != "" {
				result.FinishReason = choice.FinishReason
			}
		}
		return nil
	}

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			trimmed := strings.TrimRight(string(line), "\r\n")
			if trimmed == "" {
				if flushErr := flushEvent(); flushErr != nil {
					if errors.Is(flushErr, io.EOF) {
						break
					}
					return nil, flushErr
				}
			} else {
				eventLines = append(eventLines, trimmed)
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				// Flush any tail event before exiting.
				if flushErr := flushEvent(); flushErr != nil && !errors.Is(flushErr, io.EOF) {
					return nil, flushErr
				}
				break
			}
			return nil, fmt.Errorf("read SSE stream: %w", err)
		}
	}

	result.Content = contentBuf.String()
	result.Duration = time.Since(started)
	log.Printf("[%s] value purpose=chat_completion_stream chunks=%d content_chars=%d finish_reason=%s prompt_tokens=%d completion_tokens=%d total_tokens=%d duration_ms=%d", llmAPILogLabel(c.config), result.ChunksReceived, len(result.Content), result.FinishReason, result.Usage.PromptTokens, result.Usage.CompletionTokens, result.Usage.TotalTokens, result.Duration.Milliseconds())
	log.Printf("[%s] value purpose=chat_completion_stream content=%q tool_args=%s", llmAPILogLabel(c.config), result.Content, streamToolArgsLogValue(result.ToolCallArguments))
	return result, nil
}

func streamToolArgsLogValue(args map[int]string) string {
	if len(args) == 0 {
		return "{}"
	}
	data, err := json.Marshal(args)
	if err != nil {
		return fmt.Sprintf("%q", args)
	}
	return string(data)
}

func extractStreamToolPayload(provider, model string, result *streamResult, toolName string) (string, error) {
	if result == nil {
		return "", fmt.Errorf("LLM stream returned nil result")
	}
	if strings.EqualFold(provider, "deepseek") && modelSupportsToolCall(model) {
		for index, args := range result.ToolCallArguments {
			if strings.TrimSpace(args) == "" {
				continue
			}
			name := result.ToolCallNames[index]
			if name == "" || name == toolName {
				return args, nil
			}
		}
		return "", fmt.Errorf("LLM stream did not call %s", toolName)
	}
	for _, args := range result.ToolCallArguments {
		if strings.TrimSpace(args) != "" {
			return args, nil
		}
	}
	if content := strings.TrimSpace(result.Content); content != "" {
		return content, nil
	}
	return "", fmt.Errorf("LLM stream returned no tool arguments or content")
}
