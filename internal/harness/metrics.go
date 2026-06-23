package harness

// SummaryMetrics is the aggregate result for one harness run.
type SummaryMetrics struct {
	TotalCases        int     `json:"total_cases"`
	SuccessCases      int     `json:"success_cases"`
	FailedCases       int     `json:"failed_cases"`
	SuccessRate       float64 `json:"success_rate"`
	AverageScore      float64 `json:"average_score"`
	AverageDurationMs float64 `json:"average_duration_ms"`
	BudgetPassRate    float64 `json:"budget_pass_rate"`
	DayMatchRate      float64 `json:"day_match_rate"`
	StructurePassRate float64 `json:"structure_pass_rate"`
}

// CalculateSummary converts case-level results into stable aggregate metrics.
func CalculateSummary(results []CaseResult) SummaryMetrics {
	summary := SummaryMetrics{TotalCases: len(results)}
	if len(results) == 0 {
		return summary
	}

	var scoreSum float64
	var durationSum int64
	var budgetPassed, dayMatched, structureComplete int
	for _, result := range results {
		if result.Success {
			summary.SuccessCases++
		}
		if result.Checks.BudgetPassed {
			budgetPassed++
		}
		if result.Checks.DayMatched {
			dayMatched++
		}
		if result.Checks.StructureComplete {
			structureComplete++
		}
		scoreSum += result.Score
		durationSum += result.DurationMs
	}

	total := float64(len(results))
	summary.FailedCases = summary.TotalCases - summary.SuccessCases
	summary.SuccessRate = float64(summary.SuccessCases) / total
	summary.AverageScore = scoreSum / total
	summary.AverageDurationMs = float64(durationSum) / total
	summary.BudgetPassRate = float64(budgetPassed) / total
	summary.DayMatchRate = float64(dayMatched) / total
	summary.StructurePassRate = float64(structureComplete) / total

	return summary
}
