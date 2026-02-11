package types

import (
	"encoding/json"
	"time"
)

// Timestamp wraps time.Time to provide consistent JSON formatting
// and a clear semantic: all timestamps in Canopy are UTC.
type Timestamp struct {
	time time.Time
}

// Now returns the current time as a Timestamp in UTC.
func Now() Timestamp {
	return Timestamp{time: time.Now().UTC()}
}

// TimestampFrom creates a Timestamp from an existing time.Time, normalizing to UTC.
func TimestampFrom(t time.Time) Timestamp {
	return Timestamp{time: t.UTC()}
}

// Time returns the underlying time.Time value.
func (ts Timestamp) Time() time.Time {
	return ts.time
}

// IsZero reports whether the timestamp is unset.
func (ts Timestamp) IsZero() bool {
	return ts.time.IsZero()
}

// String returns RFC 3339 formatted string.
func (ts Timestamp) String() string {
	if ts.time.IsZero() {
		return ""
	}
	return ts.time.Format(time.RFC3339)
}

// MarshalJSON outputs RFC 3339 format. Zero timestamps marshal as null.
func (ts Timestamp) MarshalJSON() ([]byte, error) {
	if ts.time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(ts.time.Format(time.RFC3339))
}

// UnmarshalJSON parses RFC 3339 format.
func (ts *Timestamp) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		ts.time = time.Time{}
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	ts.time = t.UTC()
	return nil
}

// Timestamps is a pair of creation and optional update times.
// Immutable entities (leaves) only use CreatedAt.
// Mutable entities (deliverables) use both.
type Timestamps struct {
	CreatedAt Timestamp  `json:"created_at"`
	UpdatedAt *Timestamp `json:"updated_at,omitempty"`
}

// NewTimestamps creates a Timestamps with CreatedAt set to now and no UpdatedAt.
// Use this for immutable entities like leaves.
func NewTimestamps() Timestamps {
	return Timestamps{CreatedAt: Now()}
}

// NewMutableTimestamps creates a Timestamps with both CreatedAt and UpdatedAt set to now.
// Use this for mutable entities like deliverables.
func NewMutableTimestamps() Timestamps {
	now := Now()
	return Timestamps{CreatedAt: now, UpdatedAt: &now}
}

// Touch sets UpdatedAt to now. Panics if called on an immutable Timestamps
// (where UpdatedAt was never initialized) — this is intentional to catch
// accidental mutation of immutable entities at development time.
func (ts *Timestamps) Touch() {
	now := Now()
	ts.UpdatedAt = &now
}
