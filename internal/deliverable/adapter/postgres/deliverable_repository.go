package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/deliverable/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/deliverable/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// DeliverableRepository implements domain.DeliverableRepository using Postgres via sqlc.
type DeliverableRepository struct {
	q *sqlc.Queries
}

// NewDeliverableRepository creates a new DeliverableRepository.
func NewDeliverableRepository(db database.DBTX) *DeliverableRepository {
	return &DeliverableRepository{q: sqlc.New(db)}
}

var _ domain.DeliverableRepository = (*DeliverableRepository)(nil)

func (r *DeliverableRepository) Create(ctx context.Context, del domain.Deliverable) error {
	const op = "deliverable: create deliverable"
	return database.MapQueryError(r.q.CreateDeliverable(ctx, deliverableToCreateParams(del)), op)
}

func (r *DeliverableRepository) FindByID(ctx context.Context, id types.DeliverableID) (domain.Deliverable, error) {
	const op = "deliverable: find deliverable by id"
	row, err := r.q.FindDeliverableByID(ctx, id.String())
	if err != nil {
		return domain.Deliverable{}, database.MapQueryError(err, op)
	}
	return deliverableToDomain(row), nil
}

func (r *DeliverableRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Deliverable, error) {
	const op = "deliverable: find deliverables by workspace"
	rows, err := r.q.FindDeliverablesByWorkspace(ctx, workspaceID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return deliverablesToDomain(rows), nil
}

func (r *DeliverableRepository) Update(ctx context.Context, del domain.Deliverable) error {
	const op = "deliverable: update deliverable"
	tag, err := r.q.UpdateDeliverable(ctx, deliverableToUpdateParams(del))
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}
