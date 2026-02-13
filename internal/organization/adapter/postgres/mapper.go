package postgres

import (
	"encoding/json"

	"github.com/0xsj/canopy-backend/internal/organization/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/organization/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- org: domain → sqlc params ---

func orgToCreateParams(org domain.Organization) (sqlc.CreateOrgParams, error) {
	settingsJSON, err := json.Marshal(org.Settings())
	if err != nil {
		return sqlc.CreateOrgParams{}, err
	}
	ts := org.Timestamps()
	return sqlc.CreateOrgParams{
		ID:        org.ID().String(),
		Name:      org.Name(),
		Slug:      org.Slug(),
		Personal:  org.IsPersonal(),
		OwnerID:   org.OwnerID().String(),
		Settings:  settingsJSON,
		CreatedAt: ts.CreatedAt.Time(),
		UpdatedAt: database.TsUpdatedAt(ts),
	}, nil
}

func orgToUpdateParams(org domain.Organization) (sqlc.UpdateOrgParams, error) {
	settingsJSON, err := json.Marshal(org.Settings())
	if err != nil {
		return sqlc.UpdateOrgParams{}, err
	}
	ts := org.Timestamps()
	return sqlc.UpdateOrgParams{
		ID:        org.ID().String(),
		Name:      org.Name(),
		Slug:      org.Slug(),
		Settings:  settingsJSON,
		UpdatedAt: database.TsUpdatedAt(ts),
	}, nil
}

// --- org: sqlc model → domain ---

func orgToDomain(row sqlc.Organization) (domain.Organization, error) {
	settings := make(map[string]any)
	if len(row.Settings) > 0 {
		if err := json.Unmarshal(row.Settings, &settings); err != nil {
			return domain.Organization{}, err
		}
	}
	updatedTS := types.TimestampFrom(row.UpdatedAt)
	return domain.ReconstructOrganization(
		types.OrgIDFrom(row.ID),
		row.Name,
		row.Slug,
		row.Personal,
		types.UserIDFrom(row.OwnerID),
		settings,
		types.Timestamps{
			CreatedAt: types.TimestampFrom(row.CreatedAt),
			UpdatedAt: &updatedTS,
		},
	), nil
}

// --- org member: domain → sqlc params ---

func orgMemberToAddParams(m domain.OrgMember) sqlc.AddOrgMemberParams {
	return sqlc.AddOrgMemberParams{
		OrgID:    m.OrgID().String(),
		UserID:   m.UserID().String(),
		Role:     string(m.Role()),
		JoinedAt: m.JoinedAt().Time(),
	}
}

// --- org member: sqlc model → domain ---

func orgMemberToDomain(row sqlc.OrgMember) domain.OrgMember {
	return domain.ReconstructOrgMember(
		types.OrgIDFrom(row.OrgID),
		types.UserIDFrom(row.UserID),
		domain.Role(row.Role),
		types.TimestampFrom(row.JoinedAt),
	)
}

func orgMembersToDomain(rows []sqlc.OrgMember) []domain.OrgMember {
	members := make([]domain.OrgMember, len(rows))
	for i, row := range rows {
		members[i] = orgMemberToDomain(row)
	}
	return members
}

// --- team: domain → sqlc params ---

func teamToCreateParams(t domain.Team) sqlc.CreateTeamParams {
	ts := t.Timestamps()
	return sqlc.CreateTeamParams{
		ID:          t.ID().String(),
		OrgID:       t.OrgID().String(),
		Name:        t.Name(),
		Description: database.NullableString(t.Description()),
		CreatedAt:   ts.CreatedAt.Time(),
		UpdatedAt:   database.TsUpdatedAt(ts),
	}
}

func teamToUpdateParams(t domain.Team) sqlc.UpdateTeamParams {
	ts := t.Timestamps()
	return sqlc.UpdateTeamParams{
		ID:          t.ID().String(),
		Name:        t.Name(),
		Description: database.NullableString(t.Description()),
		UpdatedAt:   database.TsUpdatedAt(ts),
	}
}

// --- team: sqlc model → domain ---

func teamToDomain(row sqlc.Team) domain.Team {
	updatedTS := types.TimestampFrom(row.UpdatedAt)
	return domain.ReconstructTeam(
		types.TeamIDFrom(row.ID),
		types.OrgIDFrom(row.OrgID),
		row.Name,
		database.DerefString(row.Description),
		types.Timestamps{
			CreatedAt: types.TimestampFrom(row.CreatedAt),
			UpdatedAt: &updatedTS,
		},
	)
}

func teamsToDomain(rows []sqlc.Team) []domain.Team {
	teams := make([]domain.Team, len(rows))
	for i, row := range rows {
		teams[i] = teamToDomain(row)
	}
	return teams
}

// --- team member: domain → sqlc params ---

func teamMemberToAddParams(m domain.TeamMember) sqlc.AddTeamMemberParams {
	return sqlc.AddTeamMemberParams{
		TeamID:   m.TeamID().String(),
		UserID:   m.UserID().String(),
		JoinedAt: m.JoinedAt().Time(),
	}
}

// --- team member: sqlc model → domain ---

func teamMemberToDomain(row sqlc.TeamMember) domain.TeamMember {
	return domain.ReconstructTeamMember(
		types.TeamIDFrom(row.TeamID),
		types.UserIDFrom(row.UserID),
		types.TimestampFrom(row.JoinedAt),
	)
}

func teamMembersToDomain(rows []sqlc.TeamMember) []domain.TeamMember {
	members := make([]domain.TeamMember, len(rows))
	for i, row := range rows {
		members[i] = teamMemberToDomain(row)
	}
	return members
}
