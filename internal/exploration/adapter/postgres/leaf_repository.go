package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/exploration/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/exploration/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// LeafRepository implements domain.LeafRepository using Postgres via sqlc.
type LeafRepository struct {
	q *sqlc.Queries
}

// NewLeafRepository creates a new LeafRepository.
func NewLeafRepository(db database.DBTX) *LeafRepository {
	return &LeafRepository{q: sqlc.New(db)}
}

var _ domain.LeafRepository = (*LeafRepository)(nil)

func (r *LeafRepository) Create(ctx context.Context, leaf domain.Leaf) error {
	const op = "exploration: create leaf"
	params, err := leafToCreateParams(leaf)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return database.MapQueryError(r.q.CreateLeaf(ctx, params), op)
}

func (r *LeafRepository) FindByID(ctx context.Context, id types.LeafID) (domain.Leaf, error) {
	const op = "exploration: find leaf by id"
	row, err := r.q.FindLeafByID(ctx, id.String())
	if err != nil {
		return domain.Leaf{}, database.MapQueryError(err, op)
	}
	l, err := leafToDomain(row)
	if err != nil {
		return domain.Leaf{}, database.MapQueryError(err, op)
	}
	return l, nil
}

func (r *LeafRepository) FindByIDs(ctx context.Context, ids []types.LeafID) ([]domain.Leaf, error) {
	const op = "exploration: find leaves by ids"
	rows, err := r.q.FindLeavesByIDs(ctx, database.StringsFromIDs(ids))
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	leaves, err := leavesToDomain(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return leaves, nil
}

func (r *LeafRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID, filter domain.LeafFilter) ([]domain.Leaf, error) {
	const op = "exploration: find leaves by workspace"
	rows, err := r.q.FindLeavesByWorkspace(ctx, leafFilterToParams(workspaceID, filter))
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	leaves, err := leavesToDomain(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return leaves, nil
}

func (r *LeafRepository) FindByBranch(ctx context.Context, branchID types.BranchID) ([]domain.Leaf, error) {
	const op = "exploration: find leaves by branch"
	rows, err := r.q.FindLeavesByBranch(ctx, branchID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	leaves, err := leavesToDomain(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return leaves, nil
}

func (r *LeafRepository) FindBySeed(ctx context.Context, seedID types.SeedID) ([]domain.Leaf, error) {
	const op = "exploration: find leaves by seed"
	rows, err := r.q.FindLeavesBySeed(ctx, seedID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	leaves, err := leavesToDomain(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return leaves, nil
}

func (r *LeafRepository) UpdateLayer(ctx context.Context, id types.LeafID, layer domain.Layer) error {
	const op = "exploration: update leaf layer"
	tag, err := r.q.UpdateLeafLayer(ctx, sqlc.UpdateLeafLayerParams{
		ID:    id.String(),
		Layer: string(layer),
	})
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}

func (r *LeafRepository) Search(ctx context.Context, workspaceID types.WorkspaceID, query string) ([]domain.Leaf, error) {
	const op = "exploration: search leaves"
	rows, err := r.q.SearchLeavesByWorkspace(ctx, sqlc.SearchLeavesByWorkspaceParams{
		WorkspaceID:    workspaceID.String(),
		PlaintoTsquery: query,
	})
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	leaves, err := leavesToDomain(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return leaves, nil
}

func (r *LeafRepository) UpdatePosition(ctx context.Context, id types.LeafID, x, y float64) error {
	const op = "exploration: update leaf position"
	tag, err := r.q.UpdateLeafPosition(ctx, sqlc.UpdateLeafPositionParams{
		ID:        id.String(),
		PositionX: x,
		PositionY: y,
	})
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}
