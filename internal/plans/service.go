package plans

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"travel-agent/internal/domain"
)

const (
	defaultPageSize = 20
	maxTitleLength  = 80
	minTitleLength  = 2
)

// TaskLookup decouples the plan service from internal/travel. The server
// wiring layer adapts the travel TaskStore into this minimal surface.
type TaskLookup interface {
	LookupTask(ctx context.Context, taskID string) (TaskSnapshot, error)
}

// AuthorLookup resolves a user_id into a display name without forcing a
// dependency on the auth package.
type AuthorLookup interface {
	DisplayName(ctx context.Context, userID string) string
}

// Service exposes the use cases behind /me/plans, /public/plans and
// /me/current.
type Service struct {
	plans       PlanStore
	publics     PublicPlanStore
	tasks       TaskLookup
	authors     AuthorLookup
	viewDeduper PublicViewDeduper
	now         func() time.Time
}

func NewService(plans PlanStore, publics PublicPlanStore, tasks TaskLookup, authors AuthorLookup) *Service {
	return &Service{
		plans:       plans,
		publics:     publics,
		tasks:       tasks,
		authors:     authors,
		viewDeduper: NewMemoryPublicViewDeduper(0),
		now:         time.Now,
	}
}

func (s *Service) SetPublicViewDeduper(deduper PublicViewDeduper) {
	if s != nil && deduper != nil {
		s.viewDeduper = deduper
	}
}

// SaveInput is the body of POST /me/plans.
type SaveInput struct {
	TaskID string
	Title  string
	Note   string
}

// EditInput is the patch body for PATCH /me/plans/:id. Pointer fields signal
// "leave unchanged when nil".
type EditInput struct {
	Title      *string
	Note       *string
	Summary    *string
	Tags       *[]string
	Visibility *string
}

// PublishInput is the body of POST /me/plans/:id/publish. Empty fields fall
// back to plan defaults.
type PublishInput struct {
	Title   string
	Summary string
	Tags    []string
}

// CurrentView is what /me/current returns: the live running task plus the most
// recently saved plan, if any.
type CurrentView struct {
	RunningTask *RunningTask
	LatestPlan  *UserPlan
}

type RunningTask struct {
	TaskID          string
	Status          string
	DestinationCity string
	UpdatedAt       time.Time
}

func (s *Service) Save(ctx context.Context, userID string, in SaveInput) (UserPlan, error) {
	if userID == "" {
		return UserPlan{}, ErrPlanNotFound
	}
	if in.TaskID == "" {
		return UserPlan{}, ErrTaskNotFound
	}
	if existing, ok, err := s.plans.GetByTaskID(ctx, userID, in.TaskID); err != nil {
		return UserPlan{}, err
	} else if ok {
		return existing, nil
	}
	if s.tasks == nil {
		return UserPlan{}, fmt.Errorf("plan service: task lookup not configured")
	}
	snapshot, err := s.tasks.LookupTask(ctx, in.TaskID)
	if err != nil {
		return UserPlan{}, err
	}
	if snapshot.Plan == nil || snapshot.Status != TaskStatusSucceeded {
		return UserPlan{}, ErrTaskNotSucceeded
	}
	if snapshot.UserID != "" && snapshot.UserID != userID {
		return UserPlan{}, ErrTaskNotOwned
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = strings.TrimSpace(snapshot.Plan.Title)
	}
	if title == "" {
		title = strings.TrimSpace(snapshot.Request.DestinationCity) + " 旅行计划"
	}
	if err := validateTitle(title); err != nil {
		return UserPlan{}, err
	}

	now := s.now().UTC()
	plan := UserPlan{
		ID:              NewUserPlanID(),
		UserID:          userID,
		TaskID:          in.TaskID,
		Title:           title,
		Note:            strings.TrimSpace(in.Note),
		Summary:         strings.TrimSpace(snapshot.Plan.Summary),
		Tags:            tagsForPlan(snapshot.Plan, snapshot.Request),
		Plan:            cloneTravelPlan(snapshot.Plan),
		DestinationCity: snapshot.Request.DestinationCity,
		Days:            snapshot.Request.Days,
		Visibility:      VisibilityPrivate,
		PublishStatus:   PublishStatusDraft,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.plans.Create(ctx, plan); err != nil {
		return UserPlan{}, err
	}
	archive := ConversationArchive{
		ID:        NewArchiveID(),
		PlanID:    plan.ID,
		UserID:    userID,
		TaskID:    in.TaskID,
		Brief:     copyTravelRequest(snapshot.Request),
		CreatedAt: now,
	}
	if err := s.plans.CreateArchive(ctx, archive); err != nil {
		return UserPlan{}, err
	}
	return plan, nil
}

func (s *Service) Get(ctx context.Context, userID, planID string) (UserPlan, error) {
	return s.plans.Get(ctx, planID, userID)
}

func (s *Service) GetArchive(ctx context.Context, userID, planID string) (ConversationArchive, error) {
	if _, err := s.plans.Get(ctx, planID, userID); err != nil {
		return ConversationArchive{}, err
	}
	return s.plans.GetArchive(ctx, planID, userID)
}

func (s *Service) List(ctx context.Context, userID string, filter ListFilter) ([]UserPlan, int, error) {
	return s.plans.List(ctx, userID, filter)
}

func (s *Service) Rename(ctx context.Context, userID, planID, title string) (UserPlan, error) {
	plan, err := s.plans.Get(ctx, planID, userID)
	if err != nil {
		return UserPlan{}, err
	}
	if err := validateTitle(title); err != nil {
		return UserPlan{}, err
	}
	plan.Title = strings.TrimSpace(title)
	plan.UpdatedAt = s.now().UTC()
	if err := s.plans.Update(ctx, plan); err != nil {
		return UserPlan{}, err
	}
	if plan.PublishStatus == PublishStatusPublished {
		// Keep public mirror in sync but never expose private fields.
		if pub, ok, err := s.publics.GetByPlanID(ctx, plan.ID); err == nil && ok {
			pub.Title = plan.Title
			pub.UpdatedAt = plan.UpdatedAt
			_ = s.publics.Upsert(ctx, pub)
		}
	}
	return plan, nil
}

func (s *Service) Edit(ctx context.Context, userID, planID string, in EditInput) (UserPlan, error) {
	plan, err := s.plans.Get(ctx, planID, userID)
	if err != nil {
		return UserPlan{}, err
	}
	if in.Title != nil {
		if err := validateTitle(*in.Title); err != nil {
			return UserPlan{}, err
		}
		plan.Title = strings.TrimSpace(*in.Title)
	}
	if in.Note != nil {
		plan.Note = strings.TrimSpace(*in.Note)
	}
	if in.Summary != nil {
		plan.Summary = strings.TrimSpace(*in.Summary)
	}
	if in.Tags != nil {
		plan.Tags = trimNonEmpty(*in.Tags)
	}
	if in.Visibility != nil {
		v := strings.ToLower(strings.TrimSpace(*in.Visibility))
		if v != VisibilityPrivate && v != VisibilityPublic {
			return UserPlan{}, ErrInvalidVisibility
		}
		plan.Visibility = v
	}
	plan.UpdatedAt = s.now().UTC()
	if err := s.plans.Update(ctx, plan); err != nil {
		return UserPlan{}, err
	}
	if plan.PublishStatus == PublishStatusPublished {
		if pub, ok, err := s.publics.GetByPlanID(ctx, plan.ID); err == nil && ok {
			pub.Title = plan.Title
			pub.Summary = plan.Summary
			pub.Tags = append([]string{}, plan.Tags...)
			pub.UpdatedAt = plan.UpdatedAt
			_ = s.publics.Upsert(ctx, pub)
		}
	}
	return plan, nil
}

func (s *Service) Delete(ctx context.Context, userID, planID string) error {
	plan, err := s.plans.Get(ctx, planID, userID)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	if err := s.plans.SoftDelete(ctx, plan.ID, userID, now); err != nil {
		return err
	}
	if plan.PublishStatus == PublishStatusPublished {
		if pub, ok, err := s.publics.GetByPlanID(ctx, plan.ID); err == nil && ok {
			_ = s.publics.SetStatus(ctx, pub.ID, PublicPlanStatusUnpublished, now)
		}
	}
	return nil
}

func (s *Service) Publish(ctx context.Context, userID, planID string, in PublishInput) (PublicPlan, error) {
	plan, err := s.plans.Get(ctx, planID, userID)
	if err != nil {
		return PublicPlan{}, err
	}
	now := s.now().UTC()
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = plan.Title
	}
	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		summary = plan.Summary
	}
	tags := trimNonEmpty(in.Tags)
	if len(tags) == 0 {
		tags = append([]string{}, plan.Tags...)
	}
	if err := validateTitle(title); err != nil {
		return PublicPlan{}, err
	}

	pub := PublicPlan{
		PlanID:          plan.ID,
		UserID:          plan.UserID,
		Title:           title,
		Summary:         summary,
		Tags:            tags,
		Plan:            cloneTravelPlan(plan.Plan),
		DestinationCity: plan.DestinationCity,
		Days:            plan.Days,
		Status:          PublicPlanStatusPublished,
		PublishedAt:     now,
		UpdatedAt:       now,
	}
	if existing, ok, err := s.publics.GetByPlanID(ctx, plan.ID); err == nil && ok {
		pub.ID = existing.ID
		pub.PublishedAt = existing.PublishedAt
	} else {
		pub.ID = NewPublicPlanID()
	}
	if s.authors != nil {
		pub.AuthorName = s.authors.DisplayName(ctx, plan.UserID)
	}
	if err := s.publics.Upsert(ctx, pub); err != nil {
		return PublicPlan{}, err
	}

	plan.Visibility = VisibilityPublic
	plan.PublishStatus = PublishStatusPublished
	plan.UpdatedAt = now
	if err := s.plans.Update(ctx, plan); err != nil {
		return PublicPlan{}, err
	}
	stored, err := s.publics.Get(ctx, pub.ID)
	if err != nil {
		return PublicPlan{}, err
	}
	return stored, nil
}

func (s *Service) Unpublish(ctx context.Context, userID, planID string) error {
	plan, err := s.plans.Get(ctx, planID, userID)
	if err != nil {
		return err
	}
	if plan.PublishStatus != PublishStatusPublished {
		return ErrNotPublished
	}
	now := s.now().UTC()
	if pub, ok, err := s.publics.GetByPlanID(ctx, plan.ID); err == nil && ok {
		if err := s.publics.SetStatus(ctx, pub.ID, PublicPlanStatusUnpublished, now); err != nil {
			return err
		}
	}
	plan.Visibility = VisibilityPrivate
	plan.PublishStatus = PublishStatusUnpublished
	plan.UpdatedAt = now
	return s.plans.Update(ctx, plan)
}

func (s *Service) ListPublic(ctx context.Context, filter PublicListFilter) ([]PublicPlan, int, error) {
	return s.publics.List(ctx, filter)
}

func (s *Service) GetPublic(ctx context.Context, id string, viewer PublicViewer) (PublicPlan, error) {
	pub, err := s.publics.Get(ctx, id)
	if err != nil {
		return PublicPlan{}, err
	}
	if pub.Status != PublicPlanStatusPublished {
		return PublicPlan{}, ErrPublicPlanNotFound
	}
	now := s.now().UTC()
	shouldCount := true
	viewerKey := viewer.DedupKey()
	if s.viewDeduper != nil && viewerKey != "" {
		allowed, err := s.viewDeduper.Allow(ctx, pub.ID, viewerKey, now)
		shouldCount = err != nil || allowed
	}
	if !shouldCount {
		return pub, nil
	}
	// Counter increments are best-effort and must not block the read path.
	if s.incrementPublicCounter(ctx, pub.ID, CounterView, viewer, now) {
		pub.ViewCount++
		pub.HotScore = computeHotScore(pub)
	}
	return pub, nil
}

// SavePublicAsCopy creates a private user plan owned by the viewer that
// contains a snapshot of the public plan. The author's note and archive are
// NOT copied — only the rendered plan, summary, and tags.
func (s *Service) SavePublicAsCopy(ctx context.Context, viewerID, publicPlanID string) (UserPlan, error) {
	if viewerID == "" {
		return UserPlan{}, ErrPlanNotFound
	}
	pub, err := s.publics.Get(ctx, publicPlanID)
	if err != nil {
		return UserPlan{}, err
	}
	if pub.Status != PublicPlanStatusPublished {
		return UserPlan{}, ErrSourcePlanForbidden
	}
	now := s.now().UTC()
	plan := UserPlan{
		ID:                 NewUserPlanID(),
		UserID:             viewerID,
		SourcePublicPlanID: pub.ID,
		Title:              pub.Title,
		Summary:            pub.Summary,
		Tags:               append([]string{}, pub.Tags...),
		Plan:               cloneTravelPlan(pub.Plan),
		DestinationCity:    pub.DestinationCity,
		Days:               pub.Days,
		Visibility:         VisibilityPrivate,
		PublishStatus:      PublishStatusDraft,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.plans.Create(ctx, plan); err != nil {
		return UserPlan{}, err
	}
	_ = s.incrementPublicCounter(ctx, pub.ID, CounterSave, PublicViewer{UserID: viewerID}, now)
	return plan, nil
}

func (s *Service) incrementPublicCounter(ctx context.Context, publicPlanID string, kind PublicCounterKind, viewer PublicViewer, now time.Time) bool {
	if err := s.publics.IncrementCounter(ctx, publicPlanID, kind, now); err != nil {
		return false
	}
	_ = s.publics.RecordEvent(ctx, PublicPlanEvent{
		PublicPlanID: publicPlanID,
		UserID:       viewer.UserID,
		EventType:    string(kind),
		ClientHash:   viewer.ClientHash,
		CreatedAt:    now,
	})
	return true
}

// Current returns the user's currently running task (if any) plus the most
// recently saved plan. RunningTask is only populated when the caller passes a
// non-nil running snapshot; we don't reach into travel internals from here.
func (s *Service) Current(ctx context.Context, userID string, running *RunningTask) (CurrentView, error) {
	view := CurrentView{RunningTask: running}
	plans, _, err := s.plans.List(ctx, userID, ListFilter{Page: 1, PageSize: 1})
	if err != nil {
		return CurrentView{}, err
	}
	if len(plans) > 0 {
		latest := plans[0]
		view.LatestPlan = &latest
	}
	return view, nil
}

func validateTitle(title string) error {
	t := strings.TrimSpace(title)
	if t == "" {
		return ErrInvalidTitle
	}
	runes := []rune(t)
	if len(runes) < minTitleLength || len(runes) > maxTitleLength {
		return ErrInvalidTitle
	}
	return nil
}

func tagsForPlan(plan *domain.TravelPlan, req domain.TravelRequest) []string {
	tags := make([]string, 0, 4)
	if req.DestinationCity != "" {
		tags = append(tags, req.DestinationCity)
	}
	if req.Days > 0 {
		tags = append(tags, fmt.Sprintf("%d日", req.Days))
	}
	for _, interest := range req.Interests {
		if interest = strings.TrimSpace(interest); interest != "" {
			tags = append(tags, interest)
		}
	}
	return dedupeStrings(tags)
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func trimNonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if v := strings.TrimSpace(value); v != "" {
			out = append(out, v)
		}
	}
	return dedupeStrings(out)
}

func cloneTravelPlan(plan *domain.TravelPlan) *domain.TravelPlan {
	if plan == nil {
		return nil
	}
	clone := *plan
	clone.Days = append([]domain.TravelDay(nil), plan.Days...)
	clone.Warnings = append([]string(nil), plan.Warnings...)
	return &clone
}

func copyTravelRequest(req domain.TravelRequest) *domain.TravelRequest {
	clone := req
	clone.Interests = append([]string(nil), req.Interests...)
	clone.MustVisit = append([]string(nil), req.MustVisit...)
	clone.Avoid = append([]string(nil), req.Avoid...)
	clone.BudgetIncludes = append([]string(nil), req.BudgetIncludes...)
	return &clone
}

func NewUserPlanID() string {
	return prefixedRandomID("plan_")
}

func NewPublicPlanID() string {
	return prefixedRandomID("pub_")
}

func NewArchiveID() string {
	return prefixedRandomID("arc_")
}

func prefixedRandomID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		return prefix + hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano())
}
