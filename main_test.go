package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPingEndpoints(t *testing.T) {
	// Setup in-memory sqlite DB
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	// Migrate schema
	if err := db.AutoMigrate(&Ping{}); err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}

	router := SetupRouter(db)

	// POST /ping
	postBody := map[string]string{"message": "hello"}
	b, _ := json.Marshal(postBody)
	req := httptest.NewRequest(http.MethodPost, "/ping", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["message"] != "Ping registered successfully!" {
		t.Fatalf("unexpected response message: %v", resp)
	}

	// GET /pings
	req = httptest.NewRequest(http.MethodGet, "/pings", nil)
	rec = httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for GET /pings, got %d: %s", rec.Code, rec.Body.String())
	}

	var pings []Ping
	if err := json.Unmarshal(rec.Body.Bytes(), &pings); err != nil {
		t.Fatalf("failed to unmarshal pings response: %v", err)
	}

	if len(pings) != 1 {
		t.Fatalf("expected 1 ping, got %d", len(pings))
	}

	if pings[0].Message != "hello" {
		t.Fatalf("expected ping message 'hello', got '%s'", pings[0].Message)
	}
}
