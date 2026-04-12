package services

import (
	"testing"
	"time"
)

func TestPasswordHashAndVerify(t *testing.T) {
	password := "StrongPassw0rd!"
	hash, err := HashPassword(password, 10)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == password {
		t.Fatalf("password hash should not equal plaintext")
	}
	if !VerifyPassword(hash, password) {
		t.Fatalf("expected password verification to succeed")
	}
	if VerifyPassword(hash, "wrong-password") {
		t.Fatalf("expected wrong password verification to fail")
	}
}

func TestTokenGenerationUniquenessAndHashing(t *testing.T) {
	t1, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken t1 error: %v", err)
	}
	t2, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken t2 error: %v", err)
	}
	if t1 == t2 {
		t.Fatalf("expected two generated tokens to be unique")
	}

	h1 := HashToken(t1)
	h2 := HashToken(t1)
	if h1 != h2 {
		t.Fatalf("expected hashing same token to be deterministic")
	}
	if h1 == HashToken(t2) {
		t.Fatalf("expected distinct token hashes")
	}
}

func TestBackoffDurationGrowthAndCap(t *testing.T) {
	if got := backoffDuration(4); got != 0 {
		t.Fatalf("expected no lockout for <5 failures, got %v", got)
	}
	if got := backoffDuration(5); got != 1*time.Second {
		t.Fatalf("expected 1s at failure=5, got %v", got)
	}
	if got := backoffDuration(8); got != 8*time.Second {
		t.Fatalf("expected 8s at failure=8, got %v", got)
	}
	if got := backoffDuration(40); got != 15*time.Minute {
		t.Fatalf("expected capped 15m lockout, got %v", got)
	}
}
