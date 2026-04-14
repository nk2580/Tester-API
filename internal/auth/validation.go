package auth

import (
    "errors"
    "fmt"
    "strings"
)

const minPasswordLength = 8

func normalizeEmail(raw string) (string, error) {
    normalized := strings.ToLower(strings.TrimSpace(raw))
    if normalized == "" {
        return "", errors.New("email is required")
    }
    if !strings.Contains(normalized, "@") {
        return "", errors.New("email must be valid")
    }
    return normalized, nil
}

func validatePassword(password string) error {
    if len(password) < minPasswordLength {
        return fmt.Errorf("password must be at least %d characters", minPasswordLength)
    }
    return nil
}
