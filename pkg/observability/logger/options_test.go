package logger

import (
	"bytes"
	"os"
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.Level != LevelInfo {
		t.Errorf("Level = %v, want LevelInfo", opts.Level)
	}
	if !opts.Color {
		t.Error("Color = false, want true")
	}
	if !opts.Timestamps {
		t.Error("Timestamps = false, want true")
	}
	if opts.Caller {
		t.Error("Caller = true, want false")
	}
	if opts.Output != os.Stdout {
		t.Error("Output should default to os.Stdout")
	}
}

func TestFunctionalOptions(t *testing.T) {
	var buf bytes.Buffer
	opts := DefaultOptions()

	for _, opt := range []Option{
		WithLevel(LevelError),
		WithColor(false),
		WithTimestamps(false),
		WithCaller(true),
		WithOutput(&buf),
	} {
		opt(&opts)
	}

	if opts.Level != LevelError {
		t.Errorf("Level = %v, want LevelError", opts.Level)
	}
	if opts.Color {
		t.Error("Color = true, want false")
	}
	if opts.Timestamps {
		t.Error("Timestamps = true, want false")
	}
	if !opts.Caller {
		t.Error("Caller = false, want true")
	}
	if opts.Output != &buf {
		t.Error("Output should be the buffer")
	}
}
