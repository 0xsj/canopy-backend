package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func TestNewOrgMember_ValidInput(t *testing.T) {
	orgID := types.NewOrgID()
	userID := types.NewUserID()
	m, err := NewOrgMember(orgID, userID, RoleAdmin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.OrgID() != orgID {
		t.Errorf("org ID = %v, want %v", m.OrgID(), orgID)
	}
	if m.Role() != RoleAdmin {
		t.Errorf("role = %q, want %q", m.Role(), RoleAdmin)
	}
}

func TestNewOrgMember_RejectsInvalidRole(t *testing.T) {
	_, err := NewOrgMember(types.NewOrgID(), types.NewUserID(), "superadmin")
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestChangeRole_ValidTransition(t *testing.T) {
	m, _ := NewOrgMember(types.NewOrgID(), types.NewUserID(), RoleMember)

	if err := m.ChangeRole(RoleAdmin); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Role() != RoleAdmin {
		t.Errorf("role = %q, want %q", m.Role(), RoleAdmin)
	}
}

func TestChangeRole_RejectsSameRole(t *testing.T) {
	m, _ := NewOrgMember(types.NewOrgID(), types.NewUserID(), RoleMember)

	err := m.ChangeRole(RoleMember)
	if err == nil {
		t.Fatal("expected error for same role")
	}
}

func TestRolePermissions(t *testing.T) {
	tests := []struct {
		role                Role
		canManageMembers    bool
		canCreateWorkspaces bool
	}{
		{RoleOwner, true, true},
		{RoleAdmin, true, true},
		{RoleMember, false, false},
	}

	for _, tt := range tests {
		if got := tt.role.CanManageMembers(); got != tt.canManageMembers {
			t.Errorf("%s.CanManageMembers() = %v, want %v", tt.role, got, tt.canManageMembers)
		}
		if got := tt.role.CanCreateWorkspaces(); got != tt.canCreateWorkspaces {
			t.Errorf("%s.CanCreateWorkspaces() = %v, want %v", tt.role, got, tt.canCreateWorkspaces)
		}
	}
}
