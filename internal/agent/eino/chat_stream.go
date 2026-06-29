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
		"你是中文旅游需求收集助手。",
		"你会收到已经从用户消息中抽取出的结构化字段，以及仍然缺失的必填字段。",
		"请用简洁中文回复用户：确认已经收集到的信息，只追问缺失的必填字段。",
		"如果必填字段都已具备，请告诉用户信息已齐，可以生成行程。",
		"不要输出 JSON 或 Markdown，只输出一个简短段落。",
		"提示词版本：" + chatReplyPromptVersion + "。",
	}, " ")
	var sb strings.Builder
	if prior != nil {
		sb.WriteString("已收集信息：")
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
			sb.WriteString("缺失字段=" + strings.Join(prior.Missing, "、") + "; ")
		}
		if prior.IsComplete {
			sb.WriteString("信息是否完整=true; ")
		}
	}
	sb.WriteString("\n用户本轮消息：" + userMessage)
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
