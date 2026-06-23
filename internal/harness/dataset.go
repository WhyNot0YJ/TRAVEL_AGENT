package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"travel-agent/internal/domain"
)

const defaultMaxBudgetRatio = 1.1

// TravelCase binds one input request with the expectations used by the evaluator.
type TravelCase struct {
	ID          string               `json:"id"`
	Description string               `json:"description"`
	Input       domain.TravelRequest `json:"input"`
	Expectation TravelExpectation    `json:"expectation"`
}

// TravelExpectation stores deterministic constraints for one evaluation case.
type TravelExpectation struct {
	MinDays          int      `json:"min_days"`
	MaxBudgetRatio   float64  `json:"max_budget_ratio"`
	RequiredKeywords []string `json:"required_keywords"`
}

type rawTravelCase struct {
	ID          string               `json:"id"`
	Description string               `json:"description"`
	Input       domain.TravelRequest `json:"input"`
	Expectation *TravelExpectation   `json:"expectation"`
}

// LoadCases reads and validates a local JSON dataset.
func LoadCases(path string) ([]TravelCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dataset %q: %w", path, err)
	}

	var rawCases []rawTravelCase
	if err := json.Unmarshal(data, &rawCases); err != nil {
		return nil, fmt.Errorf("parse dataset %q: %w", path, err)
	}
	if len(rawCases) == 0 {
		return nil, fmt.Errorf("dataset %q is empty", path)
	}

	cases := make([]TravelCase, 0, len(rawCases))
	seen := make(map[string]struct{}, len(rawCases))
	for i, raw := range rawCases {
		tc, err := normalizeCase(i, raw, seen)
		if err != nil {
			return nil, err
		}
		cases = append(cases, tc)
	}

	return cases, nil
}

func normalizeCase(index int, raw rawTravelCase, seen map[string]struct{}) (TravelCase, error) {
	if raw.ID == "" {
		return TravelCase{}, fmt.Errorf("case at index %d has empty id", index)
	}
	if _, ok := seen[raw.ID]; ok {
		return TravelCase{}, fmt.Errorf("duplicate case id %q", raw.ID)
	}
	seen[raw.ID] = struct{}{}
	if strings.TrimSpace(raw.Description) == "" {
		return TravelCase{}, fmt.Errorf("case %q description is required", raw.ID)
	}

	if raw.Input.ID != raw.ID {
		return TravelCase{}, fmt.Errorf("case %q input.id must equal case id", raw.ID)
	}
	if raw.Input.DestinationCity == "" {
		return TravelCase{}, fmt.Errorf("case %q destination_city is required", raw.ID)
	}
	if raw.Input.Days <= 0 {
		return TravelCase{}, fmt.Errorf("case %q days must be positive", raw.ID)
	}
	if raw.Input.Budget <= 0 {
		return TravelCase{}, fmt.Errorf("case %q budget must be positive", raw.ID)
	}
	if raw.Expectation == nil {
		return TravelCase{}, fmt.Errorf("case %q expectation is required", raw.ID)
	}

	expectation := *raw.Expectation
	if expectation.MinDays <= 0 {
		return TravelCase{}, fmt.Errorf("case %q expectation.min_days must be positive", raw.ID)
	}
	if expectation.MaxBudgetRatio == 0 {
		expectation.MaxBudgetRatio = defaultMaxBudgetRatio
	}
	if expectation.MaxBudgetRatio < 1 {
		return TravelCase{}, fmt.Errorf("case %q expectation.max_budget_ratio must be >= 1", raw.ID)
	}
	if len(expectation.RequiredKeywords) == 0 {
		return TravelCase{}, fmt.Errorf("case %q expectation.required_keywords is required", raw.ID)
	}

	return TravelCase{
		ID:          raw.ID,
		Description: strings.TrimSpace(raw.Description),
		Input:       raw.Input,
		Expectation: expectation,
	}, nil
}
