package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/synthesis/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/synthesis/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// SynthesisRepository implements domain.SynthesisRepository using Postgres via sqlc.
type SynthesisRepository struct {
	q *sqlc.Queries
}

// NewSynthesisRepository creates a new SynthesisRepository.
func NewSynthesisRepository(db database.DBTX) *SynthesisRepository {
	return &SynthesisRepository{q: sqlc.New(db)}
}

var _ domain.SynthesisRepository = (*SynthesisRepository)(nil)

func (r *SynthesisRepository) Create(ctx context.Context, workflow domain.SynthesisWorkflow) error {
	const op = "synthesis: create workflow"
	return database.MapQueryError(r.q.CreateWorkflow(ctx, workflowToCreateParams(workflow)), op)
}

func (r *SynthesisRepository) FindByID(ctx context.Context, id domain.SynthesisID) (domain.SynthesisWorkflow, error) {
	const op = "synthesis: find workflow by id"
	row, err := r.q.FindWorkflowByID(ctx, id.String())
	if err != nil {
		return domain.SynthesisWorkflow{}, database.MapQueryError(err, op)
	}
	return workflowToDomain(row), nil
}

func (r *SynthesisRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.SynthesisWorkflow, error) {
	const op = "synthesis: find workflows by workspace"
	rows, err := r.q.FindWorkflowsByWorkspace(ctx, workspaceID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return workflowsToDomain(rows), nil
}

func (r *SynthesisRepository) Update(ctx context.Context, workflow domain.SynthesisWorkflow) error {
	const op = "synthesis: update workflow"
	tag, err := r.q.UpdateWorkflow(ctx, workflowToUpdateParams(workflow))
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}
