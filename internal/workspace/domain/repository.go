package domain

import (
	"context"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// WorkspaceRepository defines the persistence port for workspace entities.
type WorkspaceRepository interface {
	// Create persists a new workspace.
	Create(ctx context.Context, workspace Workspace) error

	// FindByID returns a workspace by ID.
	// Returns a NotFound error if no workspace exists with that ID.
	FindByID(ctx context.Context, id types.WorkspaceID) (Workspace, error)

	// FindByOrg returns all workspaces in an organization.
	FindByOrg(ctx context.Context, orgID types.OrgID) ([]Workspace, error)

	// FindByTeam returns all workspaces scoped to a team.
	FindByTeam(ctx context.Context, teamID types.TeamID) ([]Workspace, error)

	// Update persists changes to an existing workspace.
	Update(ctx context.Context, workspace Workspace) error
}

// WorkspaceMemberRepository defines the persistence port for workspace membership.
type WorkspaceMemberRepository interface {
	// Add persists a new membership. Returns an error if the user is
	// already a member of the workspace.
	Add(ctx context.Context, member WorkspaceMember) error

	// Remove deletes a membership. Returns a NotFound error if the
	// membership does not exist.
	Remove(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) error

	// FindByWorkspace returns all members of a workspace.
	FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]WorkspaceMember, error)

	// FindByUser returns all workspace memberships for a user within an org.
	FindByUser(ctx context.Context, orgID types.OrgID, userID types.UserID) ([]WorkspaceMember, error)

	// FindMember returns a single membership record.
	// Returns a NotFound error if the user is not a member of the workspace.
	FindMember(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) (WorkspaceMember, error)

	// UpdateRole persists a role change for an existing membership.
	UpdateRole(ctx context.Context, member WorkspaceMember) error
}

// LLMConfigRepository defines the persistence port for workspace LLM configurations.
type LLMConfigRepository interface {
	// Upsert creates or replaces the LLM config for a workspace.
	Upsert(ctx context.Context, cfg LLMConfig) error

	// FindByWorkspace returns the LLM config for a workspace.
	// Returns a NotFound error if the workspace has no custom config.
	FindByWorkspace(ctx context.Context, wsID types.WorkspaceID) (LLMConfig, error)

	// Delete removes the LLM config for a workspace.
	// Returns a NotFound error if no config exists.
	Delete(ctx context.Context, wsID types.WorkspaceID) error
}
