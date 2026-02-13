package postgres

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/synthesis/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// SynthesisRepository implements domain.SynthesisRepository using Postgres.
type SynthesisRepository struct {
	db database.DBTX
}

// NewSynthesisRepository creates a new SynthesisRepository.
func NewSynthesisRepository(db database.DBTX) *SynthesisRepository {
	return &SynthesisRepository{db: db}
}

var _ domain.SynthesisRepository = (*SynthesisRepository)(nil)

func (r *SynthesisRepository) Create(ctx context.Context, workflow domain.SynthesisWorkflow) error {
	const op = "synthesis: create workflow"
	const query = `
		INSERT INTO workflows (id, workspace_id, initiator_id, source_leaf_ids, result_leaf_id,
			status, failure_reason, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	ts := workflow.Timestamps()
	_, err := r.db.Exec(ctx, query,
		workflow.ID().String(),
		workflow.WorkspaceID().String(),
		workflow.InitiatorID().String(),
		database.StringsFromIDs(workflow.SourceLeafIDs()),
		nullableString(workflow.ResultLeafID().String()),
		string(workflow.Status()),
		nullableString(workflow.FailureReason()),
		ts.CreatedAt.Time(),
		tsUpdatedAt(ts),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *SynthesisRepository) FindByID(ctx context.Context, id domain.SynthesisID) (domain.SynthesisWorkflow, error) {
	const op = "synthesis: find workflow by id"
	const query = workflowSelectColumns + ` FROM workflows WHERE id = $1`

	return scanWorkflow(r.db.QueryRow(ctx, query, id.String()), op)
}

func (r *SynthesisRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.SynthesisWorkflow, error) {
	const op = "synthesis: find workflows by workspace"
	const query = workflowSelectColumns + ` FROM workflows WHERE workspace_id = $1 ORDER BY created_at`

	rows, err := r.db.Query(ctx, query, workspaceID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var workflows []domain.SynthesisWorkflow
	for rows.Next() {
		w, err := scanWorkflow(rows, op)
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, w)
	}
	return workflows, rows.Err()
}

func (r *SynthesisRepository) Update(ctx context.Context, workflow domain.SynthesisWorkflow) error {
	const op = "synthesis: update workflow"
	const query = `
		UPDATE workflows
		SET status = $2, result_leaf_id = $3, failure_reason = $4, updated_at = $5
		WHERE id = $1`

	ts := workflow.Timestamps()
	tag, err := r.db.Exec(ctx, query,
		workflow.ID().String(),
		string(workflow.Status()),
		nullableString(workflow.ResultLeafID().String()),
		nullableString(workflow.FailureReason()),
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

const workflowSelectColumns = `SELECT id, workspace_id, initiator_id, source_leaf_ids,
	result_leaf_id, status, failure_reason, created_at, updated_at`

func scanWorkflow(row rowScanner, op string) (domain.SynthesisWorkflow, error) {
	var (
		rawID          string
		rawWorkspace   string
		rawInitiator   string
		sourceLeafStrs []string
		rawResultLeaf  *string
		status         string
		failureReason  *string
		createdAt      time.Time
		updatedAt      time.Time
	)

	if err := row.Scan(&rawID, &rawWorkspace, &rawInitiator, &sourceLeafStrs,
		&rawResultLeaf, &status, &failureReason, &createdAt, &updatedAt); err != nil {
		return domain.SynthesisWorkflow{}, database.MapQueryError(err, op)
	}

	sourceLeafIDs := make([]types.LeafID, len(sourceLeafStrs))
	for i, s := range sourceLeafStrs {
		sourceLeafIDs[i] = types.LeafIDFrom(s)
	}

	updatedTS := types.TimestampFrom(updatedAt)
	return domain.ReconstructSynthesisWorkflow(
		domain.SynthesisIDFrom(rawID),
		types.WorkspaceIDFrom(rawWorkspace),
		types.UserIDFrom(rawInitiator),
		sourceLeafIDs,
		types.LeafIDFrom(derefString(rawResultLeaf)),
		domain.WorkflowStatus(status),
		derefString(failureReason),
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
