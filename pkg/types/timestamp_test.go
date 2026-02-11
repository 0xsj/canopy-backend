package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNow_IsUTC(t *testing.T) {
	ts := Now()
	if ts.Time().Location() != time.UTC {
		t.Errorf("Now() location = %v, want UTC", ts.Time().Location())
	}
}

func TestTimestampFrom_NormalizesToUTC(t *testing.T) {
	eastern, _ := time.LoadLocation("America/New_York")
	local := time.Date(2026, 1, 15, 10, 30, 0, 0, eastern)

	ts := TimestampFrom(local)
	if ts.Time().Location() != time.UTC {
		t.Errorf("TimestampFrom location = %v, want UTC", ts.Time().Location())
	}
	if !ts.Time().Equal(local) {
		t.Errorf("TimestampFrom time mismatch: got %v, want equivalent of %v", ts.Time(), local)
	}
}

func TestTimestamp_IsZero(t *testing.T) {
	var ts Timestamp
	if !ts.IsZero() {
		t.Error("zero-value Timestamp should be zero")
	}

	now := Now()
	if now.IsZero() {
		t.Error("Now() should not be zero")
	}
}

func TestTimestamp_String(t *testing.T) {
	ts := TimestampFrom(time.Date(2026, 2, 11, 15, 30, 0, 0, time.UTC))
	want := "2026-02-11T15:30:00Z"
	if got := ts.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestTimestamp_String_Zero(t *testing.T) {
	var ts Timestamp
	if got := ts.String(); got != "" {
		t.Errorf("zero Timestamp.String() = %q, want empty", got)
	}
}

func TestTimestamp_JSONRoundTrip(t *testing.T) {
	original := TimestampFrom(time.Date(2026, 2, 11, 15, 30, 0, 0, time.UTC))

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Timestamp
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !parsed.Time().Equal(original.Time()) {
		t.Errorf("round-trip: got %v, want %v", parsed.Time(), original.Time())
	}
}

func TestTimestamp_JSONMarshalZero(t *testing.T) {
	var ts Timestamp
	data, _ := json.Marshal(ts)
	if string(data) != "null" {
		t.Errorf("zero Timestamp JSON = %s, want null", data)
	}
}

func TestNewTimestamps_Immutable(t *testing.T) {
	ts := NewTimestamps()
	if ts.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if ts.UpdatedAt != nil {
		t.Error("UpdatedAt should be nil for immutable timestamps")
	}
}

func TestNewMutableTimestamps_HasBoth(t *testing.T) {
	ts := NewMutableTimestamps()
	if ts.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if ts.UpdatedAt == nil {
		t.Fatal("UpdatedAt should be set for mutable timestamps")
	}
	if ts.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

func TestTimestamps_Touch(t *testing.T) {
	ts := NewMutableTimestamps()
	original := ts.UpdatedAt.Time()

	// Small sleep to ensure time difference.
	time.Sleep(time.Millisecond)
	ts.Touch()

	if !ts.UpdatedAt.Time().After(original) {
		t.Error("Touch should advance UpdatedAt")
	}
}

func TestTimestamps_JSONOmitsNilUpdatedAt(t *testing.T) {
	ts := NewTimestamps()
	data, _ := json.Marshal(ts)

	var raw map[string]any
	json.Unmarshal(data, &raw)

	if _, ok := raw["updated_at"]; ok {
		t.Error("immutable Timestamps should omit updated_at from JSON")
	}
}
