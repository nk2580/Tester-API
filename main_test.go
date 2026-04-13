package main

import (
	"errors"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRunErrorPaths(t *testing.T) {
	origLoad := loadAuthConfig
	origOpen := openDB
	origNew := newApp
	origRunServer := runServer
	t.Cleanup(func() {
		loadAuthConfig = origLoad
		openDB = origOpen
		newApp = origNew
		runServer = origRunServer
	})

	loadAuthConfig = func() (AuthConfig, error) {
		return AuthConfig{}, errors.New("bad auth")
	}
	if err := run(); err == nil {
		t.Fatalf("expected auth-config error")
	}

	loadAuthConfig = func() (AuthConfig, error) {
		return AuthConfig{JWTSecret: []byte("x"), JWTTTL: time.Second}, nil
	}
	openDB = func(string) (*gorm.DB, error) {
		return nil, errors.New("bad db")
	}
	if err := run(); err == nil {
		t.Fatalf("expected db error")
	}
}

func TestRunSuccessAndServerError(t *testing.T) {
	origLoad := loadAuthConfig
	origOpen := openDB
	origNew := newApp
	origRunServer := runServer
	t.Cleanup(func() {
		loadAuthConfig = origLoad
		openDB = origOpen
		newApp = origNew
		runServer = origRunServer
	})

	db, err := gorm.Open(sqlite.Open("file:main_test?mode=memory&cache=private"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&Ping{}, &User{}); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	loadAuthConfig = func() (AuthConfig, error) {
		return AuthConfig{JWTSecret: []byte("x"), JWTTTL: time.Second}, nil
	}
	openDB = func(string) (*gorm.DB, error) {
		return db, nil
	}
	newApp = func(db *gorm.DB, authConfig AuthConfig) *App {
		app := NewApp(db, authConfig)
		app.now = func() time.Time { return time.Unix(0, 0).UTC() }
		return app
	}

	runServer = func(runner interface{ Run(addr ...string) error }) error {
		return nil
	}
	if err := run(); err != nil {
		t.Fatalf("expected run to succeed, got %v", err)
	}

	runServer = func(runner interface{ Run(addr ...string) error }) error {
		return errors.New("listen failed")
	}
	if err := run(); err == nil {
		t.Fatalf("expected server error")
	}
}

func TestMainNoFatalOnSuccessfulRun(t *testing.T) {
	origLoad := loadAuthConfig
	origOpen := openDB
	origNew := newApp
	origRunServer := runServer
	t.Cleanup(func() {
		loadAuthConfig = origLoad
		openDB = origOpen
		newApp = origNew
		runServer = origRunServer
	})

	db, err := gorm.Open(sqlite.Open("file:main_success_test?mode=memory&cache=private"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&Ping{}, &User{}); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	loadAuthConfig = func() (AuthConfig, error) {
		return AuthConfig{JWTSecret: []byte("x"), JWTTTL: time.Second}, nil
	}
	openDB = func(string) (*gorm.DB, error) {
		return db, nil
	}
	newApp = func(db *gorm.DB, authConfig AuthConfig) *App {
		return NewApp(db, authConfig)
	}
	runServer = func(runner interface{ Run(addr ...string) error }) error {
		return nil
	}

	main()
}
