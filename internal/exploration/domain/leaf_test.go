package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func TestNewLeaf_ValidInput(t *testing.T) {
	leaf, err := NewLeaf(
		types.NewWorkspaceID(),
		types.NewSeedID(),
		types.NewBranchID(),
		types.NewUserID(),
		types.LeafID{},
		"Distributed Systems",
		"An overview of distributed system patterns",
		[]string{"CAP theorem", "eventual consistency"},
		[]string{"How does this apply to real-time?"},
		[]string{"distributed", "architecture"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if leaf.ID().IsZero() {
		t.Fatal("expected non-zero ID")
	}
	if leaf.Layer() != LayerUnderstory {
		t.Errorf("layer = %q, want %q", leaf.Layer(), LayerUnderstory)
	}
	if leaf.IsSynthesis() {
		t.Error("expected non-synthesis leaf")
	}
	if !leaf.IsRoot() {
		t.Error("expected root leaf (no parent)")
	}
}

func TestNewLeaf_RejectsEmptyTitle(t *testing.T) {
	_, err := NewLeaf(
		types.NewWorkspaceID(), types.NewSeedID(), types.NewBranchID(),
		types.NewUserID(), types.LeafID{}, "", "summary",
		nil, nil, nil,
	)
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestNewLeaf_RejectsEmptySummary(t *testing.T) {
	_, err := NewLeaf(
		types.NewWorkspaceID(), types.NewSeedID(), types.NewBranchID(),
		types.NewUserID(), types.LeafID{}, "Title", "",
		nil, nil, nil,
	)
	if err == nil {
		t.Fatal("expected error for empty summary")
	}
}

func TestNewLeaf_IsImmutable(t *testing.T) {
	leaf, _ := NewLeaf(
		types.NewWorkspaceID(), types.NewSeedID(), types.NewBranchID(),
		types.NewUserID(), types.LeafID{}, "Title", "Summary",
		nil, nil, nil,
	)

	// Immutable timestamps — no UpdatedAt.
	if leaf.Timestamps().UpdatedAt != nil {
		t.Error("expected nil UpdatedAt for immutable leaf")
	}
}

func TestNewLeaf_WithParent(t *testing.T) {
	parentID := types.NewLeafID()
	leaf, _ := NewLeaf(
		types.NewWorkspaceID(), types.NewSeedID(), types.NewBranchID(),
		types.NewUserID(), parentID, "Child", "Summary",
		nil, nil, nil,
	)

	if leaf.IsRoot() {
		t.Error("expected non-root leaf with parent")
	}
	if leaf.ParentLeafID() != parentID {
		t.Errorf("parent ID = %v, want %v", leaf.ParentLeafID(), parentID)
	}
}

func TestNewSynthesisLeaf_RequiresMinTwoSources(t *testing.T) {
	ws := types.NewWorkspaceID()
	seed := types.NewSeedID()
	br := types.NewBranchID()
	author := types.NewUserID()

	_, err := NewSynthesisLeaf(ws, seed, br, author, "Title", "Summary",
		nil, nil, nil, []Source{{LeafID: "leaf_1", Title: "One"}})
	if err == nil {
		t.Fatal("expected error for < 2 sources")
	}

	leaf, err := NewSynthesisLeaf(ws, seed, br, author, "Title", "Summary",
		nil, nil, nil, []Source{
			{LeafID: "leaf_1", Title: "One"},
			{LeafID: "leaf_2", Title: "Two"},
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !leaf.IsSynthesis() {
		t.Error("expected synthesis leaf")
	}
	if len(leaf.Sources()) != 2 {
		t.Errorf("sources count = %d, want 2", len(leaf.Sources()))
	}
}
