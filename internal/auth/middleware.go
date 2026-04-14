package auth

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/nk2580/Tester-API/internal/httputil"
)

const contextUserIDKey = "auth_user_id"

// ContextUserIDKey exposes the key used to store the authenticated user ID.
func ContextUserIDKey() string {
    return contextUserIDKey
}

// NewMiddleware returns a gin middleware that validates Authorization headers.
func NewMiddleware(tokens TokenService) gin.HandlerFunc {
    return func(c *gin.Context) {
        header := c.GetHeader("Authorization")
        if header == "" {
            httputil.WriteError(c, http.StatusUnauthorized, "missing_token", "authorization token is required")
            c.Abort()
            return
        }

        parts := strings.SplitN(header, " ", 2)
        if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
            httputil.WriteError(c, http.StatusUnauthorized, "invalid_token", "invalid authorization header")
            c.Abort()
            return
        }

        userID, err := tokens.Parse(strings.TrimSpace(parts[1]))
        if err != nil {
            httputil.WriteError(c, http.StatusUnauthorized, "invalid_token", "invalid or expired token")
            c.Abort()
            return
        }

        c.Set(contextUserIDKey, userID)
        c.Next()
    }
}
