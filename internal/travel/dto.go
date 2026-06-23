package travel

import "travel-agent/internal/domain"

type CreatePlanRequest struct {
	ID              string   `json:"id"`
	DepartureCity   string   `json:"departure_city" binding:"required"`
	DestinationCity string   `json:"destination_city" binding:"required"`
	Days            int      `json:"days" binding:"required,min=1"`
	Budget          float64  `json:"budget" binding:"required,gt=0"`
	Interests       []string `json:"interests"`
	TransportMode   string   `json:"transport_mode"`
	Pace            string   `json:"pace"`
}

func (r CreatePlanRequest) ToDomain(id string) domain.TravelRequest {
	if r.ID != "" {
		id = r.ID
	}
	return domain.TravelRequest{
		ID:              id,
		DepartureCity:   r.DepartureCity,
		DestinationCity: r.DestinationCity,
		Days:            r.Days,
		Budget:          r.Budget,
		Interests:       append([]string{}, r.Interests...),
		TransportMode:   r.TransportMode,
		Pace:            r.Pace,
	}
}

type CreatePlanResponse struct {
	PlanID string             `json:"plan_id"`
	Plan   *domain.TravelPlan `json:"plan"`
}

type GetPlanResponse struct {
	PlanID string             `json:"plan_id"`
	Plan   *domain.TravelPlan `json:"plan"`
}

type ErrorResponse struct {
	RequestID string `json:"request_id"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}
