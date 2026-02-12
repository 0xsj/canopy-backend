package domain

import (
	"fmt"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// WorkspaceRole represents a member's role within a workspace.
// Workspace roles are separate from organization-level roles.
type WorkspaceRole string

const (
	RoleParticipant WorkspaceRole = "participant"
	RoleLoreKeeper  WorkspaceRole = "lore_keeper"
)

// IsValid reports whether the role is a recognized value.
func (r WorkspaceRole) IsValid() bool {
	switch r {
	case RoleParticipant, RoleLoreKeeper:
		return true
	}
	return false
}

// WorkspaceMember represents a user's membership in a workspace.
// Only members of the owning organization can be workspace members.
type WorkspaceMember struct {
	workspaceID types.WorkspaceID
	userID      types.UserID
	role        WorkspaceRole
	joinedAt    types.Timestamp
}

// NewWorkspaceMember creates a new workspace membership.
func NewWorkspaceMember(workspaceID types.WorkspaceID, userID types.UserID, role WorkspaceRole) (WorkspaceMember, error) {
	if workspaceID.IsZero() {
		return WorkspaceMember{}, fmt.Errorf("workspace: workspace ID is required")
	}
	if userID.IsZero() {
		return WorkspaceMember{}, fmt.Errorf("workspace: user ID is required")
	}
	if !role.IsValid() {
		return WorkspaceMember{}, fmt.Errorf("workspace: invalid role %q", role)
	}

	return WorkspaceMember{
		workspaceID: workspaceID,
		userID:      userID,
		role:        role,
		joinedAt:    types.Now(),
	}, nil
}

// ReconstructWorkspaceMember builds a WorkspaceMember from trusted data.
func ReconstructWorkspaceMember(
	workspaceID types.WorkspaceID,
	userID types.UserID,
	role WorkspaceRole,
	joinedAt types.Timestamp,
) WorkspaceMember {
	return WorkspaceMember{
		workspaceID: workspaceID,
		userID:      userID,
		role:        role,
		joinedAt:    joinedAt,
	}
}

// ChangeRole updates the member's workspace role.
func (m *WorkspaceMember) ChangeRole(newRole WorkspaceRole) error {
	if !newRole.IsValid() {
		return fmt.Errorf("workspace: invalid role %q", newRole)
	}
	if newRole == m.role {
		return fmt.Errorf("workspace: member already has role %q", newRole)
	}
	m.role = newRole
	return nil
}

func (m WorkspaceMember) WorkspaceID() types.WorkspaceID { return m.workspaceID }
func (m WorkspaceMember) UserID() types.UserID           { return m.userID }
func (m WorkspaceMember) Role() WorkspaceRole            { return m.role }
func (m WorkspaceMember) JoinedAt() types.Timestamp      { return m.joinedAt }
