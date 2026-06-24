package travel

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"travel-agent/internal/domain"
)

type MySQLTaskStore struct {
	db *sql.DB
}

func NewMySQLTaskStore(db *sql.DB) *MySQLTaskStore {
	return &MySQLTaskStore{db: db}
}

func (s *MySQLTaskStore) Create(ctx context.Context, task Task) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql task store is not initialized")
	}
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if err := insertTask(ctx, tx, task); err != nil {
			return err
		}
		return upsertPlan(ctx, tx, task)
	})
}

func (s *MySQLTaskStore) Update(ctx context.Context, task Task) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql task store is not initialized")
	}
	return s.withTx(ctx, func(tx *sql.Tx) error {
		requestJSON, err := json.Marshal(task.Request)
		if err != nil {
			return fmt.Errorf("marshal task request: %w", err)
		}
		result, err := tx.ExecContext(ctx, `
UPDATE travel_tasks
SET request_id = ?, request_hash = ?, status = ?, request_json = ?, error_text = ?, updated_at = ?
WHERE id = ?`,
			nullableString(task.RequestID),
			task.RequestHash,
			task.Status,
			string(requestJSON),
			nullableString(task.Error),
			normalizeTaskTime(task.UpdatedAt),
			task.ID,
		)
		if err != nil {
			return fmt.Errorf("update travel task: %w", err)
		}
		affected, err := result.RowsAffected()
		if err == nil && affected == 0 {
			return ErrTaskNotFound
		}
		return upsertPlan(ctx, tx, task)
	})
}

func (s *MySQLTaskStore) Get(ctx context.Context, id string) (Task, error) {
	if s == nil || s.db == nil {
		return Task{}, fmt.Errorf("mysql task store is not initialized")
	}
	return scanTask(s.db.QueryRowContext(ctx, `
SELECT t.id, COALESCE(t.request_id, ''), t.request_hash, t.status, t.request_json, COALESCE(t.error_text, ''), t.created_at, t.updated_at, p.plan_json
FROM travel_tasks t
LEFT JOIN travel_plans p ON p.task_id = t.id
WHERE t.id = ?`, id))
}

func (s *MySQLTaskStore) FindByHash(ctx context.Context, requestHash string) (Task, bool, error) {
	if s == nil || s.db == nil {
		return Task{}, false, fmt.Errorf("mysql task store is not initialized")
	}
	task, err := scanTask(s.db.QueryRowContext(ctx, `
SELECT t.id, COALESCE(t.request_id, ''), t.request_hash, t.status, t.request_json, COALESCE(t.error_text, ''), t.created_at, t.updated_at, p.plan_json
FROM travel_tasks t
LEFT JOIN travel_plans p ON p.task_id = t.id
WHERE t.request_hash = ?`, requestHash))
	if errors.Is(err, ErrTaskNotFound) {
		return Task{}, false, nil
	}
	return task, err == nil, err
}

func (s *MySQLTaskStore) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin mysql transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit mysql transaction: %w", err)
	}
	return nil
}

func insertTask(ctx context.Context, tx *sql.Tx, task Task) error {
	requestJSON, err := json.Marshal(task.Request)
	if err != nil {
		return fmt.Errorf("marshal task request: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO travel_tasks (id, request_id, request_hash, status, request_json, error_text, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID,
		nullableString(task.RequestID),
		task.RequestHash,
		task.Status,
		string(requestJSON),
		nullableString(task.Error),
		normalizeTaskTime(task.CreatedAt),
		normalizeTaskTime(task.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert travel task: %w", err)
	}
	return nil
}

func upsertPlan(ctx context.Context, tx *sql.Tx, task Task) error {
	if task.Plan == nil {
		return nil
	}
	planJSON, err := json.Marshal(task.Plan)
	if err != nil {
		return fmt.Errorf("marshal task plan: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO travel_plans (task_id, plan_json, budget_total, day_count, warning_count, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
	plan_json = VALUES(plan_json),
	budget_total = VALUES(budget_total),
	day_count = VALUES(day_count),
	warning_count = VALUES(warning_count),
	updated_at = VALUES(updated_at)`,
		task.ID,
		string(planJSON),
		task.Plan.Budget.Total,
		len(task.Plan.Days),
		len(task.Plan.Warnings),
		normalizeTaskTime(task.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("upsert travel plan: %w", err)
	}
	return nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(row taskScanner) (Task, error) {
	var (
		task        Task
		status      string
		requestJSON []byte
		planJSON    sql.NullString
	)
	if err := row.Scan(
		&task.ID,
		&task.RequestID,
		&task.RequestHash,
		&status,
		&requestJSON,
		&task.Error,
		&task.CreatedAt,
		&task.UpdatedAt,
		&planJSON,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrTaskNotFound
		}
		return Task{}, fmt.Errorf("scan travel task: %w", err)
	}
	task.Status = TaskStatus(status)
	if err := json.Unmarshal(requestJSON, &task.Request); err != nil {
		return Task{}, fmt.Errorf("decode task request: %w", err)
	}
	if planJSON.Valid && planJSON.String != "" {
		var plan domain.TravelPlan
		if err := json.Unmarshal([]byte(planJSON.String), &plan); err != nil {
			return Task{}, fmt.Errorf("decode task plan: %w", err)
		}
		task.Plan = &plan
	}
	return task, nil
}

func nullableString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func normalizeTaskTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}
