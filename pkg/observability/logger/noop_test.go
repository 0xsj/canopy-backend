package logger

import "testing"

func TestNoopLogger_SatisfiesInterface(t *testing.T) {
	var _ Logger = NewNoop()
}

func TestNoopLogger_MethodsDoNotPanic(t *testing.T) {
	log := NewNoop()
	log.Debug("msg", String("k", "v"))
	log.Info("msg")
	log.Warn("msg")
	log.Error("msg")
}

func TestNoopLogger_WithReturnsSelf(t *testing.T) {
	log := NewNoop()
	child := log.With(String("k", "v"))

	if child != log {
		t.Error("Noop.With() should return the same instance")
	}
}
