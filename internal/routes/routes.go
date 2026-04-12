package routes

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/nk2580/Tester-API/internal/api"
	"github.com/nk2580/Tester-API/internal/config"
	"github.com/nk2580/Tester-API/internal/handlers"
	"github.com/nk2580/Tester-API/internal/middleware"
	"github.com/nk2580/Tester-API/internal/services"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

type Dependencies struct {
	EmailSender services.EmailSender
}

func NewRouter(cfg config.Config, db *gorm.DB, logger *slog.Logger) (*gin.Engine, error) {
	return NewRouterWithDependencies(cfg, db, logger, Dependencies{})
}

func NewRouterWithDependencies(cfg config.Config, db *gorm.DB, logger *slog.Logger, deps Dependencies) (*gin.Engine, error) {
	if cfg.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logging(logger))
	r.Use(middleware.SecurityHeaders(cfg.AllowedOrigins))

	auditor := services.NewAuditService(db)
	emailSender := deps.EmailSender
	if emailSender == nil {
		emailSender = services.NewEmailSender(cfg, logger)
	}
	authService := services.NewAuthService(db, cfg, emailSender, auditor)
	authHandler := handlers.NewAuthHandler(authService)
	pingHandler := handlers.NewPingHandler(db)

	authLimiter := services.NewLimiterStore(rate.Limit(0.3), 5)
	registerLimiter := services.NewLimiterStore(rate.Limit(0.2), 3)
	resetLimiter := services.NewLimiterStore(rate.Limit(0.2), 3)

	r.GET("/health", handlers.Health)

	r.POST("/ping", pingHandler.Create)
	r.GET("/pings", pingHandler.List)
	r.GET("/hello", pingHandler.Hello)
	r.GET("/time", pingHandler.Time)

	auth := r.Group("/auth")
	auth.POST("/register", middleware.RateLimit(registerLimiter, "register"), authHandler.Register)
	auth.POST("/login", middleware.RateLimit(authLimiter, "login"), authHandler.Login)
	auth.POST("/password-reset/request", middleware.RateLimit(resetLimiter, "reset_request"), authHandler.PasswordResetRequest)
	auth.POST("/password-reset/confirm", middleware.RateLimit(resetLimiter, "reset_confirm"), authHandler.PasswordResetConfirm)
	auth.POST("/logout", middleware.Authn(authService), authHandler.Logout)
	auth.GET("/me", middleware.Authn(authService), authHandler.Me)

	r.NoRoute(func(c *gin.Context) {
		api.Error(c, 404, "not_found", "Route not found")
	})

	return r, nil
}
