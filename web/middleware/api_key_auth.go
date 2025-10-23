package middleware

import (
	"os"
	"strings"

	"github.com/mhsanaei/3x-ui/v2/web/service"
	"github.com/mhsanaei/3x-ui/v2/web/session"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// ApiKeyAuthMiddleware authorizes API requests using an API key without changing existing session-based auth.
// If XUI_API_KEY env var is set and a request to panel API includes a matching token in either
// X-API-Key header or Authorization: Bearer <token>, the middleware sets a login session for the duration
// of the request (cookie will be issued as a side effect).
func ApiKeyAuthMiddleware() gin.HandlerFunc {
	expected := strings.TrimSpace(os.Getenv("XUI_API_KEY"))
	return func(c *gin.Context) {
		// If feature is disabled, do nothing
		if expected == "" {
			c.Next()
			return
		}

		// Scope ONLY to new public API: '{basePath}api/*'
		basePathAny, _ := c.Get("base_path")
		basePath, _ := basePathAny.(string)
		path := c.Request.URL.Path
		allowed := false
		if basePath != "" {
			allowed = strings.HasPrefix(path, basePath+"api/")
		} else {
			allowed = strings.HasPrefix(path, "/api/")
		}
		if !allowed {
			c.Next()
			return
		}

		// Allow unauthenticated ping endpoint
		pingPath := "/api/ping"
		if basePath != "" {
			pingPath = basePath + "api/ping"
		}
		if path == pingPath {
			c.Next()
			return
		}

		// If already logged in via session, continue
		if session.IsLogin(c) {
			c.Next()
			return
		}

		// Extract API key from headers
		token := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if token == "" {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				token = strings.TrimSpace(auth[7:])
			}
		}

		if token == "" || token != expected {
			c.Next()
			return
		}

		// Authenticate as the first user and set session
		var userService service.UserService
		user, err := userService.GetFirstUser()
		if err == nil && user != nil {
			session.SetLoginUser(c, user)
			_ = sessions.Default(c).Save()
		}

		c.Next()
	}
}
