package api

import (
	"errors"
	"testing"

	"github.com/nk2580/Tester-API/internal/store"
)

// TestCreatePing_Success verifies that CreatePing successfully saves a valid ping
func TestCreatePing_Success(t *testing.T) {
	// Given an empty in-memory store
	memStore := store.NewMemoryStore()

	// And a valid ping payload with message "hello, world"
	ping := &store.Ping{
		Message: "hello, world",
	}

	// When the CreatePing business function is invoked with that payload
	err := CreatePing(memStore, ping)

	// Then the function returns no error
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// And the store contains exactly one Ping whose Message == "hello, world"
	pings, err := memStore.GetPings()
	if err != nil {
		t.Fatalf("failed to get pings: %v", err)
	}

	if len(pings) != 1 {
		t.Fatalf("expected 1 ping, got %d", len(pings))
	}

	if pings[0].Message != "hello, world" {
		t.Errorf("expected message 'hello, world', got '%s'", pings[0].Message)
	}
}

// TestCreatePing_ValidationError verifies that CreatePing fails when message is empty
func TestCreatePing_ValidationError(t *testing.T) {
	// Given an empty in-memory store
	memStore := store.NewMemoryStore()

	// Test cases for empty messages
	testCases := []struct {
		name    string
		message string
	}{
		{"empty string", ""},
		{"only spaces", "   "},
		{"only tabs", "\t\t"},
		{"only newlines", "\n\n"},
		{"mixed whitespace", " \t\n "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear the store for each test case
			memStore.Clear()

			// And a ping payload with empty/whitespace message
			ping := &store.Ping{
				Message: tc.message,
			}

			// When the CreatePing business function is invoked with that payload
			err := CreatePing(memStore, ping)

			// Then the function returns a validation error (client-side error)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}

			if !errors.Is(err, ErrEmptyMessage) {
				t.Errorf("expected ErrEmptyMessage, got: %v", err)
			}

			// And the store remains empty
			pings, err := memStore.GetPings()
			if err != nil {
				t.Fatalf("failed to get pings: %v", err)
			}

			if len(pings) != 0 {
				t.Errorf("expected store to be empty, got %d pings", len(pings))
			}
		})
	}
}

// TestCreatePing_StorageError verifies that CreatePing surfaces storage errors
func TestCreatePing_StorageError(t *testing.T) {
	// Given a store implementation that always returns an error on Save
	memStore := store.NewMemoryStore()
	expectedError := errors.New("simulated storage failure")
	memStore.SetErrorOnSave(expectedError)

	// And a valid ping payload
	ping := &store.Ping{
		Message: "valid message",
	}

	// When the CreatePing business function is invoked with a valid payload
	err := CreatePing(memStore, ping)

	// Then the function returns an error indicating an internal storage failure
	if err == nil {
		t.Fatal("expected storage error, got nil")
	}

	if !errors.Is(err, expectedError) {
		t.Errorf("expected error '%v', got '%v'", expectedError, err)
	}

	// And the store did not persist the ping (verify via another store without error)
	memStore.SetErrorOnSave(nil) // clear error for verification
	pings, err := memStore.GetPings()
	if err != nil {
		t.Fatalf("failed to get pings: %v", err)
	}

	if len(pings) != 0 {
		t.Errorf("expected store to be empty after failed save, got %d pings", len(pings))
	}
}

// TestListPings_ReturnsSeeded verifies that ListPings returns existing records
func TestListPings_ReturnsSeeded(t *testing.T) {
	// Given an in-memory store seeded with two pings ("a", "b")
	memStore := store.NewMemoryStore()
	seededPings := []store.Ping{
		{Message: "a"},
		{Message: "b"},
	}
	memStore.Seed(seededPings)

	// When the ListPings business function is invoked
	pings, err := ListPings(memStore)

	// Then the function returns no error
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// And the function returns an array with two pings matching the seeded messages
	if len(pings) != 2 {
		t.Fatalf("expected 2 pings, got %d", len(pings))
	}

	// Verify messages match (order should be preserved)
	if pings[0].Message != "a" {
		t.Errorf("expected first ping message 'a', got '%s'", pings[0].Message)
	}

	if pings[1].Message != "b" {
		t.Errorf("expected second ping message 'b', got '%s'", pings[1].Message)
	}
}

// TestListPings_EmptyStore verifies that ListPings returns empty array when no pings exist
func TestListPings_EmptyStore(t *testing.T) {
	// Given an empty in-memory store
	memStore := store.NewMemoryStore()

	// When the ListPings business function is invoked
	pings, err := ListPings(memStore)

	// Then the function returns no error
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// And the function returns an empty array
	if len(pings) != 0 {
		t.Errorf("expected empty array, got %d pings", len(pings))
	}
}

// TestCreatePing_TrimsWhitespace verifies that CreatePing trims whitespace from messages
func TestCreatePing_TrimsWhitespace(t *testing.T) {
	// Given an empty in-memory store
	memStore := store.NewMemoryStore()

	// And a ping payload with leading/trailing whitespace
	ping := &store.Ping{
		Message: "  hello world  ",
	}

	// When the CreatePing business function is invoked
	err := CreatePing(memStore, ping)

	// Then the function returns no error
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// And the stored message has whitespace trimmed
	pings, err := memStore.GetPings()
	if err != nil {
		t.Fatalf("failed to get pings: %v", err)
	}

	if len(pings) != 1 {
		t.Fatalf("expected 1 ping, got %d", len(pings))
	}

	if pings[0].Message != "hello world" {
		t.Errorf("expected trimmed message 'hello world', got '%s'", pings[0].Message)
	}
}
