package eino

import (
	"context"
	"time"

	"travel-agent/internal/agent"
)

func appendTrace(ctx context.Context, state TravelPlanningState, step, message string, started time.Time, success bool) TravelPlanningState {
	duration := time.Since(started)
	state.Trace = append(state.Trace, TraceEvent{
		Step:       step,
		Message:    message,
		DurationMs: duration.Milliseconds(),
		Success:    success,
	})
	status := "success"
	if !success {
		status = "error"
	}
	agent.ReportPlannerEvent(ctx, agent.PlannerTraceEvent{
		Name:           step,
		Kind:           "node",
		Status:         status,
		Duration:       duration,
		FallbackReason: message,
	})
	return state
}
