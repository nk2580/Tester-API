package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nk2580/Tester-API/internal/api"
	"github.com/nk2580/Tester-API/internal/services"
)

func RateLimit(store *services.LimiterStore, keyPrefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !store.Allow(keyPrefix + ":" + c.ClientIP()) {
			api.Error(c, http.StatusTooManyRequests, "rate_limited", "Too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}
