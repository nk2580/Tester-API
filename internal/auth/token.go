package auth

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Token represents an issued access token response payload.
type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// TokenService describes functionality required to issue and validate tokens.
type TokenService interface {
	Generate(userID uint) (Token, error)
	Parse(tokenString string) (uint, error)
}

// JWTService implements TokenService backed by HMAC JWTs.
type JWTService struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewJWTService constructs a JWTService with the provided secret, TTL, and clock.
func NewJWTService(secret []byte, ttl time.Duration, now func() time.Time) *JWTService {
	if now == nil {
		now = time.Now
	}
	return &JWTService{secret: secret, ttl: ttl, now: now}
}

// ErrInvalidToken is returned when a token cannot be parsed or validated.
var ErrInvalidToken = errors.New("invalid token")

// Generate creates a signed JWT for the provided user ID.
func (s *JWTService) Generate(userID uint) (Token, error) {
	issuedAt := s.now().UTC()
	expiresAt := issuedAt.Add(s.ttl)

	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatUint(uint64(userID), 10),
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return Token{}, err
	}

	return Token{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.ttl / time.Second),
	}, nil
}

// Parse validates the token string and returns the embedded user ID.
func (s *JWTService) Parse(tokenString string) (uint, error) {
	var claims jwt.RegisteredClaims
	parsed, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	}, jwt.WithTimeFunc(s.now))
	if err != nil || !parsed.Valid {
		return 0, ErrInvalidToken
	}

	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(s.now()) {
		return 0, ErrInvalidToken
	}

	sub := claims.Subject
	if strings.TrimSpace(sub) == "" {
		return 0, ErrInvalidToken
	}

	id, err := strconv.ParseUint(sub, 10, 64)
	if err != nil {
		return 0, ErrInvalidToken
	}

	return uint(id), nil
}
