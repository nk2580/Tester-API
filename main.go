package main

import (
	"log"
)

var (
	loadAuthConfig = loadAuthConfigFromEnv
	openDB         = openDatabase
	newApp         = NewApp
	runServer      = func(runner interface{ Run(addr ...string) error }) error {
		return runner.Run(serverAddress)
	}
)

const serverAddress = ":8080"

func run() error {
	authConfig, err := loadAuthConfig()
	if err != nil {
		return err
	}

	db, err := openDB("db/data.db")
	if err != nil {
		return err
	}

	app := newApp(db, authConfig)
	return runServer(app.Router())
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
