package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func TestNewConnection_ValidInput(t *testing.T) {
	leafIDs := []types.LeafID{types.NewLeafID(), types.NewLeafID()}
	conn, err := NewConnection(types.NewWorkspaceID(), types.NewUserID(), leafIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if conn.ID().IsZero() {
		t.Fatal("expected non-zero ID")
	}
	if len(conn.LeafIDs()) != 2 {
		t.Errorf("leaf IDs count = %d, want 2", len(conn.LeafIDs()))
	}
}

func TestNewConnection_RejectsFewerThanTwoLeaves(t *testing.T) {
	_, err := NewConnection(types.NewWorkspaceID(), types.NewUserID(), []types.LeafID{types.NewLeafID()})
	if err == nil {
		t.Fatal("expected error for < 2 leaves")
	}
}

func TestNewConnection_RejectsDuplicateLeaves(t *testing.T) {
	leafID := types.NewLeafID()
	_, err := NewConnection(types.NewWorkspaceID(), types.NewUserID(), []types.LeafID{leafID, leafID})
	if err == nil {
		t.Fatal("expected error for duplicate leaves")
	}
}

func TestNewConnection_RejectsZeroLeafID(t *testing.T) {
	_, err := NewConnection(types.NewWorkspaceID(), types.NewUserID(), []types.LeafID{types.NewLeafID(), {}})
	if err == nil {
		t.Fatal("expected error for zero leaf ID")
	}
}
