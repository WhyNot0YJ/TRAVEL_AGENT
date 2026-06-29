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
			"cost":             costInfoSchema(),
			"duration_minutes": integerSchema(),
		},
		"time", "type", "name", "address", "reason", "estimated_cost", "cost", "duration_minutes",
	)
}

func travelBudgetSchema() map[string]any {
	return objectSchema(
		map[string]any{
			"transport":   numberSchema(),
			"food":        numberSchema(),
			"hotel":       numberSchema(),
			"ticket":      numberSchema(),
			"total":       numberSchema(),
			"known_total": numberSchema(),
			"complete":    booleanSchema(),
			"currency":    stringSchema(),
			"items":       arraySchema(budgetLineSchema()),
			"missing":     arraySchema(stringSchema()),
		},
		"transport", "food", "hotel", "ticket", "total", "known_total", "complete", "currency", "items", "missing",
	)
}

func costInfoSchema() map[string]any {
	return objectSchema(
		map[string]any{
			"amount":   nullableNumberSchema(),
			"currency": stringSchema(),
			"unit":     stringSchema(),
			"status":   enumSchema("available", "unavailable", "not_applicable"),
			"source":   stringSchema(),
			"display":  stringSchema(),
			"included": booleanSchema(),
		},
		"amount", "currency", "unit", "status", "source", "display", "included",
	)
}

func budgetLineSchema() map[string]any {
	return objectSchema(
		map[string]any{
			"key":      stringSchema(),
			"label":    stringSchema(),
			"amount":   nullableNumberSchema(),
			"currency": stringSchema(),
			"status":   enumSchema("available", "unavailable", "not_applicable"),
			"source":   stringSchema(),
			"display":  stringSchema(),
			"included": booleanSchema(),
		},
		"key", "label", "amount", "currency", "status", "source", "display", "included",
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

func nullableNumberSchema() map[string]any {
	return map[string]any{"type": []string{"number", "null"}}
}

func integerSchema() map[string]any {
	return map[string]any{"type": "integer"}
}

func booleanSchema() map[string]any {
	return map[string]any{"type": "boolean"}
}

func enumSchema(values ...string) map[string]any {
	return map[string]any{"type": "string", "enum": values}
}
