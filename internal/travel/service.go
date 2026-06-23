package travel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"travel-agent/internal/agent"
	"travel-agent/internal/domain"
)

var ErrPlanNotFound = errors.New("plan not found")

type TravelPlanService struct {
	planner agent.TravelPlanner
	store   PlanStore
}

func NewTravelPlanService(planner agent.TravelPlanner, store PlanStore) *TravelPlanService {
	return &TravelPlanService{planner: planner, store: store}
}

func (s *TravelPlanService) CreatePlan(ctx context.Context, req CreatePlanRequest) (CreatePlanResponse, error) {
	if s == nil || s.planner == nil || s.store == nil {
		return CreatePlanResponse{}, fmt.Errorf("travel plan service is not initialized")
	}
	planID := newPlanID()
	domainReq := req.ToDomain(planID)
	plan, err := s.planner.Plan(ctx, domainReq)
	if err != nil {
		return CreatePlanResponse{}, err
	}
	if err := s.store.Save(ctx, planID, plan); err != nil {
		return CreatePlanResponse{}, err
	}
	return CreatePlanResponse{PlanID: planID, Plan: plan}, nil
}

func (s *TravelPlanService) GetPlan(ctx context.Context, id string) (GetPlanResponse, error) {
	if s == nil || s.store == nil {
		return GetPlanResponse{}, fmt.Errorf("travel plan service is not initialized")
	}
	plan, ok, err := s.store.Get(ctx, id)
	if err != nil {
		return GetPlanResponse{}, err
	}
	if !ok {
		return GetPlanResponse{}, ErrPlanNotFound
	}
	return GetPlanResponse{PlanID: id, Plan: plan}, nil
}

func newPlanID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "plan_" + hex.EncodeToString(b[:])
	}
	return "plan_fallback"
}

func ValidatePlan(plan *domain.TravelPlan) error {
	if plan == nil {
		return fmt.Errorf("plan is nil")
	}
	if plan.Title == "" {
		return fmt.Errorf("plan title is empty")
	}
	return nil
}
