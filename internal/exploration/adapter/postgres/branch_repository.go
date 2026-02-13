package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/exploration/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/exploration/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// BranchRepository implements domain.BranchRepository using Postgres via sqlc.
type BranchRepository struct {
	q *sqlc.Queries
}

// NewBranchRepository creates a new BranchRepository.
func NewBranchRepository(db database.DBTX) *BranchRepository {
	return &BranchRepository{q: sqlc.New(db)}
}

var _ domain.BranchRepository = (*BranchRepository)(nil)

func (r *BranchRepository) Create(ctx context.Context, branch domain.Branch) error {
	const op = "exploration: create branch"
	return database.MapQueryError(r.q.CreateBranch(ctx, branchToCreateParams(branch)), op)
}

func (r *BranchRepository) FindByID(ctx context.Context, id types.BranchID) (domain.Branch, error) {
	const op = "exploration: find branch by id"
	row, err := r.q.FindBranchByID(ctx, id.String())
	if err != nil {
		return domain.Branch{}, database.MapQueryError(err, op)
	}
	return branchToDomain(row), nil
}

func (r *BranchRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Branch, error) {
	const op = "exploration: find branches by workspace"
	rows, err := r.q.FindBranchesByWorkspace(ctx, workspaceID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return branchesToDomain(rows), nil
}

func (r *BranchRepository) FindBySeed(ctx context.Context, seedID types.SeedID) ([]domain.Branch, error) {
	const op = "exploration: find branches by seed"
	rows, err := r.q.FindBranchesBySeed(ctx, seedID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return branchesToDomain(rows), nil
}

func (r *BranchRepository) FindByAuthor(ctx context.Context, workspaceID types.WorkspaceID, authorID types.UserID) ([]domain.Branch, error) {
	const op = "exploration: find branches by author"
	rows, err := r.q.FindBranchesByAuthor(ctx, sqlc.FindBranchesByAuthorParams{
		WorkspaceID: workspaceID.String(),
		AuthorID:    authorID.String(),
	})
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return branchesToDomain(rows), nil
}

func (r *BranchRepository) Update(ctx context.Context, branch domain.Branch) error {
	const op = "exploration: update branch"
	tag, err := r.q.UpdateBranch(ctx, branchToUpdateParams(branch))
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}
