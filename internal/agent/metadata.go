package agent

import (
	"context"
	"time"
)

type PlannerMetadata struct {
	PlannerType   string              `json:"planner_type,omitempty"`
	PromptVersion string              `json:"prompt_version,omitempty"`
	ToolMode      string              `json:"tool_mode,omitempty"`
	TokenUsage    TokenUsage          `json:"token_usage,omitempty"`
	Trace         []PlannerTraceEvent `json:"trace,omitempty"`
}

type PlannerTraceEvent struct {
	Name           string        `json:"name"`
	Kind           string        `json:"kind"`
	Provider       string        `json:"provider,omitempty"`
	Status         string        `json:"status"`
	Duration       time.Duration `json:"duration"`
	FallbackReason string        `json:"fallback_reason,omitempty"`
}

type TokenUsage struct {
	PromptTokens     int  `json:"prompt_tokens,omitempty"`
	CompletionTokens int  `json:"completion_tokens,omitempty"`
	TotalTokens      int  `json:"total_tokens,omitempty"`
	Known            bool `json:"known"`
}

type TraceablePlanner interface {
	Metadata() PlannerMetadata
}

type PlannerEventReporter interface {
	ReportPlannerEvent(ctx context.Context, event PlannerTraceEvent)
}

// LLMDeltaReporter receives token-level streaming deltas from the LLM.
// Implementations are expected to forward deltas to the user-visible SSE channel.
// SawAnyDelta lets non-streaming code paths know whether streaming has already
// produced output, so they can avoid re-emitting the same content via fallbacks.
type LLMDeltaReporter interface {
	ReportLLMDelta(ctx context.Context, delta string)
	ReportLLMDone(ctx context.Context, fullText string)
	SawAnyDelta() bool
}

type plannerEventReporterKey struct{}
type llmDeltaReporterKey struct{}
type plannerOptionsKey struct{}

type PlannerOptions struct {
	TestMode  bool
	AgentMode string
}

func WithPlannerOptions(ctx context.Context, options PlannerOptions) context.Context {
	return context.WithValue(ctx, plannerOptionsKey{}, options)
}

func PlannerOptionsFromContext(ctx context.Context) PlannerOptions {
	options, _ := ctx.Value(plannerOptionsKey{}).(PlannerOptions)
	return options
}

func WithPlannerEventReporter(ctx context.Context, reporter PlannerEventReporter) context.Context {
	if reporter == nil {
		return ctx
	}
	return context.WithValue(ctx, plannerEventReporterKey{}, reporter)
}

func ReportPlannerEvent(ctx context.Context, event PlannerTraceEvent) {
	reporter, ok := ctx.Value(plannerEventReporterKey{}).(PlannerEventReporter)
	if !ok || reporter == nil {
		return
	}
	reporter.ReportPlannerEvent(ctx, event)
}

func WithLLMDeltaReporter(ctx context.Context, reporter LLMDeltaReporter) context.Context {
	if reporter == nil {
		return ctx
	}
	return context.WithValue(ctx, llmDeltaReporterKey{}, reporter)
}

func LLMDeltaReporterFromContext(ctx context.Context) LLMDeltaReporter {
	reporter, _ := ctx.Value(llmDeltaReporterKey{}).(LLMDeltaReporter)
	return reporter
}
