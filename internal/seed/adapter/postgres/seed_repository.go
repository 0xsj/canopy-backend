package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/0xsj/canopy-backend/internal/seed/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// SeedRepository implements domain.SeedRepository using Postgres.
type SeedRepository struct {
	db database.DBTX
}

// NewSeedRepository creates a new SeedRepository.
func NewSeedRepository(db database.DBTX) *SeedRepository {
	return &SeedRepository{db: db}
}

var _ domain.SeedRepository = (*SeedRepository)(nil)

func (r *SeedRepository) Create(ctx context.Context, seed domain.Seed) error {
	const op = "seed: create seed"
	const query = `
		INSERT INTO seeds (id, workspace_id, author_id, title, description, constraints, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	constraintsJSON, err := json.Marshal(seed.Constraints())
	if err != nil {
		return database.MapQueryError(err, op)
	}

	ts := seed.Timestamps()
	_, err = r.db.Exec(ctx, query,
		seed.ID().String(),
		seed.WorkspaceID().String(),
		seed.AuthorID().String(),
		seed.Title(),
		nullableString(seed.Description()),
		constraintsJSON,
		seed.Tags(),
		ts.CreatedAt.Time(),
		tsUpdatedAt(ts),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *SeedRepository) FindByID(ctx context.Context, id types.SeedID) (domain.Seed, error) {
	const op = "seed: find seed by id"
	const query = `
		SELECT id, workspace_id, author_id, title, description, constraints, tags, created_at, updated_at
		FROM seeds WHERE id = $1`

	return scanSeed(r.db.QueryRow(ctx, query, id.String()), op)
}

func (r *SeedRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Seed, error) {
	const op = "seed: find seeds by workspace"
	const query = `
		SELECT id, workspace_id, author_id, title, description, constraints, tags, created_at, updated_at
		FROM seeds WHERE workspace_id = $1 ORDER BY created_at`

	rows, err := r.db.Query(ctx, query, workspaceID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var seeds []domain.Seed
	for rows.Next() {
		s, err := scanSeed(rows, op)
		if err != nil {
			return nil, err
		}
		seeds = append(seeds, s)
	}
	return seeds, rows.Err()
}

func (r *SeedRepository) Update(ctx context.Context, seed domain.Seed) error {
	const op = "seed: update seed"
	const query = `
		UPDATE seeds
		SET constraints = $2, tags = $3, updated_at = $4
		WHERE id = $1`

	constraintsJSON, err := json.Marshal(seed.Constraints())
	if err != nil {
		return database.MapQueryError(err, op)
	}

	ts := seed.Timestamps()
	tag, err := r.db.Exec(ctx, query,
		seed.ID().String(),
		constraintsJSON,
		seed.Tags(),
		tsUpdatedAt(ts),
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

func scanSeed(row rowScanner, op string) (domain.Seed, error) {
	var (
		rawID           string
		rawWorkspaceID  string
		rawAuthorID     string
		title           string
		description     *string
		constraintsJSON []byte
		tags            []string
		createdAt       time.Time
		updatedAt       time.Time
	)

	if err := row.Scan(&rawID, &rawWorkspaceID, &rawAuthorID, &title, &description, &constraintsJSON, &tags, &createdAt, &updatedAt); err != nil {
		return domain.Seed{}, database.MapQueryError(err, op)
	}

	constraints := make(map[string]any)
	if len(constraintsJSON) > 0 {
		if err := json.Unmarshal(constraintsJSON, &constraints); err != nil {
			return domain.Seed{}, database.MapQueryError(err, op)
		}
	}

	updatedTS := types.TimestampFrom(updatedAt)
	return domain.ReconstructSeed(
		types.SeedIDFrom(rawID),
		types.WorkspaceIDFrom(rawWorkspaceID),
		types.UserIDFrom(rawAuthorID),
		title,
		derefString(description),
		constraints,
		tags,
		types.Timestamps{
			CreatedAt: types.TimestampFrom(createdAt),
			UpdatedAt: &updatedTS,
		},
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

func tsUpdatedAt(ts types.Timestamps) time.Time {
	if ts.UpdatedAt != nil {
		return ts.UpdatedAt.Time()
	}
	return ts.CreatedAt.Time()
}
