package postgres

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/exploration/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// ConnectionRepository implements domain.ConnectionRepository using Postgres.
type ConnectionRepository struct {
	db database.DBTX
}

// NewConnectionRepository creates a new ConnectionRepository.
func NewConnectionRepository(db database.DBTX) *ConnectionRepository {
	return &ConnectionRepository{db: db}
}

var _ domain.ConnectionRepository = (*ConnectionRepository)(nil)

func (r *ConnectionRepository) Create(ctx context.Context, conn domain.Connection) error {
	const op = "exploration: create connection"
	const query = `
		INSERT INTO connections (id, workspace_id, author_id, leaf_ids, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.Exec(ctx, query,
		conn.ID().String(),
		conn.WorkspaceID().String(),
		conn.AuthorID().String(),
		database.StringsFromIDs(conn.LeafIDs()),
		conn.Timestamps().CreatedAt.Time(),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *ConnectionRepository) FindByID(ctx context.Context, id types.ConnectionID) (domain.Connection, error) {
	const op = "exploration: find connection by id"
	const query = `
		SELECT id, workspace_id, author_id, leaf_ids, created_at
		FROM connections WHERE id = $1`

	return scanConnection(r.db.QueryRow(ctx, query, id.String()), op)
}

func (r *ConnectionRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Connection, error) {
	const op = "exploration: find connections by workspace"
	const query = `
		SELECT id, workspace_id, author_id, leaf_ids, created_at
		FROM connections WHERE workspace_id = $1 ORDER BY created_at`

	rows, err := r.db.Query(ctx, query, workspaceID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var conns []domain.Connection
	for rows.Next() {
		c, err := scanConnection(rows, op)
		if err != nil {
			return nil, err
		}
		conns = append(conns, c)
	}
	return conns, rows.Err()
}

func (r *ConnectionRepository) FindByLeaf(ctx context.Context, leafID types.LeafID) ([]domain.Connection, error) {
	const op = "exploration: find connections by leaf"
	const query = `
		SELECT id, workspace_id, author_id, leaf_ids, created_at
		FROM connections WHERE $1 = ANY(leaf_ids) ORDER BY created_at`

	rows, err := r.db.Query(ctx, query, leafID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var conns []domain.Connection
	for rows.Next() {
		c, err := scanConnection(rows, op)
		if err != nil {
			return nil, err
		}
		conns = append(conns, c)
	}
	return conns, rows.Err()
}

// --- helpers ---

func scanConnection(row rowScanner, op string) (domain.Connection, error) {
	var (
		rawID        string
		rawWorkspace string
		rawAuthor    string
		leafIDStrs   []string
		createdAt    time.Time
	)

	if err := row.Scan(&rawID, &rawWorkspace, &rawAuthor, &leafIDStrs, &createdAt); err != nil {
		return domain.Connection{}, database.MapQueryError(err, op)
	}

	leafIDs := make([]types.LeafID, len(leafIDStrs))
	for i, s := range leafIDStrs {
		leafIDs[i] = types.LeafIDFrom(s)
	}

	return domain.ReconstructConnection(
		types.ConnectionIDFrom(rawID),
		types.WorkspaceIDFrom(rawWorkspace),
		types.UserIDFrom(rawAuthor),
		leafIDs,
		types.Timestamps{CreatedAt: types.TimestampFrom(createdAt)},
	), nil
}
