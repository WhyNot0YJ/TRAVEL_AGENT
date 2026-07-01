package plans

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"travel-agent/internal/domain"
)

// MySQLPlanStore implements PlanStore on top of MySQL. Read paths always
// include user_id in the WHERE clause to enforce ownership at the DB layer.
type MySQLPlanStore struct {
	db *sql.DB
}

func NewMySQLPlanStore(db *sql.DB) *MySQLPlanStore {
	return &MySQLPlanStore{db: db}
}

func (s *MySQLPlanStore) Create(ctx context.Context, plan UserPlan) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql plan store not initialized")
	}
	planJSON, err := json.Marshal(plan.Plan)
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	tagsJSON, err := json.Marshal(nonNilTags(plan.Tags))
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO user_plans (id, user_id, task_id, source_public_plan_id, title, note, summary, tags_json, plan_json, destination_city, days, visibility, publish_status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		plan.ID,
		plan.UserID,
		nullableString(plan.TaskID),
		nullableString(plan.SourcePublicPlanID),
		plan.Title,
		nullableString(plan.Note),
		nullableString(plan.Summary),
		string(tagsJSON),
		string(planJSON),
		plan.DestinationCity,
		plan.Days,
		plan.Visibility,
		plan.PublishStatus,
		plan.CreatedAt.UTC(),
		plan.UpdatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert user_plan: %w", err)
	}
	return nil
}

func (s *MySQLPlanStore) Get(ctx context.Context, planID, userID string) (UserPlan, error) {
	if s == nil || s.db == nil {
		return UserPlan{}, fmt.Errorf("mysql plan store not initialized")
	}
	row := s.db.QueryRowContext(ctx, planSelectColumns+`
FROM user_plans
WHERE id = ? AND user_id = ? AND deleted_at IS NULL`, planID, userID)
	plan, err := scanUserPlan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return UserPlan{}, ErrPlanNotFound
	}
	return plan, err
}

func (s *MySQLPlanStore) GetByID(ctx context.Context, planID string) (UserPlan, error) {
	if s == nil || s.db == nil {
		return UserPlan{}, fmt.Errorf("mysql plan store not initialized")
	}
	row := s.db.QueryRowContext(ctx, planSelectColumns+`
FROM user_plans WHERE id = ?`, planID)
	plan, err := scanUserPlan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return UserPlan{}, ErrPlanNotFound
	}
	return plan, err
}

func (s *MySQLPlanStore) GetByTaskID(ctx context.Context, userID, taskID string) (UserPlan, bool, error) {
	if s == nil || s.db == nil {
		return UserPlan{}, false, fmt.Errorf("mysql plan store not initialized")
	}
	row := s.db.QueryRowContext(ctx, planSelectColumns+`
FROM user_plans
WHERE user_id = ? AND task_id = ? AND deleted_at IS NULL
ORDER BY updated_at DESC
LIMIT 1`, userID, taskID)
	plan, err := scanUserPlan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return UserPlan{}, false, nil
	}
	if err != nil {
		return UserPlan{}, false, err
	}
	return plan, true, nil
}

func (s *MySQLPlanStore) List(ctx context.Context, userID string, filter ListFilter) ([]UserPlan, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("mysql plan store not initialized")
	}
	page, size := normalizePagination(filter.Page, filter.PageSize)
	args := []any{userID}
	where := []string{"user_id = ?", "deleted_at IS NULL"}
	if filter.Visibility != "" {
		where = append(where, "visibility = ?")
		args = append(args, filter.Visibility)
	}
	if filter.PublishStatus != "" {
		where = append(where, "publish_status = ?")
		args = append(args, filter.PublishStatus)
	}
	if filter.DestinationCity != "" {
		where = append(where, "destination_city = ?")
		args = append(args, filter.DestinationCity)
	}
	if filter.Query != "" {
		where = append(where, "(title LIKE ? OR destination_city LIKE ? OR summary LIKE ?)")
		like := "%" + filter.Query + "%"
		args = append(args, like, like, like)
	}
	whereClause := strings.Join(where, " AND ")

	countRow := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_plans WHERE "+whereClause, args...)
	var total int
	if err := countRow.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count plans: %w", err)
	}

	listArgs := append(append([]any{}, args...), size, (page-1)*size)
	rows, err := s.db.QueryContext(ctx, planSelectColumns+`
FROM user_plans
WHERE `+whereClause+`
ORDER BY updated_at DESC
LIMIT ? OFFSET ?`, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()
	var plans []UserPlan
	for rows.Next() {
		plan, err := scanUserPlan(rows)
		if err != nil {
			return nil, 0, err
		}
		plans = append(plans, plan)
	}
	return plans, total, rows.Err()
}

func (s *MySQLPlanStore) Update(ctx context.Context, plan UserPlan) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql plan store not initialized")
	}
	planJSON, err := json.Marshal(plan.Plan)
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	tagsJSON, err := json.Marshal(nonNilTags(plan.Tags))
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE user_plans
SET title = ?, note = ?, summary = ?, tags_json = ?, plan_json = ?, destination_city = ?, days = ?, visibility = ?, publish_status = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		plan.Title,
		nullableString(plan.Note),
		nullableString(plan.Summary),
		string(tagsJSON),
		string(planJSON),
		plan.DestinationCity,
		plan.Days,
		plan.Visibility,
		plan.PublishStatus,
		plan.UpdatedAt.UTC(),
		plan.ID,
		plan.UserID,
	)
	if err != nil {
		return fmt.Errorf("update plan: %w", err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return ErrPlanNotFound
	}
	return nil
}

func (s *MySQLPlanStore) SoftDelete(ctx context.Context, planID, userID string, deletedAt time.Time) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql plan store not initialized")
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE user_plans SET deleted_at = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		deletedAt.UTC(), deletedAt.UTC(), planID, userID)
	if err != nil {
		return fmt.Errorf("soft delete plan: %w", err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return ErrPlanNotFound
	}
	return nil
}

func (s *MySQLPlanStore) CreateArchive(ctx context.Context, archive ConversationArchive) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql plan store not initialized")
	}
	briefJSON, err := jsonOrNull(archive.Brief)
	if err != nil {
		return err
	}
	messagesJSON, err := jsonOrNull(archive.Messages)
	if err != nil {
		return err
	}
	eventsJSON, err := jsonOrNull(archive.Events)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO plan_conversation_archives (id, plan_id, user_id, task_id, brief_json, messages_json, events_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		archive.ID,
		archive.PlanID,
		archive.UserID,
		nullableString(archive.TaskID),
		briefJSON,
		messagesJSON,
		eventsJSON,
		archive.CreatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert archive: %w", err)
	}
	return nil
}

func (s *MySQLPlanStore) GetArchive(ctx context.Context, planID, userID string) (ConversationArchive, error) {
	if s == nil || s.db == nil {
		return ConversationArchive{}, fmt.Errorf("mysql plan store not initialized")
	}
	row := s.db.QueryRowContext(ctx, `
SELECT id, plan_id, user_id, COALESCE(task_id, ''), brief_json, messages_json, events_json, created_at
FROM plan_conversation_archives WHERE plan_id = ? AND user_id = ?`, planID, userID)
	var (
		archive       ConversationArchive
		briefBytes    sql.NullString
		messagesBytes sql.NullString
		eventsBytes   sql.NullString
	)
	err := row.Scan(&archive.ID, &archive.PlanID, &archive.UserID, &archive.TaskID, &briefBytes, &messagesBytes, &eventsBytes, &archive.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ConversationArchive{}, ErrPlanNotFound
	}
	if err != nil {
		return ConversationArchive{}, fmt.Errorf("scan archive: %w", err)
	}
	if briefBytes.Valid && briefBytes.String != "" {
		var brief domain.TravelRequest
		if err := json.Unmarshal([]byte(briefBytes.String), &brief); err == nil {
			archive.Brief = &brief
		}
	}
	if messagesBytes.Valid && messagesBytes.String != "" {
		_ = json.Unmarshal([]byte(messagesBytes.String), &archive.Messages)
	}
	if eventsBytes.Valid && eventsBytes.String != "" {
		_ = json.Unmarshal([]byte(eventsBytes.String), &archive.Events)
	}
	return archive, nil
}

// MySQLPublicPlanStore is the publish-side persistence.
type MySQLPublicPlanStore struct {
	db *sql.DB
}

func NewMySQLPublicPlanStore(db *sql.DB) *MySQLPublicPlanStore {
	return &MySQLPublicPlanStore{db: db}
}

func (s *MySQLPublicPlanStore) Upsert(ctx context.Context, plan PublicPlan) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql public plan store not initialized")
	}
	planJSON, err := json.Marshal(plan.Plan)
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	tagsJSON, err := json.Marshal(nonNilTags(plan.Tags))
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO public_plans (id, plan_id, user_id, title, summary, tags_json, plan_json, destination_city, days, status, hot_score, published_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  title = VALUES(title),
  summary = VALUES(summary),
  tags_json = VALUES(tags_json),
  plan_json = VALUES(plan_json),
  destination_city = VALUES(destination_city),
  days = VALUES(days),
  status = VALUES(status),
  updated_at = VALUES(updated_at)`,
		plan.ID,
		plan.PlanID,
		plan.UserID,
		plan.Title,
		nullableString(plan.Summary),
		string(tagsJSON),
		string(planJSON),
		plan.DestinationCity,
		plan.Days,
		plan.Status,
		plan.HotScore,
		plan.PublishedAt.UTC(),
		plan.UpdatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("upsert public_plan: %w", err)
	}
	return nil
}

func (s *MySQLPublicPlanStore) Get(ctx context.Context, id string) (PublicPlan, error) {
	if s == nil || s.db == nil {
		return PublicPlan{}, fmt.Errorf("mysql public plan store not initialized")
	}
	row := s.db.QueryRowContext(ctx, publicPlanSelectColumns+`
FROM public_plans p LEFT JOIN users u ON u.id = p.user_id WHERE p.id = ?`, id)
	plan, err := scanPublicPlan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return PublicPlan{}, ErrPublicPlanNotFound
	}
	return plan, err
}

func (s *MySQLPublicPlanStore) GetByPlanID(ctx context.Context, planID string) (PublicPlan, bool, error) {
	if s == nil || s.db == nil {
		return PublicPlan{}, false, fmt.Errorf("mysql public plan store not initialized")
	}
	row := s.db.QueryRowContext(ctx, publicPlanSelectColumns+`
FROM public_plans p LEFT JOIN users u ON u.id = p.user_id WHERE p.plan_id = ?`, planID)
	plan, err := scanPublicPlan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return PublicPlan{}, false, nil
	}
	if err != nil {
		return PublicPlan{}, false, err
	}
	return plan, true, nil
}

func (s *MySQLPublicPlanStore) List(ctx context.Context, filter PublicListFilter) ([]PublicPlan, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("mysql public plan store not initialized")
	}
	page, size := normalizePagination(filter.Page, filter.PageSize)
	args := []any{PublicPlanStatusPublished}
	where := []string{"p.status = ?"}
	if filter.DestinationCity != "" {
		where = append(where, "p.destination_city = ?")
		args = append(args, filter.DestinationCity)
	}
	if filter.Days > 0 {
		where = append(where, "p.days = ?")
		args = append(args, filter.Days)
	}
	if filter.Query != "" {
		where = append(where, "(p.title LIKE ? OR p.summary LIKE ? OR p.destination_city LIKE ?)")
		like := "%" + filter.Query + "%"
		args = append(args, like, like, like)
	}
	if filter.Interest != "" {
		where = append(where, "JSON_SEARCH(p.tags_json, 'one', ?) IS NOT NULL")
		args = append(args, filter.Interest)
	}
	whereClause := strings.Join(where, " AND ")

	countRow := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM public_plans p WHERE "+whereClause, args...)
	var total int
	if err := countRow.Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count public plans: %w", err)
	}

	orderBy := "p.hot_score DESC, p.published_at DESC"
	if filter.Sort == "latest" {
		orderBy = "p.published_at DESC"
	}
	listArgs := append(append([]any{}, args...), size, (page-1)*size)
	rows, err := s.db.QueryContext(ctx, publicPlanSelectColumns+`
FROM public_plans p LEFT JOIN users u ON u.id = p.user_id
WHERE `+whereClause+`
ORDER BY `+orderBy+`
LIMIT ? OFFSET ?`, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list public plans: %w", err)
	}
	defer rows.Close()
	var plans []PublicPlan
	for rows.Next() {
		plan, err := scanPublicPlan(rows)
		if err != nil {
			return nil, 0, err
		}
		plans = append(plans, plan)
	}
	return plans, total, rows.Err()
}

func (s *MySQLPublicPlanStore) SetStatus(ctx context.Context, id, status string, updatedAt time.Time) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql public plan store not initialized")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE public_plans SET status = ?, updated_at = ? WHERE id = ?`,
		status, updatedAt.UTC(), id)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return ErrPublicPlanNotFound
	}
	return nil
}

func (s *MySQLPublicPlanStore) IncrementCounter(ctx context.Context, id string, kind PublicCounterKind, now time.Time) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql public plan store not initialized")
	}
	column := ""
	switch kind {
	case CounterView:
		column = "view_count"
	case CounterSave:
		column = "save_count"
	case CounterCopy:
		column = "copy_count"
	default:
		return fmt.Errorf("unknown counter kind %q", kind)
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE public_plans
SET `+column+` = `+column+` + 1,
    hot_score = CASE
      WHEN ? = 'view' THEN (view_count + 1) + save_count * 5 + copy_count * 3
      WHEN ? = 'save' THEN view_count + (save_count + 1) * 5 + copy_count * 3
      WHEN ? = 'copy' THEN view_count + save_count * 5 + (copy_count + 1) * 3
      ELSE view_count + save_count * 5 + copy_count * 3
    END,
    updated_at = ?
WHERE id = ?`,
		string(kind), string(kind), string(kind), now.UTC(), id)
	if err != nil {
		return fmt.Errorf("increment %s: %w", kind, err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return ErrPublicPlanNotFound
	}
	return nil
}

func (s *MySQLPublicPlanStore) RecordEvent(ctx context.Context, event PublicPlanEvent) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql public plan store not initialized")
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO public_plan_events (public_plan_id, user_id, event_type, client_hash, created_at)
VALUES (?, ?, ?, ?, ?)`,
		event.PublicPlanID,
		nullableString(event.UserID),
		event.EventType,
		nullableString(event.ClientHash),
		event.CreatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert public_plan_event: %w", err)
	}
	return nil
}

const planSelectColumns = `
SELECT id, user_id, COALESCE(task_id, ''), COALESCE(source_public_plan_id, ''), title, COALESCE(note, ''), COALESCE(summary, ''), COALESCE(tags_json, '[]'), plan_json, destination_city, days, visibility, publish_status, created_at, updated_at, deleted_at`

const publicPlanSelectColumns = `
SELECT p.id, p.plan_id, p.user_id, COALESCE(u.display_name, ''), p.title, COALESCE(p.summary, ''), COALESCE(p.tags_json, '[]'), p.plan_json, p.destination_city, p.days, p.status, p.view_count, p.save_count, p.copy_count, p.hot_score, p.published_at, p.updated_at`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUserPlan(row rowScanner) (UserPlan, error) {
	var (
		plan      UserPlan
		taskID    string
		source    string
		note      string
		summary   string
		tagsJSON  string
		planJSON  []byte
		deletedAt sql.NullTime
	)
	if err := row.Scan(
		&plan.ID,
		&plan.UserID,
		&taskID,
		&source,
		&plan.Title,
		&note,
		&summary,
		&tagsJSON,
		&planJSON,
		&plan.DestinationCity,
		&plan.Days,
		&plan.Visibility,
		&plan.PublishStatus,
		&plan.CreatedAt,
		&plan.UpdatedAt,
		&deletedAt,
	); err != nil {
		return UserPlan{}, err
	}
	plan.TaskID = taskID
	plan.SourcePublicPlanID = source
	plan.Note = note
	plan.Summary = summary
	if tagsJSON != "" {
		var tags []string
		if err := json.Unmarshal([]byte(tagsJSON), &tags); err == nil {
			plan.Tags = tags
		}
	}
	if len(planJSON) > 0 {
		var travelPlan domain.TravelPlan
		if err := json.Unmarshal(planJSON, &travelPlan); err == nil {
			plan.Plan = &travelPlan
		}
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		plan.DeletedAt = &t
	}
	return plan, nil
}

func scanPublicPlan(row rowScanner) (PublicPlan, error) {
	var (
		plan     PublicPlan
		summary  string
		tagsJSON string
		planJSON []byte
	)
	if err := row.Scan(
		&plan.ID,
		&plan.PlanID,
		&plan.UserID,
		&plan.AuthorName,
		&plan.Title,
		&summary,
		&tagsJSON,
		&planJSON,
		&plan.DestinationCity,
		&plan.Days,
		&plan.Status,
		&plan.ViewCount,
		&plan.SaveCount,
		&plan.CopyCount,
		&plan.HotScore,
		&plan.PublishedAt,
		&plan.UpdatedAt,
	); err != nil {
		return PublicPlan{}, err
	}
	plan.Summary = summary
	if tagsJSON != "" {
		var tags []string
		if err := json.Unmarshal([]byte(tagsJSON), &tags); err == nil {
			plan.Tags = tags
		}
	}
	if len(planJSON) > 0 {
		var travelPlan domain.TravelPlan
		if err := json.Unmarshal(planJSON, &travelPlan); err == nil {
			plan.Plan = &travelPlan
		}
	}
	return plan, nil
}

func nullableString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func jsonOrNull(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch v := value.(type) {
	case []ArchivedMessage:
		if len(v) == 0 {
			return nil, nil
		}
	case []ArchivedEvent:
		if len(v) == 0 {
			return nil, nil
		}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	if string(data) == "null" {
		return nil, nil
	}
	return string(data), nil
}
