package harness

// SummaryMetrics is the aggregate result for one harness run.
type SummaryMetrics struct {
	TotalCases               int     `json:"total_cases"`
	SuccessCases             int     `json:"success_cases"`
	FailedCases              int     `json:"failed_cases"`
	SuccessRate              float64 `json:"success_rate"`
	AverageScore             float64 `json:"average_score"`
	AverageDurationMs        float64 `json:"average_duration_ms"`
	BudgetPassRate           float64 `json:"budget_pass_rate"`
	DayMatchRate             float64 `json:"day_match_rate"`
	StructurePassRate        float64 `json:"structure_pass_rate"`
	ToolFallbackRate         float64 `json:"tool_fallback_rate"`
	LLMFallbackRate          float64 `json:"llm_fallback_rate"`
	ExternalAPISuccessRate   float64 `json:"external_api_success_rate"`
	AverageNodeDurationMs    float64 `json:"average_node_duration_ms"`
	AverageTokenUsage        float64 `json:"average_token_usage"`
	RouteFeasibilityPassRate float64 `json:"route_feasibility_pass_rate"`
	WarningRate              float64 `json:"warning_rate"`
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
	var toolFallbackCases, llmFallbackCases, warningCases, routeFeasible int
	var externalSuccess, externalCalls int
	var nodeDurationSum int64
	var nodeDurationSamples int
	var tokenSum, tokenSamples int
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
		if len(result.Warnings) > 0 {
			warningCases++
		}
		if result.Diagnostics.ToolFallbacks > 0 {
			toolFallbackCases++
		}
		if result.Diagnostics.LLMFallbacks > 0 {
			llmFallbackCases++
		}
		if result.Checks.RouteFeasible {
			routeFeasible++
		}
		externalSuccess += result.Diagnostics.ExternalAPISuccesses
		externalCalls += result.Diagnostics.ExternalAPICalls
		nodeDurationSum += result.Diagnostics.NodeDurationMs
		nodeDurationSamples += result.Diagnostics.NodeDurationSamples
		if result.Diagnostics.TokenUsageKnown {
			tokenSum += result.Diagnostics.TotalTokens
			tokenSamples++
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
	summary.ToolFallbackRate = float64(toolFallbackCases) / total
	summary.LLMFallbackRate = float64(llmFallbackCases) / total
	if externalCalls > 0 {
		summary.ExternalAPISuccessRate = float64(externalSuccess) / float64(externalCalls)
	}
	if nodeDurationSamples > 0 {
		summary.AverageNodeDurationMs = float64(nodeDurationSum) / float64(nodeDurationSamples)
	}
	if tokenSamples > 0 {
		summary.AverageTokenUsage = float64(tokenSum) / float64(tokenSamples)
	}
	summary.RouteFeasibilityPassRate = float64(routeFeasible) / total
	summary.WarningRate = float64(warningCases) / total

	return summary
}
