package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nk2580/Tester-API/internal/api"
	"github.com/nk2580/Tester-API/internal/services"
)

func Authn(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			api.Unauthorized(c)
			c.Abort()
			return
		}
		token := strings.TrimSpace(authHeader[len("Bearer "):])
		user, session, err := authService.Authenticate(token)
		if err != nil {
			api.Error(c, http.StatusUnauthorized, "unauthorized", "Invalid token")
			c.Abort()
			return
		}
		c.Set("auth_user", user)
		c.Set("auth_session_id", session.ID)
		c.Next()
	}
}
