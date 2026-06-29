package eino

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"travel-agent/internal/domain"
)

const (
	nodeParse        = "parse_travel_request"
	nodePOI          = "search_pois"
	nodeWeather      = "get_weather"
	nodeRoute        = "compute_route"
	nodeBudget       = "estimate_budget"
	nodeOptimize     = "optimize_itinerary"
	nodeRouteCheck   = "validate_route_feasibility"
	nodeGeneratePlan = "generate_travel_plan"
	nodeValidatePlan = "validate_plan"
)

func buildTravelGraph(ctx context.Context, generator TravelPlanGenerator) (compose.Runnable[domain.TravelRequest, *domain.TravelPlan], error) {
	if generator == nil {
		generator = deterministicPlanGenerator{}
	}
	tools := defaultToolSet()

	graph := compose.NewGraph[domain.TravelRequest, *domain.TravelPlan]()

	if err := graph.AddLambdaNode(nodeParse, compose.InvokableLambda(parseTravelRequestNode)); err != nil {
		return nil, fmt.Errorf("add parse node: %w", err)
	}
	if err := graph.AddLambdaNode(nodePOI, compose.InvokableLambda(searchPOIsToolNode(tools.POI))); err != nil {
		return nil, fmt.Errorf("add poi node: %w", err)
	}
	if err := graph.AddLambdaNode(nodeWeather, compose.InvokableLambda(getWeatherToolNode(tools.Weather))); err != nil {
		return nil, fmt.Errorf("add weather node: %w", err)
	}
	if err := graph.AddLambdaNode(nodeRoute, compose.InvokableLambda(computeRouteToolNode(tools.Route))); err != nil {
		return nil, fmt.Errorf("add route node: %w", err)
	}
	if err := graph.AddLambdaNode(nodeBudget, compose.InvokableLambda(estimateBudgetToolNode(tools.Budget))); err != nil {
		return nil, fmt.Errorf("add budget node: %w", err)
	}
	if err := graph.AddLambdaNode(nodeOptimize, compose.InvokableLambda(optimizeItineraryNode)); err != nil {
		return nil, fmt.Errorf("add optimize node: %w", err)
	}
	if err := graph.AddLambdaNode(nodeRouteCheck, compose.InvokableLambda(validateRouteFeasibilityNode)); err != nil {
		return nil, fmt.Errorf("add route feasibility node: %w", err)
	}
	if err := graph.AddLambdaNode(nodeGeneratePlan, compose.InvokableLambda(generateTravelPlanNode(generator))); err != nil {
		return nil, fmt.Errorf("add generate node: %w", err)
	}
	if err := graph.AddLambdaNode(nodeValidatePlan, compose.InvokableLambda(validatePlanNode)); err != nil {
		return nil, fmt.Errorf("add validate node: %w", err)
	}

	edges := [][2]string{
		{compose.START, nodeParse},
		{nodeParse, nodePOI},
		{nodePOI, nodeWeather},
		{nodeWeather, nodeRoute},
		{nodeRoute, nodeOptimize},
		{nodeOptimize, nodeBudget},
		{nodeBudget, nodeRouteCheck},
		{nodeRouteCheck, nodeGeneratePlan},
		{nodeGeneratePlan, nodeValidatePlan},
		{nodeValidatePlan, compose.END},
	}
	for _, edge := range edges {
		if err := graph.AddEdge(edge[0], edge[1]); err != nil {
			return nil, fmt.Errorf("add edge %s -> %s: %w", edge[0], edge[1], err)
		}
	}

	runnable, err := graph.Compile(ctx, compose.WithGraphName("travel_agent_eino_planner"), compose.WithMaxRunSteps(20))
	if err != nil {
		return nil, fmt.Errorf("compile travel graph: %w", err)
	}
	return runnable, nil
}
