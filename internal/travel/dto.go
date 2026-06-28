package travel

import "travel-agent/internal/domain"

type CreatePlanRequest struct {
	ID               string   `json:"id"`
	DepartureCity    string   `json:"departure_city" binding:"required"`
	DestinationCity  string   `json:"destination_city" binding:"required"`
	Days             int      `json:"days" binding:"required,min=1"`
	Budget           float64  `json:"budget" binding:"required,gt=0"`
	Interests        []string `json:"interests"`
	Travelers        int      `json:"travelers" binding:"required,min=1"`
	DateRange        string   `json:"date_range"`
	TransportMode    string   `json:"transport_mode"`
	Pace             string   `json:"pace"`
	WalkingTolerance string   `json:"walking_tolerance"`
	HotelArea        string   `json:"hotel_area"`
	MustVisit        []string `json:"must_visit"`
	Avoid            []string `json:"avoid"`
	TravelerType     string   `json:"traveler_type"`
	BudgetType       string   `json:"budget_type"`
	BudgetIncludes   []string `json:"budget_includes"`
	TestMode         bool     `json:"test_mode"`
	AgentMode        string   `json:"agent_mode"`
}

func (r CreatePlanRequest) ToDomain(id string) domain.TravelRequest {
	if r.ID != "" {
		id = r.ID
	}
	return domain.NormalizeTravelBrief(domain.TravelRequest{
		ID:               id,
		DepartureCity:    r.DepartureCity,
		DestinationCity:  r.DestinationCity,
		Days:             r.Days,
		Budget:           r.Budget,
		Interests:        append([]string{}, r.Interests...),
		Travelers:        r.Travelers,
		DateRange:        r.DateRange,
		TransportMode:    r.TransportMode,
		Pace:             r.Pace,
		WalkingTolerance: r.WalkingTolerance,
		HotelArea:        r.HotelArea,
		MustVisit:        append([]string{}, r.MustVisit...),
		Avoid:            append([]string{}, r.Avoid...),
		TravelerType:     r.TravelerType,
		BudgetType:       r.BudgetType,
		BudgetIncludes:   append([]string{}, r.BudgetIncludes...),
	})
}

type CreateTaskResponse struct {
	TaskID      string     `json:"task_id"`
	RequestHash string     `json:"request_hash"`
	Status      TaskStatus `json:"status"`
	Cached      bool       `json:"cached"`
}

type GetTaskResponse struct {
	TaskID      string             `json:"task_id"`
	RequestHash string             `json:"request_hash"`
	Status      TaskStatus         `json:"status"`
	Plan        *domain.TravelPlan `json:"plan,omitempty"`
	Error       string             `json:"error,omitempty"`
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
}

type ErrorResponse struct {
	RequestID string `json:"request_id"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

type ChatRequest struct {
	Message          string   `json:"message" binding:"required"`
	DepartureCity    string   `json:"departure_city"`
	DestinationCity  string   `json:"destination_city"`
	Days             int      `json:"days"`
	Budget           float64  `json:"budget"`
	Interests        []string `json:"interests"`
	Travelers        int      `json:"travelers"`
	DateRange        string   `json:"date_range"`
	TransportMode    string   `json:"transport_mode"`
	Pace             string   `json:"pace"`
	WalkingTolerance string   `json:"walking_tolerance"`
	HotelArea        string   `json:"hotel_area"`
	MustVisit        []string `json:"must_visit"`
	Avoid            []string `json:"avoid"`
	TravelerType     string   `json:"traveler_type"`
	BudgetType       string   `json:"budget_type"`
	BudgetIncludes   []string `json:"budget_includes"`
	TestMode         bool     `json:"test_mode"`
	AgentMode        string   `json:"agent_mode"`
}

type ChatResponse struct {
	DepartureCity    string   `json:"departure_city,omitempty"`
	DestinationCity  string   `json:"destination_city,omitempty"`
	Days             int      `json:"days,omitempty"`
	Budget           float64  `json:"budget,omitempty"`
	Interests        []string `json:"interests,omitempty"`
	Travelers        int      `json:"travelers,omitempty"`
	DateRange        string   `json:"date_range,omitempty"`
	TransportMode    string   `json:"transport_mode,omitempty"`
	Pace             string   `json:"pace,omitempty"`
	WalkingTolerance string   `json:"walking_tolerance,omitempty"`
	HotelArea        string   `json:"hotel_area,omitempty"`
	MustVisit        []string `json:"must_visit,omitempty"`
	Avoid            []string `json:"avoid,omitempty"`
	TravelerType     string   `json:"traveler_type,omitempty"`
	BudgetType       string   `json:"budget_type,omitempty"`
	BudgetIncludes   []string `json:"budget_includes,omitempty"`
	Reply            string   `json:"reply"`
	Missing          []string `json:"missing"`
	IsComplete       bool     `json:"is_complete"`
	AgentMode        string   `json:"agent_mode"`
}
