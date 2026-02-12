package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func TestNewWorkspaceMember_ValidInput(t *testing.T) {
	wsID := types.NewWorkspaceID()
	userID := types.NewUserID()
	m, err := NewWorkspaceMember(wsID, userID, RoleParticipant)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.WorkspaceID() != wsID {
		t.Errorf("workspace ID = %v, want %v", m.WorkspaceID(), wsID)
	}
	if m.Role() != RoleParticipant {
		t.Errorf("role = %q, want %q", m.Role(), RoleParticipant)
	}
}

func TestNewWorkspaceMember_RejectsInvalidRole(t *testing.T) {
	_, err := NewWorkspaceMember(types.NewWorkspaceID(), types.NewUserID(), "admin")
	if err == nil {
		t.Fatal("expected error for invalid workspace role")
	}
}

func TestWorkspaceMember_ChangeRole(t *testing.T) {
	m, _ := NewWorkspaceMember(types.NewWorkspaceID(), types.NewUserID(), RoleParticipant)

	if err := m.ChangeRole(RoleLoreKeeper); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Role() != RoleLoreKeeper {
		t.Errorf("role = %q, want %q", m.Role(), RoleLoreKeeper)
	}
}

func TestWorkspaceMember_ChangeRole_RejectsSame(t *testing.T) {
	m, _ := NewWorkspaceMember(types.NewWorkspaceID(), types.NewUserID(), RoleParticipant)

	err := m.ChangeRole(RoleParticipant)
	if err == nil {
		t.Fatal("expected error for same role")
	}
}
