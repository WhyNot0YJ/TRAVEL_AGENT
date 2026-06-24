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

type plannerEventReporterKey struct{}

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
