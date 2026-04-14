package server

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/nk2580/Tester-API/internal/auth"
    "github.com/nk2580/Tester-API/internal/ping"
    timehandler "github.com/nk2580/Tester-API/internal/time"
)

// RouterConfig enumerates the handlers and middleware injected into the router.
type RouterConfig struct {
    AuthHandler   *auth.Handler
    PingHandler   *ping.Handler
    TimeHandler   *timehandler.Handler
    AuthMiddleware gin.HandlerFunc
}

func newRouter(cfg RouterConfig) *gin.Engine {
    r := gin.Default()

    r.POST("/ping", cfg.PingHandler.Create)
    r.GET("/pings", cfg.PingHandler.List)
    r.GET("/hello", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"message": "Hello World"})
    })
    r.GET("/time", cfg.TimeHandler.Get)

    authGroup := r.Group("/auth")
    authGroup.POST("/signup", cfg.AuthHandler.Signup)
    authGroup.POST("/login", cfg.AuthHandler.Login)

    protected := authGroup.Group("")
    protected.Use(cfg.AuthMiddleware)
    protected.GET("/me", cfg.AuthHandler.Me)

    return r
}
