package auth

import (
	"context"
	"testing"
)

func TestWithClaims_FromClaims_Roundtrip(t *testing.T) {
	original := Claims{
		Subject: "user_abc123",
		Email:   "test@canopy.dev",
	}

	ctx := WithClaims(context.Background(), original)
	got, ok := FromClaims(ctx)

	if !ok {
		t.Fatal("FromClaims returned ok=false, want true")
	}
	if got.Subject != original.Subject {
		t.Errorf("Subject = %q, want %q", got.Subject, original.Subject)
	}
	if got.Email != original.Email {
		t.Errorf("Email = %q, want %q", got.Email, original.Email)
	}
}

func TestFromClaims_EmptyContext(t *testing.T) {
	got, ok := FromClaims(context.Background())

	if ok {
		t.Error("FromClaims should return ok=false on empty context")
	}
	if !got.IsZero() {
		t.Error("FromClaims should return zero claims on empty context")
	}
}

func TestMustFromClaims_Success(t *testing.T) {
	c := Claims{Subject: "user_abc123"}
	ctx := WithClaims(context.Background(), c)

	got := MustFromClaims(ctx)
	if got.Subject != c.Subject {
		t.Errorf("Subject = %q, want %q", got.Subject, c.Subject)
	}
}

func TestMustFromClaims_PanicsOnEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustFromClaims should panic on empty context")
		}
	}()

	MustFromClaims(context.Background())
}
