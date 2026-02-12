package domain

import (
	"fmt"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// Role represents a member's role within an organization.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// ValidRoles is the set of recognized roles.
var ValidRoles = []Role{RoleOwner, RoleAdmin, RoleMember}

// IsValid reports whether the role is a recognized value.
func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember:
		return true
	}
	return false
}

// CanManageMembers reports whether this role can add/remove members.
func (r Role) CanManageMembers() bool {
	return r == RoleOwner || r == RoleAdmin
}

// CanCreateWorkspaces reports whether this role can create workspaces.
func (r Role) CanCreateWorkspaces() bool {
	return r == RoleOwner || r == RoleAdmin
}

// OrgMember represents a user's membership in an organization.
type OrgMember struct {
	orgID    types.OrgID
	userID   types.UserID
	role     Role
	joinedAt types.Timestamp
}

// NewOrgMember creates a new organization membership.
func NewOrgMember(orgID types.OrgID, userID types.UserID, role Role) (OrgMember, error) {
	if orgID.IsZero() {
		return OrgMember{}, fmt.Errorf("organization: org ID is required")
	}
	if userID.IsZero() {
		return OrgMember{}, fmt.Errorf("organization: user ID is required")
	}
	if !role.IsValid() {
		return OrgMember{}, fmt.Errorf("organization: invalid role %q", role)
	}

	return OrgMember{
		orgID:    orgID,
		userID:   userID,
		role:     role,
		joinedAt: types.Now(),
	}, nil
}

// ReconstructOrgMember builds an OrgMember from trusted data.
func ReconstructOrgMember(
	orgID types.OrgID,
	userID types.UserID,
	role Role,
	joinedAt types.Timestamp,
) OrgMember {
	return OrgMember{
		orgID:    orgID,
		userID:   userID,
		role:     role,
		joinedAt: joinedAt,
	}
}

// ChangeRole updates the member's role. Cannot change to the same role.
func (m *OrgMember) ChangeRole(newRole Role) error {
	if !newRole.IsValid() {
		return fmt.Errorf("organization: invalid role %q", newRole)
	}
	if newRole == m.role {
		return fmt.Errorf("organization: member already has role %q", newRole)
	}
	m.role = newRole
	return nil
}

func (m OrgMember) OrgID() types.OrgID        { return m.orgID }
func (m OrgMember) UserID() types.UserID      { return m.userID }
func (m OrgMember) Role() Role                { return m.role }
func (m OrgMember) JoinedAt() types.Timestamp { return m.joinedAt }
