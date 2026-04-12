package services

import (
	"errors"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

var (
	errPasswordTooShort = errors.New("password must be at least 12 characters")
	errPasswordMissing  = errors.New("password must include upper, lower, digit, and special character")
	emailRegex          = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
)

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ValidateEmail(email string) error {
	email = NormalizeEmail(email)
	if len(email) < 3 || len(email) > 320 || !emailRegex.MatchString(email) {
		return errors.New("invalid email")
	}
	return nil
}

func ValidatePasswordPolicy(password string) error {
	if len(password) < 12 {
		return errPasswordTooShort
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return errPasswordMissing
	}
	return nil
}

func HashPassword(password string, cost int) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
