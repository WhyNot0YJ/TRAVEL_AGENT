package plans

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

// PlanStore is the persistence boundary for user plans + archives. Repository
// implementations enforce user_id scoping at every read.
type PlanStore interface {
	Create(ctx context.Context, plan UserPlan) error
	Get(ctx context.Context, planID, userID string) (UserPlan, error)
	GetByTaskID(ctx context.Context, userID, taskID string) (UserPlan, bool, error)
	List(ctx context.Context, userID string, filter ListFilter) ([]UserPlan, int, error)
	Update(ctx context.Context, plan UserPlan) error
	SoftDelete(ctx context.Context, planID, userID string, deletedAt time.Time) error
	CreateArchive(ctx context.Context, archive ConversationArchive) error
	GetArchive(ctx context.Context, planID, userID string) (ConversationArchive, error)
	GetByID(ctx context.Context, planID string) (UserPlan, error) // for internal joins
}

// PublicPlanStore manages the publish-side view. List queries are public so
// they accept no user_id; private guards are enforced in the service layer.
type PublicPlanStore interface {
	Upsert(ctx context.Context, plan PublicPlan) error
	Get(ctx context.Context, id string) (PublicPlan, error)
	GetByPlanID(ctx context.Context, planID string) (PublicPlan, bool, error)
	List(ctx context.Context, filter PublicListFilter) ([]PublicPlan, int, error)
	SetStatus(ctx context.Context, id, status string, updatedAt time.Time) error
	IncrementCounter(ctx context.Context, id string, kind PublicCounterKind, now time.Time) error
	RecordEvent(ctx context.Context, event PublicPlanEvent) error
}

type PublicCounterKind string

const (
	CounterView PublicCounterKind = "view"
	CounterSave PublicCounterKind = "save"
	CounterCopy PublicCounterKind = "copy"
)

// MemoryPlanStore is a single-process fallback. Useful in tests and when MySQL
// is not configured.
type MemoryPlanStore struct {
	mu       sync.RWMutex
	plans    map[string]UserPlan
	archives map[string]ConversationArchive // keyed by plan_id
}

func NewMemoryPlanStore() *MemoryPlanStore {
	return &MemoryPlanStore{plans: map[string]UserPlan{}, archives: map[string]ConversationArchive{}}
}

func (s *MemoryPlanStore) Create(ctx context.Context, plan UserPlan) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[plan.ID] = plan
	return nil
}

func (s *MemoryPlanStore) Get(ctx context.Context, planID, userID string) (UserPlan, error) {
	if err := ctx.Err(); err != nil {
		return UserPlan{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	plan, ok := s.plans[planID]
	if !ok || plan.UserID != userID || plan.IsDeleted() {
		return UserPlan{}, ErrPlanNotFound
	}
	return plan, nil
}

func (s *MemoryPlanStore) GetByID(ctx context.Context, planID string) (UserPlan, error) {
	if err := ctx.Err(); err != nil {
		return UserPlan{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	plan, ok := s.plans[planID]
	if !ok {
		return UserPlan{}, ErrPlanNotFound
	}
	return plan, nil
}

func (s *MemoryPlanStore) GetByTaskID(ctx context.Context, userID, taskID string) (UserPlan, bool, error) {
	if err := ctx.Err(); err != nil {
		return UserPlan{}, false, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, plan := range s.plans {
		if plan.UserID == userID && plan.TaskID == taskID && !plan.IsDeleted() {
			return plan, true, nil
		}
	}
	return UserPlan{}, false, nil
}

func (s *MemoryPlanStore) List(ctx context.Context, userID string, filter ListFilter) ([]UserPlan, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var matched []UserPlan
	q := strings.ToLower(strings.TrimSpace(filter.Query))
	for _, plan := range s.plans {
		if plan.UserID != userID || plan.IsDeleted() {
			continue
		}
		if filter.Visibility != "" && plan.Visibility != filter.Visibility {
			continue
		}
		if filter.PublishStatus != "" && plan.PublishStatus != filter.PublishStatus {
			continue
		}
		if filter.DestinationCity != "" && !strings.EqualFold(plan.DestinationCity, filter.DestinationCity) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(plan.Title), q) && !strings.Contains(strings.ToLower(plan.DestinationCity), q) {
			continue
		}
		matched = append(matched, plan)
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].UpdatedAt.After(matched[j].UpdatedAt) })
	total := len(matched)
	page, size := normalizePagination(filter.Page, filter.PageSize)
	start := (page - 1) * size
	if start >= total {
		return []UserPlan{}, total, nil
	}
	end := start + size
	if end > total {
		end = total
	}
	return matched[start:end], total, nil
}

func (s *MemoryPlanStore) Update(ctx context.Context, plan UserPlan) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.plans[plan.ID]
	if !ok || current.UserID != plan.UserID {
		return ErrPlanNotFound
	}
	s.plans[plan.ID] = plan
	return nil
}

func (s *MemoryPlanStore) SoftDelete(ctx context.Context, planID, userID string, deletedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.plans[planID]
	if !ok || plan.UserID != userID {
		return ErrPlanNotFound
	}
	t := deletedAt
	plan.DeletedAt = &t
	plan.UpdatedAt = deletedAt
	s.plans[planID] = plan
	return nil
}

func (s *MemoryPlanStore) CreateArchive(ctx context.Context, archive ConversationArchive) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.archives[archive.PlanID] = archive
	return nil
}

func (s *MemoryPlanStore) GetArchive(ctx context.Context, planID, userID string) (ConversationArchive, error) {
	if err := ctx.Err(); err != nil {
		return ConversationArchive{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	archive, ok := s.archives[planID]
	if !ok || archive.UserID != userID {
		return ConversationArchive{}, ErrPlanNotFound
	}
	return archive, nil
}

// MemoryPublicPlanStore is the in-memory implementation for development.
type MemoryPublicPlanStore struct {
	mu          sync.RWMutex
	plans       map[string]PublicPlan
	planToPubID map[string]string
	events      []PublicPlanEvent
}

func NewMemoryPublicPlanStore() *MemoryPublicPlanStore {
	return &MemoryPublicPlanStore{plans: map[string]PublicPlan{}, planToPubID: map[string]string{}}
}

func (s *MemoryPublicPlanStore) Upsert(ctx context.Context, plan PublicPlan) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existingID, ok := s.planToPubID[plan.PlanID]; ok && existingID != plan.ID {
		// keep existing ID stable
		plan.ID = existingID
	}
	if existing, ok := s.plans[plan.ID]; ok {
		// preserve counters across re-publish
		plan.ViewCount = existing.ViewCount
		plan.SaveCount = existing.SaveCount
		plan.CopyCount = existing.CopyCount
	}
	plan.HotScore = computeHotScore(plan)
	s.plans[plan.ID] = plan
	s.planToPubID[plan.PlanID] = plan.ID
	return nil
}

func (s *MemoryPublicPlanStore) Get(ctx context.Context, id string) (PublicPlan, error) {
	if err := ctx.Err(); err != nil {
		return PublicPlan{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	plan, ok := s.plans[id]
	if !ok {
		return PublicPlan{}, ErrPublicPlanNotFound
	}
	return plan, nil
}

func (s *MemoryPublicPlanStore) GetByPlanID(ctx context.Context, planID string) (PublicPlan, bool, error) {
	if err := ctx.Err(); err != nil {
		return PublicPlan{}, false, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.planToPubID[planID]
	if !ok {
		return PublicPlan{}, false, nil
	}
	plan, ok := s.plans[id]
	return plan, ok, nil
}

func (s *MemoryPublicPlanStore) List(ctx context.Context, filter PublicListFilter) ([]PublicPlan, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var matched []PublicPlan
	q := strings.ToLower(strings.TrimSpace(filter.Query))
	for _, plan := range s.plans {
		if plan.Status != PublicPlanStatusPublished {
			continue
		}
		if filter.DestinationCity != "" && !strings.EqualFold(plan.DestinationCity, filter.DestinationCity) {
			continue
		}
		if filter.Days > 0 && plan.Days != filter.Days {
			continue
		}
		if filter.Interest != "" {
			tagged := false
			for _, tag := range plan.Tags {
				if strings.EqualFold(tag, filter.Interest) {
					tagged = true
					break
				}
			}
			if !tagged {
				continue
			}
		}
		if q != "" {
			haystack := strings.ToLower(plan.Title + " " + plan.Summary + " " + plan.DestinationCity + " " + strings.Join(plan.Tags, " "))
			if !strings.Contains(haystack, q) {
				continue
			}
		}
		matched = append(matched, plan)
	}
	sort.Slice(matched, func(i, j int) bool {
		if filter.Sort == "latest" {
			return matched[i].PublishedAt.After(matched[j].PublishedAt)
		}
		if matched[i].HotScore == matched[j].HotScore {
			return matched[i].PublishedAt.After(matched[j].PublishedAt)
		}
		return matched[i].HotScore > matched[j].HotScore
	})
	total := len(matched)
	page, size := normalizePagination(filter.Page, filter.PageSize)
	start := (page - 1) * size
	if start >= total {
		return []PublicPlan{}, total, nil
	}
	end := start + size
	if end > total {
		end = total
	}
	return matched[start:end], total, nil
}

func (s *MemoryPublicPlanStore) SetStatus(ctx context.Context, id, status string, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.plans[id]
	if !ok {
		return ErrPublicPlanNotFound
	}
	plan.Status = status
	plan.UpdatedAt = updatedAt
	s.plans[id] = plan
	return nil
}

func (s *MemoryPublicPlanStore) IncrementCounter(ctx context.Context, id string, kind PublicCounterKind, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.plans[id]
	if !ok {
		return ErrPublicPlanNotFound
	}
	switch kind {
	case CounterView:
		plan.ViewCount++
	case CounterSave:
		plan.SaveCount++
	case CounterCopy:
		plan.CopyCount++
	}
	plan.HotScore = computeHotScore(plan)
	plan.UpdatedAt = now
	s.plans[id] = plan
	return nil
}

func (s *MemoryPublicPlanStore) RecordEvent(ctx context.Context, event PublicPlanEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *MemoryPublicPlanStore) Events() []PublicPlanEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]PublicPlanEvent(nil), s.events...)
}

func computeHotScore(p PublicPlan) int64 {
	return p.ViewCount + p.SaveCount*5 + p.CopyCount*3
}

func normalizePagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
