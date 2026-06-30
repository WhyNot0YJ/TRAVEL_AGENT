package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// MySQLUserStore persists users in MySQL. SQL keeps a UNIQUE index on email so
// the duplicate check race is handled at the DB level.
type MySQLUserStore struct {
	db *sql.DB
}

func NewMySQLUserStore(db *sql.DB) *MySQLUserStore {
	return &MySQLUserStore{db: db}
}

func (s *MySQLUserStore) Create(ctx context.Context, user User) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql user store not initialized")
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO users (id, email, display_name, password_hash, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.ID,
		user.Email,
		user.DisplayName,
		user.PasswordHash,
		user.Status,
		user.CreatedAt.UTC(),
		user.UpdatedAt.UTC(),
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrEmailExists
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (s *MySQLUserStore) GetByID(ctx context.Context, id string) (User, error) {
	if s == nil || s.db == nil {
		return User{}, fmt.Errorf("mysql user store not initialized")
	}
	return scanUser(s.db.QueryRowContext(ctx, `
SELECT id, email, display_name, password_hash, status, created_at, updated_at
FROM users WHERE id = ?`, id))
}

func (s *MySQLUserStore) GetByEmail(ctx context.Context, email string) (User, error) {
	if s == nil || s.db == nil {
		return User{}, fmt.Errorf("mysql user store not initialized")
	}
	return scanUser(s.db.QueryRowContext(ctx, `
SELECT id, email, display_name, password_hash, status, created_at, updated_at
FROM users WHERE email = ?`, email))
}

type MySQLSessionStore struct {
	db *sql.DB
}

func NewMySQLSessionStore(db *sql.DB) *MySQLSessionStore {
	return &MySQLSessionStore{db: db}
}

func (s *MySQLSessionStore) Create(ctx context.Context, session Session) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql session store not initialized")
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO user_sessions (id, user_id, token_hash, expires_at, created_at, revoked_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		session.ID,
		session.UserID,
		session.TokenHash,
		session.ExpiresAt.UTC(),
		session.CreatedAt.UTC(),
		nullableTime(session.RevokedAt),
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (s *MySQLSessionStore) GetByTokenHash(ctx context.Context, tokenHash string) (Session, error) {
	if s == nil || s.db == nil {
		return Session{}, fmt.Errorf("mysql session store not initialized")
	}
	row := s.db.QueryRowContext(ctx, `
SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
FROM user_sessions WHERE token_hash = ?`, tokenHash)
	var (
		session Session
		revoked sql.NullTime
	)
	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.ExpiresAt,
		&session.CreatedAt,
		&revoked,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("scan session: %w", err)
	}
	if revoked.Valid {
		t := revoked.Time
		session.RevokedAt = &t
	}
	return session, nil
}

func (s *MySQLSessionStore) Revoke(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql session store not initialized")
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE user_sessions SET revoked_at = ? WHERE token_hash = ? AND revoked_at IS NULL`,
		revokedAt.UTC(), tokenHash)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return ErrSessionNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (User, error) {
	var user User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.PasswordHash,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("scan user: %w", err)
	}
	return user, nil
}

func nullableTime(t *time.Time) sql.NullTime {
	if t == nil || t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t.UTC(), Valid: true}
}

// isDuplicateKeyError best-effort detects MySQL ER_DUP_ENTRY (1062). Avoids a
// hard dependency on go-sql-driver internals so sqlmock-based tests can drive
// the same code path with a generic error.
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	type mysqlErr interface{ Number() uint16 }
	var m mysqlErr
	if errors.As(err, &m) && m.Number() == 1062 {
		return true
	}
	// Fallback for sqlmock and anything that just returns the textual error.
	msg := err.Error()
	return contains(msg, "Duplicate entry") || contains(msg, "duplicate key")
}

func contains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	if len(haystack) < len(needle) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
