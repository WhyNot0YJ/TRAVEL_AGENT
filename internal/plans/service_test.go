package plans

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"travel-agent/internal/domain"
)

type stubTaskLookup struct {
	tasks map[string]TaskSnapshot
}

func (s stubTaskLookup) LookupTask(ctx context.Context, taskID string) (TaskSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return TaskSnapshot{}, err
	}
	task, ok := s.tasks[taskID]
	if !ok {
		return TaskSnapshot{}, ErrTaskNotFound
	}
	return task, nil
}

type stubAuthorLookup struct{ name string }

func (s stubAuthorLookup) DisplayName(ctx context.Context, userID string) string { return s.name }

func newTestService(t *testing.T, snapshots ...TaskSnapshot) (*Service, *MemoryPlanStore, *MemoryPublicPlanStore) {
	t.Helper()
	plans := NewMemoryPlanStore()
	publics := NewMemoryPublicPlanStore()
	tasks := stubTaskLookup{tasks: map[string]TaskSnapshot{}}
	for _, snap := range snapshots {
		tasks.tasks[snap.TaskID] = snap
	}
	svc := NewService(plans, publics, tasks, stubAuthorLookup{name: "Alice"})
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	return svc, plans, publics
}

func sampleSnapshot(taskID, userID string) TaskSnapshot {
	return TaskSnapshot{
		TaskID: taskID,
		UserID: userID,
		Status: TaskStatusSucceeded,
		Plan: &domain.TravelPlan{
			Title:   "杭州 3 日旅行",
			Summary: "西湖与灵隐",
			Days:    []domain.TravelDay{{Day: 1}, {Day: 2}, {Day: 3}},
			Budget:  domain.TravelBudget{Total: 2000},
		},
		Request: domain.TravelRequest{
			DepartureCity:   "上海",
			DestinationCity: "杭州",
			Days:            3,
			Budget:          3000,
			Interests:       []string{"自然风光", "美食"},
			Travelers:       2,
		},
	}
}

func TestServiceSavePlanAttachesArchive(t *testing.T) {
	snap := sampleSnapshot("task_a", "user_a")
	svc, plans, _ := newTestService(t, snap)
	plan, err := svc.Save(context.Background(), "user_a", SaveInput{TaskID: "task_a"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if plan.UserID != "user_a" || plan.Visibility != VisibilityPrivate || plan.PublishStatus != PublishStatusDraft {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if !contains(plan.Tags, "杭州") || !contains(plan.Tags, "3日") || !contains(plan.Tags, "美食") {
		t.Fatalf("expected tags from request, got %v", plan.Tags)
	}
	archive, err := plans.GetArchive(context.Background(), plan.ID, "user_a")
	if err != nil {
		t.Fatalf("GetArchive: %v", err)
	}
	if archive.Brief == nil || archive.Brief.DestinationCity != "杭州" {
		t.Fatalf("expected archived brief, got %+v", archive)
	}
}

func TestServiceSaveDeduplicatesByTask(t *testing.T) {
	snap := sampleSnapshot("task_a", "user_a")
	svc, _, _ := newTestService(t, snap)
	first, err := svc.Save(context.Background(), "user_a", SaveInput{TaskID: "task_a"})
	if err != nil {
		t.Fatalf("first Save: %v", err)
	}
	second, err := svc.Save(context.Background(), "user_a", SaveInput{TaskID: "task_a", Title: "ignored"})
	if err != nil {
		t.Fatalf("second Save: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected same plan id on duplicate save, got %s and %s", first.ID, second.ID)
	}
}

func TestServiceSaveRejectsUnsucceededTask(t *testing.T) {
	snap := sampleSnapshot("task_a", "user_a")
	snap.Status = "running"
	svc, _, _ := newTestService(t, snap)
	_, err := svc.Save(context.Background(), "user_a", SaveInput{TaskID: "task_a"})
	if !errors.Is(err, ErrTaskNotSucceeded) {
		t.Fatalf("expected ErrTaskNotSucceeded, got %v", err)
	}
}

func TestServiceSaveRejectsTaskOwnedByOther(t *testing.T) {
	snap := sampleSnapshot("task_a", "user_other")
	svc, _, _ := newTestService(t, snap)
	_, err := svc.Save(context.Background(), "user_a", SaveInput{TaskID: "task_a"})
	if !errors.Is(err, ErrTaskNotOwned) {
		t.Fatalf("expected ErrTaskNotOwned, got %v", err)
	}
}

func TestServiceGetReturnsErrPlanNotFoundForOtherUser(t *testing.T) {
	snap := sampleSnapshot("task_a", "user_a")
	svc, _, _ := newTestService(t, snap)
	plan, err := svc.Save(context.Background(), "user_a", SaveInput{TaskID: "task_a"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := svc.Get(context.Background(), "user_b", plan.ID); !errors.Is(err, ErrPlanNotFound) {
		t.Fatalf("expected ErrPlanNotFound when accessing another user's plan, got %v", err)
	}
}

func TestServiceRenameAndEditUpdateMetadata(t *testing.T) {
	snap := sampleSnapshot("task_a", "user_a")
	svc, _, _ := newTestService(t, snap)
	plan, err := svc.Save(context.Background(), "user_a", SaveInput{TaskID: "task_a"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	renamed, err := svc.Rename(context.Background(), "user_a", plan.ID, "杭州亲子轻松路线")
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if renamed.Title != "杭州亲子轻松路线" {
		t.Fatalf("unexpected title: %q", renamed.Title)
	}

	tags := []string{"杭州", "亲子", "美食"}
	note := "周末适合出行"
	edited, err := svc.Edit(context.Background(), "user_a", plan.ID, EditInput{Note: &note, Tags: &tags})
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if edited.Note != note || len(edited.Tags) != 3 {
		t.Fatalf("unexpected edited plan: %+v", edited)
	}
}

func TestServiceRenameRejectsShortTitle(t *testing.T) {
	snap := sampleSnapshot("task_a", "user_a")
	svc, _, _ := newTestService(t, snap)
	plan, err := svc.Save(context.Background(), "user_a", SaveInput{TaskID: "task_a"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := svc.Rename(context.Background(), "user_a", plan.ID, "x"); !errors.Is(err, ErrInvalidTitle) {
		t.Fatalf("expected ErrInvalidTitle, got %v", err)
	}
}

func TestServicePublishUnpublishRoundTrip(t *testing.T) {
	snap := sampleSnapshot("task_a", "user_a")
	svc, _, publics := newTestService(t, snap)
	plan, err := svc.Save(context.Background(), "user_a", SaveInput{TaskID: "task_a"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	pub, err := svc.Publish(context.Background(), "user_a", plan.ID, PublishInput{Tags: []string{"杭州", "3日", "美食"}})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if pub.Status != PublicPlanStatusPublished || pub.AuthorName != "Alice" {
		t.Fatalf("unexpected publish result: %+v", pub)
	}
	results, total, err := svc.ListPublic(context.Background(), PublicListFilter{Sort: "hot"})
	if err != nil {
		t.Fatalf("ListPublic: %v", err)
	}
	if total != 1 || len(results) != 1 || results[0].ID != pub.ID {
		t.Fatalf("expected published plan in list, got total=%d results=%v", total, results)
	}

	if err := svc.Unpublish(context.Background(), "user_a", plan.ID); err != nil {
		t.Fatalf("Unpublish: %v", err)
	}
	results, total, err = svc.ListPublic(context.Background(), PublicListFilter{Sort: "hot"})
	if err != nil {
		t.Fatalf("ListPublic after unpublish: %v", err)
	}
	if total != 0 || len(results) != 0 {
		t.Fatalf("expected unpublish to remove from list, got total=%d", total)
	}

	// Confirm GetPublic on unpublished id returns not found.
	if _, err := publics.Get(context.Background(), pub.ID); err != nil {
		t.Fatalf("Get from store should still succeed: %v", err)
	}
	if _, err := svc.GetPublic(context.Background(), pub.ID, ""); !errors.Is(err, ErrPublicPlanNotFound) {
		t.Fatalf("expected ErrPublicPlanNotFound, got %v", err)
	}
}

func TestServiceDeleteDeactivatesPublicMirror(t *testing.T) {
	snap := sampleSnapshot("task_a", "user_a")
	svc, _, _ := newTestService(t, snap)
	plan, err := svc.Save(context.Background(), "user_a", SaveInput{TaskID: "task_a"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := svc.Publish(context.Background(), "user_a", plan.ID, PublishInput{}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if err := svc.Delete(context.Background(), "user_a", plan.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(context.Background(), "user_a", plan.ID); !errors.Is(err, ErrPlanNotFound) {
		t.Fatalf("expected deleted plan to be hidden, got %v", err)
	}
	results, total, err := svc.ListPublic(context.Background(), PublicListFilter{})
	if err != nil {
		t.Fatalf("ListPublic: %v", err)
	}
	if total != 0 || len(results) != 0 {
		t.Fatalf("expected deletion to suppress public listing, got %d", total)
	}
}

func TestServiceSavePublicAsCopyCreatesPrivateClone(t *testing.T) {
	authorSnap := sampleSnapshot("task_a", "user_author")
	svc, plans, _ := newTestService(t, authorSnap)
	original, err := svc.Save(context.Background(), "user_author", SaveInput{TaskID: "task_a"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	pub, err := svc.Publish(context.Background(), "user_author", original.ID, PublishInput{})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	copyPlan, err := svc.SavePublicAsCopy(context.Background(), "user_viewer", pub.ID)
	if err != nil {
		t.Fatalf("SavePublicAsCopy: %v", err)
	}
	if copyPlan.UserID != "user_viewer" || copyPlan.SourcePublicPlanID != pub.ID || copyPlan.Visibility != VisibilityPrivate {
		t.Fatalf("unexpected copy: %+v", copyPlan)
	}
	if copyPlan.ID == original.ID {
		t.Fatal("expected new id for copy")
	}
	// Viewer should not see author's archive.
	if _, err := plans.GetArchive(context.Background(), copyPlan.ID, "user_viewer"); !errors.Is(err, ErrPlanNotFound) {
		t.Fatalf("expected no archive on copy, got %v", err)
	}
}

func TestServiceListPublicSortAndSearch(t *testing.T) {
	authorSnap := sampleSnapshot("task_a", "user_author")
	svc, _, _ := newTestService(t, authorSnap)
	plan, err := svc.Save(context.Background(), "user_author", SaveInput{TaskID: "task_a"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := svc.Publish(context.Background(), "user_author", plan.ID, PublishInput{Title: "杭州亲子游", Tags: []string{"杭州", "亲子"}}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	results, _, err := svc.ListPublic(context.Background(), PublicListFilter{Query: "亲子"})
	if err != nil {
		t.Fatalf("ListPublic: %v", err)
	}
	if len(results) != 1 || !strings.Contains(results[0].Title, "亲子") {
		t.Fatalf("expected tag/title search to match, got %v", results)
	}

	noMatch, _, err := svc.ListPublic(context.Background(), PublicListFilter{DestinationCity: "苏州"})
	if err != nil {
		t.Fatalf("ListPublic: %v", err)
	}
	if len(noMatch) != 0 {
		t.Fatalf("expected no match for unknown destination, got %v", noMatch)
	}
}

func TestServiceCurrentReturnsLatestPlan(t *testing.T) {
	snap := sampleSnapshot("task_a", "user_a")
	svc, _, _ := newTestService(t, snap)
	plan, err := svc.Save(context.Background(), "user_a", SaveInput{TaskID: "task_a"})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	view, err := svc.Current(context.Background(), "user_a", &RunningTask{TaskID: "task_b", Status: "running"})
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if view.RunningTask == nil || view.RunningTask.TaskID != "task_b" {
		t.Fatalf("expected running_task, got %+v", view.RunningTask)
	}
	if view.LatestPlan == nil || view.LatestPlan.ID != plan.ID {
		t.Fatalf("expected latest_plan to match saved plan, got %+v", view.LatestPlan)
	}
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
