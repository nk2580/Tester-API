package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nk2580/Tester-API/internal/auth"
	"github.com/nk2580/Tester-API/internal/config"
	"github.com/nk2580/Tester-API/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testMailer struct {
	verificationTokens []string
	resetTokens        []string
	confirmations      []string
}

func (m *testMailer) SendVerificationEmail(_ string, token string) error {
	m.verificationTokens = append(m.verificationTokens, token)
	return nil
}

func (m *testMailer) SendPasswordResetEmail(_ string, token string) error {
	m.resetTokens = append(m.resetTokens, token)
	return nil
}

func (m *testMailer) SendPasswordResetConfirmationEmail(toEmail string) error {
	m.confirmations = append(m.confirmations, toEmail)
	return nil
}

func newTestServer(t *testing.T) (*httptest.Server, *testMailer, config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Ping{},
		&models.User{},
		&models.Session{},
		&models.VerificationToken{},
		&models.PasswordResetToken{},
		&models.AuditEvent{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := config.Config{
		SessionCookieName:          "session_token",
		SessionTTL:                 time.Hour,
		VerificationTokenTTL:       time.Hour,
		PasswordResetTokenTTL:      time.Hour,
		ResendVerificationCooldown: 0,
		LockoutDuration:            time.Minute,
		MaxFailedLoginAttempts:     5,
		CookieSecure:               false,
		PasswordMinLength:          12,
	}
	mailer := &testMailer{}
	application := New(db, cfg, mailer, time.Now)
	r := gin.New()
	application.RegisterRoutes(r)
	server := httptest.NewServer(r)
	t.Cleanup(server.Close)
	return server, mailer, cfg
}

func postJSON(t *testing.T, client *http.Client, url string, payload any, cookies ...*http.Cookie) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func get(t *testing.T, client *http.Client, url string, cookies ...*http.Cookie) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func findCookie(resp *http.Response, name string) *http.Cookie {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func TestRegisterVerifyAndLoginFlow(t *testing.T) {
	server, mailer, cfg := newTestServer(t)
	client := server.Client()

	resp := postJSON(t, client, server.URL+"/auth/register", map[string]string{
		"email":    "user@example.com",
		"password": "VeryStrong#123",
	})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected %d got %d", http.StatusAccepted, resp.StatusCode)
	}
	if len(mailer.verificationTokens) != 1 {
		t.Fatalf("expected one verification token, got %d", len(mailer.verificationTokens))
	}

	resp = postJSON(t, client, server.URL+"/auth/login", map[string]string{
		"email":    "user@example.com",
		"password": "VeryStrong#123",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unverified login to fail with 401, got %d", resp.StatusCode)
	}

	resp = postJSON(t, client, server.URL+"/auth/verify-email", map[string]string{"token": mailer.verificationTokens[0]})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected verify status 200 got %d", resp.StatusCode)
	}

	resp = postJSON(t, client, server.URL+"/auth/login", map[string]string{
		"email":    "user@example.com",
		"password": "VeryStrong#123",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected login status 200 got %d", resp.StatusCode)
	}

	sessionCookie := findCookie(resp, cfg.SessionCookieName)
	if sessionCookie == nil || strings.TrimSpace(sessionCookie.Value) == "" {
		t.Fatalf("expected non-empty session cookie")
	}

	resp = get(t, client, server.URL+"/auth/me", sessionCookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected /auth/me status 200 got %d", resp.StatusCode)
	}
}

func TestForgotResetInvalidatesSessionsAndRejectsReuse(t *testing.T) {
	server, mailer, cfg := newTestServer(t)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	resp := postJSON(t, client, server.URL+"/auth/register", map[string]string{
		"email":    "reset@example.com",
		"password": "VeryStrong#123",
	})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("register expected %d got %d", http.StatusAccepted, resp.StatusCode)
	}

	resp = postJSON(t, client, server.URL+"/auth/verify-email", map[string]string{"token": mailer.verificationTokens[0]})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("verify expected 200 got %d", resp.StatusCode)
	}

	resp = postJSON(t, client, server.URL+"/auth/login", map[string]string{
		"email":    "reset@example.com",
		"password": "VeryStrong#123",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login expected 200 got %d", resp.StatusCode)
	}
	oldSession := findCookie(resp, cfg.SessionCookieName)
	if oldSession == nil {
		t.Fatalf("expected session cookie")
	}

	resp = postJSON(t, client, server.URL+"/auth/forgot-password", map[string]string{"email": "reset@example.com"})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("forgot expected %d got %d", http.StatusAccepted, resp.StatusCode)
	}
	if len(mailer.resetTokens) != 1 {
		t.Fatalf("expected one reset token got %d", len(mailer.resetTokens))
	}

	resp = postJSON(t, client, server.URL+"/auth/reset-password", map[string]string{
		"token":        mailer.resetTokens[0],
		"new_password": "Different#12345",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reset expected 200 got %d", resp.StatusCode)
	}
	if len(mailer.confirmations) != 1 {
		t.Fatalf("expected one reset confirmation email")
	}

	resp = get(t, client, server.URL+"/auth/me", oldSession)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected old session invalidated with 401 got %d", resp.StatusCode)
	}

	resp = postJSON(t, client, server.URL+"/auth/reset-password", map[string]string{
		"token":        mailer.resetTokens[0],
		"new_password": "Another#12345",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected reused token rejection 400 got %d", resp.StatusCode)
	}

	resp = postJSON(t, client, server.URL+"/auth/login", map[string]string{
		"email":    "reset@example.com",
		"password": "VeryStrong#123",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old password should fail, expected 401 got %d", resp.StatusCode)
	}

	resp = postJSON(t, client, server.URL+"/auth/login", map[string]string{
		"email":    "reset@example.com",
		"password": "Different#12345",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("new password should pass, expected 200 got %d", resp.StatusCode)
	}
}

func TestLoginLockout(t *testing.T) {
	server, mailer, _ := newTestServer(t)
	client := server.Client()

	resp := postJSON(t, client, server.URL+"/auth/register", map[string]string{
		"email":    "lock@example.com",
		"password": "VeryStrong#123",
	})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("register expected 202 got %d", resp.StatusCode)
	}

	resp = postJSON(t, client, server.URL+"/auth/verify-email", map[string]string{"token": mailer.verificationTokens[0]})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("verify expected 200 got %d", resp.StatusCode)
	}

	for i := 0; i < 6; i++ {
		resp = postJSON(t, client, server.URL+"/auth/login", map[string]string{
			"email":    "lock@example.com",
			"password": "WrongPassword#1",
		})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected unauthorized on failed attempt %d got %d", i+1, resp.StatusCode)
		}
	}

	resp = postJSON(t, client, server.URL+"/auth/login", map[string]string{
		"email":    "lock@example.com",
		"password": "VeryStrong#123",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected locked account to return 401 got %d", resp.StatusCode)
	}
}

var _ auth.Mailer = (*testMailer)(nil)
