package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Report is the full JSON payload emitted by the harness.
type Report struct {
	GeneratedAt time.Time      `json:"generated_at"`
	PlannerType string         `json:"planner_type"`
	Summary     SummaryMetrics `json:"summary"`
	Cases       []CaseResult   `json:"cases"`
}

// ReportWriter persists and prints a completed harness report.
type ReportWriter interface {
	Write(ctx context.Context, report Report) error
}

// JSONConsoleReportWriter writes a pretty JSON report and a compact console summary.
type JSONConsoleReportWriter struct {
	ReportPath string
	Output     io.Writer
}

// NewJSONConsoleReportWriter creates the default report writer used by the CLI.
func NewJSONConsoleReportWriter(reportPath string, output io.Writer) *JSONConsoleReportWriter {
	if output == nil {
		output = io.Discard
	}
	return &JSONConsoleReportWriter{ReportPath: reportPath, Output: output}
}

// Write emits a human-readable console report and a machine-readable JSON file.
func (w *JSONConsoleReportWriter) Write(ctx context.Context, report Report) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := w.writeJSON(report); err != nil {
		return err
	}
	printConsoleReport(w.Output, report)
	return nil
}

func (w *JSONConsoleReportWriter) writeJSON(report Report) error {
	dir := filepath.Dir(w.ReportPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create report directory %q: %w", dir, err)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(w.ReportPath, data, 0o644); err != nil {
		return fmt.Errorf("write report %q: %w", w.ReportPath, err)
	}
	return nil
}

func printConsoleReport(w io.Writer, report Report) {
	s := report.Summary
	fmt.Fprintln(w, "# Travel Agent Harness Report")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Planner Type: %s\n", report.PlannerType)
	fmt.Fprintf(w, "Total Cases: %d\n", s.TotalCases)
	fmt.Fprintf(w, "Success Cases: %d\n", s.SuccessCases)
	fmt.Fprintf(w, "Failed Cases: %d\n", s.FailedCases)
	fmt.Fprintf(w, "Success Rate: %.2f%%\n", s.SuccessRate*100)
	fmt.Fprintf(w, "Average Score: %.2f\n", s.AverageScore)
	fmt.Fprintf(w, "Average Duration: %.0fms\n", s.AverageDurationMs)
	fmt.Fprintf(w, "Budget Pass Rate: %.2f%%\n", s.BudgetPassRate*100)
	fmt.Fprintf(w, "Day Match Rate: %.2f%%\n", s.DayMatchRate*100)
	fmt.Fprintf(w, "Structure Pass Rate: %.2f%%\n", s.StructurePassRate*100)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Case Results:")
	fmt.Fprintln(w)
	for _, result := range report.Cases {
		status := "FAIL"
		if result.Success {
			status = "PASS"
		}
		errText := ""
		if len(result.Errors) > 0 {
			errText = " errors=[" + strings.Join(result.Errors, "; ") + "]"
		}
		fmt.Fprintf(w, "* %s %s score=%.0f duration=%dms%s\n", result.CaseID, status, result.Score, result.DurationMs, errText)
	}
}
