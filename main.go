package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/nk2580/Tester-API/internal/app"
	"github.com/nk2580/Tester-API/internal/auth"
	"github.com/nk2580/Tester-API/internal/config"
	"github.com/nk2580/Tester-API/internal/store"
)

func main() {
	cfg := config.Load()
	db, err := store.OpenSQLite(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("database initialization failed: %v", err)
	}

	r := gin.Default()
	application := app.New(db, cfg, auth.LogMailer{}, nil)
	application.RegisterRoutes(r)

	if err := r.Run(cfg.ServerAddr); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
