package postgres

import (
	"encoding/json"

	"github.com/0xsj/canopy-backend/internal/workspace/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- workspace: domain → sqlc params ---

func workspaceToCreateParams(w domain.Workspace) (sqlc.CreateWorkspaceParams, error) {
	configJSON, err := json.Marshal(w.Configuration())
	if err != nil {
		return sqlc.CreateWorkspaceParams{}, err
	}
	ts := w.Timestamps()
	return sqlc.CreateWorkspaceParams{
		ID:             w.ID().String(),
		OrgID:          w.OrgID().String(),
		TeamID:         database.NullableString(w.TeamID().String()),
		Name:           w.Name(),
		Description:    database.NullableString(w.Description()),
		Phase:          string(w.Phase()),
		LoreKeeperMode: string(w.LoreKeeperMode()),
		LoreKeeperID:   database.NullableString(w.LoreKeeperID().String()),
		Configuration:  configJSON,
		CreatedAt:      ts.CreatedAt.Time(),
		UpdatedAt:      database.TsUpdatedAt(ts),
	}, nil
}

func workspaceToUpdateParams(w domain.Workspace) (sqlc.UpdateWorkspaceParams, error) {
	configJSON, err := json.Marshal(w.Configuration())
	if err != nil {
		return sqlc.UpdateWorkspaceParams{}, err
	}
	ts := w.Timestamps()
	return sqlc.UpdateWorkspaceParams{
		ID:             w.ID().String(),
		Name:           w.Name(),
		Description:    database.NullableString(w.Description()),
		Phase:          string(w.Phase()),
		LoreKeeperMode: string(w.LoreKeeperMode()),
		LoreKeeperID:   database.NullableString(w.LoreKeeperID().String()),
		Configuration:  configJSON,
		UpdatedAt:      database.TsUpdatedAt(ts),
	}, nil
}

// --- workspace: sqlc model → domain ---

func workspaceToDomain(row sqlc.Workspace) (domain.Workspace, error) {
	config := make(map[string]any)
	if len(row.Configuration) > 0 {
		if err := json.Unmarshal(row.Configuration, &config); err != nil {
			return domain.Workspace{}, err
		}
	}
	updatedTS := types.TimestampFrom(row.UpdatedAt)
	return domain.ReconstructWorkspace(
		types.WorkspaceIDFrom(row.ID),
		types.OrgIDFrom(row.OrgID),
		types.TeamIDFrom(database.DerefString(row.TeamID)),
		row.Name,
		database.DerefString(row.Description),
		domain.Phase(row.Phase),
		domain.LoreKeeperMode(row.LoreKeeperMode),
		types.UserIDFrom(database.DerefString(row.LoreKeeperID)),
		config,
		types.Timestamps{
			CreatedAt: types.TimestampFrom(row.CreatedAt),
			UpdatedAt: &updatedTS,
		},
	), nil
}

func workspacesToDomain(rows []sqlc.Workspace) ([]domain.Workspace, error) {
	workspaces := make([]domain.Workspace, len(rows))
	for i, row := range rows {
		w, err := workspaceToDomain(row)
		if err != nil {
			return nil, err
		}
		workspaces[i] = w
	}
	return workspaces, nil
}

// --- workspace member: domain → sqlc params ---

func memberToAddParams(m domain.WorkspaceMember) sqlc.AddWorkspaceMemberParams {
	return sqlc.AddWorkspaceMemberParams{
		WorkspaceID: m.WorkspaceID().String(),
		UserID:      m.UserID().String(),
		Role:        string(m.Role()),
		JoinedAt:    m.JoinedAt().Time(),
	}
}

// --- workspace member: sqlc model → domain ---

func memberToDomain(row sqlc.WorkspaceMember) domain.WorkspaceMember {
	return domain.ReconstructWorkspaceMember(
		types.WorkspaceIDFrom(row.WorkspaceID),
		types.UserIDFrom(row.UserID),
		domain.WorkspaceRole(row.Role),
		types.TimestampFrom(row.JoinedAt),
	)
}

func membersToDomain(rows []sqlc.WorkspaceMember) []domain.WorkspaceMember {
	members := make([]domain.WorkspaceMember, len(rows))
	for i, row := range rows {
		members[i] = memberToDomain(row)
	}
	return members
}
