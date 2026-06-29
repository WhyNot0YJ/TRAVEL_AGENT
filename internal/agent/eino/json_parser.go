package eino

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"travel-agent/internal/domain"
)

func parseTravelPlanArguments(raw string, req domain.TravelRequest) (*domain.TravelPlan, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("travel plan arguments are empty")
	}

	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	var plan domain.TravelPlan
	if err := decoder.Decode(&plan); err != nil {
		return nil, fmt.Errorf("decode travel plan arguments: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("travel plan arguments contain trailing data")
	}
	if err := validateGeneratedPlan(req, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func validateGeneratedPlan(req domain.TravelRequest, plan *domain.TravelPlan) error {
	if plan == nil {
		return fmt.Errorf("plan is nil")
	}
	if strings.TrimSpace(plan.Title) == "" {
		return fmt.Errorf("plan title is empty")
	}
	if strings.TrimSpace(plan.Summary) == "" {
		return fmt.Errorf("plan summary is empty")
	}
	if !strings.Contains(plan.Title+" "+plan.Summary, req.DestinationCity) {
		return fmt.Errorf("title or summary must contain destination city %q", req.DestinationCity)
	}
	if len(plan.Days) != req.Days {
		return fmt.Errorf("expected %d days, got %d", req.Days, len(plan.Days))
	}
	if plan.Budget.Total < 0 || plan.Budget.KnownTotal < 0 {
		return fmt.Errorf("plan budget total is negative")
	}
	budgetLimit := req.Budget
	if domain.IsBudgetPerPerson(req.BudgetType) && req.Travelers > 0 {
		budgetLimit *= float64(req.Travelers)
	}
	if plan.Budget.Total > budgetLimit*1.1 {
		return fmt.Errorf("plan budget total %.2f exceeds request budget threshold %.2f", plan.Budget.Total, budgetLimit*1.1)
	}
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
			return fmt.Errorf("budget.%s is negative", budget.name)
		}
	}
	if err := validateBudgetDetails(plan.Budget); err != nil {
		return err
	}
	for i, day := range plan.Days {
		expected := i + 1
		if day.Day != expected {
			return fmt.Errorf("day number mismatch at index %d: got %d want %d", i, day.Day, expected)
		}
		if strings.TrimSpace(day.Theme) == "" {
			return fmt.Errorf("day %d theme is empty", day.Day)
		}
		if len(day.Items) == 0 {
			return fmt.Errorf("day %d has no items", day.Day)
		}
		for idx, item := range day.Items {
			if strings.TrimSpace(item.Time) == "" {
				return fmt.Errorf("day %d item %d time is empty", day.Day, idx)
			}
			if strings.TrimSpace(item.Type) == "" {
				return fmt.Errorf("day %d item %d type is empty", day.Day, idx)
			}
			if strings.TrimSpace(item.Name) == "" {
				return fmt.Errorf("day %d item %d name is empty", day.Day, idx)
			}
			if strings.TrimSpace(item.Address) == "" {
				return fmt.Errorf("day %d item %d address is empty", day.Day, idx)
			}
			if strings.TrimSpace(item.Reason) == "" {
				return fmt.Errorf("day %d item %d reason is empty", day.Day, idx)
			}
			if item.EstimatedCost < 0 {
				return fmt.Errorf("day %d item %d estimated cost is negative", day.Day, idx)
			}
			if err := validateCostInfo(item.Cost); err != nil {
				return fmt.Errorf("day %d item %d cost is invalid: %w", day.Day, idx, err)
			}
			if item.DurationMinutes < 0 {
				return fmt.Errorf("day %d item %d duration is negative", day.Day, idx)
			}
		}
	}
	return nil
}

func validateBudgetDetails(budget domain.TravelBudget) error {
	if budget.Currency == "" {
		return fmt.Errorf("budget currency is empty")
	}
	if budget.Total != budget.KnownTotal {
		return fmt.Errorf("budget total %.2f must equal known_total %.2f", budget.Total, budget.KnownTotal)
	}
	for idx, item := range budget.Items {
		if strings.TrimSpace(item.Key) == "" || strings.TrimSpace(item.Label) == "" {
			return fmt.Errorf("budget item %d key or label is empty", idx)
		}
		if err := validateBudgetLine(item); err != nil {
			return fmt.Errorf("budget item %d is invalid: %w", idx, err)
		}
	}
	return nil
}

func validateBudgetLine(item domain.BudgetLine) error {
	switch item.Status {
	case domain.CostAvailable:
		if item.Amount == nil {
			return fmt.Errorf("available amount is nil")
		}
		if *item.Amount < 0 {
			return fmt.Errorf("amount is negative")
		}
		if !item.Included {
			return fmt.Errorf("available budget item must be included")
		}
	case domain.CostUnavailable:
		if item.Amount != nil {
			return fmt.Errorf("unavailable amount must be null")
		}
		if item.Included {
			return fmt.Errorf("unavailable budget item must not be included")
		}
	case domain.CostNotApplicable:
		if item.Included {
			return fmt.Errorf("not_applicable budget item must not be included")
		}
	default:
		return fmt.Errorf("unknown status %q", item.Status)
	}
	return nil
}

func validateCostInfo(cost domain.CostInfo) error {
	switch cost.Status {
	case domain.CostAvailable:
		if cost.Amount == nil {
			return fmt.Errorf("available amount is nil")
		}
		if *cost.Amount < 0 {
			return fmt.Errorf("amount is negative")
		}
		if !cost.Included {
			return fmt.Errorf("available cost must be included")
		}
	case domain.CostUnavailable:
		if cost.Amount != nil {
			return fmt.Errorf("unavailable amount must be null")
		}
		if cost.Included {
			return fmt.Errorf("unavailable cost must not be included")
		}
	case domain.CostNotApplicable:
		if cost.Included {
			return fmt.Errorf("not_applicable cost must not be included")
		}
	default:
		return fmt.Errorf("unknown status %q", cost.Status)
	}
	return nil
}
