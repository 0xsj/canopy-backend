package postgres

import (
	"encoding/json"

	"github.com/0xsj/canopy-backend/internal/session/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/session/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- domain → sqlc params ---

func sessionToCreateParams(s domain.Session) (sqlc.CreateSessionParams, error) {
	messagesJSON, err := json.Marshal(s.Messages())
	if err != nil {
		return sqlc.CreateSessionParams{}, err
	}
	ts := s.Timestamps()
	return sqlc.CreateSessionParams{
		ID:           s.ID().String(),
		WorkspaceID:  s.WorkspaceID().String(),
		UserID:       s.UserID().String(),
		SeedID:       s.SeedID().String(),
		ParentLeafID: database.NullableString(s.ParentLeafID().String()),
		SessionType:  string(s.Type()),
		Status:       string(s.Status()),
		Messages:     messagesJSON,
		CreatedAt:    ts.CreatedAt.Time(),
		UpdatedAt:    database.TsUpdatedAt(ts),
	}, nil
}

func sessionToUpdateParams(s domain.Session) (sqlc.UpdateSessionParams, error) {
	messagesJSON, err := json.Marshal(s.Messages())
	if err != nil {
		return sqlc.UpdateSessionParams{}, err
	}
	ts := s.Timestamps()
	return sqlc.UpdateSessionParams{
		ID:        s.ID().String(),
		Status:    string(s.Status()),
		Messages:  messagesJSON,
		UpdatedAt: database.TsUpdatedAt(ts),
	}, nil
}

// --- sqlc model → domain ---

func sessionToDomain(row sqlc.Session) (domain.Session, error) {
	var messages []domain.Message
	if len(row.Messages) > 0 {
		if err := json.Unmarshal(row.Messages, &messages); err != nil {
			return domain.Session{}, err
		}
	}
	updatedTS := types.TimestampFrom(row.UpdatedAt)
	return domain.ReconstructSession(
		domain.SessionIDFrom(row.ID),
		types.WorkspaceIDFrom(row.WorkspaceID),
		types.UserIDFrom(row.UserID),
		types.SeedIDFrom(row.SeedID),
		types.LeafIDFrom(database.DerefString(row.ParentLeafID)),
		domain.SessionType(row.SessionType),
		domain.SessionStatus(row.Status),
		messages,
		types.Timestamps{
			CreatedAt: types.TimestampFrom(row.CreatedAt),
			UpdatedAt: &updatedTS,
		},
	), nil
}

func sessionsToDomain(rows []sqlc.Session) ([]domain.Session, error) {
	sessions := make([]domain.Session, len(rows))
	for i, row := range rows {
		s, err := sessionToDomain(row)
		if err != nil {
			return nil, err
		}
		sessions[i] = s
	}
	return sessions, nil
}
