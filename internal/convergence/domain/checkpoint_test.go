package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func TestNewCheckpoint_ValidInput(t *testing.T) {
	cp, err := NewCheckpoint(types.NewWorkspaceID(), []types.LeafID{types.NewLeafID()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cp.Status() != CheckpointOpen {
		t.Errorf("status = %q, want %q", cp.Status(), CheckpointOpen)
	}
}

func TestNewCheckpoint_RejectsEmptyLeaves(t *testing.T) {
	_, err := NewCheckpoint(types.NewWorkspaceID(), nil)
	if err == nil {
		t.Fatal("expected error for empty leaf list")
	}
}

func TestCheckpoint_RecordSignal(t *testing.T) {
	cp, _ := NewCheckpoint(types.NewWorkspaceID(), []types.LeafID{types.NewLeafID()})
	sig, _ := NewConsensusSignal(cp.ID(), types.NewUserID(), PositionAlign, "")

	if err := cp.RecordSignal(sig); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cp.Signals()) != 1 {
		t.Errorf("signals count = %d, want 1", len(cp.Signals()))
	}
}

func TestCheckpoint_RejectsDuplicateSignal(t *testing.T) {
	cp, _ := NewCheckpoint(types.NewWorkspaceID(), []types.LeafID{types.NewLeafID()})
	userID := types.NewUserID()
	sig1, _ := NewConsensusSignal(cp.ID(), userID, PositionAlign, "")
	sig2, _ := NewConsensusSignal(cp.ID(), userID, PositionConcern, "changed mind")

	_ = cp.RecordSignal(sig1)
	err := cp.RecordSignal(sig2)
	if err == nil {
		t.Fatal("expected error for duplicate signal from same user")
	}
}

func TestCheckpoint_RejectsSignalAfterResolve(t *testing.T) {
	cp, _ := NewCheckpoint(types.NewWorkspaceID(), []types.LeafID{types.NewLeafID()})
	_ = cp.Resolve()

	sig, _ := NewConsensusSignal(cp.ID(), types.NewUserID(), PositionAlign, "")
	err := cp.RecordSignal(sig)
	if err == nil {
		t.Fatal("expected error for signal on resolved checkpoint")
	}
}

func TestCheckpoint_HasConsensus(t *testing.T) {
	cp, _ := NewCheckpoint(types.NewWorkspaceID(), []types.LeafID{types.NewLeafID()})

	// No signals — no consensus.
	if cp.HasConsensus() {
		t.Error("expected no consensus with zero signals")
	}

	// All aligned — consensus.
	sig1, _ := NewConsensusSignal(cp.ID(), types.NewUserID(), PositionAlign, "")
	sig2, _ := NewConsensusSignal(cp.ID(), types.NewUserID(), PositionAlign, "")
	_ = cp.RecordSignal(sig1)
	_ = cp.RecordSignal(sig2)

	if !cp.HasConsensus() {
		t.Error("expected consensus when all aligned")
	}
}

func TestCheckpoint_HasConsensus_FalseWithConcern(t *testing.T) {
	cp, _ := NewCheckpoint(types.NewWorkspaceID(), []types.LeafID{types.NewLeafID()})

	sig1, _ := NewConsensusSignal(cp.ID(), types.NewUserID(), PositionAlign, "")
	sig2, _ := NewConsensusSignal(cp.ID(), types.NewUserID(), PositionConcern, "not sure")
	_ = cp.RecordSignal(sig1)
	_ = cp.RecordSignal(sig2)

	if cp.HasConsensus() {
		t.Error("expected no consensus with a concern")
	}
}

func TestCheckpoint_Resolve_OnlyOnce(t *testing.T) {
	cp, _ := NewCheckpoint(types.NewWorkspaceID(), []types.LeafID{types.NewLeafID()})

	if err := cp.Resolve(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := cp.Resolve()
	if err == nil {
		t.Fatal("expected error for double resolve")
	}
}
