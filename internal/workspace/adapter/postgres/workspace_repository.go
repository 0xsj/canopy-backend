package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/workspace/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// WorkspaceRepository implements domain.WorkspaceRepository using Postgres via sqlc.
type WorkspaceRepository struct {
	q *sqlc.Queries
}

// NewWorkspaceRepository creates a new WorkspaceRepository.
func NewWorkspaceRepository(db database.DBTX) *WorkspaceRepository {
	return &WorkspaceRepository{q: sqlc.New(db)}
}

var _ domain.WorkspaceRepository = (*WorkspaceRepository)(nil)

func (r *WorkspaceRepository) Create(ctx context.Context, workspace domain.Workspace) error {
	const op = "workspace: create workspace"
	params, err := workspaceToCreateParams(workspace)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return database.MapQueryError(r.q.CreateWorkspace(ctx, params), op)
}

func (r *WorkspaceRepository) FindByID(ctx context.Context, id types.WorkspaceID) (domain.Workspace, error) {
	const op = "workspace: find workspace by id"
	row, err := r.q.FindWorkspaceByID(ctx, id.String())
	if err != nil {
		return domain.Workspace{}, database.MapQueryError(err, op)
	}
	w, err := workspaceToDomain(row)
	if err != nil {
		return domain.Workspace{}, database.MapQueryError(err, op)
	}
	return w, nil
}

func (r *WorkspaceRepository) FindByOrg(ctx context.Context, orgID types.OrgID) ([]domain.Workspace, error) {
	const op = "workspace: find workspaces by org"
	rows, err := r.q.FindWorkspacesByOrg(ctx, orgID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	workspaces, err := workspacesToDomain(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return workspaces, nil
}

func (r *WorkspaceRepository) FindByTeam(ctx context.Context, teamID types.TeamID) ([]domain.Workspace, error) {
	const op = "workspace: find workspaces by team"
	s := teamID.String()
	rows, err := r.q.FindWorkspacesByTeam(ctx, &s)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	workspaces, err := workspacesToDomain(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return workspaces, nil
}

func (r *WorkspaceRepository) Update(ctx context.Context, workspace domain.Workspace) error {
	const op = "workspace: update workspace"
	params, err := workspaceToUpdateParams(workspace)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	tag, err := r.q.UpdateWorkspace(ctx, params)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}
