package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/discussion/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/discussion/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// ThreadRepository implements domain.ThreadRepository using Postgres via sqlc.
type ThreadRepository struct {
	q *sqlc.Queries
}

// NewThreadRepository creates a new ThreadRepository.
func NewThreadRepository(db database.DBTX) *ThreadRepository {
	return &ThreadRepository{q: sqlc.New(db)}
}

var _ domain.ThreadRepository = (*ThreadRepository)(nil)

func (r *ThreadRepository) Save(ctx context.Context, thread domain.Thread) error {
	const op = "discussion: save thread"
	params, err := threadToSaveParams(thread)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return database.MapQueryError(r.q.SaveThread(ctx, params), op)
}

func (r *ThreadRepository) FindByLeaf(ctx context.Context, leafID types.LeafID) (domain.Thread, error) {
	const op = "discussion: find thread by leaf"
	row, err := r.q.FindThreadByLeaf(ctx, leafID.String())
	if err != nil {
		return domain.Thread{}, database.MapQueryError(err, op)
	}
	t, err := threadToDomain(row)
	if err != nil {
		return domain.Thread{}, database.MapQueryError(err, op)
	}
	return t, nil
}

func (r *ThreadRepository) Exists(ctx context.Context, leafID types.LeafID) (bool, error) {
	const op = "discussion: check thread exists"
	exists, err := r.q.ThreadExists(ctx, leafID.String())
	if err != nil {
		return false, database.MapQueryError(err, op)
	}
	return exists, nil
}
