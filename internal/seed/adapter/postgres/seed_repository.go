package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/seed/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/seed/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// SeedRepository implements domain.SeedRepository using Postgres via sqlc.
type SeedRepository struct {
	q *sqlc.Queries
}

// NewSeedRepository creates a new SeedRepository.
func NewSeedRepository(db database.DBTX) *SeedRepository {
	return &SeedRepository{q: sqlc.New(db)}
}

var _ domain.SeedRepository = (*SeedRepository)(nil)

func (r *SeedRepository) Create(ctx context.Context, seed domain.Seed) error {
	const op = "seed: create seed"
	params, err := seedToCreateParams(seed)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return database.MapQueryError(r.q.CreateSeed(ctx, params), op)
}

func (r *SeedRepository) FindByID(ctx context.Context, id types.SeedID) (domain.Seed, error) {
	const op = "seed: find seed by id"
	row, err := r.q.FindSeedByID(ctx, id.String())
	if err != nil {
		return domain.Seed{}, database.MapQueryError(err, op)
	}
	s, err := seedToDomain(row)
	if err != nil {
		return domain.Seed{}, database.MapQueryError(err, op)
	}
	return s, nil
}

func (r *SeedRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Seed, error) {
	const op = "seed: find seeds by workspace"
	rows, err := r.q.FindSeedsByWorkspace(ctx, workspaceID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	seeds, err := seedsToDomain(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return seeds, nil
}

func (r *SeedRepository) Update(ctx context.Context, seed domain.Seed) error {
	const op = "seed: update seed"
	params, err := seedToUpdateParams(seed)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	tag, err := r.q.UpdateSeed(ctx, params)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}
