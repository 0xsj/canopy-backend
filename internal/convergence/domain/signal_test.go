package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func TestNewSignal_ValidUpvote(t *testing.T) {
	sig, err := NewSignal(types.NewWorkspaceID(), types.NewLeafID(), types.NewUserID(), SignalUpvote, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sig.Type() != SignalUpvote {
		t.Errorf("type = %q, want %q", sig.Type(), SignalUpvote)
	}
}

func TestNewSignal_FlagRequiresAnnotation(t *testing.T) {
	_, err := NewSignal(types.NewWorkspaceID(), types.NewLeafID(), types.NewUserID(), SignalFlag, "")
	if err == nil {
		t.Fatal("expected error for flag without annotation")
	}

	sig, err := NewSignal(types.NewWorkspaceID(), types.NewLeafID(), types.NewUserID(), SignalFlag, "Needs clarification")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig.Annotation() != "Needs clarification" {
		t.Errorf("annotation = %q, want %q", sig.Annotation(), "Needs clarification")
	}
}

func TestNewSignal_RejectsInvalidType(t *testing.T) {
	_, err := NewSignal(types.NewWorkspaceID(), types.NewLeafID(), types.NewUserID(), "star", "")
	if err == nil {
		t.Fatal("expected error for invalid signal type")
	}
}
