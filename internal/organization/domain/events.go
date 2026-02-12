package domain

import "time"

// Event subjects published by the organization context.
const (
	SubjectOrgCreated        = "organization.org.created"
	SubjectMemberAdded       = "organization.member.added"
	SubjectMemberRemoved     = "organization.member.removed"
	SubjectMemberRoleChanged = "organization.member.role_changed"
	SubjectTeamCreated       = "organization.team.created"
	SubjectTeamMemberAdded   = "organization.team.member_added"
	SubjectTeamMemberRemoved = "organization.team.member_removed"
)

// OrgCreatedData is published when a new organization is created.
type OrgCreatedData struct {
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Personal  bool      `json:"personal"`
	OwnerID   string    `json:"owner_id"`
	Timestamp time.Time `json:"timestamp"`
}

// MemberAddedData is published when a user is added to an organization.
type MemberAddedData struct {
	OrgID     string    `json:"org_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	Timestamp time.Time `json:"timestamp"`
}

// MemberRemovedData is published when a user is removed from an organization.
type MemberRemovedData struct {
	OrgID     string    `json:"org_id"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}

// MemberRoleChangedData is published when a member's role changes.
type MemberRoleChangedData struct {
	OrgID        string    `json:"org_id"`
	UserID       string    `json:"user_id"`
	PreviousRole string    `json:"previous_role"`
	NewRole      string    `json:"new_role"`
	Timestamp    time.Time `json:"timestamp"`
}

// TeamCreatedData is published when a new team is created.
type TeamCreatedData struct {
	TeamID    string    `json:"team_id"`
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
}

// TeamMemberAddedData is published when a user is added to a team.
type TeamMemberAddedData struct {
	TeamID    string    `json:"team_id"`
	OrgID     string    `json:"org_id"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}

// TeamMemberRemovedData is published when a user is removed from a team.
type TeamMemberRemovedData struct {
	TeamID    string    `json:"team_id"`
	OrgID     string    `json:"org_id"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}
