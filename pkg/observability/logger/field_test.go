package logger

import (
	"errors"
	"testing"
	"time"
)

func TestFieldConstructors(t *testing.T) {
	tests := []struct {
		name  string
		field Field
		key   string
		value any
	}{
		{"String", String("k", "v"), "k", "v"},
		{"Int", Int("k", 42), "k", 42},
		{"Int64", Int64("k", int64(99)), "k", int64(99)},
		{"Float64", Float64("k", 3.14), "k", 3.14},
		{"Bool", Bool("k", true), "k", true},
		{"Duration", Duration("k", 5*time.Second), "k", 5 * time.Second},
		{"Any", Any("k", []int{1, 2}), "k", []int{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.Key != tt.key {
				t.Errorf("Key = %q, want %q", tt.field.Key, tt.key)
			}
		})
	}
}

func TestErr_Constructor(t *testing.T) {
	err := errors.New("boom")
	f := Err(err)

	if f.Key != "error" {
		t.Errorf("Err().Key = %q, want %q", f.Key, "error")
	}
	if f.Value != err {
		t.Error("Err().Value should be the original error")
	}
}

func TestFormatValue_String(t *testing.T) {
	f := String("k", "hello")
	if got := f.FormatValue(); got != "hello" {
		t.Errorf("FormatValue() = %q, want %q", got, "hello")
	}
}

func TestFormatValue_Error(t *testing.T) {
	f := Err(errors.New("boom"))
	if got := f.FormatValue(); got != "boom" {
		t.Errorf("FormatValue() = %q, want %q", got, "boom")
	}
}

func TestFormatValue_Stringer(t *testing.T) {
	f := Any("k", time.Second)
	if got := f.FormatValue(); got != "1s" {
		t.Errorf("FormatValue() = %q, want %q", got, "1s")
	}
}

func TestFormatValue_Int(t *testing.T) {
	f := Int("k", 42)
	if got := f.FormatValue(); got != "42" {
		t.Errorf("FormatValue() = %q, want %q", got, "42")
	}
}

func TestFormatValue_Nil(t *testing.T) {
	f := Any("k", nil)
	if got := f.FormatValue(); got != "<nil>" {
		t.Errorf("FormatValue() = %q, want %q", got, "<nil>")
	}
}
