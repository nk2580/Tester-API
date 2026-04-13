package main

import (
	"log"
)

func main() {
	authConfig, err := loadAuthConfigFromEnv()
	if err != nil {
		log.Fatalf("failed to load auth config: %v", err)
	}

	db, err := openDatabase("db/data.db")
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	app := NewApp(db, authConfig)
	r := app.Router()

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
