package eino

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"travel-agent/internal/domain"
)

// EinoTravelPlanner implements agent.TravelPlanner through a CloudWeGo Eino graph.
type EinoTravelPlanner struct {
	graph compose.Runnable[domain.TravelRequest, *domain.TravelPlan]
}

// NewEinoTravelPlanner builds and compiles the Eino travel planning graph.
func NewEinoTravelPlanner() (*EinoTravelPlanner, error) {
	return NewEinoTravelPlannerWithGenerator(newDefaultPlanGenerator())
}

func NewEinoTravelPlannerWithGenerator(generator TravelPlanGenerator) (*EinoTravelPlanner, error) {
	graph, err := buildTravelGraph(context.Background(), generator)
	if err != nil {
		return nil, err
	}
	return &EinoTravelPlanner{graph: graph}, nil
}

// Plan invokes the compiled Eino graph and returns a structured TravelPlan.
func (p *EinoTravelPlanner) Plan(ctx context.Context, req domain.TravelRequest) (*domain.TravelPlan, error) {
	if p == nil || p.graph == nil {
		return nil, fmt.Errorf("eino travel planner is not initialized")
	}
	plan, err := p.graph.Invoke(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("invoke eino travel graph: %w", err)
	}
	return plan, nil
}
