package server

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = newRequestID()
		}
		c.Set(requestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic request_id=%s error=%v stack=%s", requestID(c), recovered, string(debug.Stack()))
				c.AbortWithStatusJSON(http.StatusInternalServerError, errorBody(c, "internal_error", "internal server error"))
			}
		}()
		c.Next()
	}
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		log.Printf("request_id=%s method=%s path=%s status=%d duration=%s", requestID(c), c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(started))
	}
}

// CORS now requires an explicit allowlist when credentials must flow. An
// empty allowedOrigins falls back to the open dev policy, but production
// deployments should always pass real origins (the cookie auth needs it).
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowSet := map[string]struct{}{}
	for _, origin := range allowedOrigins {
		allowSet[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if len(allowSet) == 0 {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if _, ok := allowSet[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Access-Control-Allow-Headers", "Content-Type, X-Request-ID, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func requestID(c *gin.Context) string {
	if value, ok := c.Get(requestIDKey); ok {
		if text, ok := value.(string); ok {
			return text
		}
	}
	return ""
}

func errorBody(c *gin.Context, code, message string) gin.H {
	return gin.H{
		"request_id": requestID(c),
		"code":       code,
		"message":    message,
	}
}

func newRequestID() string {
	return time.Now().UTC().Format("20060102150405.000000000")
}
