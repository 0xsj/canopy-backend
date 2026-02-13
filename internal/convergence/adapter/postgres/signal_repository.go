package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/convergence/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/convergence/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// SignalRepository implements domain.SignalRepository using Postgres via sqlc.
type SignalRepository struct {
	q *sqlc.Queries
}

// NewSignalRepository creates a new SignalRepository.
func NewSignalRepository(db database.DBTX) *SignalRepository {
	return &SignalRepository{q: sqlc.New(db)}
}

var _ domain.SignalRepository = (*SignalRepository)(nil)

func (r *SignalRepository) Create(ctx context.Context, signal domain.Signal) error {
	const op = "convergence: create signal"
	return database.MapQueryError(r.q.CreateSignal(ctx, signalToCreateParams(signal)), op)
}

func (r *SignalRepository) Delete(ctx context.Context, id domain.SignalID) error {
	const op = "convergence: delete signal"
	tag, err := r.q.DeleteSignal(ctx, id.String())
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}

func (r *SignalRepository) FindByLeaf(ctx context.Context, leafID types.LeafID) ([]domain.Signal, error) {
	const op = "convergence: find signals by leaf"
	rows, err := r.q.FindSignalsByLeaf(ctx, leafID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return signalsToDomain(rows), nil
}

func (r *SignalRepository) FindByUser(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) ([]domain.Signal, error) {
	const op = "convergence: find signals by user"
	rows, err := r.q.FindSignalsByUser(ctx, sqlc.FindSignalsByUserParams{
		WorkspaceID: workspaceID.String(),
		UserID:      userID.String(),
	})
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return signalsToDomain(rows), nil
}

func (r *SignalRepository) FindByLeafAndUser(ctx context.Context, leafID types.LeafID, userID types.UserID) (domain.Signal, error) {
	const op = "convergence: find signal by leaf and user"
	row, err := r.q.FindSignalByLeafAndUser(ctx, sqlc.FindSignalByLeafAndUserParams{
		LeafID: leafID.String(),
		UserID: userID.String(),
	})
	if err != nil {
		return domain.Signal{}, database.MapQueryError(err, op)
	}
	return signalToDomain(row), nil
}

func (r *SignalRepository) CountByLeaf(ctx context.Context, leafID types.LeafID) (map[domain.SignalType]int, error) {
	const op = "convergence: count signals by leaf"
	rows, err := r.q.CountSignalsByLeaf(ctx, leafID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return countRowsToMap(rows), nil
}
