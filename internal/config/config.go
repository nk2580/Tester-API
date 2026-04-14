package config

import (
    "errors"
    "fmt"
    "os"
    "strconv"
    "strings"
    "time"
)

const (
    defaultJWTTTLSeconds = 3600
    defaultDBPath        = "db/data.db"
    defaultHTTPAddress   = ":8080"
)

// Auth holds JWT configuration for the API.
type Auth struct {
    Secret []byte
    TTL    time.Duration
}

// Config aggregates the runtime configuration values needed by the server.
type Config struct {
    DBPath      string
    HTTPAddress string
    Auth        Auth
}

// Load reads configuration from environment variables and applies defaults.
func Load() (Config, error) {
    secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
    if secret == "" {
        return Config{}, errors.New("JWT_SECRET is required")
    }

    ttlSeconds := defaultJWTTTLSeconds
    if raw := strings.TrimSpace(os.Getenv("JWT_TTL")); raw != "" {
        parsed, err := strconv.Atoi(raw)
        if err != nil || parsed <= 0 {
            return Config{}, fmt.Errorf("JWT_TTL must be a positive integer number of seconds")
        }
        ttlSeconds = parsed
    }

    dbPath := strings.TrimSpace(os.Getenv("DB_PATH"))
    if dbPath == "" {
        dbPath = defaultDBPath
    }

    addr := strings.TrimSpace(os.Getenv("HTTP_ADDRESS"))
    if addr == "" {
        addr = defaultHTTPAddress
    }

    return Config{
        DBPath:      dbPath,
        HTTPAddress: addr,
        Auth: Auth{
            Secret: []byte(secret),
            TTL:    time.Duration(ttlSeconds) * time.Second,
        },
    }, nil
}
