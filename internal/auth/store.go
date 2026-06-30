package auth

import (
	"context"
	"sync"
	"time"
)

// UserStore is the persistence boundary for users. The same interface backs
// both the in-memory dev fallback and the MySQL implementation.
type UserStore interface {
	Create(ctx context.Context, user User) error
	GetByID(ctx context.Context, id string) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
}

// SessionStore manages session lifecycle. Storage only ever sees TokenHash —
// the plaintext token never lives at rest.
type SessionStore interface {
	Create(ctx context.Context, session Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (Session, error)
	Revoke(ctx context.Context, tokenHash string, revokedAt time.Time) error
}

// MemoryUserStore is the dev fallback when MySQL is not configured. All data
// is lost on process restart.
type MemoryUserStore struct {
	mu    sync.RWMutex
	byID  map[string]User
	email map[string]string
}

func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{byID: map[string]User{}, email: map[string]string{}}
}

func (s *MemoryUserStore) Create(ctx context.Context, user User) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.email[user.Email]; ok {
		return ErrEmailExists
	}
	s.byID[user.ID] = user
	s.email[user.Email] = user.ID
	return nil
}

func (s *MemoryUserStore) GetByID(ctx context.Context, id string) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.byID[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

func (s *MemoryUserStore) GetByEmail(ctx context.Context, email string) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.email[email]
	if !ok {
		return User{}, ErrUserNotFound
	}
	user := s.byID[id]
	return user, nil
}

type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session // keyed by token_hash
}

func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{sessions: map[string]Session{}}
}

func (s *MemorySessionStore) Create(ctx context.Context, session Session) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.TokenHash] = session
	return nil
}

func (s *MemorySessionStore) GetByTokenHash(ctx context.Context, tokenHash string) (Session, error) {
	if err := ctx.Err(); err != nil {
		return Session{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[tokenHash]
	if !ok {
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (s *MemorySessionStore) Revoke(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[tokenHash]
	if !ok {
		return ErrSessionNotFound
	}
	t := revokedAt
	session.RevokedAt = &t
	s.sessions[tokenHash] = session
	return nil
}
