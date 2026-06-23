package travel

import (
	"context"
	"fmt"
	"sync"

	"travel-agent/internal/domain"
)

type PlanStore interface {
	Save(ctx context.Context, id string, plan *domain.TravelPlan) error
	Get(ctx context.Context, id string) (*domain.TravelPlan, bool, error)
}

type MemoryPlanStore struct {
	mu    sync.RWMutex
	plans map[string]*domain.TravelPlan
}

func NewMemoryPlanStore() *MemoryPlanStore {
	return &MemoryPlanStore{plans: map[string]*domain.TravelPlan{}}
}

func (s *MemoryPlanStore) Save(ctx context.Context, id string, plan *domain.TravelPlan) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("plan id is required")
	}
	if plan == nil {
		return fmt.Errorf("plan is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[id] = plan
	return nil
}

func (s *MemoryPlanStore) Get(ctx context.Context, id string) (*domain.TravelPlan, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	plan, ok := s.plans[id]
	return plan, ok, nil
}
