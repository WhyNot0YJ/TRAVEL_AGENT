package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CookieConfig captures everything we need to issue and clear the session
// cookie consistently. Domain/Secure come from environment so production can
// flip Secure=true behind HTTPS without code changes.
type CookieConfig struct {
	Name     string
	Domain   string
	Secure   bool
	Path     string
	SameSite http.SameSite
}

func (c CookieConfig) WithDefaults() CookieConfig {
	out := c
	if out.Name == "" {
		out.Name = "travel_agent_session"
	}
	if out.Path == "" {
		out.Path = "/"
	}
	if out.SameSite == 0 {
		out.SameSite = http.SameSiteLaxMode
	}
	return out
}

type Handler struct {
	service *Service
	cookie  CookieConfig
}

func NewHandler(service *Service, cookie CookieConfig) *Handler {
	return &Handler{service: service, cookie: cookie.WithDefaults()}
}

type registerBody struct {
	Email       string `json:"email" binding:"required"`
	Password    string `json:"password" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
}

type loginBody struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Register(c *gin.Context) {
	var body registerBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respond(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	issued, err := h.service.Register(c.Request.Context(), RegisterInput{
		Email:       body.Email,
		Password:    body.Password,
		DisplayName: body.DisplayName,
	})
	if errors.Is(err, ErrEmailExists) {
		respond(c, http.StatusConflict, "email_exists", "email already registered")
		return
	}
	if errors.Is(err, ErrInvalidEmail) {
		respond(c, http.StatusBadRequest, "invalid_email", "email is invalid")
		return
	}
	if errors.Is(err, ErrPasswordTooShort) {
		respond(c, http.StatusBadRequest, "password_too_short", "password is too short")
		return
	}
	if errors.Is(err, ErrDisplayNameMissing) {
		respond(c, http.StatusBadRequest, "display_name_required", "display_name is required")
		return
	}
	if err != nil {
		respond(c, http.StatusInternalServerError, "register_failed", err.Error())
		return
	}
	h.SetSessionCookie(c, issued)
	c.JSON(http.StatusCreated, gin.H{"user": NewUserDTO(issued.User)})
}

func (h *Handler) Login(c *gin.Context) {
	var body loginBody
	if err := c.ShouldBindJSON(&body); err != nil {
		respond(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	issued, err := h.service.Login(c.Request.Context(), LoginInput{Email: body.Email, Password: body.Password})
	if errors.Is(err, ErrInvalidCredentials) {
		respond(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	if err != nil {
		respond(c, http.StatusInternalServerError, "login_failed", err.Error())
		return
	}
	h.SetSessionCookie(c, issued)
	c.JSON(http.StatusOK, gin.H{"user": NewUserDTO(issued.User)})
}

func (h *Handler) Logout(c *gin.Context) {
	token := h.tokenFromRequest(c)
	if token != "" {
		_ = h.service.Logout(c.Request.Context(), token)
	}
	h.ClearSessionCookie(c)
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, ok := UserFromGin(c)
	if !ok {
		respond(c, http.StatusUnauthorized, "unauthenticated", "not signed in")
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": NewUserDTO(user)})
}

func (h *Handler) SetSessionCookie(c *gin.Context, issued Issued) {
	maxAge := int(time.Until(issued.Session.ExpiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	cookie := &http.Cookie{
		Name:     h.cookie.Name,
		Value:    issued.Plaintext,
		Path:     h.cookie.Path,
		Domain:   h.cookie.Domain,
		Expires:  issued.Session.ExpiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: h.cookie.SameSite,
	}
	http.SetCookie(c.Writer, cookie)
}

func (h *Handler) ClearSessionCookie(c *gin.Context) {
	cookie := &http.Cookie{
		Name:     h.cookie.Name,
		Value:    "",
		Path:     h.cookie.Path,
		Domain:   h.cookie.Domain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: h.cookie.SameSite,
	}
	http.SetCookie(c.Writer, cookie)
}

func (h *Handler) tokenFromRequest(c *gin.Context) string {
	if cookie, err := c.Cookie(h.cookie.Name); err == nil && cookie != "" {
		return cookie
	}
	if header := c.GetHeader("Authorization"); header != "" {
		const prefix = "Bearer "
		if strings.HasPrefix(header, prefix) {
			return strings.TrimSpace(header[len(prefix):])
		}
	}
	return ""
}

func respond(c *gin.Context, status int, code, message string) {
	requestID := c.Writer.Header().Get("X-Request-ID")
	c.JSON(status, gin.H{
		"request_id": requestID,
		"code":       code,
		"message":    message,
	})
}
