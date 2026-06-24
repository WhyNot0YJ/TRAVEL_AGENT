package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"travel-agent/internal/agent"
	einoagent "travel-agent/internal/agent/eino"
	"travel-agent/internal/harness"
)

func main() {
	datasetPath := flag.String("dataset", "testdata/travel_cases.json", "path to travel evaluation dataset")
	reportPath := flag.String("report", "reports/eval_report.json", "path to JSON evaluation report")
	plannerType := flag.String("planner", "mock", "planner implementation: mock or eino")
	repeat := flag.Int("repeat", 1, "number of times to repeat the dataset")
	concurrency := flag.Int("concurrency", 1, "number of concurrent planner evaluations")
	flag.Parse()

	planner, err := buildPlanner(*plannerType)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	writer := harness.NewJSONConsoleReportWriter(*reportPath, os.Stdout)
	runner := harness.NewRunner(*datasetPath, planner, writer)
	runner.PlannerType = *plannerType
	runner.Repeat = *repeat
	runner.Concurrency = *concurrency
	if _, err := runner.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "harness failed: %v\n", err)
		os.Exit(1)
	}
}

func buildPlanner(plannerType string) (agent.TravelPlanner, error) {
	switch plannerType {
	case "mock":
		return agent.NewMockPlanner(), nil
	case "eino":
		return einoagent.NewEinoTravelPlanner()
	default:
		return nil, fmt.Errorf("unsupported planner type: %s", plannerType)
	}
}
