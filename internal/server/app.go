package server

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nk2580/Tester-API/internal/auth"
	"github.com/nk2580/Tester-API/internal/data"
	"github.com/nk2580/Tester-API/internal/ping"
	timehandler "github.com/nk2580/Tester-API/internal/time"
)

// Dependencies enumerates the services required to build the HTTP server.
type Dependencies struct {
    UserStore    data.UserStore
    PingStore    data.PingStore
    TokenService auth.TokenService
    TimeProvider func() time.Time
}

// App composes the HTTP router and exposes helpers for running it.
type App struct {
    engine *gin.Engine
}

// NewApp wires the router using the provided dependencies.
func NewApp(deps Dependencies) *App {
    if deps.TimeProvider == nil {
        deps.TimeProvider = time.Now
    }

	if deps.UserStore == nil || deps.PingStore == nil || deps.TokenService == nil {
		panic(fmt.Sprintf(\"server dependencies must not be nil: %+v\", deps))
	}

    authHandler := auth.NewHandler(deps.UserStore, deps.TokenService)
    pingHandler := ping.NewHandler(deps.PingStore)
    timeHandler := timehandler.NewHandler(deps.TimeProvider)

    engine := newRouter(RouterConfig{
        AuthHandler:   authHandler,
        PingHandler:   pingHandler,
        TimeHandler:   timeHandler,
        AuthMiddleware: auth.NewMiddleware(deps.TokenService),
    })

    return &App{engine: engine}
}

// Router exposes the configured gin engine for tests.
func (a *App) Router() *gin.Engine {
    return a.engine
}

// Run starts the HTTP server on the provided address.
func (a *App) Run(addr string) error {
    return a.engine.Run(addr)
}
