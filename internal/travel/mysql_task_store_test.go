package travel

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"travel-agent/internal/domain"
)

func TestMySQLTaskStoreCreateAndGet(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewMySQLTaskStore(db)
	task := mysqlTestTask()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO travel_tasks")).
		WithArgs(task.ID, task.RequestHash, task.Status, sqlmock.AnyArg(), nil, task.CreatedAt.UTC(), task.UpdatedAt.UTC()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := store.Create(context.Background(), task); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	rows := sqlmock.NewRows([]string{"id", "request_hash", "status", "request_json", "error_text", "created_at", "updated_at", "plan_json"}).
		AddRow(task.ID, task.RequestHash, string(task.Status), mustJSONValue(t, task.Request), "", task.CreatedAt.UTC(), task.UpdatedAt.UTC(), nil)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT t.id, t.request_hash, t.status")).
		WithArgs(task.ID).
		WillReturnRows(rows)

	got, err := store.Get(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.ID != task.ID || got.Request.DestinationCity != task.Request.DestinationCity || got.Plan != nil {
		t.Fatalf("unexpected task: %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMySQLTaskStoreUpdatePersistsPlan(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewMySQLTaskStore(db)
	task := mysqlTestTask()
	task.Status = TaskSucceeded
	task.Plan = &domain.TravelPlan{
		Title:   "杭州2日游",
		Summary: "杭州2日计划",
		Days: []domain.TravelDay{
			{Day: 1, Theme: "西湖", Items: []domain.TravelItem{{Time: "09:30", Type: "nature", Name: "西湖", Address: "杭州", Reason: "自然风光", DurationMinutes: 120}}},
			{Day: 2, Theme: "灵隐", Items: []domain.TravelItem{{Time: "09:30", Type: "culture", Name: "灵隐寺", Address: "杭州", Reason: "文化", DurationMinutes: 120}}},
		},
		Budget:   domain.TravelBudget{Total: 1200},
		Warnings: []string{"check weather"},
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE travel_tasks")).
		WithArgs(task.RequestHash, task.Status, sqlmock.AnyArg(), nil, task.UpdatedAt.UTC(), task.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO travel_plans")).
		WithArgs(task.ID, sqlmock.AnyArg(), task.Plan.Budget.Total, len(task.Plan.Days), len(task.Plan.Warnings), task.UpdatedAt.UTC()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := store.Update(context.Background(), task); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	rows := sqlmock.NewRows([]string{"id", "request_hash", "status", "request_json", "error_text", "created_at", "updated_at", "plan_json"}).
		AddRow(task.ID, task.RequestHash, string(task.Status), mustJSONValue(t, task.Request), "", task.CreatedAt.UTC(), task.UpdatedAt.UTC(), mustJSONValue(t, task.Plan))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT t.id, t.request_hash, t.status")).
		WithArgs(task.RequestHash).
		WillReturnRows(rows)

	got, ok, err := store.FindByHash(context.Background(), task.RequestHash)
	if err != nil {
		t.Fatalf("FindByHash returned error: %v", err)
	}
	if !ok || got.Plan == nil || got.Plan.Budget.Total != 1200 {
		t.Fatalf("unexpected find result ok=%t task=%#v", ok, got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMySQLTaskStoreUpdateMissingTask(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewMySQLTaskStore(db)
	task := mysqlTestTask()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE travel_tasks")).
		WithArgs(task.RequestHash, task.Status, sqlmock.AnyArg(), nil, task.UpdatedAt.UTC(), task.ID).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	if err := store.Update(context.Background(), task); err != ErrTaskNotFound {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func mysqlTestTask() Task {
	now := time.Date(2026, 6, 24, 10, 0, 0, 0, time.UTC)
	return Task{
		ID:          "task_mysql",
		RequestHash: "hash_mysql",
		Status:      TaskPending,
		Request: domain.TravelRequest{
			ID:              "case_mysql",
			DepartureCity:   "上海",
			DestinationCity: "杭州",
			Days:            2,
			Budget:          3000,
			Interests:       []string{"自然风光"},
			TransportMode:   "train_taxi",
			Pace:            "balanced",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func mustJSONValue(t *testing.T, value any) driver.Value {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return string(data)
}
