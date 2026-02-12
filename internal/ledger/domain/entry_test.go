package domain

import (
	"testing"
)

func TestNewSystemEntry_ValidInput(t *testing.T) {
	entry, err := NewSystemEntry(
		"identity.user.registered",
		map[string]any{"user_id": "usr_abc123"},
		"identity",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry.EventSubject() != "identity.user.registered" {
		t.Errorf("subject = %q, want %q", entry.EventSubject(), "identity.user.registered")
	}
	if entry.SourceContext() != "identity" {
		t.Errorf("source context = %q, want %q", entry.SourceContext(), "identity")
	}
}

func TestNewSystemEntry_RejectsEmptySubject(t *testing.T) {
	_, err := NewSystemEntry("", nil, "identity")
	if err == nil {
		t.Fatal("expected error for empty subject")
	}
}

func TestNewDomainEntry_ValidInput(t *testing.T) {
	entry, err := NewDomainEntry(
		"usr_abc123",
		ActionCreated,
		"leaf",
		"leaf_def456",
		"org_111",
		"ws_222",
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry.ActorID() != "usr_abc123" {
		t.Errorf("actor ID = %q, want %q", entry.ActorID(), "usr_abc123")
	}
	if entry.Action() != ActionCreated {
		t.Errorf("action = %q, want %q", entry.Action(), ActionCreated)
	}
	if entry.ResourceType() != "leaf" {
		t.Errorf("resource type = %q, want %q", entry.ResourceType(), "leaf")
	}
}

func TestNewDomainEntry_RejectsEmptyActor(t *testing.T) {
	_, err := NewDomainEntry("", ActionCreated, "leaf", "leaf_1", "", "", nil)
	if err == nil {
		t.Fatal("expected error for empty actor ID")
	}
}

func TestNewDomainEntry_RejectsEmptyResourceType(t *testing.T) {
	_, err := NewDomainEntry("usr_1", ActionCreated, "", "leaf_1", "", "", nil)
	if err == nil {
		t.Fatal("expected error for empty resource type")
	}
}
