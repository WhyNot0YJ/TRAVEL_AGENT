package eino

const submitTravelPlanToolName = "submit_travel_plan"

func travelPlanJSONSchema() map[string]any {
	return objectSchema(
		map[string]any{
			"title":    stringSchema(),
			"summary":  stringSchema(),
			"days":     arraySchema(travelDaySchema()),
			"budget":   travelBudgetSchema(),
			"warnings": arraySchema(stringSchema()),
		},
		"title", "summary", "days", "budget", "warnings",
	)
}

func travelDaySchema() map[string]any {
	return objectSchema(
		map[string]any{
			"day":   integerSchema(),
			"theme": stringSchema(),
			"items": arraySchema(travelItemSchema()),
		},
		"day", "theme", "items",
	)
}

func travelItemSchema() map[string]any {
	return objectSchema(
		map[string]any{
			"time":             stringSchema(),
			"type":             stringSchema(),
			"name":             stringSchema(),
			"address":          stringSchema(),
			"reason":           stringSchema(),
			"estimated_cost":   numberSchema(),
			"duration_minutes": integerSchema(),
		},
		"time", "type", "name", "address", "reason", "estimated_cost", "duration_minutes",
	)
}

func travelBudgetSchema() map[string]any {
	return objectSchema(
		map[string]any{
			"transport": numberSchema(),
			"food":      numberSchema(),
			"hotel":     numberSchema(),
			"ticket":    numberSchema(),
			"total":     numberSchema(),
		},
		"transport", "food", "hotel", "ticket", "total",
	)
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}
}

func arraySchema(items map[string]any) map[string]any {
	return map[string]any{
		"type":  "array",
		"items": items,
	}
}

func stringSchema() map[string]any {
	return map[string]any{"type": "string"}
}

func numberSchema() map[string]any {
	return map[string]any{"type": "number"}
}

func integerSchema() map[string]any {
	return map[string]any{"type": "integer"}
}
