package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// WorkspaceRepository implements domain.WorkspaceRepository using Postgres.
type WorkspaceRepository struct {
	db database.DBTX
}

// NewWorkspaceRepository creates a new WorkspaceRepository.
func NewWorkspaceRepository(db database.DBTX) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

var _ domain.WorkspaceRepository = (*WorkspaceRepository)(nil)

func (r *WorkspaceRepository) Create(ctx context.Context, ws domain.Workspace) error {
	const op = "workspace: create workspace"
	const query = `
		INSERT INTO workspaces (id, org_id, team_id, name, description, phase, lore_keeper_mode, lore_keeper_id, configuration, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	configJSON, err := json.Marshal(ws.Configuration())
	if err != nil {
		return database.MapQueryError(err, op)
	}

	ts := ws.Timestamps()
	_, err = r.db.Exec(ctx, query,
		ws.ID().String(),
		ws.OrgID().String(),
		nullableString(ws.TeamID().String()),
		ws.Name(),
		nullableString(ws.Description()),
		string(ws.Phase()),
		string(ws.LoreKeeperMode()),
		nullableString(ws.LoreKeeperID().String()),
		configJSON,
		ts.CreatedAt.Time(),
		tsUpdatedAt(ts),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *WorkspaceRepository) FindByID(ctx context.Context, id types.WorkspaceID) (domain.Workspace, error) {
	const op = "workspace: find workspace by id"
	const query = `
		SELECT id, org_id, team_id, name, description, phase, lore_keeper_mode, lore_keeper_id, configuration, created_at, updated_at
		FROM workspaces WHERE id = $1`

	return scanWorkspace(r.db.QueryRow(ctx, query, id.String()), op)
}

func (r *WorkspaceRepository) FindByOrg(ctx context.Context, orgID types.OrgID) ([]domain.Workspace, error) {
	const op = "workspace: find workspaces by org"
	const query = `
		SELECT id, org_id, team_id, name, description, phase, lore_keeper_mode, lore_keeper_id, configuration, created_at, updated_at
		FROM workspaces WHERE org_id = $1 ORDER BY created_at`

	return r.queryWorkspaces(ctx, query, op, orgID.String())
}

func (r *WorkspaceRepository) FindByTeam(ctx context.Context, teamID types.TeamID) ([]domain.Workspace, error) {
	const op = "workspace: find workspaces by team"
	const query = `
		SELECT id, org_id, team_id, name, description, phase, lore_keeper_mode, lore_keeper_id, configuration, created_at, updated_at
		FROM workspaces WHERE team_id = $1 ORDER BY created_at`

	return r.queryWorkspaces(ctx, query, op, teamID.String())
}

func (r *WorkspaceRepository) Update(ctx context.Context, ws domain.Workspace) error {
	const op = "workspace: update workspace"
	const query = `
		UPDATE workspaces
		SET name = $2, description = $3, phase = $4, lore_keeper_mode = $5,
			lore_keeper_id = $6, team_id = $7, configuration = $8, updated_at = $9
		WHERE id = $1`

	configJSON, err := json.Marshal(ws.Configuration())
	if err != nil {
		return database.MapQueryError(err, op)
	}

	ts := ws.Timestamps()
	tag, err := r.db.Exec(ctx, query,
		ws.ID().String(),
		ws.Name(),
		nullableString(ws.Description()),
		string(ws.Phase()),
		string(ws.LoreKeeperMode()),
		nullableString(ws.LoreKeeperID().String()),
		nullableString(ws.TeamID().String()),
		configJSON,
		tsUpdatedAt(ts),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}

// --- helpers ---

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *WorkspaceRepository) queryWorkspaces(ctx context.Context, query, op string, args ...any) ([]domain.Workspace, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var workspaces []domain.Workspace
	for rows.Next() {
		ws, err := scanWorkspace(rows, op)
		if err != nil {
			return nil, err
		}
		workspaces = append(workspaces, ws)
	}
	return workspaces, rows.Err()
}

func scanWorkspace(row rowScanner, op string) (domain.Workspace, error) {
	var (
		rawID          string
		rawOrgID       string
		rawTeamID      *string
		name           string
		description    *string
		phase          string
		loreKeeperMode string
		rawLoreKeeper  *string
		configJSON     []byte
		createdAt      time.Time
		updatedAt      time.Time
	)

	if err := row.Scan(&rawID, &rawOrgID, &rawTeamID, &name, &description, &phase, &loreKeeperMode, &rawLoreKeeper, &configJSON, &createdAt, &updatedAt); err != nil {
		return domain.Workspace{}, database.MapQueryError(err, op)
	}

	config := make(map[string]any)
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &config); err != nil {
			return domain.Workspace{}, database.MapQueryError(err, op)
		}
	}

	updatedTS := types.TimestampFrom(updatedAt)
	return domain.ReconstructWorkspace(
		types.WorkspaceIDFrom(rawID),
		types.OrgIDFrom(rawOrgID),
		types.TeamIDFrom(derefString(rawTeamID)),
		name,
		derefString(description),
		domain.Phase(phase),
		domain.LoreKeeperMode(loreKeeperMode),
		types.UserIDFrom(derefString(rawLoreKeeper)),
		config,
		types.Timestamps{
			CreatedAt: types.TimestampFrom(createdAt),
			UpdatedAt: &updatedTS,
		},
	), nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func tsUpdatedAt(ts types.Timestamps) time.Time {
	if ts.UpdatedAt != nil {
		return ts.UpdatedAt.Time()
	}
	return ts.CreatedAt.Time()
}
