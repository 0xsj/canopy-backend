package postgres

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/deliverable/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// DeliverableRepository implements domain.DeliverableRepository using Postgres.
type DeliverableRepository struct {
	db database.DBTX
}

// NewDeliverableRepository creates a new DeliverableRepository.
func NewDeliverableRepository(db database.DBTX) *DeliverableRepository {
	return &DeliverableRepository{db: db}
}

var _ domain.DeliverableRepository = (*DeliverableRepository)(nil)

func (r *DeliverableRepository) Create(ctx context.Context, del domain.Deliverable) error {
	const op = "deliverable: create deliverable"
	const query = `
		INSERT INTO deliverables (id, workspace_id, format, content, source_leaf_ids, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	ts := del.Timestamps()
	_, err := r.db.Exec(ctx, query,
		del.ID().String(),
		del.WorkspaceID().String(),
		string(del.Format()),
		del.Content(),
		database.StringsFromIDs(del.SourceLeafIDs()),
		del.Version(),
		ts.CreatedAt.Time(),
		tsUpdatedAt(ts),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *DeliverableRepository) FindByID(ctx context.Context, id types.DeliverableID) (domain.Deliverable, error) {
	const op = "deliverable: find deliverable by id"
	const query = deliverableSelectColumns + ` FROM deliverables WHERE id = $1`

	return scanDeliverable(r.db.QueryRow(ctx, query, id.String()), op)
}

func (r *DeliverableRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Deliverable, error) {
	const op = "deliverable: find deliverables by workspace"
	const query = deliverableSelectColumns + ` FROM deliverables WHERE workspace_id = $1 ORDER BY created_at`

	rows, err := r.db.Query(ctx, query, workspaceID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var deliverables []domain.Deliverable
	for rows.Next() {
		d, err := scanDeliverable(rows, op)
		if err != nil {
			return nil, err
		}
		deliverables = append(deliverables, d)
	}
	return deliverables, rows.Err()
}

func (r *DeliverableRepository) Update(ctx context.Context, del domain.Deliverable) error {
	const op = "deliverable: update deliverable"
	const query = `
		UPDATE deliverables
		SET format = $2, content = $3, source_leaf_ids = $4, version = $5, updated_at = $6
		WHERE id = $1`

	ts := del.Timestamps()
	tag, err := r.db.Exec(ctx, query,
		del.ID().String(),
		string(del.Format()),
		del.Content(),
		database.StringsFromIDs(del.SourceLeafIDs()),
		del.Version(),
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

const deliverableSelectColumns = `SELECT id, workspace_id, format, content, source_leaf_ids, version, created_at, updated_at`

func scanDeliverable(row rowScanner, op string) (domain.Deliverable, error) {
	var (
		rawID          string
		rawWorkspace   string
		format         string
		content        string
		sourceLeafStrs []string
		version        int
		createdAt      time.Time
		updatedAt      time.Time
	)

	if err := row.Scan(&rawID, &rawWorkspace, &format, &content, &sourceLeafStrs, &version, &createdAt, &updatedAt); err != nil {
		return domain.Deliverable{}, database.MapQueryError(err, op)
	}

	sourceLeafIDs := make([]types.LeafID, len(sourceLeafStrs))
	for i, s := range sourceLeafStrs {
		sourceLeafIDs[i] = types.LeafIDFrom(s)
	}

	updatedTS := types.TimestampFrom(updatedAt)
	return domain.ReconstructDeliverable(
		types.DeliverableIDFrom(rawID),
		types.WorkspaceIDFrom(rawWorkspace),
		domain.Format(format),
		content,
		sourceLeafIDs,
		version,
		types.Timestamps{
			CreatedAt: types.TimestampFrom(createdAt),
			UpdatedAt: &updatedTS,
		},
	), nil
}

func tsUpdatedAt(ts types.Timestamps) time.Time {
	if ts.UpdatedAt != nil {
		return ts.UpdatedAt.Time()
	}
	return ts.CreatedAt.Time()
}
