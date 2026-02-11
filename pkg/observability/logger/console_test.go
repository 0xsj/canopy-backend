package logger

import (
	"bytes"
	"strings"
	"testing"
)

func newTestLogger(opts ...Option) (Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	defaults := []Option{
		WithOutput(&buf),
		WithColor(false),
		WithTimestamps(false),
	}
	return NewConsole(append(defaults, opts...)...), &buf
}

func TestConsoleLogger_SatisfiesInterface(t *testing.T) {
	var _ Logger = NewConsole()
}

func TestConsoleLogger_BasicOutput(t *testing.T) {
	log, buf := newTestLogger()
	log.Info("hello world")

	got := buf.String()
	if !strings.Contains(got, "INFO") {
		t.Errorf("output missing level, got %q", got)
	}
	if !strings.Contains(got, "hello world") {
		t.Errorf("output missing message, got %q", got)
	}
}

func TestConsoleLogger_FieldsInOutput(t *testing.T) {
	log, buf := newTestLogger()
	log.Info("request", String("method", "GET"), Int("status", 200))

	got := buf.String()
	if !strings.Contains(got, "method=GET") {
		t.Errorf("output missing field, got %q", got)
	}
	if !strings.Contains(got, "status=200") {
		t.Errorf("output missing field, got %q", got)
	}
}

func TestConsoleLogger_LevelFiltering(t *testing.T) {
	log, buf := newTestLogger(WithLevel(LevelWarn))

	log.Debug("should not appear")
	log.Info("should not appear")
	log.Warn("should appear")
	log.Error("should appear")

	got := buf.String()
	if strings.Contains(got, "should not appear") {
		t.Errorf("filtered levels should not appear, got %q", got)
	}
	if !strings.Contains(got, "WARN") {
		t.Errorf("WARN should appear, got %q", got)
	}
	if !strings.Contains(got, "ERROR") {
		t.Errorf("ERROR should appear, got %q", got)
	}
}

func TestConsoleLogger_WithPresetFields(t *testing.T) {
	log, buf := newTestLogger()
	child := log.With(String("request_id", "abc"))
	child.Info("handling request", String("path", "/leaf"))

	got := buf.String()
	if !strings.Contains(got, "request_id=abc") {
		t.Errorf("output missing preset field, got %q", got)
	}
	if !strings.Contains(got, "path=/leaf") {
		t.Errorf("output missing call-site field, got %q", got)
	}
}

func TestConsoleLogger_WithDoesNotMutateParent(t *testing.T) {
	log, buf := newTestLogger()
	_ = log.With(String("child_key", "child_val"))
	log.Info("parent log")

	got := buf.String()
	if strings.Contains(got, "child_key") {
		t.Errorf("parent should not have child fields, got %q", got)
	}
}

func TestConsoleLogger_Timestamps(t *testing.T) {
	log, buf := newTestLogger(WithTimestamps(true))
	log.Info("with time")

	got := buf.String()
	// Timestamp format is HH:MM:SS.mmm — should contain a colon.
	if !strings.Contains(got, ":") {
		t.Errorf("expected timestamp with colon, got %q", got)
	}
}

func TestConsoleLogger_NoTimestamps(t *testing.T) {
	log, buf := newTestLogger(WithTimestamps(false))
	log.Info("no time")

	got := buf.String()
	// Without timestamps, line should start directly with the level.
	if !strings.HasPrefix(strings.TrimSpace(got), "INFO") {
		t.Errorf("expected line to start with level, got %q", got)
	}
}

func TestConsoleLogger_Caller(t *testing.T) {
	log, buf := newTestLogger(WithCaller(true))
	log.Info("with caller")

	got := buf.String()
	if !strings.Contains(got, "console_test.go:") {
		t.Errorf("expected caller info, got %q", got)
	}
}

func TestConsoleLogger_AllLevels(t *testing.T) {
	tests := []struct {
		name   string
		logFn  func(Logger)
		expect string
	}{
		{"Debug", func(l Logger) { l.Debug("msg") }, "DEBUG"},
		{"Info", func(l Logger) { l.Info("msg") }, "INFO"},
		{"Warn", func(l Logger) { l.Warn("msg") }, "WARN"},
		{"Error", func(l Logger) { l.Error("msg") }, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, buf := newTestLogger(WithLevel(LevelDebug))
			tt.logFn(log)
			if !strings.Contains(buf.String(), tt.expect) {
				t.Errorf("output missing %q, got %q", tt.expect, buf.String())
			}
		})
	}
}

func TestConsoleLogger_ColorOutput(t *testing.T) {
	var buf bytes.Buffer
	log := NewConsole(
		WithOutput(&buf),
		WithColor(true),
		WithTimestamps(false),
	)
	log.Error("colored")

	got := buf.String()
	// ANSI escape code for red should be present.
	if !strings.Contains(got, "\033[31m") {
		t.Errorf("expected ANSI red in output, got %q", got)
	}
}

func TestConsoleLogger_NoColorOutput(t *testing.T) {
	log, buf := newTestLogger(WithColor(false))
	log.Error("plain")

	got := buf.String()
	if strings.Contains(got, "\033[") {
		t.Errorf("expected no ANSI codes, got %q", got)
	}
}
