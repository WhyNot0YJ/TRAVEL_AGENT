package harness

import (
	"fmt"
	"strings"

	"travel-agent/internal/domain"
)

// EvaluationChecks records the boolean dimensions used for aggregate metrics.
type EvaluationChecks struct {
	PlannerSucceeded  bool `json:"planner_succeeded"`
	DayMatched        bool `json:"day_matched"`
	BudgetPassed      bool `json:"budget_passed"`
	StructureComplete bool `json:"structure_complete"`
	KeywordsMatched   bool `json:"keywords_matched"`
	NoIllegalFields   bool `json:"no_illegal_fields"`
}

// CaseResult is the per-case output stored in the report.
type CaseResult struct {
	CaseID      string             `json:"case_id"`
	Description string             `json:"description"`
	Success     bool               `json:"success"`
	DurationMs  int64              `json:"duration_ms"`
	Score       float64            `json:"score"`
	Errors      []string           `json:"errors"`
	Warnings    []string           `json:"warnings"`
	Checks      EvaluationChecks   `json:"checks"`
	Plan        *domain.TravelPlan `json:"plan"`
}

// Evaluator applies deterministic quality checks to planner output.
type Evaluator struct{}

// NewEvaluator returns a stateless evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// Evaluate scores a single planner result without mutating the plan.
func (e *Evaluator) Evaluate(tc TravelCase, plan *domain.TravelPlan, plannerErr error, durationMs int64) CaseResult {
	result := CaseResult{
		CaseID:      tc.ID,
		Description: tc.Description,
		DurationMs:  durationMs,
		Errors:      []string{},
		Warnings:    []string{},
		Plan:        plan,
	}

	if plannerErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("planner error: %v", plannerErr))
		result.Score = 0
		return result
	}
	if plan == nil {
		result.Errors = append(result.Errors, "planner returned nil plan")
		result.Score = 0
		return result
	}

	result.Checks.PlannerSucceeded = true
	result.Checks.DayMatched = checkDays(tc, plan, &result)
	result.Checks.BudgetPassed = checkBudget(tc, plan, &result)
	result.Checks.StructureComplete = checkStructure(plan, &result)
	result.Checks.KeywordsMatched = checkKeywords(tc, plan, &result)
	result.Checks.NoIllegalFields = checkIllegalFields(plan, &result)
	result.Score = score(result.Checks)
	result.Success = len(result.Errors) == 0

	return result
}

func checkDays(tc TravelCase, plan *domain.TravelPlan, result *CaseResult) bool {
	ok := true
	if len(plan.Days) != tc.Input.Days {
		result.Errors = append(result.Errors, fmt.Sprintf("expected %d days, got %d", tc.Input.Days, len(plan.Days)))
		ok = false
	}
	if len(plan.Days) < tc.Expectation.MinDays {
		result.Errors = append(result.Errors, fmt.Sprintf("expected at least %d days, got %d", tc.Expectation.MinDays, len(plan.Days)))
		ok = false
	}
	for i, day := range plan.Days {
		expected := i + 1
		if day.Day != expected {
			result.Errors = append(result.Errors, fmt.Sprintf("day index at position %d should be %d, got %d", i, expected, day.Day))
			ok = false
		}
	}
	return ok
}

func checkBudget(tc TravelCase, plan *domain.TravelPlan, result *CaseResult) bool {
	limit := tc.Input.Budget * tc.Expectation.MaxBudgetRatio
	if plan.Budget.Total > limit {
		result.Errors = append(result.Errors, fmt.Sprintf("budget total %.2f exceeds limit %.2f", plan.Budget.Total, limit))
		return false
	}
	return true
}

func checkStructure(plan *domain.TravelPlan, result *CaseResult) bool {
	ok := true
	if strings.TrimSpace(plan.Title) == "" {
		result.Errors = append(result.Errors, "title is empty")
		ok = false
	}
	if strings.TrimSpace(plan.Summary) == "" {
		result.Errors = append(result.Errors, "summary is empty")
		ok = false
	}
	if len(plan.Days) == 0 {
		result.Errors = append(result.Errors, "days is empty")
		ok = false
	}
	for _, day := range plan.Days {
		if strings.TrimSpace(day.Theme) == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("day %d theme is empty", day.Day))
		}
		if len(day.Items) == 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("day %d has no items", day.Day))
			ok = false
			continue
		}
		for idx, item := range day.Items {
			if strings.TrimSpace(item.Name) == "" {
				result.Errors = append(result.Errors, fmt.Sprintf("day %d item %d name is empty", day.Day, idx))
				ok = false
			}
			if strings.TrimSpace(item.Type) == "" {
				result.Errors = append(result.Errors, fmt.Sprintf("day %d item %d type is empty", day.Day, idx))
				ok = false
			}
			if strings.TrimSpace(item.Reason) == "" {
				result.Errors = append(result.Errors, fmt.Sprintf("day %d item %d reason is empty", day.Day, idx))
				ok = false
			}
		}
	}
	return ok
}

func checkKeywords(tc TravelCase, plan *domain.TravelPlan, result *CaseResult) bool {
	search := strings.Join([]string{plan.Title, plan.Summary, flattenPlanNames(plan)}, " ")
	ok := true
	for _, keyword := range tc.Expectation.RequiredKeywords {
		if !strings.Contains(search, keyword) {
			result.Errors = append(result.Errors, fmt.Sprintf("required keyword %q not found", keyword))
			ok = false
		}
	}
	if !strings.Contains(plan.Title+" "+plan.Summary, tc.Input.DestinationCity) {
		result.Errors = append(result.Errors, fmt.Sprintf("title or summary must contain destination city %q", tc.Input.DestinationCity))
		ok = false
	}
	return ok
}

func checkIllegalFields(plan *domain.TravelPlan, result *CaseResult) bool {
	ok := true
	budgets := []struct {
		name  string
		value float64
	}{
		{"transport", plan.Budget.Transport},
		{"food", plan.Budget.Food},
		{"hotel", plan.Budget.Hotel},
		{"ticket", plan.Budget.Ticket},
		{"total", plan.Budget.Total},
	}
	for _, budget := range budgets {
		if budget.value < 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("budget.%s is negative", budget.name))
			ok = false
		}
	}
	for _, day := range plan.Days {
		if day.Day <= 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("day number %d is invalid", day.Day))
			ok = false
		}
		for _, item := range day.Items {
			if item.EstimatedCost < 0 {
				result.Errors = append(result.Errors, fmt.Sprintf("item %q has negative estimated cost", item.Name))
				ok = false
			}
			if item.DurationMinutes < 0 {
				result.Errors = append(result.Errors, fmt.Sprintf("item %q has negative duration", item.Name))
				ok = false
			}
		}
	}
	return ok
}

func flattenPlanNames(plan *domain.TravelPlan) string {
	var parts []string
	for _, day := range plan.Days {
		parts = append(parts, day.Theme)
		for _, item := range day.Items {
			parts = append(parts, item.Name, item.Address, item.Reason)
		}
	}
	return strings.Join(parts, " ")
}

func score(checks EvaluationChecks) float64 {
	var total float64
	if checks.PlannerSucceeded {
		total += 20
	}
	if checks.DayMatched {
		total += 20
	}
	if checks.BudgetPassed {
		total += 20
	}
	if checks.StructureComplete {
		total += 20
	}
	if checks.KeywordsMatched {
		total += 10
	}
	if checks.NoIllegalFields {
		total += 10
	}
	if total < 0 {
		return 0
	}
	if total > 100 {
		return 100
	}
	return total
}
