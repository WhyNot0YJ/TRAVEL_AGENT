package eino

import (
	"encoding/json"
	"fmt"
	"strings"
)

const travelPlanPromptVersion = "travel-plan-v1"

func buildTravelPlanMessages(state TravelPlanningState) ([]chatMessage, error) {
	contextPayload := struct {
		Request   any `json:"request"`
		POIs      any `json:"pois"`
		Weather   any `json:"weather"`
		Routes    any `json:"routes"`
		Budget    any `json:"budget"`
		Itinerary any `json:"itinerary"`
	}{
		Request:   state.Request,
		POIs:      state.POIs,
		Weather:   state.Weather,
		Routes:    state.Routes,
		Budget:    state.Budget,
		Itinerary: state.Itinerary,
	}
	data, err := json.Marshal(contextPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal planning context: %w", err)
	}

	system := strings.Join([]string{
		"You are a travel planning component.",
		"Prompt version: " + travelPlanPromptVersion + ".",
		"Use the provided planning context to produce a practical itinerary.",
		"Return the final plan only through the configured structured-output channel.",
		"Do not include secrets, API keys, hidden reasoning, or external claims not supported by the context.",
	}, " ")
	user := fmt.Sprintf("Planning context:\n%s", string(data))
	return []chatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}, nil
}
