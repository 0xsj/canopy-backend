package v1

import (
	"github.com/0xsj/canopy-backend/internal/organization/domain"
	"github.com/0xsj/canopy-backend/pkg/types"
)

type OrgResponse struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Slug       string           `json:"slug"`
	IsPersonal bool             `json:"is_personal"`
	OwnerID    string           `json:"owner_id"`
	Settings   map[string]any   `json:"settings,omitempty"`
	CreatedAt  types.Timestamp  `json:"created_at"`
	UpdatedAt  *types.Timestamp `json:"updated_at,omitempty"`
}

func OrgFromDomain(o domain.Organization) OrgResponse {
	ts := o.Timestamps()
	return OrgResponse{
		ID:         o.ID().String(),
		Name:       o.Name(),
		Slug:       o.Slug(),
		IsPersonal: o.IsPersonal(),
		OwnerID:    o.OwnerID().String(),
		Settings:   o.Settings(),
		CreatedAt:  ts.CreatedAt,
		UpdatedAt:  ts.UpdatedAt,
	}
}

type OrgMemberResponse struct {
	OrgID    string          `json:"org_id"`
	UserID   string          `json:"user_id"`
	Role     string          `json:"role"`
	JoinedAt types.Timestamp `json:"joined_at"`
}

func OrgMemberFromDomain(m domain.OrgMember) OrgMemberResponse {
	return OrgMemberResponse{
		OrgID:    m.OrgID().String(),
		UserID:   m.UserID().String(),
		Role:     string(m.Role()),
		JoinedAt: m.JoinedAt(),
	}
}

func OrgMembersFromDomain(members []domain.OrgMember) []OrgMemberResponse {
	out := make([]OrgMemberResponse, len(members))
	for i, m := range members {
		out[i] = OrgMemberFromDomain(m)
	}
	return out
}

type TeamResponse struct {
	ID          string           `json:"id"`
	OrgID       string           `json:"org_id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	CreatedAt   types.Timestamp  `json:"created_at"`
	UpdatedAt   *types.Timestamp `json:"updated_at,omitempty"`
}

func TeamFromDomain(t domain.Team) TeamResponse {
	ts := t.Timestamps()
	return TeamResponse{
		ID:          t.ID().String(),
		OrgID:       t.OrgID().String(),
		Name:        t.Name(),
		Description: t.Description(),
		CreatedAt:   ts.CreatedAt,
		UpdatedAt:   ts.UpdatedAt,
	}
}

func TeamsFromDomain(teams []domain.Team) []TeamResponse {
	out := make([]TeamResponse, len(teams))
	for i, t := range teams {
		out[i] = TeamFromDomain(t)
	}
	return out
}

type TeamMemberResponse struct {
	TeamID   string          `json:"team_id"`
	UserID   string          `json:"user_id"`
	JoinedAt types.Timestamp `json:"joined_at"`
}

func TeamMemberFromDomain(m domain.TeamMember) TeamMemberResponse {
	return TeamMemberResponse{
		TeamID:   m.TeamID().String(),
		UserID:   m.UserID().String(),
		JoinedAt: m.JoinedAt(),
	}
}

func TeamMembersFromDomain(members []domain.TeamMember) []TeamMemberResponse {
	out := make([]TeamMemberResponse, len(members))
	for i, m := range members {
		out[i] = TeamMemberFromDomain(m)
	}
	return out
}
