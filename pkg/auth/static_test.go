package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStaticValidator_ReturnsClaims(t *testing.T) {
	expected := Claims{
		Subject:   "user_test123",
		Email:     "dev@canopy.dev",
		Issuer:    "static",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	v := NewStaticValidator(expected)
	got, err := v.Validate(context.Background(), "any-token-value")

	if err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
	if got.Subject != expected.Subject {
		t.Errorf("Subject = %q, want %q", got.Subject, expected.Subject)
	}
	if got.Email != expected.Email {
		t.Errorf("Email = %q, want %q", got.Email, expected.Email)
	}
}

func TestStaticValidator_IgnoresToken(t *testing.T) {
	v := NewStaticValidator(Claims{Subject: "user_1"})

	for _, token := range []string{"", "abc", "eyJhbGciOiJSUzI1NiJ9.test.sig"} {
		got, err := v.Validate(context.Background(), token)
		if err != nil {
			t.Fatalf("Validate(%q) error = %v", token, err)
		}
		if got.Subject != "user_1" {
			t.Errorf("Validate(%q) Subject = %q, want user_1", token, got.Subject)
		}
	}
}

func TestFailingValidator_ReturnsError(t *testing.T) {
	want := errors.New("token revoked")
	v := NewFailingValidator(want)

	_, err := v.Validate(context.Background(), "any-token")
	if err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
	if err.Error() != want.Error() {
		t.Errorf("error = %q, want %q", err.Error(), want.Error())
	}
}

func TestFailingValidator_ReturnsZeroClaims(t *testing.T) {
	v := NewFailingValidator(errors.New("fail"))

	claims, _ := v.Validate(context.Background(), "any-token")
	if !claims.IsZero() {
		t.Error("failing validator should return zero claims")
	}
}
