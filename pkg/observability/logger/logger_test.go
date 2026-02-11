package logger

import (
	"context"
	"testing"
)

func TestWithContext_And_FromContext(t *testing.T) {
	log := NewConsole()
	ctx := WithContext(context.Background(), log)

	got := FromContext(ctx)
	if got == nil {
		t.Fatal("FromContext returned nil")
	}

	// Should be the same logger we put in.
	if got != log {
		t.Error("FromContext returned a different logger")
	}
}

func TestFromContext_ReturnsNoopWhenMissing(t *testing.T) {
	ctx := context.Background()
	got := FromContext(ctx)

	if got == nil {
		t.Fatal("FromContext returned nil, want noop")
	}

	// Should be a noop — calling methods should not panic.
	got.Debug("should not panic")
	got.Info("should not panic")
	got.Warn("should not panic")
	got.Error("should not panic")
	got.With(String("k", "v")).Info("should not panic")
}
