package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	sqlite3 "github.com/mattn/go-sqlite3"
	"gorm.io/gorm"
)

func TestLoadJWTTTLSecondsFromEnv(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("JWT_TTL", "")
		ttl, err := loadJWTTTLSecondsFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ttl != defaultJWTTTLSeconds {
			t.Fatalf("expected %d, got %d", defaultJWTTTLSeconds, ttl)
		}
	})

	t.Run("valid", func(t *testing.T) {
		t.Setenv("JWT_TTL", "90")
		ttl, err := loadJWTTTLSecondsFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ttl != 90 {
			t.Fatalf("expected 90, got %d", ttl)
		}
	})

	for _, tc := range []string{"abc", "0", "-4"} {
		t.Run("invalid_"+tc, func(t *testing.T) {
			t.Setenv("JWT_TTL", tc)
			_, err := loadJWTTTLSecondsFromEnv()
			if err == nil {
				t.Fatalf("expected error for ttl=%q", tc)
			}
		})
	}
}

func TestParseBearerToken(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		wantErr bool
	}{
		{name: "valid", header: "Bearer token123"},
		{name: "valid_case_insensitive", header: "bearer token123"},
		{name: "missing", header: "", wantErr: true},
		{name: "missing_value", header: "Bearer   ", wantErr: true},
		{name: "wrong_prefix", header: "Token token123", wantErr: true},
		{name: "single_part", header: "Bearer", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := parseBearerToken(tc.header)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if token != "token123" {
				t.Fatalf("unexpected token: %q", token)
			}
		})
	}
}

func TestParseTokenUserID(t *testing.T) {
	now := time.Date(2026, time.April, 13, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		claims  jwt.MapClaims
		wantErr bool
	}{
		{
			name: "valid",
			claims: jwt.MapClaims{
				"sub": "42",
				"exp": float64(now.Add(time.Minute).Unix()),
			},
		},
		{
			name: "missing_subject",
			claims: jwt.MapClaims{
				"exp": float64(now.Add(time.Minute).Unix()),
			},
			wantErr: true,
		},
		{
			name: "invalid_subject",
			claims: jwt.MapClaims{
				"sub": "abc",
				"exp": float64(now.Add(time.Minute).Unix()),
			},
			wantErr: true,
		},
		{
			name: "missing_exp",
			claims: jwt.MapClaims{
				"sub": "42",
			},
			wantErr: true,
		},
		{
			name: "expired",
			claims: jwt.MapClaims{
				"sub": "42",
				"exp": float64(now.Add(-time.Second).Unix()),
			},
			wantErr: true,
		},
		{
			name: "exp_equal_now",
			claims: jwt.MapClaims{
				"sub": "42",
				"exp": float64(now.Unix()),
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id, err := parseTokenUserID(tc.claims, now)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != 42 {
				t.Fatalf("expected 42, got %d", id)
			}
		})
	}
}

func TestAuthUserIDFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	if _, ok := authUserIDFromContext(ctx); ok {
		t.Fatalf("expected missing key to fail")
	}

	ctx.Set(string(authUserIDContextKey), "bad")
	if _, ok := authUserIDFromContext(ctx); ok {
		t.Fatalf("expected invalid type to fail")
	}

	ctx.Set(string(authUserIDContextKey), uint(7))
	id, ok := authUserIDFromContext(ctx)
	if !ok || id != 7 {
		t.Fatalf("expected id=7, got id=%d ok=%v", id, ok)
	}
}

func TestOpenDatabase(t *testing.T) {
	db, err := openDatabase(fmt.Sprintf("file:%s?mode=memory&cache=private", t.Name()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := db.Create(&Ping{Message: "hello"}).Error; err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	_, err = openDatabase("/definitely-not-a-real-dir/test.db")
	if err == nil {
		t.Fatalf("expected openDatabase to fail for invalid path")
	}
}

func TestCreateAndListPings(t *testing.T) {
	app := testApp(t)
	router := app.Router()

	bad := performJSONRequest(t, router, http.MethodPost, "/ping", map[string]any{"id": "bad"}, nil)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, bad.Code)
	}

	ok := performJSONRequest(t, router, http.MethodPost, "/ping", map[string]any{"message": "hi"}, nil)
	if ok.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, ok.Code, ok.Body.String())
	}

	list := performJSONRequest(t, router, http.MethodGet, "/pings", nil, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, list.Code)
	}

	var pings []map[string]any
	if err := json.Unmarshal(list.Body.Bytes(), &pings); err != nil {
		t.Fatalf("failed to decode list: %v", err)
	}
	if len(pings) != 1 {
		t.Fatalf("expected 1 ping, got %d", len(pings))
	}

	if err := app.db.Exec("DROP TABLE pings").Error; err != nil {
		t.Fatalf("failed to drop pings table: %v", err)
	}

	createFail := performJSONRequest(t, router, http.MethodPost, "/ping", map[string]any{"message": "again"}, nil)
	if createFail.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, createFail.Code)
	}

	listFail := performJSONRequest(t, router, http.MethodGet, "/pings", nil, nil)
	if listFail.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, listFail.Code)
	}
}

func TestGetTimeEndpoint(t *testing.T) {
	app := testApp(t)
	router := app.Router()

	invalid := performJSONRequest(t, router, http.MethodGet, "/time", nil, map[string]string{"X-Timezone": "No/Such_Zone"})
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, invalid.Code)
	}

	valid := performJSONRequest(t, router, http.MethodGet, "/time", nil, map[string]string{"X-Timezone": "Australia/Brisbane"})
	if valid.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d body=%s", http.StatusOK, valid.Code, valid.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(valid.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}
	if payload["timezone"] != "Australia/Brisbane" {
		t.Fatalf("unexpected timezone: %v", payload["timezone"])
	}
}

func TestSignupValidationErrors(t *testing.T) {
	app := testApp(t)
	router := app.Router()

	badJSON := httptest.NewRequest(http.MethodPost, "/auth/signup", strings.NewReader("{"))
	badJSON.Header.Set("Content-Type", "application/json")
	badJSONResp := httptest.NewRecorder()
	router.ServeHTTP(badJSONResp, badJSON)
	if badJSONResp.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, badJSONResp.Code)
	}

	invalidEmail := performJSONRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"email":    "not-an-email",
		"password": "supersecret",
	}, nil)
	if invalidEmail.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, invalidEmail.Code)
	}

	invalidPassword := performJSONRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"email":    "user@example.com",
		"password": "short",
	}, nil)
	if invalidPassword.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, invalidPassword.Code)
	}

	// bcrypt rejects passwords longer than 72 bytes; this exercises the internal-error branch.
	tooLongPassword := performJSONRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"email":    "toolong@example.com",
		"password": strings.Repeat("a", 73),
	}, nil)
	if tooLongPassword.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, tooLongPassword.Code)
	}
}

func TestLoginValidationAndMissingUser(t *testing.T) {
	app := testApp(t)
	router := app.Router()

	badJSON := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("{"))
	badJSON.Header.Set("Content-Type", "application/json")
	badJSONResp := httptest.NewRecorder()
	router.ServeHTTP(badJSONResp, badJSON)
	if badJSONResp.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, badJSONResp.Code)
	}

	invalidEmail := performJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"email":    "not-an-email",
		"password": "supersecret",
	}, nil)
	if invalidEmail.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, invalidEmail.Code)
	}

	unknownUser := performJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"email":    "nobody@example.com",
		"password": "supersecret",
	}, nil)
	if unknownUser.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, unknownUser.Code)
	}
}

func TestAuthMiddlewareInvalidSigningMethodAndSubject(t *testing.T) {
	app := testApp(t)
	router := app.Router()

	signup := performJSONRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"email":    "user@example.com",
		"password": "supersecret",
	}, nil)
	if signup.Code != http.StatusCreated {
		t.Fatalf("failed to create test user: %d", signup.Code)
	}

	claims := jwt.MapClaims{"sub": "1", "exp": app.now().Add(time.Hour).Unix()}
	noneToken, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to create none token: %v", err)
	}

	noneResp := performJSONRequest(t, router, http.MethodGet, "/auth/me", nil, map[string]string{
		"Authorization": "Bearer " + noneToken,
	})
	if noneResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, noneResp.Code)
	}

	badSubClaims := jwt.MapClaims{"sub": "abc", "exp": app.now().Add(time.Hour).Unix()}
	badSubToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, badSubClaims).SignedString(app.authConfig.JWTSecret)
	if err != nil {
		t.Fatalf("failed to create bad-sub token: %v", err)
	}

	badSubResp := performJSONRequest(t, router, http.MethodGet, "/auth/me", nil, map[string]string{
		"Authorization": "Bearer " + badSubToken,
	})
	if badSubResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, badSubResp.Code)
	}
}

func TestMeUnauthorizedWhenUserNoLongerExists(t *testing.T) {
	app := testApp(t)
	router := app.Router()

	signupResp := performJSONRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"email":    "user@example.com",
		"password": "supersecret",
	}, nil)
	if signupResp.Code != http.StatusCreated {
		t.Fatalf("signup failed: %d", signupResp.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(signupResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse signup payload: %v", err)
	}
	token := payload["token"].(map[string]any)["access_token"].(string)

	if err := app.db.Where("1 = 1").Delete(&User{}).Error; err != nil {
		t.Fatalf("failed to delete users: %v", err)
	}

	resp := performJSONRequest(t, router, http.MethodGet, "/auth/me", nil, map[string]string{
		"Authorization": "Bearer " + token,
	})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, resp.Code)
	}
}

func TestMeHandlerContextBranches(t *testing.T) {
	app := testApp(t)

	newContext := func() (*gin.Context, *httptest.ResponseRecorder) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		return c, rec
	}

	c1, rec1 := newContext()
	app.me(c1)
	if rec1.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec1.Code)
	}

	c2, rec2 := newContext()
	c2.Set(string(authUserIDContextKey), "bad-type")
	app.me(c2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec2.Code)
	}

	c3, rec3 := newContext()
	c3.Set(string(authUserIDContextKey), uint(1))
	if err := app.db.Exec("DROP TABLE users").Error; err != nil {
		t.Fatalf("failed to drop users table: %v", err)
	}
	app.me(c3)
	if rec3.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rec3.Code)
	}
}

func TestGenerateTokenAndHelpers(t *testing.T) {
	app := testApp(t)
	token, err := app.generateToken(9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token.TokenType != "Bearer" {
		t.Fatalf("unexpected token type: %s", token.TokenType)
	}
	if token.ExpiresIn != int64(time.Hour/time.Second) {
		t.Fatalf("unexpected expires_in: %d", token.ExpiresIn)
	}

	if _, err := normalizeEmail(" "); err == nil {
		t.Fatalf("expected normalizeEmail to fail for empty email")
	}
	if _, err := normalizeEmail("bad-email"); err == nil {
		t.Fatalf("expected normalizeEmail to fail for invalid email")
	}
	normalized, err := normalizeEmail(" USER@EXAMPLE.COM ")
	if err != nil || normalized != "user@example.com" {
		t.Fatalf("unexpected normalizeEmail result: %q, %v", normalized, err)
	}

	if err := validatePassword("short"); err == nil {
		t.Fatalf("expected validatePassword error")
	}
	if err := validatePassword("long-enough"); err != nil {
		t.Fatalf("unexpected validatePassword error: %v", err)
	}

	app.signToken = func(*jwt.Token) (string, error) {
		return "", errors.New("sign failed")
	}
	if _, err := app.generateToken(9); err == nil {
		t.Fatalf("expected generateToken to fail when signer fails")
	}
}

func TestSignupAndLoginInternalErrors(t *testing.T) {
	app := testApp(t)
	router := app.Router()

	if err := app.db.Exec("DROP TABLE users").Error; err != nil {
		t.Fatalf("failed to drop users table: %v", err)
	}

	signup := performJSONRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"email":    "user@example.com",
		"password": "supersecret",
	}, nil)
	if signup.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, signup.Code)
	}

	login := performJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"email":    "user@example.com",
		"password": "supersecret",
	}, nil)
	if login.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, login.Code)
	}
}

func TestSignupAndLoginTokenGenerationFailure(t *testing.T) {
	app := testApp(t)
	app.signToken = func(*jwt.Token) (string, error) {
		return "", errors.New("sign failed")
	}
	router := app.Router()

	signup := performJSONRequest(t, router, http.MethodPost, "/auth/signup", map[string]string{
		"email":    "user@example.com",
		"password": "supersecret",
	}, nil)
	if signup.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, signup.Code)
	}

	login := performJSONRequest(t, router, http.MethodPost, "/auth/login", map[string]string{
		"email":    "user@example.com",
		"password": "supersecret",
	}, nil)
	if login.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, login.Code)
	}
}

func TestIsDuplicateErrorAndWriteError(t *testing.T) {
	if !isDuplicateError(gorm.ErrDuplicatedKey) {
		t.Fatalf("expected gorm duplicated key to be treated as duplicate")
	}

	sqlErr := sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintUnique}
	if !isDuplicateError(sqlErr) {
		t.Fatalf("expected sqlite unique constraint to be treated as duplicate")
	}

	if !isDuplicateError(errors.New("UNIQUE constraint failed: users.email")) {
		t.Fatalf("expected unique constraint message to be treated as duplicate")
	}

	if isDuplicateError(errors.New("some other error")) {
		t.Fatalf("did not expect generic error to be duplicate")
	}

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	writeError(c, http.StatusTeapot, "test_code", "test message")
	if rec.Code != http.StatusTeapot {
		t.Fatalf("expected %d, got %d", http.StatusTeapot, rec.Code)
	}

	var payload map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode error payload: %v", err)
	}
	if payload["error"]["code"] != "test_code" || payload["error"]["message"] != "test message" {
		t.Fatalf("unexpected error payload: %+v", payload)
	}
}
