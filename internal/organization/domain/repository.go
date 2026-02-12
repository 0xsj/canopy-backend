package domain

import (
	"context"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// OrgRepository defines the persistence port for organization entities.
type OrgRepository interface {
	// Create persists a new organization. Returns an error if the slug
	// already exists.
	Create(ctx context.Context, org Organization) error

	// FindByID returns an organization by ID.
	// Returns a NotFound error if no organization exists with that ID.
	FindByID(ctx context.Context, id types.OrgID) (Organization, error)

	// FindBySlug returns an organization by its unique slug.
	// Returns a NotFound error if no organization exists with that slug.
	FindBySlug(ctx context.Context, slug string) (Organization, error)

	// FindPersonalByOwner returns the personal organization for a user.
	// Returns a NotFound error if no personal org exists for that user.
	FindPersonalByOwner(ctx context.Context, ownerID types.UserID) (Organization, error)

	// Update persists changes to an existing organization.
	Update(ctx context.Context, org Organization) error
}

// MemberRepository defines the persistence port for organization membership.
type MemberRepository interface {
	// Add persists a new membership. Returns an error if the user is
	// already a member of the organization.
	Add(ctx context.Context, member OrgMember) error

	// Remove deletes a membership. Returns a NotFound error if the
	// membership does not exist.
	Remove(ctx context.Context, orgID types.OrgID, userID types.UserID) error

	// FindByOrg returns all members of an organization.
	FindByOrg(ctx context.Context, orgID types.OrgID) ([]OrgMember, error)

	// FindByUser returns all memberships for a user (supports multi-org).
	FindByUser(ctx context.Context, userID types.UserID) ([]OrgMember, error)

	// FindMember returns a single membership record.
	// Returns a NotFound error if the user is not a member of the org.
	FindMember(ctx context.Context, orgID types.OrgID, userID types.UserID) (OrgMember, error)

	// UpdateRole persists a role change for an existing membership.
	UpdateRole(ctx context.Context, member OrgMember) error

	// CountByRole returns the number of members with a given role in an org.
	// Useful for enforcing invariants like "at least one owner".
	CountByRole(ctx context.Context, orgID types.OrgID, role Role) (int, error)
}

// TeamRepository defines the persistence port for team entities and membership.
type TeamRepository interface {
	// Create persists a new team. Returns an error if a team with the
	// same name already exists in the organization.
	Create(ctx context.Context, team Team) error

	// FindByID returns a team by ID.
	FindByID(ctx context.Context, id types.TeamID) (Team, error)

	// FindByOrg returns all teams in an organization.
	FindByOrg(ctx context.Context, orgID types.OrgID) ([]Team, error)

	// Update persists changes to an existing team.
	Update(ctx context.Context, team Team) error

	// Delete removes a team. Returns a NotFound error if the team
	// does not exist.
	Delete(ctx context.Context, id types.TeamID) error

	// AddMember adds a user to a team. Returns an error if the user
	// is already a member of the team.
	AddMember(ctx context.Context, member TeamMember) error

	// RemoveMember removes a user from a team.
	RemoveMember(ctx context.Context, teamID types.TeamID, userID types.UserID) error

	// FindMembers returns all members of a team.
	FindMembers(ctx context.Context, teamID types.TeamID) ([]TeamMember, error)

	// FindTeamsByUser returns all teams a user belongs to within an org.
	FindTeamsByUser(ctx context.Context, orgID types.OrgID, userID types.UserID) ([]Team, error)
}
