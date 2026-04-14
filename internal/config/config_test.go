package config_test

import (
    "testing"
    "time"

    "github.com/nk2580/Tester-API/internal/config"
)

func TestLoadConfigValidationAndDefaults(t *testing.T) {
    t.Run("missing secret", func(t *testing.T) {
        t.Setenv("JWT_SECRET", "")
        t.Setenv("JWT_TTL", "")
        t.Setenv("DB_PATH", "")
        t.Setenv("HTTP_ADDRESS", "")
        if _, err := config.Load(); err == nil {
            t.Fatalf("expected error")
        }
    })

    t.Run("default values", func(t *testing.T) {
        t.Setenv("JWT_SECRET", "secret")
        t.Setenv("JWT_TTL", "")
        t.Setenv("DB_PATH", "")
        t.Setenv("HTTP_ADDRESS", "")

        cfg, err := config.Load()
        if err != nil {
            t.Fatalf("unexpected err: %v", err)
        }
        if cfg.Auth.TTL == 0 {
            t.Fatalf("expected ttl")
        }
        if cfg.DBPath == "" || cfg.HTTPAddress == "" {
            t.Fatalf("expected defaults")
        }
    })

    t.Run("custom values", func(t *testing.T) {
        t.Setenv("JWT_SECRET", "secret")
        t.Setenv("JWT_TTL", "7200")
        t.Setenv("DB_PATH", "custom.db")
        t.Setenv("HTTP_ADDRESS", ":9000")

        cfg, err := config.Load()
        if err != nil {
            t.Fatalf("unexpected err: %v", err)
        }
        if cfg.Auth.TTL != 2*time.Hour {
            t.Fatalf("expected ttl 2h, got %v", cfg.Auth.TTL)
        }
        if cfg.DBPath != "custom.db" || cfg.HTTPAddress != ":9000" {
            t.Fatalf("expected provided overrides")
        }
    })
}
