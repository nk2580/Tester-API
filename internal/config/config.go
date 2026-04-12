package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env                         string
	ListenAddr                  string
	DBPath                      string
	BaseURL                     string
	AppSecret                   string
	AccessTokenTTL              time.Duration
	PasswordResetTokenTTL       time.Duration
	BcryptCost                  int
	AllowedOrigins              []string
	EmailFrom                   string
	EmailProvider               string
	SMTPHost                    string
	SMTPPort                    int
	SMTPUser                    string
	SMTPPassword                string
	EnableDetailedAuthResponses bool
	LogLevel                    slog.Level
}

func Load() (Config, error) {
	cfg := Config{
		Env:                   getEnv("APP_ENV", "dev"),
		ListenAddr:            getEnv("LISTEN_ADDR", ":8080"),
		DBPath:                getEnv("DB_PATH", "db/data.db"),
		BaseURL:               getEnv("BASE_URL", "http://localhost:8080"),
		AppSecret:             getEnv("APP_SECRET", "dev-secret-change-me"),
		AccessTokenTTL:        mustParseDuration("ACCESS_TOKEN_TTL", "15m"),
		PasswordResetTokenTTL: mustParseDuration("PASSWORD_RESET_TOKEN_TTL", "30m"),
		BcryptCost:            mustParseInt("BCRYPT_COST", 12),
		AllowedOrigins:        parseCSV(getEnv("ALLOWED_ORIGINS", "http://localhost:3000")),
		EmailFrom:             getEnv("EMAIL_FROM", "noreply@testerapi.local"),
		EmailProvider:         getEnv("EMAIL_PROVIDER", "log"),
		SMTPHost:              getEnv("SMTP_HOST", ""),
		SMTPPort:              mustParseInt("SMTP_PORT", 587),
		SMTPUser:              getEnv("SMTP_USER", ""),
		SMTPPassword:          getEnv("SMTP_PASSWORD", ""),
		LogLevel:              parseLogLevel(getEnv("LOG_LEVEL", "INFO")),
	}

	if cfg.BcryptCost < 10 || cfg.BcryptCost > 14 {
		return Config{}, fmt.Errorf("BCRYPT_COST must be between 10 and 14")
	}

	if cfg.Env != "dev" && cfg.Env != "test" {
		if len(cfg.AppSecret) < 32 {
			return Config{}, errors.New("APP_SECRET must be set to at least 32 characters in non-dev/test environments")
		}
		if strings.EqualFold(cfg.EmailProvider, "smtp") {
			if cfg.SMTPHost == "" || cfg.SMTPUser == "" || cfg.SMTPPassword == "" {
				return Config{}, errors.New("SMTP_HOST/SMTP_USER/SMTP_PASSWORD are required when EMAIL_PROVIDER=smtp")
			}
		}
	}

	return cfg, nil
}

func getEnv(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}

func mustParseDuration(key, def string) time.Duration {
	val := getEnv(key, def)
	d, err := time.ParseDuration(val)
	if err != nil {
		panic(fmt.Sprintf("invalid duration for %s: %v", key, err))
	}
	return d
}

func mustParseInt(key string, def int) int {
	val := getEnv(key, "")
	if val == "" {
		return def
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		panic(fmt.Sprintf("invalid int for %s: %v", key, err))
	}
	return i
}

func parseCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
