package postgres

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/exploration/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// BranchRepository implements domain.BranchRepository using Postgres.
type BranchRepository struct {
	db database.DBTX
}

// NewBranchRepository creates a new BranchRepository.
func NewBranchRepository(db database.DBTX) *BranchRepository {
	return &BranchRepository{db: db}
}

var _ domain.BranchRepository = (*BranchRepository)(nil)

func (r *BranchRepository) Create(ctx context.Context, branch domain.Branch) error {
	const op = "exploration: create branch"
	const query = `
		INSERT INTO branches (id, workspace_id, seed_id, author_id, root_leaf_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.Exec(ctx, query,
		branch.ID().String(),
		branch.WorkspaceID().String(),
		branch.SeedID().String(),
		branch.AuthorID().String(),
		nullableString(branch.RootLeafID().String()),
		branch.Timestamps().CreatedAt.Time(),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *BranchRepository) FindByID(ctx context.Context, id types.BranchID) (domain.Branch, error) {
	const op = "exploration: find branch by id"
	const query = `
		SELECT id, workspace_id, seed_id, author_id, root_leaf_id, created_at
		FROM branches WHERE id = $1`

	return scanBranch(r.db.QueryRow(ctx, query, id.String()), op)
}

func (r *BranchRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Branch, error) {
	const op = "exploration: find branches by workspace"
	const query = `
		SELECT id, workspace_id, seed_id, author_id, root_leaf_id, created_at
		FROM branches WHERE workspace_id = $1 ORDER BY created_at`

	return r.queryBranches(ctx, query, op, workspaceID.String())
}

func (r *BranchRepository) FindBySeed(ctx context.Context, seedID types.SeedID) ([]domain.Branch, error) {
	const op = "exploration: find branches by seed"
	const query = `
		SELECT id, workspace_id, seed_id, author_id, root_leaf_id, created_at
		FROM branches WHERE seed_id = $1 ORDER BY created_at`

	return r.queryBranches(ctx, query, op, seedID.String())
}

func (r *BranchRepository) FindByAuthor(ctx context.Context, workspaceID types.WorkspaceID, authorID types.UserID) ([]domain.Branch, error) {
	const op = "exploration: find branches by author"
	const query = `
		SELECT id, workspace_id, seed_id, author_id, root_leaf_id, created_at
		FROM branches WHERE workspace_id = $1 AND author_id = $2 ORDER BY created_at`

	return r.queryBranches(ctx, query, op, workspaceID.String(), authorID.String())
}

func (r *BranchRepository) Update(ctx context.Context, branch domain.Branch) error {
	const op = "exploration: update branch"
	const query = `UPDATE branches SET root_leaf_id = $2 WHERE id = $1`

	tag, err := r.db.Exec(ctx, query,
		branch.ID().String(),
		nullableString(branch.RootLeafID().String()),
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

func (r *BranchRepository) queryBranches(ctx context.Context, query, op string, args ...any) ([]domain.Branch, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var branches []domain.Branch
	for rows.Next() {
		b, err := scanBranch(rows, op)
		if err != nil {
			return nil, err
		}
		branches = append(branches, b)
	}
	return branches, rows.Err()
}

func scanBranch(row rowScanner, op string) (domain.Branch, error) {
	var (
		rawID        string
		rawWorkspace string
		rawSeed      string
		rawAuthor    string
		rawRootLeaf  *string
		createdAt    time.Time
	)

	if err := row.Scan(&rawID, &rawWorkspace, &rawSeed, &rawAuthor, &rawRootLeaf, &createdAt); err != nil {
		return domain.Branch{}, database.MapQueryError(err, op)
	}

	return domain.ReconstructBranch(
		types.BranchIDFrom(rawID),
		types.WorkspaceIDFrom(rawWorkspace),
		types.SeedIDFrom(rawSeed),
		types.UserIDFrom(rawAuthor),
		types.LeafIDFrom(derefString(rawRootLeaf)),
		types.Timestamps{CreatedAt: types.TimestampFrom(createdAt)},
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
