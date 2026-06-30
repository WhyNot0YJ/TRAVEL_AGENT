package auth

import "time"

// User is the authoritative user record. Password material never leaves the
// auth package — handlers must use UserDTO when serialising to clients.
type User struct {
	ID           string
	Email        string
	DisplayName  string
	PasswordHash string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

const (
	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"
)

// UserDTO is the JSON shape returned by the API. It deliberately omits hashes
// and any other server-only fields.
type UserDTO struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

func NewUserDTO(u User) UserDTO {
	return UserDTO{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Status:      u.Status,
		CreatedAt:   u.CreatedAt.UTC().Format(time.RFC3339),
	}
}
