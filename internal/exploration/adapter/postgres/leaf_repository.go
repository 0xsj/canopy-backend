package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/0xsj/canopy-backend/internal/exploration/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// LeafRepository implements domain.LeafRepository using Postgres.
type LeafRepository struct {
	db database.DBTX
}

// NewLeafRepository creates a new LeafRepository.
func NewLeafRepository(db database.DBTX) *LeafRepository {
	return &LeafRepository{db: db}
}

var _ domain.LeafRepository = (*LeafRepository)(nil)

func (r *LeafRepository) Create(ctx context.Context, leaf domain.Leaf) error {
	const op = "exploration: create leaf"
	const query = `
		INSERT INTO leaves (id, workspace_id, seed_id, branch_id, author_id, parent_leaf_id,
			title, summary, key_points, open_questions, tags, layer, sources, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	sourcesJSON, err := marshalSources(leaf.Sources())
	if err != nil {
		return database.MapQueryError(err, op)
	}

	metadataJSON, err := json.Marshal(leaf.Metadata())
	if err != nil {
		return database.MapQueryError(err, op)
	}

	_, err = r.db.Exec(ctx, query,
		leaf.ID().String(),
		leaf.WorkspaceID().String(),
		leaf.SeedID().String(),
		leaf.BranchID().String(),
		leaf.AuthorID().String(),
		nullableString(leaf.ParentLeafID().String()),
		leaf.Title(),
		leaf.Summary(),
		leaf.KeyPoints(),
		leaf.OpenQuestions(),
		leaf.Tags(),
		string(leaf.Layer()),
		sourcesJSON,
		metadataJSON,
		leaf.Timestamps().CreatedAt.Time(),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *LeafRepository) FindByID(ctx context.Context, id types.LeafID) (domain.Leaf, error) {
	const op = "exploration: find leaf by id"
	const query = leafSelectColumns + ` FROM leaves WHERE id = $1`

	return scanLeaf(r.db.QueryRow(ctx, query, id.String()), op)
}

func (r *LeafRepository) FindByIDs(ctx context.Context, ids []types.LeafID) ([]domain.Leaf, error) {
	const op = "exploration: find leaves by ids"
	const query = leafSelectColumns + ` FROM leaves WHERE id = ANY($1)`

	return r.queryLeaves(ctx, query, op, database.StringsFromIDs(ids))
}

func (r *LeafRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID, filter domain.LeafFilter) ([]domain.Leaf, error) {
	const op = "exploration: find leaves by workspace"

	where := []string{"workspace_id = $1"}
	args := []any{workspaceID.String()}
	idx := 2

	if filter.AuthorID != nil {
		where = append(where, fmt.Sprintf("author_id = $%d", idx))
		args = append(args, filter.AuthorID.String())
		idx++
	}
	if filter.SeedID != nil {
		where = append(where, fmt.Sprintf("seed_id = $%d", idx))
		args = append(args, filter.SeedID.String())
		idx++
	}
	if filter.Layer != nil {
		where = append(where, fmt.Sprintf("layer = $%d", idx))
		args = append(args, string(*filter.Layer))
		idx++
	}
	if len(filter.Tags) > 0 {
		where = append(where, fmt.Sprintf("tags && $%d", idx))
		args = append(args, filter.Tags)
		idx++
	}

	query := leafSelectColumns + " FROM leaves WHERE " + strings.Join(where, " AND ") + " ORDER BY created_at"
	return r.queryLeaves(ctx, query, op, args...)
}

func (r *LeafRepository) FindByBranch(ctx context.Context, branchID types.BranchID) ([]domain.Leaf, error) {
	const op = "exploration: find leaves by branch"
	const query = leafSelectColumns + ` FROM leaves WHERE branch_id = $1 ORDER BY created_at`

	return r.queryLeaves(ctx, query, op, branchID.String())
}

func (r *LeafRepository) FindBySeed(ctx context.Context, seedID types.SeedID) ([]domain.Leaf, error) {
	const op = "exploration: find leaves by seed"
	const query = leafSelectColumns + ` FROM leaves WHERE seed_id = $1 ORDER BY created_at`

	return r.queryLeaves(ctx, query, op, seedID.String())
}

func (r *LeafRepository) UpdateLayer(ctx context.Context, id types.LeafID, layer domain.Layer) error {
	const op = "exploration: update leaf layer"
	const query = `UPDATE leaves SET layer = $2 WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, id.String(), string(layer))
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}

// --- helpers ---

const leafSelectColumns = `SELECT id, workspace_id, seed_id, branch_id, author_id, parent_leaf_id,
	title, summary, key_points, open_questions, tags, layer, sources, metadata, created_at`

func (r *LeafRepository) queryLeaves(ctx context.Context, query, op string, args ...any) ([]domain.Leaf, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var leaves []domain.Leaf
	for rows.Next() {
		l, err := scanLeaf(rows, op)
		if err != nil {
			return nil, err
		}
		leaves = append(leaves, l)
	}
	return leaves, rows.Err()
}

func scanLeaf(row rowScanner, op string) (domain.Leaf, error) {
	var (
		rawID         string
		rawWorkspace  string
		rawSeed       string
		rawBranch     string
		rawAuthor     string
		rawParentLeaf *string
		title         string
		summary       string
		keyPoints     []string
		openQuestions []string
		tags          []string
		layer         string
		sourcesJSON   []byte
		metadataJSON  []byte
		createdAt     time.Time
	)

	if err := row.Scan(
		&rawID, &rawWorkspace, &rawSeed, &rawBranch, &rawAuthor, &rawParentLeaf,
		&title, &summary, &keyPoints, &openQuestions, &tags, &layer,
		&sourcesJSON, &metadataJSON, &createdAt,
	); err != nil {
		return domain.Leaf{}, database.MapQueryError(err, op)
	}

	sources, err := unmarshalSources(sourcesJSON)
	if err != nil {
		return domain.Leaf{}, database.MapQueryError(err, op)
	}

	metadata := make(map[string]any)
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
			return domain.Leaf{}, database.MapQueryError(err, op)
		}
	}

	return domain.ReconstructLeaf(
		types.LeafIDFrom(rawID),
		types.WorkspaceIDFrom(rawWorkspace),
		types.SeedIDFrom(rawSeed),
		types.BranchIDFrom(rawBranch),
		types.UserIDFrom(rawAuthor),
		types.LeafIDFrom(derefString(rawParentLeaf)),
		title,
		summary,
		keyPoints,
		openQuestions,
		tags,
		domain.Layer(layer),
		sources,
		metadata,
		types.Timestamps{CreatedAt: types.TimestampFrom(createdAt)},
	), nil
}

func marshalSources(sources []domain.Source) ([]byte, error) {
	if sources == nil {
		return nil, nil
	}
	return json.Marshal(sources)
}

func unmarshalSources(data []byte) ([]domain.Source, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var sources []domain.Source
	if err := json.Unmarshal(data, &sources); err != nil {
		return nil, err
	}
	return sources, nil
}
