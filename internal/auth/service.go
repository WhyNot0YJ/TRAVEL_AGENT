package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

const (
	// DefaultPasswordMinLength is conservative; ops can override with config.
	DefaultPasswordMinLength = 8
	// DefaultSessionTTL keeps users signed in for 7 days unless they log out.
	DefaultSessionTTL = 7 * 24 * time.Hour
)

type Config struct {
	PasswordMinLength int
	SessionTTL        time.Duration
}

func (c Config) WithDefaults() Config {
	out := c
	if out.PasswordMinLength <= 0 {
		out.PasswordMinLength = DefaultPasswordMinLength
	}
	if out.SessionTTL <= 0 {
		out.SessionTTL = DefaultSessionTTL
	}
	return out
}

// Service is the auth use-case layer. Handler / middleware depend on this
// surface so future changes (e.g. OAuth) can extend without touching the wire.
type Service struct {
	users    UserStore
	sessions SessionStore
	cfg      Config
	now      func() time.Time
}

func NewService(users UserStore, sessions SessionStore, cfg Config) *Service {
	return &Service{users: users, sessions: sessions, cfg: cfg.WithDefaults(), now: time.Now}
}

type RegisterInput struct {
	Email       string
	Password    string
	DisplayName string
}

type LoginInput struct {
	Email    string
	Password string
}

// Issued is what the handler turns into a Set-Cookie header. Plaintext token
// must NEVER be persisted; only TokenHash lives in the SessionStore.
type Issued struct {
	User      User
	Session   Session
	Plaintext string
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (Issued, error) {
	if s == nil || s.users == nil || s.sessions == nil {
		return Issued{}, fmt.Errorf("auth service is not initialized")
	}
	email := normalizeEmail(in.Email)
	if email == "" {
		return Issued{}, ErrInvalidEmail
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return Issued{}, ErrInvalidEmail
	}
	display := strings.TrimSpace(in.DisplayName)
	if display == "" {
		return Issued{}, ErrDisplayNameMissing
	}
	if len(display) > 80 {
		display = display[:80]
	}
	if len([]byte(in.Password)) < s.cfg.PasswordMinLength {
		return Issued{}, ErrPasswordTooShort
	}

	hash, err := HashPassword(in.Password)
	if err != nil {
		return Issued{}, err
	}
	now := s.now().UTC()
	user := User{
		ID:           NewUserID(),
		Email:        email,
		DisplayName:  display,
		PasswordHash: hash,
		Status:       UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return Issued{}, err
	}
	return s.issueSession(ctx, user)
}

func (s *Service) Login(ctx context.Context, in LoginInput) (Issued, error) {
	if s == nil || s.users == nil || s.sessions == nil {
		return Issued{}, fmt.Errorf("auth service is not initialized")
	}
	email := normalizeEmail(in.Email)
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// Stable error: do not leak whether email exists.
		return Issued{}, ErrInvalidCredentials
	}
	if user.Status != UserStatusActive {
		return Issued{}, ErrInvalidCredentials
	}
	if err := VerifyPassword(user.PasswordHash, in.Password); err != nil {
		return Issued{}, ErrInvalidCredentials
	}
	return s.issueSession(ctx, user)
}

func (s *Service) Logout(ctx context.Context, plaintextToken string) error {
	if s == nil || s.sessions == nil {
		return fmt.Errorf("auth service is not initialized")
	}
	if plaintextToken == "" {
		return nil
	}
	hash := HashSessionToken(plaintextToken)
	err := s.sessions.Revoke(ctx, hash, s.now().UTC())
	if err == nil || err == ErrSessionNotFound {
		return nil
	}
	return err
}

// Authenticate validates a plaintext session token. Returns the User if the
// session exists and is currently active. ErrSessionNotFound and
// ErrSessionExpired are returned distinctly so middleware can clear the cookie.
func (s *Service) Authenticate(ctx context.Context, plaintextToken string) (User, error) {
	if s == nil || s.users == nil || s.sessions == nil {
		return User{}, fmt.Errorf("auth service is not initialized")
	}
	if plaintextToken == "" {
		return User{}, ErrUnauthenticated
	}
	hash := HashSessionToken(plaintextToken)
	session, err := s.sessions.GetByTokenHash(ctx, hash)
	if err != nil {
		return User{}, err
	}
	if !session.Active(s.now().UTC()) {
		return User{}, ErrSessionExpired
	}
	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil {
		return User{}, err
	}
	if user.Status != UserStatusActive {
		return User{}, ErrUnauthenticated
	}
	return user, nil
}

func (s *Service) issueSession(ctx context.Context, user User) (Issued, error) {
	plaintext, hash, err := NewSessionToken()
	if err != nil {
		return Issued{}, err
	}
	now := s.now().UTC()
	session := Session{
		ID:        NewSessionID(),
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: now.Add(s.cfg.SessionTTL),
		CreatedAt: now,
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return Issued{}, err
	}
	return Issued{User: user, Session: session, Plaintext: plaintext}, nil
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func NewUserID() string {
	return prefixedRandomID("user_")
}

func NewSessionID() string {
	return prefixedRandomID("sess_")
}

func prefixedRandomID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		return prefix + hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano())
}
