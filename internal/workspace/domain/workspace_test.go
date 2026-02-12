package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func TestNewWorkspace_ValidInput(t *testing.T) {
	orgID := types.NewOrgID()
	ws, err := NewWorkspace(orgID, "Sprint Planning", "Weekly planning session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ws.ID().IsZero() {
		t.Fatal("expected non-zero ID")
	}
	if ws.Phase() != PhaseFloor {
		t.Errorf("phase = %q, want %q", ws.Phase(), PhaseFloor)
	}
	if ws.LoreKeeperMode() != LoreKeeperHuman {
		t.Errorf("lore keeper mode = %q, want %q", ws.LoreKeeperMode(), LoreKeeperHuman)
	}
}

func TestNewWorkspace_RejectsEmptyName(t *testing.T) {
	_, err := NewWorkspace(types.NewOrgID(), "", "")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestTransitionPhase_ForwardOnly(t *testing.T) {
	ws, _ := NewWorkspace(types.NewOrgID(), "Test", "")

	// Valid forward transitions.
	if err := ws.TransitionPhase(PhaseUnderstory); err != nil {
		t.Fatalf("floor → understory: %v", err)
	}
	if err := ws.TransitionPhase(PhaseCanopy); err != nil {
		t.Fatalf("understory → canopy: %v", err)
	}
	if err := ws.TransitionPhase(PhaseEmergent); err != nil {
		t.Fatalf("canopy → emergent: %v", err)
	}
}

func TestTransitionPhase_RejectsBackward(t *testing.T) {
	ws, _ := NewWorkspace(types.NewOrgID(), "Test", "")
	_ = ws.TransitionPhase(PhaseUnderstory)

	err := ws.TransitionPhase(PhaseFloor)
	if err == nil {
		t.Fatal("expected error for backward transition")
	}
}

func TestTransitionPhase_RejectsSkipping(t *testing.T) {
	ws, _ := NewWorkspace(types.NewOrgID(), "Test", "")

	err := ws.TransitionPhase(PhaseCanopy)
	if err == nil {
		t.Fatal("expected error for skipping a phase")
	}
}

func TestSetLoreKeeper_HumanRequiresUserID(t *testing.T) {
	ws, _ := NewWorkspace(types.NewOrgID(), "Test", "")

	err := ws.SetLoreKeeper(LoreKeeperHuman, types.UserID{})
	if err == nil {
		t.Fatal("expected error for human mode without user ID")
	}

	userID := types.NewUserID()
	if err := ws.SetLoreKeeper(LoreKeeperHuman, userID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.LoreKeeperID() != userID {
		t.Errorf("lore keeper ID = %v, want %v", ws.LoreKeeperID(), userID)
	}
}

func TestSetLoreKeeper_AIIgnoresUserID(t *testing.T) {
	ws, _ := NewWorkspace(types.NewOrgID(), "Test", "")

	if err := ws.SetLoreKeeper(LoreKeeperAI, types.NewUserID()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ws.LoreKeeperID().IsZero() {
		t.Error("expected zero lore keeper ID for AI mode")
	}
}

func TestAssignTeam_OnlyOnce(t *testing.T) {
	ws, _ := NewWorkspace(types.NewOrgID(), "Test", "")

	if err := ws.AssignTeam(types.NewTeamID()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := ws.AssignTeam(types.NewTeamID())
	if err == nil {
		t.Fatal("expected error for second team assignment")
	}
}
