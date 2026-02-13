package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/exploration/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/exploration/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// ConnectionRepository implements domain.ConnectionRepository using Postgres via sqlc.
type ConnectionRepository struct {
	q *sqlc.Queries
}

// NewConnectionRepository creates a new ConnectionRepository.
func NewConnectionRepository(db database.DBTX) *ConnectionRepository {
	return &ConnectionRepository{q: sqlc.New(db)}
}

var _ domain.ConnectionRepository = (*ConnectionRepository)(nil)

func (r *ConnectionRepository) Create(ctx context.Context, conn domain.Connection) error {
	const op = "exploration: create connection"
	return database.MapQueryError(r.q.CreateConnection(ctx, connectionToCreateParams(conn)), op)
}

func (r *ConnectionRepository) FindByID(ctx context.Context, id types.ConnectionID) (domain.Connection, error) {
	const op = "exploration: find connection by id"
	row, err := r.q.FindConnectionByID(ctx, id.String())
	if err != nil {
		return domain.Connection{}, database.MapQueryError(err, op)
	}
	return connectionToDomain(row), nil
}

func (r *ConnectionRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Connection, error) {
	const op = "exploration: find connections by workspace"
	rows, err := r.q.FindConnectionsByWorkspace(ctx, workspaceID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return connectionsToDomain(rows), nil
}

func (r *ConnectionRepository) FindByLeaf(ctx context.Context, leafID types.LeafID) ([]domain.Connection, error) {
	const op = "exploration: find connections by leaf"
	rows, err := r.q.FindConnectionsByLeaf(ctx, leafID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return connectionsToDomain(rows), nil
}
