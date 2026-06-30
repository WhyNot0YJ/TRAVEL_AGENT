package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	svc := NewService(NewMemoryUserStore(), NewMemorySessionStore(), Config{PasswordMinLength: 8, SessionTTL: time.Hour})
	svc.now = func() time.Time { return time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC) }
	return svc
}

func TestServiceRegisterAndLogin(t *testing.T) {
	svc := newTestService(t)
	issued, err := svc.Register(context.Background(), RegisterInput{Email: "Alice@example.com", Password: "hunter2hunter", DisplayName: "Alice"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if issued.User.Email != "alice@example.com" {
		t.Fatalf("expected normalized email, got %q", issued.User.Email)
	}
	if issued.Plaintext == "" {
		t.Fatal("expected plaintext token")
	}
	if issued.User.PasswordHash == "" || issued.User.PasswordHash == "hunter2hunter" {
		t.Fatalf("password must be hashed, got %q", issued.User.PasswordHash)
	}

	got, err := svc.Login(context.Background(), LoginInput{Email: "alice@example.com", Password: "hunter2hunter"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if got.User.ID != issued.User.ID {
		t.Fatalf("expected same user id, got %s vs %s", got.User.ID, issued.User.ID)
	}
}

func TestServiceRejectsDuplicateEmail(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "longenough!", DisplayName: "A"}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, err := svc.Register(context.Background(), RegisterInput{Email: "a@b.com", Password: "longenough!", DisplayName: "Other"})
	if !errors.Is(err, ErrEmailExists) {
		t.Fatalf("expected ErrEmailExists, got %v", err)
	}
}

func TestServiceRejectsShortPassword(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Register(context.Background(), RegisterInput{Email: "short@example.com", Password: "short", DisplayName: "S"})
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestServiceRejectsInvalidEmail(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Register(context.Background(), RegisterInput{Email: "not-an-email", Password: "longenough!", DisplayName: "X"})
	if !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestServiceLoginUnknownUserReturnsStableError(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Login(context.Background(), LoginInput{Email: "ghost@example.com", Password: "whatever123"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestServiceLoginWrongPasswordReturnsStableError(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.Register(context.Background(), RegisterInput{Email: "u@e.com", Password: "longenough!", DisplayName: "U"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	_, err := svc.Login(context.Background(), LoginInput{Email: "u@e.com", Password: "wrongpassword"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestServiceAuthenticateRevokesAfterLogout(t *testing.T) {
	svc := newTestService(t)
	issued, err := svc.Register(context.Background(), RegisterInput{Email: "u@e.com", Password: "longenough!", DisplayName: "U"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	user, err := svc.Authenticate(context.Background(), issued.Plaintext)
	if err != nil || user.ID != issued.User.ID {
		t.Fatalf("expected authenticate to succeed, got user=%v err=%v", user, err)
	}
	if err := svc.Logout(context.Background(), issued.Plaintext); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := svc.Authenticate(context.Background(), issued.Plaintext); err == nil {
		t.Fatal("expected error after logout")
	}
}

func TestServiceAuthenticateRejectsExpiredSession(t *testing.T) {
	svc := NewService(NewMemoryUserStore(), NewMemorySessionStore(), Config{PasswordMinLength: 8, SessionTTL: time.Minute})
	frozen := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return frozen }
	issued, err := svc.Register(context.Background(), RegisterInput{Email: "u@e.com", Password: "longenough!", DisplayName: "U"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	svc.now = func() time.Time { return frozen.Add(2 * time.Hour) }
	_, err = svc.Authenticate(context.Background(), issued.Plaintext)
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}
}

func TestHandlerRegisterAndMe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService(t)
	handler := NewHandler(svc, CookieConfig{Name: "ta_session"})
	router := gin.New()
	router.POST("/auth/register", handler.Register)
	router.GET("/auth/me", handler.Optional(), handler.Me)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"email":"alice@example.com","password":"longenough!","display_name":"Alice"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	cookieHeader := rec.Header().Get("Set-Cookie")
	if cookieHeader == "" {
		t.Fatal("expected Set-Cookie header")
	}
	if !strings.Contains(cookieHeader, "ta_session=") || !strings.Contains(cookieHeader, "HttpOnly") {
		t.Fatalf("expected HttpOnly session cookie, got %q", cookieHeader)
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req2.Header.Set("Cookie", strings.SplitN(cookieHeader, ";", 2)[0])
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	if !strings.Contains(rec2.Body.String(), "alice@example.com") {
		t.Fatalf("expected user payload, got %s", rec2.Body.String())
	}
}

func TestHandlerLoginInvalidReturns401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService(t)
	handler := NewHandler(svc, CookieConfig{Name: "ta_session"})
	router := gin.New()
	router.POST("/auth/login", handler.Login)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"missing@example.com","password":"longenough!"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid_credentials") {
		t.Fatalf("expected stable error code, got %s", rec.Body.String())
	}
}

func TestRequireAuthBlocksAnonymous(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService(t)
	handler := NewHandler(svc, CookieConfig{Name: "ta_session"})
	router := gin.New()
	router.GET("/me/secret", handler.RequireAuth(), func(c *gin.Context) {
		user, _ := UserFromGin(c)
		c.JSON(http.StatusOK, gin.H{"id": user.ID})
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/me/secret", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuthAllowsValidSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService(t)
	handler := NewHandler(svc, CookieConfig{Name: "ta_session"})
	router := gin.New()
	router.GET("/me/secret", handler.RequireAuth(), func(c *gin.Context) {
		user, _ := UserFromGin(c)
		userID := UserIDFromContext(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"id": user.ID, "ctx": userID})
	})

	issued, err := svc.Register(context.Background(), RegisterInput{Email: "u@e.com", Password: "longenough!", DisplayName: "U"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me/secret", nil)
	req.AddCookie(&http.Cookie{Name: "ta_session", Value: issued.Plaintext})
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), issued.User.ID) {
		t.Fatalf("expected user id in body, got %s", rec.Body.String())
	}
}

func TestNewSessionTokenIsRandom(t *testing.T) {
	a, ahash, err := NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken: %v", err)
	}
	b, bhash, err := NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken: %v", err)
	}
	if a == b || ahash == bhash {
		t.Fatal("expected unique token / hash pairs")
	}
	if HashSessionToken(a) != ahash {
		t.Fatal("HashSessionToken must be deterministic")
	}
}
