package postgres

import (
	"encoding/json"
	"time"

	"github.com/0xsj/canopy-backend/internal/convergence/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/convergence/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- signal: domain → sqlc params ---

func signalToCreateParams(s domain.Signal) sqlc.CreateSignalParams {
	return sqlc.CreateSignalParams{
		ID:          s.ID().String(),
		WorkspaceID: s.WorkspaceID().String(),
		LeafID:      s.LeafID().String(),
		UserID:      s.UserID().String(),
		SignalType:  string(s.Type()),
		Annotation:  database.NullableString(s.Annotation()),
		CreatedAt:   s.Timestamps().CreatedAt.Time(),
	}
}

// --- signal: sqlc model → domain ---

func signalToDomain(row sqlc.Signal) domain.Signal {
	return domain.ReconstructSignal(
		domain.SignalIDFrom(row.ID),
		types.WorkspaceIDFrom(row.WorkspaceID),
		types.LeafIDFrom(row.LeafID),
		types.UserIDFrom(row.UserID),
		domain.SignalType(row.SignalType),
		database.DerefString(row.Annotation),
		types.Timestamps{CreatedAt: types.TimestampFrom(row.CreatedAt)},
	)
}

func signalsToDomain(rows []sqlc.Signal) []domain.Signal {
	signals := make([]domain.Signal, len(rows))
	for i, row := range rows {
		signals[i] = signalToDomain(row)
	}
	return signals
}

// --- signal: count rows → map ---

func countRowsToMap(rows []sqlc.CountSignalsByLeafRow) map[domain.SignalType]int {
	counts := make(map[domain.SignalType]int, len(rows))
	for _, row := range rows {
		counts[domain.SignalType(row.SignalType)] = int(row.Count)
	}
	return counts
}

// --- checkpoint: domain → sqlc params ---

func checkpointToCreateParams(c domain.Checkpoint) (sqlc.CreateCheckpointParams, error) {
	signalsJSON, err := marshalConsensusSignals(c.Signals())
	if err != nil {
		return sqlc.CreateCheckpointParams{}, err
	}
	ts := c.Timestamps()
	return sqlc.CreateCheckpointParams{
		ID:          c.ID().String(),
		WorkspaceID: c.WorkspaceID().String(),
		LeafIds:     leafIDsToStrings(c.LeafIDs()),
		Status:      string(c.Status()),
		Signals:     signalsJSON,
		CreatedAt:   ts.CreatedAt.Time(),
		UpdatedAt:   database.TsUpdatedAt(ts),
	}, nil
}

func checkpointToUpdateParams(c domain.Checkpoint) (sqlc.UpdateCheckpointParams, error) {
	signalsJSON, err := marshalConsensusSignals(c.Signals())
	if err != nil {
		return sqlc.UpdateCheckpointParams{}, err
	}
	ts := c.Timestamps()
	return sqlc.UpdateCheckpointParams{
		ID:        c.ID().String(),
		Status:    string(c.Status()),
		Signals:   signalsJSON,
		UpdatedAt: database.TsUpdatedAt(ts),
	}, nil
}

// --- checkpoint: sqlc model → domain ---

func checkpointToDomain(row sqlc.Checkpoint) (domain.Checkpoint, error) {
	signals, err := unmarshalConsensusSignals(row.Signals)
	if err != nil {
		return domain.Checkpoint{}, err
	}
	updatedTS := types.TimestampFrom(row.UpdatedAt)
	return domain.ReconstructCheckpoint(
		types.CheckpointIDFrom(row.ID),
		types.WorkspaceIDFrom(row.WorkspaceID),
		leafIDsFromStrings(row.LeafIds),
		domain.CheckpointStatus(row.Status),
		signals,
		types.Timestamps{
			CreatedAt: types.TimestampFrom(row.CreatedAt),
			UpdatedAt: &updatedTS,
		},
	), nil
}

func checkpointsToDomain(rows []sqlc.Checkpoint) ([]domain.Checkpoint, error) {
	checkpoints := make([]domain.Checkpoint, len(rows))
	for i, row := range rows {
		c, err := checkpointToDomain(row)
		if err != nil {
			return nil, err
		}
		checkpoints[i] = c
	}
	return checkpoints, nil
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

// --- leaf ID helpers ---

func leafIDsToStrings(ids []types.LeafID) []string {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = id.String()
	}
	return strs
}

func leafIDsFromStrings(strs []string) []types.LeafID {
	ids := make([]types.LeafID, len(strs))
	for i, s := range strs {
		ids[i] = types.LeafIDFrom(s)
	}
	return ids
}
