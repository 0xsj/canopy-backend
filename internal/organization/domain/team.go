package domain

import (
	"fmt"
	"strings"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// Team is an optional functional grouping within an organization
// (e.g., engineering, design, product). Not every member needs to
// belong to a team. Teams enable bulk workspace invitations and
// team-scoped views.
type Team struct {
	id          types.TeamID
	orgID       types.OrgID
	name        string
	description string
	timestamps  types.Timestamps
}

// NewTeam creates a new team within an organization.
func NewTeam(orgID types.OrgID, name string) (Team, error) {
	if orgID.IsZero() {
		return Team{}, fmt.Errorf("organization: org ID is required for team")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return Team{}, fmt.Errorf("organization: team name is required")
	}

	return Team{
		id:         types.NewTeamID(),
		orgID:      orgID,
		name:       name,
		timestamps: types.NewMutableTimestamps(),
	}, nil
}

// ReconstructTeam builds a Team from trusted data.
func ReconstructTeam(
	id types.TeamID,
	orgID types.OrgID,
	name string,
	description string,
	timestamps types.Timestamps,
) Team {
	return Team{
		id:          id,
		orgID:       orgID,
		name:        name,
		description: description,
		timestamps:  timestamps,
	}
}

// UpdateDetails changes the team's name and/or description.
// Empty name is ignored (no change). Empty description clears it.
func (t *Team) UpdateDetails(name, description string) error {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	changed := false

	if name != "" && name != t.name {
		t.name = name
		changed = true
	}
	if description != t.description {
		t.description = description
		changed = true
	}

	if !changed {
		return fmt.Errorf("organization: no team changes")
	}

	t.timestamps.Touch()
	return nil
}

func (t Team) ID() types.TeamID             { return t.id }
func (t Team) OrgID() types.OrgID           { return t.orgID }
func (t Team) Name() string                 { return t.name }
func (t Team) Description() string          { return t.description }
func (t Team) Timestamps() types.Timestamps { return t.timestamps }

// TeamMember represents a user's membership in a team. Teams are optional —
// a user can be an org member without belonging to any team.
type TeamMember struct {
	teamID   types.TeamID
	userID   types.UserID
	joinedAt types.Timestamp
}

// NewTeamMember creates a new team membership.
func NewTeamMember(teamID types.TeamID, userID types.UserID) (TeamMember, error) {
	if teamID.IsZero() {
		return TeamMember{}, fmt.Errorf("organization: team ID is required")
	}
	if userID.IsZero() {
		return TeamMember{}, fmt.Errorf("organization: user ID is required for team member")
	}

	return TeamMember{
		teamID:   teamID,
		userID:   userID,
		joinedAt: types.Now(),
	}, nil
}

// ReconstructTeamMember builds a TeamMember from trusted data.
func ReconstructTeamMember(
	teamID types.TeamID,
	userID types.UserID,
	joinedAt types.Timestamp,
) TeamMember {
	return TeamMember{
		teamID:   teamID,
		userID:   userID,
		joinedAt: joinedAt,
	}
}

func (m TeamMember) TeamID() types.TeamID      { return m.teamID }
func (m TeamMember) UserID() types.UserID      { return m.userID }
func (m TeamMember) JoinedAt() types.Timestamp { return m.joinedAt }
