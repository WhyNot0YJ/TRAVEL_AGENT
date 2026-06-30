package auth

import "errors"

var (
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
	ErrPasswordTooShort   = errors.New("password too short")
	ErrInvalidEmail       = errors.New("email is invalid")
	ErrDisplayNameMissing = errors.New("display name is required")
	ErrUnauthenticated    = errors.New("unauthenticated")
)
