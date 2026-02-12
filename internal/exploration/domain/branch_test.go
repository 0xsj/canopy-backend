package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func TestNewBranch_ValidInput(t *testing.T) {
	branch, err := NewBranch(types.NewWorkspaceID(), types.NewSeedID(), types.NewUserID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if branch.ID().IsZero() {
		t.Fatal("expected non-zero ID")
	}
	if !branch.RootLeafID().IsZero() {
		t.Error("expected zero root leaf ID initially")
	}
}

func TestSetRootLeaf_OnlyOnce(t *testing.T) {
	branch, _ := NewBranch(types.NewWorkspaceID(), types.NewSeedID(), types.NewUserID())

	if err := branch.SetRootLeaf(types.NewLeafID()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := branch.SetRootLeaf(types.NewLeafID())
	if err == nil {
		t.Fatal("expected error for second root leaf assignment")
	}
}

func TestSetRootLeaf_RejectsZeroID(t *testing.T) {
	branch, _ := NewBranch(types.NewWorkspaceID(), types.NewSeedID(), types.NewUserID())

	err := branch.SetRootLeaf(types.LeafID{})
	if err == nil {
		t.Fatal("expected error for zero leaf ID")
	}
}
