package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/0xsj/canopy-backend/internal/convergence/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// CheckpointRepository implements domain.CheckpointRepository using Postgres.
type CheckpointRepository struct {
	db database.DBTX
}

// NewCheckpointRepository creates a new CheckpointRepository.
func NewCheckpointRepository(db database.DBTX) *CheckpointRepository {
	return &CheckpointRepository{db: db}
}

var _ domain.CheckpointRepository = (*CheckpointRepository)(nil)

func (r *CheckpointRepository) Create(ctx context.Context, checkpoint domain.Checkpoint) error {
	const op = "convergence: create checkpoint"
	const query = `
		INSERT INTO checkpoints (id, workspace_id, leaf_ids, status, signals, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	signalsJSON, err := marshalConsensusSignals(checkpoint.Signals())
	if err != nil {
		return database.MapQueryError(err, op)
	}

	ts := checkpoint.Timestamps()
	_, err = r.db.Exec(ctx, query,
		checkpoint.ID().String(),
		checkpoint.WorkspaceID().String(),
		database.StringsFromIDs(checkpoint.LeafIDs()),
		string(checkpoint.Status()),
		signalsJSON,
		ts.CreatedAt.Time(),
		tsUpdatedAt(ts),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *CheckpointRepository) FindByID(ctx context.Context, id types.CheckpointID) (domain.Checkpoint, error) {
	const op = "convergence: find checkpoint by id"
	const query = checkpointSelectColumns + ` FROM checkpoints WHERE id = $1`

	return scanCheckpoint(r.db.QueryRow(ctx, query, id.String()), op)
}

func (r *CheckpointRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Checkpoint, error) {
	const op = "convergence: find checkpoints by workspace"
	const query = checkpointSelectColumns + ` FROM checkpoints WHERE workspace_id = $1 ORDER BY created_at`

	return r.queryCheckpoints(ctx, query, op, workspaceID.String())
}

func (r *CheckpointRepository) FindOpenByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Checkpoint, error) {
	const op = "convergence: find open checkpoints by workspace"
	const query = checkpointSelectColumns + ` FROM checkpoints WHERE workspace_id = $1 AND status = 'open' ORDER BY created_at`

	return r.queryCheckpoints(ctx, query, op, workspaceID.String())
}

func (r *CheckpointRepository) Update(ctx context.Context, checkpoint domain.Checkpoint) error {
	const op = "convergence: update checkpoint"
	const query = `
		UPDATE checkpoints
		SET status = $2, signals = $3, updated_at = $4
		WHERE id = $1`

	signalsJSON, err := marshalConsensusSignals(checkpoint.Signals())
	if err != nil {
		return database.MapQueryError(err, op)
	}

	ts := checkpoint.Timestamps()
	tag, err := r.db.Exec(ctx, query,
		checkpoint.ID().String(),
		string(checkpoint.Status()),
		signalsJSON,
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

const checkpointSelectColumns = `SELECT id, workspace_id, leaf_ids, status, signals, created_at, updated_at`

func (r *CheckpointRepository) queryCheckpoints(ctx context.Context, query, op string, args ...any) ([]domain.Checkpoint, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var checkpoints []domain.Checkpoint
	for rows.Next() {
		c, err := scanCheckpoint(rows, op)
		if err != nil {
			return nil, err
		}
		checkpoints = append(checkpoints, c)
	}
	return checkpoints, rows.Err()
}

func scanCheckpoint(row rowScanner, op string) (domain.Checkpoint, error) {
	var (
		rawID        string
		rawWorkspace string
		leafIDStrs   []string
		status       string
		signalsJSON  []byte
		createdAt    time.Time
		updatedAt    time.Time
	)

	if err := row.Scan(&rawID, &rawWorkspace, &leafIDStrs, &status, &signalsJSON, &createdAt, &updatedAt); err != nil {
		return domain.Checkpoint{}, database.MapQueryError(err, op)
	}

	leafIDs := make([]types.LeafID, len(leafIDStrs))
	for i, s := range leafIDStrs {
		leafIDs[i] = types.LeafIDFrom(s)
	}

	signals, err := unmarshalConsensusSignals(signalsJSON)
	if err != nil {
		return domain.Checkpoint{}, database.MapQueryError(err, op)
	}

	updatedTS := types.TimestampFrom(updatedAt)
	return domain.ReconstructCheckpoint(
		types.CheckpointIDFrom(rawID),
		types.WorkspaceIDFrom(rawWorkspace),
		leafIDs,
		domain.CheckpointStatus(status),
		signals,
		types.Timestamps{
			CreatedAt: types.TimestampFrom(createdAt),
			UpdatedAt: &updatedTS,
		},
	), nil
}

// --- consensus signal JSONB DTO ---

type consensusSignalDTO struct {
	CheckpointID string    `json:"checkpoint_id"`
	UserID       string    `json:"user_id"`
	Position     string    `json:"position"`
	Explanation  string    `json:"explanation,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

func marshalConsensusSignals(signals []domain.ConsensusSignal) ([]byte, error) {
	if signals == nil {
		return []byte("[]"), nil
	}
	dtos := make([]consensusSignalDTO, len(signals))
	for i, s := range signals {
		dtos[i] = consensusSignalDTO{
			CheckpointID: s.CheckpointID().String(),
			UserID:       s.UserID().String(),
			Position:     string(s.Position()),
			Explanation:  s.Explanation(),
			CreatedAt:    s.CreatedAt().Time(),
		}
	}
	return json.Marshal(dtos)
}

func unmarshalConsensusSignals(data []byte) ([]domain.ConsensusSignal, error) {
	var dtos []consensusSignalDTO
	if err := json.Unmarshal(data, &dtos); err != nil {
		return nil, err
	}
	signals := make([]domain.ConsensusSignal, len(dtos))
	for i, dto := range dtos {
		signals[i] = domain.ReconstructConsensusSignal(
			types.CheckpointIDFrom(dto.CheckpointID),
			types.UserIDFrom(dto.UserID),
			domain.Position(dto.Position),
			dto.Explanation,
			types.TimestampFrom(dto.CreatedAt),
		)
	}
	return signals, nil
}

func tsUpdatedAt(ts types.Timestamps) time.Time {
	if ts.UpdatedAt != nil {
		return ts.UpdatedAt.Time()
	}
	return ts.CreatedAt.Time()
}
