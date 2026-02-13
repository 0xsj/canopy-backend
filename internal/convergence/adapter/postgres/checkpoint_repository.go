package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/convergence/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/convergence/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// CheckpointRepository implements domain.CheckpointRepository using Postgres via sqlc.
type CheckpointRepository struct {
	q *sqlc.Queries
}

// NewCheckpointRepository creates a new CheckpointRepository.
func NewCheckpointRepository(db database.DBTX) *CheckpointRepository {
	return &CheckpointRepository{q: sqlc.New(db)}
}

var _ domain.CheckpointRepository = (*CheckpointRepository)(nil)

func (r *CheckpointRepository) Create(ctx context.Context, checkpoint domain.Checkpoint) error {
	const op = "convergence: create checkpoint"
	params, err := checkpointToCreateParams(checkpoint)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return database.MapQueryError(r.q.CreateCheckpoint(ctx, params), op)
}

func (r *CheckpointRepository) FindByID(ctx context.Context, id types.CheckpointID) (domain.Checkpoint, error) {
	const op = "convergence: find checkpoint by id"
	row, err := r.q.FindCheckpointByID(ctx, id.String())
	if err != nil {
		return domain.Checkpoint{}, database.MapQueryError(err, op)
	}
	c, err := checkpointToDomain(row)
	if err != nil {
		return domain.Checkpoint{}, database.MapQueryError(err, op)
	}
	return c, nil
}

func (r *CheckpointRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Checkpoint, error) {
	const op = "convergence: find checkpoints by workspace"
	rows, err := r.q.FindCheckpointsByWorkspace(ctx, workspaceID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	checkpoints, err := checkpointsToDomain(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return checkpoints, nil
}

func (r *CheckpointRepository) FindOpenByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Checkpoint, error) {
	const op = "convergence: find open checkpoints by workspace"
	rows, err := r.q.FindOpenCheckpointsByWorkspace(ctx, workspaceID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	checkpoints, err := checkpointsToDomain(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return checkpoints, nil
}

func (r *CheckpointRepository) Update(ctx context.Context, checkpoint domain.Checkpoint) error {
	const op = "convergence: update checkpoint"
	params, err := checkpointToUpdateParams(checkpoint)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	tag, err := r.q.UpdateCheckpoint(ctx, params)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}
