package main

import (
    "log"
    "time"

    "github.com/nk2580/Tester-API/internal/auth"
    "github.com/nk2580/Tester-API/internal/config"
    datasqlite "github.com/nk2580/Tester-API/internal/data/sqlite"
    "github.com/nk2580/Tester-API/internal/server"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("failed to load config: %v", err)
    }

    db, err := datasqlite.Open(cfg.DBPath)
    if err != nil {
        log.Fatalf("failed to open database: %v", err)
    }

    store := datasqlite.NewStore(db)
    if err := store.AutoMigrate(); err != nil {
        log.Fatalf("failed to run database migrations: %v", err)
    }

    tokenService := auth.NewJWTService(cfg.Auth.Secret, cfg.Auth.TTL, time.Now)

    app := server.NewApp(server.Dependencies{
        UserStore:    store,
        PingStore:    store,
        TokenService: tokenService,
        TimeProvider: time.Now,
    })

    if err := app.Run(cfg.HTTPAddress); err != nil {
        log.Fatalf("server exited: %v", err)
    }
}
