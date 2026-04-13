package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testApp(t *testing.T) *App {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=private", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(&Ping{}, &User{}); err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	app := NewApp(db, AuthConfig{JWTSecret: []byte("test-secret"), JWTTTL: time.Hour})
	app.now = func() time.Time {
		return time.Date(2026, time.April, 13, 12, 0, 0, 0, time.UTC)
	}

	return app
}

func performJSONRequest(t *testing.T, router http.Handler, method string, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func TestSignupAndDuplicateEmail(t *testing.T) {
	app := testApp(t)
	router := app.Router()

	resp := performJSONRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"email":    "USER@example.com",
		"password": "supersecret",
	}, nil)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d body=%s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var first map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &first); err != nil {
		t.Fatalf("invalid signup response: %v", err)
	}
	user := first["user"].(map[string]any)
	if user["email"] != "user@example.com" {
		t.Fatalf("expected normalized email, got %v", user["email"])
	}
	tokenObj := first["token"].(map[string]any)
	if tokenObj["access_token"] == "" {
		t.Fatalf("expected access token")
	}

	respDup := performJSONRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"email":    "user@example.com",
		"password": "supersecret",
	}, nil)
	if respDup.Code != http.StatusConflict {
		t.Fatalf("expected %d, got %d body=%s", http.StatusConflict, respDup.Code, respDup.Body.String())
	}
}

func TestLoginSuccessAndInvalidCredentials(t *testing.T) {
	app := testApp(t)
	router := app.Router()

	_ = performJSONRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"email":    "user@example.com",
		"password": "supersecret",
	}, nil)

	loginResp := performJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"email":    "user@example.com",
		"password": "supersecret",
	}, nil)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, loginResp.Code, loginResp.Body.String())
	}

	badLoginResp := performJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"email":    "user@example.com",
		"password": "wrong-password",
	}, nil)
	if badLoginResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d body=%s", http.StatusUnauthorized, badLoginResp.Code, badLoginResp.Body.String())
	}
}

func TestAuthMiddlewareMissingInvalidExpiredAndValidToken(t *testing.T) {
	app := testApp(t)
	router := app.Router()

	signupResp := performJSONRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"email":    "user@example.com",
		"password": "supersecret",
	}, nil)

	var payload map[string]any
	if err := json.Unmarshal(signupResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse signup response: %v", err)
	}
	token := payload["token"].(map[string]any)["access_token"].(string)

	missingTokenResp := performJSONRequest(t, router, http.MethodGet, "/auth/me", nil, nil)
	if missingTokenResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d body=%s", http.StatusUnauthorized, missingTokenResp.Code, missingTokenResp.Body.String())
	}

	invalidTokenResp := performJSONRequest(t, router, http.MethodGet, "/auth/me", nil, map[string]string{
		"Authorization": "Bearer not-a-token",
	})
	if invalidTokenResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d body=%s", http.StatusUnauthorized, invalidTokenResp.Code, invalidTokenResp.Body.String())
	}

	expiredClaims := jwt.MapClaims{
		"sub": strconv.FormatUint(1, 10),
		"iat": app.now().Add(-2 * time.Hour).Unix(),
		"exp": app.now().Add(-1 * time.Hour).Unix(),
	}
	expiredToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims).SignedString(app.authConfig.JWTSecret)
	if err != nil {
		t.Fatalf("failed to create expired token: %v", err)
	}

	expiredTokenResp := performJSONRequest(t, router, http.MethodGet, "/auth/me", nil, map[string]string{
		"Authorization": "Bearer " + expiredToken,
	})
	if expiredTokenResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d body=%s", http.StatusUnauthorized, expiredTokenResp.Code, expiredTokenResp.Body.String())
	}

	validResp := performJSONRequest(t, router, http.MethodGet, "/auth/me", nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
	if validResp.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, validResp.Code, validResp.Body.String())
	}
}

func TestPublicRoutesRemainAccessible(t *testing.T) {
	app := testApp(t)
	router := app.Router()

	helloResp := performJSONRequest(t, router, http.MethodGet, "/hello", nil, nil)
	if helloResp.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, helloResp.Code)
	}

	timeResp := performJSONRequest(t, router, http.MethodGet, "/time", nil, nil)
	if timeResp.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, timeResp.Code)
	}
}
