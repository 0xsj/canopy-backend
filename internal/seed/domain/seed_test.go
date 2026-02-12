package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func TestNewSeed_ValidInput(t *testing.T) {
	seed, err := NewSeed(types.NewWorkspaceID(), types.NewUserID(), "Distributed Systems", "Explore patterns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if seed.ID().IsZero() {
		t.Fatal("expected non-zero ID")
	}
	if seed.Title() != "Distributed Systems" {
		t.Errorf("title = %q, want %q", seed.Title(), "Distributed Systems")
	}
}

func TestNewSeed_RejectsEmptyTitle(t *testing.T) {
	_, err := NewSeed(types.NewWorkspaceID(), types.NewUserID(), "", "")
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestSeed_UpdateConstraints(t *testing.T) {
	seed, _ := NewSeed(types.NewWorkspaceID(), types.NewUserID(), "Topic", "")

	seed.UpdateConstraints(map[string]any{"max_depth": 3})
	if seed.Constraints()["max_depth"] != 3 {
		t.Errorf("constraints = %v, want max_depth=3", seed.Constraints())
	}
}

func TestSeed_UpdateTags(t *testing.T) {
	seed, _ := NewSeed(types.NewWorkspaceID(), types.NewUserID(), "Topic", "")

	seed.UpdateTags([]string{"architecture", "patterns"})
	if len(seed.Tags()) != 2 {
		t.Errorf("tags count = %d, want 2", len(seed.Tags()))
	}
}
