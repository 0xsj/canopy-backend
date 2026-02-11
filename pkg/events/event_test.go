package events

import (
	"testing"
	"time"
)

func TestNew_CreatesEvent(t *testing.T) {
	type payload struct {
		Title string `json:"title"`
	}

	event, err := New("leaf_created", "ws_abc123", payload{Title: "test leaf"})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if event.ID == "" {
		t.Error("ID is empty")
	}
	if len(event.ID) != 32 {
		t.Errorf("ID length = %d, want 32", len(event.ID))
	}
	if event.Type != "leaf_created" {
		t.Errorf("Type = %q, want leaf_created", event.Type)
	}
	if event.WorkspaceID != "ws_abc123" {
		t.Errorf("WorkspaceID = %q, want ws_abc123", event.WorkspaceID)
	}
	if event.OccurredAt.IsZero() {
		t.Error("OccurredAt is zero")
	}
	if time.Since(event.OccurredAt) > time.Second {
		t.Errorf("OccurredAt = %v, should be recent", event.OccurredAt)
	}
	if event.Data == nil {
		t.Error("Data is nil")
	}
}

func TestNew_UniqueIDs(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		event, err := New("test", "ws_1", nil)
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		if seen[event.ID] {
			t.Fatalf("duplicate ID %q on iteration %d", event.ID, i)
		}
		seen[event.ID] = true
	}
}

func TestNew_MarshalError(t *testing.T) {
	// Channels cannot be marshaled to JSON.
	_, err := New("test", "ws_1", make(chan int))
	if err == nil {
		t.Error("New() = nil, want marshal error")
	}
}

func TestEvent_Decode(t *testing.T) {
	type payload struct {
		Title  string `json:"title"`
		Points int    `json:"points"`
	}

	original := payload{Title: "brainstorm", Points: 5}
	event, err := New("leaf_created", "ws_abc", original)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	var decoded payload
	if err := event.Decode(&decoded); err != nil {
		t.Fatalf("Decode() error: %v", err)
	}
	if decoded.Title != original.Title {
		t.Errorf("Title = %q, want %q", decoded.Title, original.Title)
	}
	if decoded.Points != original.Points {
		t.Errorf("Points = %d, want %d", decoded.Points, original.Points)
	}
}

func TestEvent_Decode_NilData(t *testing.T) {
	event, err := New("test", "ws_1", nil)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	var target map[string]any
	if err := event.Decode(&target); err != nil {
		t.Errorf("Decode(nil data) = %v, want nil", err)
	}
}
