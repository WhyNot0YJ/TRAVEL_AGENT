package harness

import "testing"

func TestCalculateSummaryIncludesDiagnostics(t *testing.T) {
	results := []CaseResult{
		{
			Success:    true,
			DurationMs: 10,
			Score:      100,
			Warnings: []string{
				"tool fallback: tool=poi provider=amap stage=request category=provider_error mock_fallback=true reason=x",
				"LLM trace: prompt_version=travel-plan-v1 duration_ms=20 prompt_tokens=3 completion_tokens=4 total_tokens=7",
				"route feasibility: check=poi_coordinates score=90 message=x",
			},
			Checks: EvaluationChecks{
				BudgetPassed:      true,
				DayMatched:        true,
				StructureComplete: true,
				RouteFeasible:     false,
			},
			Diagnostics: analyzeWarnings([]string{
				"tool fallback: tool=poi provider=amap stage=request category=provider_error mock_fallback=true reason=x",
				"LLM trace: prompt_version=travel-plan-v1 duration_ms=20 prompt_tokens=3 completion_tokens=4 total_tokens=7",
				"route feasibility: check=poi_coordinates score=90 message=x",
			}),
		},
		{
			Success:    true,
			DurationMs: 30,
			Score:      90,
			Warnings: []string{
				"LLM fallback: prompt_version=travel-plan-v1 category=retry_exhausted attempts=2 duration_ms=40 reason=x",
			},
			Checks: EvaluationChecks{
				BudgetPassed:      true,
				DayMatched:        true,
				StructureComplete: true,
				RouteFeasible:     true,
			},
			Diagnostics: analyzeWarnings([]string{
				"LLM fallback: prompt_version=travel-plan-v1 category=retry_exhausted attempts=2 duration_ms=40 reason=x",
			}),
		},
	}

	summary := CalculateSummary(results)
	if summary.ToolFallbackRate != 0.5 {
		t.Fatalf("unexpected tool fallback rate: %.2f", summary.ToolFallbackRate)
	}
	if summary.LLMFallbackRate != 0.5 {
		t.Fatalf("unexpected llm fallback rate: %.2f", summary.LLMFallbackRate)
	}
	if summary.WarningRate != 1 {
		t.Fatalf("unexpected warning rate: %.2f", summary.WarningRate)
	}
	if summary.RouteFeasibilityPassRate != 0.5 {
		t.Fatalf("unexpected route feasibility pass rate: %.2f", summary.RouteFeasibilityPassRate)
	}
	if summary.AverageNodeDurationMs != 30 {
		t.Fatalf("unexpected node duration: %.2f", summary.AverageNodeDurationMs)
	}
	if summary.AverageTokenUsage != 7 {
		t.Fatalf("unexpected token usage: %.2f", summary.AverageTokenUsage)
	}
}
