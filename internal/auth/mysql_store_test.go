package auth

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMySQLUserStoreCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewMySQLUserStore(db)
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	user := User{
		ID:           "user_abc",
		Email:        "alice@example.com",
		DisplayName:  "Alice",
		PasswordHash: "hashed",
		Status:       UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users")).
		WithArgs(user.ID, user.Email, user.DisplayName, user.PasswordHash, user.Status, now.UTC(), now.UTC()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := store.Create(context.Background(), user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMySQLUserStoreCreateMapsDuplicate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewMySQLUserStore(db)
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	user := User{
		ID:           "user_abc",
		Email:        "alice@example.com",
		DisplayName:  "Alice",
		PasswordHash: "hashed",
		Status:       UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users")).
		WithArgs(user.ID, user.Email, user.DisplayName, user.PasswordHash, user.Status, now.UTC(), now.UTC()).
		WillReturnError(errors.New("Error 1062: Duplicate entry"))

	if err := store.Create(context.Background(), user); !errors.Is(err, ErrEmailExists) {
		t.Fatalf("expected ErrEmailExists, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMySQLUserStoreGetByEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewMySQLUserStore(db)
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"id", "email", "display_name", "password_hash", "status", "created_at", "updated_at"}).
		AddRow("user_abc", "alice@example.com", "Alice", "hashed", UserStatusActive, now, now)
	mock.ExpectQuery(regexp.QuoteMeta("FROM users WHERE email = ?")).WithArgs("alice@example.com").WillReturnRows(rows)

	got, err := store.GetByEmail(context.Background(), "alice@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got.ID != "user_abc" {
		t.Fatalf("unexpected user: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestMySQLSessionStoreLifecycle(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewMySQLSessionStore(db)
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	session := Session{
		ID:        "sess_abc",
		UserID:    "user_abc",
		TokenHash: "hash_abc",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO user_sessions")).
		WithArgs(session.ID, session.UserID, session.TokenHash, session.ExpiresAt.UTC(), session.CreatedAt.UTC(), nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := store.Create(context.Background(), session); err != nil {
		t.Fatalf("Create: %v", err)
	}

	rows := sqlmock.NewRows([]string{"id", "user_id", "token_hash", "expires_at", "created_at", "revoked_at"}).
		AddRow(session.ID, session.UserID, session.TokenHash, session.ExpiresAt, session.CreatedAt, nil)
	mock.ExpectQuery(regexp.QuoteMeta("FROM user_sessions WHERE token_hash = ?")).WithArgs(session.TokenHash).WillReturnRows(rows)
	got, err := store.GetByTokenHash(context.Background(), session.TokenHash)
	if err != nil {
		t.Fatalf("GetByTokenHash: %v", err)
	}
	if got.ID != session.ID {
		t.Fatalf("unexpected session: %+v", got)
	}

	mock.ExpectExec(regexp.QuoteMeta("UPDATE user_sessions SET revoked_at = ?")).
		WithArgs(now.UTC(), session.TokenHash).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := store.Revoke(context.Background(), session.TokenHash, now); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
