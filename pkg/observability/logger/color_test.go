package logger

import (
	"strings"
	"testing"
)

func TestColorize(t *testing.T) {
	got := colorize(colorRed, "ERROR")
	if !strings.HasPrefix(got, "\033[31m") {
		t.Errorf("expected red prefix, got %q", got)
	}
	if !strings.HasSuffix(got, "\033[0m") {
		t.Errorf("expected reset suffix, got %q", got)
	}
	if !strings.Contains(got, "ERROR") {
		t.Errorf("expected text in output, got %q", got)
	}
}

func TestLevelColor_AllLevels(t *testing.T) {
	tests := []struct {
		level Level
		color string
		text  string
	}{
		{LevelDebug, colorGray, "DEBUG"},
		{LevelInfo, colorGreen, "INFO"},
		{LevelWarn, colorYellow, "WARN"},
		{LevelError, colorRed, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := levelColor(tt.level)
			if !strings.Contains(got, tt.color) {
				t.Errorf("levelColor(%v) missing color code, got %q", tt.level, got)
			}
			if !strings.Contains(got, tt.text) {
				t.Errorf("levelColor(%v) missing text, got %q", tt.level, got)
			}
		})
	}
}

func TestLevelPlain(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, " INFO"},
		{LevelWarn, " WARN"},
		{LevelError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := levelPlain(tt.level); got != tt.want {
				t.Errorf("levelPlain(%v) = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}
