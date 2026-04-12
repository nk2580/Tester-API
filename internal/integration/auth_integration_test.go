package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nk2580/Tester-API/internal/config"
	"github.com/nk2580/Tester-API/internal/db"
	"github.com/nk2580/Tester-API/internal/models"
	"github.com/nk2580/Tester-API/internal/routes"
	"github.com/nk2580/Tester-API/internal/services"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type capturingEmailSender struct {
	mu   sync.Mutex
	link string
}

func (s *capturingEmailSender) SendPasswordResetEmail(_ string, resetLink string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.link = resetLink
	return nil
}

func (s *capturingEmailSender) LastToken(t *testing.T) string {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.link == "" {
		t.Fatalf("no reset link captured")
	}
	u, err := url.Parse(s.link)
	if err != nil {
		t.Fatalf("parse reset link: %v", err)
	}
	return u.Query().Get("token")
}

func setupTestServer(t *testing.T) (*httptest.Server, *gorm.DB, *capturingEmailSender) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := config.Config{
		Env:                   "test",
		ListenAddr:            ":0",
		DBPath:                dbPath,
		BaseURL:               "http://example.test",
		AppSecret:             strings.Repeat("a", 40),
		AccessTokenTTL:        15 * time.Minute,
		PasswordResetTokenTTL: 30 * time.Minute,
		BcryptCost:            10,
		AllowedOrigins:        []string{"http://localhost:3000"},
		EmailFrom:             "noreply@example.test",
		EmailProvider:         "log",
		SMTPPort:              587,
		LogLevel:              slog.LevelError,
	}

	gormDB, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.RunMigrations(gormDB); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	sender := &capturingEmailSender{}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	r, err := routes.NewRouterWithDependencies(cfg, gormDB, logger, routes.Dependencies{EmailSender: sender})
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	return httptest.NewServer(r), gormDB, sender
}

func doJSON(t *testing.T, method, url string, body any, headers map[string]string) (*http.Response, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	var payload map[string]any
	if len(data) > 0 {
		_ = json.Unmarshal(data, &payload)
	}
	return res, payload
}

func TestAuthFlowRegisterLoginLogoutProtected(t *testing.T) {
	srv, _, _ := setupTestServer(t)
	defer srv.Close()

	res, _ := doJSON(t, http.MethodPost, srv.URL+"/auth/register", map[string]string{
		"email":    "USER@example.com",
		"password": "StrongPassw0rd!",
	}, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 register, got %d", res.StatusCode)
	}

	res, payload := doJSON(t, http.MethodPost, srv.URL+"/auth/login", map[string]string{
		"email":    "user@example.com",
		"password": "StrongPassw0rd!",
	}, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 login, got %d", res.StatusCode)
	}
	token, _ := payload["access_token"].(string)
	if token == "" {
		t.Fatalf("expected access token")
	}

	res, me := doJSON(t, http.MethodGet, srv.URL+"/auth/me", nil, map[string]string{"Authorization": "Bearer " + token})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 /auth/me, got %d", res.StatusCode)
	}
	if me["email"] != "user@example.com" {
		t.Fatalf("expected normalized email, got %v", me["email"])
	}

	res, _ = doJSON(t, http.MethodPost, srv.URL+"/auth/logout", nil, map[string]string{"Authorization": "Bearer " + token})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 logout, got %d", res.StatusCode)
	}

	res, _ = doJSON(t, http.MethodGet, srv.URL+"/auth/me", nil, map[string]string{"Authorization": "Bearer " + token})
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d", res.StatusCode)
	}
}

func TestPasswordResetUnknownEmailReturnsGenericSuccess(t *testing.T) {
	srv, _, _ := setupTestServer(t)
	defer srv.Close()

	res, _ := doJSON(t, http.MethodPost, srv.URL+"/auth/password-reset/request", map[string]string{"email": "unknown@example.com"}, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown reset request, got %d", res.StatusCode)
	}
}

func TestPasswordResetHappyPathReplayAndSessionInvalidation(t *testing.T) {
	srv, gormDB, sender := setupTestServer(t)
	defer srv.Close()

	_, _ = doJSON(t, http.MethodPost, srv.URL+"/auth/register", map[string]string{
		"email":    "reset-user@example.com",
		"password": "StrongPassw0rd!",
	}, nil)
	_, loginPayload := doJSON(t, http.MethodPost, srv.URL+"/auth/login", map[string]string{
		"email":    "reset-user@example.com",
		"password": "StrongPassw0rd!",
	}, nil)
	oldToken, _ := loginPayload["access_token"].(string)

	res, _ := doJSON(t, http.MethodPost, srv.URL+"/auth/password-reset/request", map[string]string{"email": "reset-user@example.com"}, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 reset request, got %d", res.StatusCode)
	}

	resetToken := sender.LastToken(t)
	res, _ = doJSON(t, http.MethodPost, srv.URL+"/auth/password-reset/confirm", map[string]string{
		"token":        resetToken,
		"new_password": "NewStrongPassw0rd!",
	}, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 reset confirm, got %d", res.StatusCode)
	}

	res, _ = doJSON(t, http.MethodPost, srv.URL+"/auth/password-reset/confirm", map[string]string{
		"token":        resetToken,
		"new_password": "AnotherStrongPassw0rd!",
	}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 on token replay, got %d", res.StatusCode)
	}

	res, _ = doJSON(t, http.MethodGet, srv.URL+"/auth/me", nil, map[string]string{"Authorization": "Bearer " + oldToken})
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 old session invalidated, got %d", res.StatusCode)
	}

	res, payload := doJSON(t, http.MethodPost, srv.URL+"/auth/login", map[string]string{
		"email":    "reset-user@example.com",
		"password": "NewStrongPassw0rd!",
	}, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected login with new password to succeed, got %d", res.StatusCode)
	}
	if payload["access_token"] == "" {
		t.Fatalf("expected access token after new-password login")
	}

	var usedCount int64
	if err := gormDB.Model(&models.PasswordResetToken{}).Where("used_at IS NOT NULL").Count(&usedCount).Error; err != nil {
		t.Fatalf("count used tokens: %v", err)
	}
	if usedCount == 0 {
		t.Fatalf("expected used token record")
	}
}

func TestPasswordResetExpiredTokenFails(t *testing.T) {
	srv, gormDB, sender := setupTestServer(t)
	defer srv.Close()

	_, _ = doJSON(t, http.MethodPost, srv.URL+"/auth/register", map[string]string{
		"email":    "expired-reset@example.com",
		"password": "StrongPassw0rd!",
	}, nil)

	_, _ = doJSON(t, http.MethodPost, srv.URL+"/auth/password-reset/request", map[string]string{"email": "expired-reset@example.com"}, nil)
	resetToken := sender.LastToken(t)

	hash := services.HashToken(resetToken)
	if err := gormDB.Model(&models.PasswordResetToken{}).Where("token_hash = ?", hash).Update("expires_at", time.Now().UTC().Add(-1*time.Minute)).Error; err != nil {
		t.Fatalf("expire token: %v", err)
	}

	res, _ := doJSON(t, http.MethodPost, srv.URL+"/auth/password-reset/confirm", map[string]string{
		"token":        resetToken,
		"new_password": "NewStrongPassw0rd!",
	}, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 expired token, got %d", res.StatusCode)
	}
}

func TestLegacyEndpointsRegression(t *testing.T) {
	srv, _, _ := setupTestServer(t)
	defer srv.Close()

	res, _ := doJSON(t, http.MethodGet, srv.URL+"/hello", nil, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected /hello 200, got %d", res.StatusCode)
	}

	res, _ = doJSON(t, http.MethodGet, srv.URL+"/time", nil, map[string]string{"X-Timezone": "UTC"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected /time 200, got %d", res.StatusCode)
	}

	res, _ = doJSON(t, http.MethodPost, srv.URL+"/ping", map[string]string{"message": "hello"}, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected /ping 200, got %d", res.StatusCode)
	}

	res, _ = doJSON(t, http.MethodGet, srv.URL+"/pings", nil, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected /pings 200, got %d", res.StatusCode)
	}
}
