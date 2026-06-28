package eino

import (
	"context"
	"strings"

	"travel-agent/internal/agent"
)

const chatReplyPromptVersion = "chat-reply-v1"

// streamChatReply rewrites the assistant's reply via streaming chat completion
// for the chat info collection link. Returns the accumulated text on success;
// returns "" on any error so the caller can fall back to a deterministic reply.
func (c *openAICompatibleClient) streamChatReply(ctx context.Context, prior *agent.TravelInfoResult, userMessage string, reporter agent.LLMDeltaReporter) string {
	if c == nil || reporter == nil || !c.config.StreamEnabled {
		return ""
	}
	messages := buildChatReplyMessages(prior, userMessage)
	cfg := c.config
	payload := buildChatCompletionRequest(cfg, messages, nil, "", "", "quick")
	payload.Tools = nil
	payload.ToolChoice = nil
	payload.ResponseFormat = nil
	payload.Thinking = nil

	result, err := c.chatCompletionStream(ctx, payload, func(delta string) { reporter.ReportLLMDelta(ctx, delta) }, nil)
	if err != nil {
		return ""
	}
	full := strings.TrimSpace(result.Content)
	if full == "" {
		return ""
	}
	reporter.ReportLLMDone(ctx, full)
	return full
}

func buildChatReplyMessages(prior *agent.TravelInfoResult, userMessage string) []chatMessage {
	system := strings.Join([]string{
		"You are a Chinese travel requirement collection assistant.",
		"You will be given the structured fields already extracted from the user's message and the missing required fields.",
		"Reply to the user in concise Chinese: confirm what you have, ask only for missing required fields.",
		"If every required field is present, tell the user the information is ready for itinerary generation.",
		"Do not output JSON or markdown. One short paragraph.",
		"Prompt version: " + chatReplyPromptVersion + ".",
	}, " ")
	var sb strings.Builder
	if prior != nil {
		sb.WriteString("Already collected: ")
		if prior.DepartureCity != "" {
			sb.WriteString("出发地=" + prior.DepartureCity + "; ")
		}
		if prior.DestinationCity != "" {
			sb.WriteString("目的地=" + prior.DestinationCity + "; ")
		}
		if prior.Days > 0 {
			sb.WriteString("天数=" + intToStr(prior.Days) + "; ")
		}
		if prior.Budget > 0 {
			sb.WriteString("预算=" + floatToStr(prior.Budget) + "; ")
		}
		if prior.Travelers > 0 {
			sb.WriteString("人数=" + intToStr(prior.Travelers) + "; ")
		}
		if len(prior.Interests) > 0 {
			sb.WriteString("偏好=" + strings.Join(prior.Interests, "、") + "; ")
		}
		if prior.DateRange != "" {
			sb.WriteString("日期=" + prior.DateRange + "; ")
		}
		if prior.Pace != "" {
			sb.WriteString("节奏=" + prior.Pace + "; ")
		}
		if prior.TransportMode != "" {
			sb.WriteString("交通=" + prior.TransportMode + "; ")
		}
		if prior.WalkingTolerance != "" {
			sb.WriteString("步行=" + prior.WalkingTolerance + "; ")
		}
		if len(prior.MustVisit) > 0 {
			sb.WriteString("必去=" + strings.Join(prior.MustVisit, "、") + "; ")
		}
		if len(prior.Avoid) > 0 {
			sb.WriteString("避开=" + strings.Join(prior.Avoid, "、") + "; ")
		}
		if len(prior.Missing) > 0 {
			sb.WriteString("Missing=" + strings.Join(prior.Missing, "、") + "; ")
		}
		if prior.IsComplete {
			sb.WriteString("IsComplete=true; ")
		}
	}
	sb.WriteString("\nUser said: " + userMessage)
	return []chatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: sb.String()},
	}
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

func floatToStr(f float64) string {
	return formatBudget(f)
}
