package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func newTestSession(t *testing.T) *Session {
	t.Helper()
	s, err := NewSession(types.NewWorkspaceID(), types.NewUserID(), types.NewSeedID(), types.LeafID{}, nil, SessionExploration)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return &s
}

func TestNewSession_ValidInput(t *testing.T) {
	s := newTestSession(t)

	if s.Status() != StatusActive {
		t.Errorf("status = %q, want %q", s.Status(), StatusActive)
	}
	if s.Type() != SessionExploration {
		t.Errorf("type = %q, want %q", s.Type(), SessionExploration)
	}
	if len(s.Messages()) != 0 {
		t.Errorf("messages = %d, want 0", len(s.Messages()))
	}
}

func TestSession_AddMessage(t *testing.T) {
	s := newTestSession(t)

	if err := s.AddMessage(Message{Role: "user", Content: "Hello"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Messages()) != 1 {
		t.Errorf("messages = %d, want 1", len(s.Messages()))
	}
}

func TestSession_StateMachine_HappyPath(t *testing.T) {
	s := newTestSession(t)

	// active → checkpoint → completed
	if err := s.EnterCheckpoint(); err != nil {
		t.Fatalf("enter checkpoint: %v", err)
	}
	if s.Status() != StatusCheckpoint {
		t.Errorf("status = %q, want %q", s.Status(), StatusCheckpoint)
	}

	if err := s.Complete(); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if s.Status() != StatusCompleted {
		t.Errorf("status = %q, want %q", s.Status(), StatusCompleted)
	}
}

func TestSession_StateMachine_Abandon(t *testing.T) {
	s := newTestSession(t)

	if err := s.Abandon(); err != nil {
		t.Fatalf("abandon: %v", err)
	}
	if s.Status() != StatusAbandoned {
		t.Errorf("status = %q, want %q", s.Status(), StatusAbandoned)
	}
}

func TestSession_RejectsMessageAfterCompletion(t *testing.T) {
	s := newTestSession(t)
	_ = s.Complete()

	err := s.AddMessage(Message{Role: "user", Content: "Hello"})
	if err == nil {
		t.Fatal("expected error for message on completed session")
	}
}

func TestSession_RejectsCheckpointOnNonActive(t *testing.T) {
	s := newTestSession(t)
	_ = s.EnterCheckpoint()

	err := s.EnterCheckpoint()
	if err == nil {
		t.Fatal("expected error for checkpoint on non-active session")
	}
}

func TestSession_RejectsDoubleComplete(t *testing.T) {
	s := newTestSession(t)
	_ = s.Complete()

	err := s.Complete()
	if err == nil {
		t.Fatal("expected error for double complete")
	}
}

func TestSession_CanAddMessageDuringCheckpoint(t *testing.T) {
	s := newTestSession(t)
	_ = s.EnterCheckpoint()

	if err := s.AddMessage(Message{Role: "assistant", Content: "Shaping..."}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
