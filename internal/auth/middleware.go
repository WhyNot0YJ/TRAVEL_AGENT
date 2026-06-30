package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// userContextKey is private to this package; handlers read the User via
// UserFromGin or UserIDFromContext.
const userContextKey = "auth.user"

// Optional resolves the session if present and injects the user into the gin
// context, but never blocks the request. Useful for endpoints that adapt
// behaviour for logged-in users without requiring auth.
func (h *Handler) Optional() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := h.resolve(c)
		if ok {
			h.attach(c, user)
		}
		c.Next()
	}
}

// RequireAuth blocks anonymous requests and clears stale cookies. Returns 401.
func (h *Handler) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := h.resolve(c)
		if !ok {
			respond(c, http.StatusUnauthorized, "unauthenticated", "sign in required")
			c.Abort()
			return
		}
		h.attach(c, user)
		c.Next()
	}
}

func (h *Handler) resolve(c *gin.Context) (User, bool) {
	token := h.tokenFromRequest(c)
	if token == "" {
		return User{}, false
	}
	user, err := h.service.Authenticate(c.Request.Context(), token)
	if err != nil {
		// Stale or expired cookie — clear it so the browser stops sending it.
		if errors.Is(err, ErrSessionExpired) || errors.Is(err, ErrSessionNotFound) {
			h.ClearSessionCookie(c)
		}
		return User{}, false
	}
	return user, true
}

func (h *Handler) attach(c *gin.Context, user User) {
	c.Set(userContextKey, user)
	ctx := WithUserID(c.Request.Context(), user.ID)
	c.Request = c.Request.WithContext(ctx)
}

// UserFromGin extracts the authenticated user from the request context. Use
// this in handlers behind RequireAuth.
func UserFromGin(c *gin.Context) (User, bool) {
	value, ok := c.Get(userContextKey)
	if !ok {
		return User{}, false
	}
	user, ok := value.(User)
	return user, ok
}
