package harness

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
	Repeat      int
	Concurrency int
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
		Repeat:      1,
		Concurrency: 1,
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
	if r.Repeat <= 0 {
		r.Repeat = 1
	}
	if r.Concurrency <= 0 {
		r.Concurrency = 1
	}

	cases, err := LoadCases(r.DatasetPath)
	if err != nil {
		return Report{}, err
	}

	jobs := make([]TravelCase, 0, len(cases)*r.Repeat)
	for repeat := 1; repeat <= r.Repeat; repeat++ {
		for _, tc := range cases {
			if r.Repeat > 1 {
				tc.ID = fmt.Sprintf("%s#%d", tc.ID, repeat)
			}
			jobs = append(jobs, tc)
		}
	}

	results := make([]CaseResult, 0, len(jobs))
	if r.Concurrency == 1 {
		for _, tc := range jobs {
			if err := ctx.Err(); err != nil {
				return Report{}, err
			}
			results = append(results, r.runCase(ctx, tc))
		}
	} else {
		jobCh := make(chan TravelCase)
		resultCh := make(chan CaseResult, len(jobs))
		var wg sync.WaitGroup
		for i := 0; i < r.Concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for tc := range jobCh {
					resultCh <- r.runCase(ctx, tc)
				}
			}()
		}
		for _, tc := range jobs {
			if err := ctx.Err(); err != nil {
				close(jobCh)
				wg.Wait()
				close(resultCh)
				return Report{}, err
			}
			jobCh <- tc
		}
		close(jobCh)
		wg.Wait()
		close(resultCh)
		for result := range resultCh {
			results = append(results, result)
		}
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

func (r *Runner) runCase(ctx context.Context, tc TravelCase) CaseResult {
	start := time.Now()
	plan, planErr := r.Planner.Plan(ctx, tc.Input)
	durationMs := time.Since(start).Milliseconds()
	return r.Evaluator.Evaluate(tc, plan, planErr, durationMs)
}
