package harness

import (
	"context"
	"errors"
	"time"

	"travel-agent/internal/agent"
)

// Runner orchestrates dataset loading, planner calls, evaluation, and report writing.
type Runner struct {
	DatasetPath string
	PlannerType string
	Planner     agent.TravelPlanner
	Evaluator   *Evaluator
	Writer      ReportWriter
	Now         func() time.Time
}

// NewRunner wires the default harness components around a TravelPlanner.
func NewRunner(datasetPath string, planner agent.TravelPlanner, writer ReportWriter) *Runner {
	return &Runner{
		DatasetPath: datasetPath,
		PlannerType: "unknown",
		Planner:     planner,
		Evaluator:   NewEvaluator(),
		Writer:      writer,
		Now:         time.Now,
	}
}

// Run executes every case and continues after individual planner failures.
func (r *Runner) Run(ctx context.Context) (Report, error) {
	if r.Planner == nil {
		return Report{}, errors.New("planner is required")
	}
	if r.Evaluator == nil {
		r.Evaluator = NewEvaluator()
	}
	if r.Now == nil {
		r.Now = time.Now
	}
	if r.PlannerType == "" {
		r.PlannerType = "unknown"
	}

	cases, err := LoadCases(r.DatasetPath)
	if err != nil {
		return Report{}, err
	}

	results := make([]CaseResult, 0, len(cases))
	for _, tc := range cases {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		start := time.Now()
		plan, planErr := r.Planner.Plan(ctx, tc.Input)
		durationMs := time.Since(start).Milliseconds()
		results = append(results, r.Evaluator.Evaluate(tc, plan, planErr, durationMs))
	}

	report := Report{
		GeneratedAt: r.Now().UTC(),
		PlannerType: r.PlannerType,
		Summary:     CalculateSummary(results),
		Cases:       results,
	}
	if r.Writer != nil {
		if err := r.Writer.Write(ctx, report); err != nil {
			return Report{}, err
		}
	}
	return report, nil
}
