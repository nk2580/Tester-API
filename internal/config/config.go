package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabasePath               string
	ServerAddr                 string
	SessionCookieName          string
	SessionTTL                 time.Duration
	VerificationTokenTTL       time.Duration
	PasswordResetTokenTTL      time.Duration
	ResendVerificationCooldown time.Duration
	LockoutDuration            time.Duration
	MaxFailedLoginAttempts     int
	CookieSecure               bool
	CookieDomain               string
	PasswordMinLength          int
}

func Load() Config {
	return Config{
		DatabasePath:               getEnv("DB_PATH", "db/data.db"),
		ServerAddr:                 getEnv("SERVER_ADDR", ":8080"),
		SessionCookieName:          getEnv("SESSION_COOKIE_NAME", "session_token"),
		SessionTTL:                 getDuration("SESSION_TTL", 24*time.Hour),
		VerificationTokenTTL:       getDuration("VERIFICATION_TOKEN_TTL", 24*time.Hour),
		PasswordResetTokenTTL:      getDuration("PASSWORD_RESET_TOKEN_TTL", 30*time.Minute),
		ResendVerificationCooldown: getDuration("RESEND_VERIFICATION_COOLDOWN", 60*time.Second),
		LockoutDuration:            getDuration("LOCKOUT_DURATION", 15*time.Minute),
		MaxFailedLoginAttempts:     getInt("MAX_FAILED_LOGIN_ATTEMPTS", 5),
		CookieSecure:               getBool("COOKIE_SECURE", false),
		CookieDomain:               strings.TrimSpace(os.Getenv("COOKIE_DOMAIN")),
		PasswordMinLength:          getInt("PASSWORD_MIN_LENGTH", 12),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
