package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword produces a bcrypt hash with the default cost. Bcrypt rejects
// passwords longer than 72 bytes, which is fine for our minimum-length policy.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password is empty")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword returns nil only when the plaintext matches the stored hash.
// Callers must NEVER differentiate "wrong password" from "user not found" in
// API responses.
func VerifyPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// NewSessionToken returns (plaintextToken, tokenHash). Plaintext is shown to
// the user once via cookie; the hash is what we persist.
func NewSessionToken() (string, string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", "", err
	}
	plaintext := hex.EncodeToString(raw[:])
	return plaintext, HashSessionToken(plaintext), nil
}

// HashSessionToken is deterministic SHA-256. Bcrypt is unnecessary because the
// plaintext is already a 256-bit cryptographically random value, and we need
// constant-time exact lookup by hash.
func HashSessionToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}
