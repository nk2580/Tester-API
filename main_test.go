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

func TestPingHandler_PersistsPing(t *testing.T) {
	// Open an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	// Auto-migrate the Ping schema
	if err := db.AutoMigrate(&Ping{}); err != nil {
		t.Fatalf("failed to migrate in-memory database: %v", err)
	}

	// Setup router with the in-memory DB
	router := SetupRouter(db)

	// Prepare request body
	payload := map[string]string{"message": "hello unit test"}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/ping", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(rec, req)

	// Assert HTTP 200
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d; body: %s", rec.Code, rec.Body.String())
	}

	// Verify the DB contains exactly one Ping with the expected message
	var pings []Ping
	if err := db.Find(&pings).Error; err != nil {
		t.Fatalf("failed to query pings: %v", err)
	}

	if len(pings) != 1 {
		t.Fatalf("expected 1 ping, found %d", len(pings))
	}

	if pings[0].Message != "hello unit test" {
		t.Fatalf("unexpected ping message: %q", pings[0].Message)
	}
}
