package auth

import (
	"testing"
	"time"
)

func TestClaims_IsExpired_True(t *testing.T) {
	c := Claims{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	if !c.IsExpired() {
		t.Error("claims should be expired")
	}
}

func TestClaims_IsExpired_False(t *testing.T) {
	c := Claims{
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if c.IsExpired() {
		t.Error("claims should not be expired")
	}
}

func TestClaims_IsZero_Empty(t *testing.T) {
	var c Claims
	if !c.IsZero() {
		t.Error("zero-value claims should be zero")
	}
}

func TestClaims_IsZero_WithSubject(t *testing.T) {
	c := Claims{Subject: "user_abc123"}
	if c.IsZero() {
		t.Error("claims with subject should not be zero")
	}
}
