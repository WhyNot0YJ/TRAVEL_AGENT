package eino

import "time"

func appendTrace(state TravelPlanningState, step, message string, started time.Time, success bool) TravelPlanningState {
	state.Trace = append(state.Trace, TraceEvent{
		Step:       step,
		Message:    message,
		DurationMs: time.Since(started).Milliseconds(),
		Success:    success,
	})
	return state
}
