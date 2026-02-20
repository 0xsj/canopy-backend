package v1

import (
	"github.com/0xsj/canopy-backend/internal/session/domain"
	"github.com/0xsj/canopy-backend/pkg/types"
)

type SessionResponse struct {
	ID            string           `json:"id"`
	WorkspaceID   string           `json:"workspace_id"`
	UserID        string           `json:"user_id"`
	SeedID        string           `json:"seed_id"`
	ParentLeafID  string           `json:"parent_leaf_id,omitempty"`
	SourceLeafIDs []string         `json:"source_leaf_ids,omitempty"`
	Type          string           `json:"type"`
	Status        string           `json:"status"`
	Messages      []domain.Message `json:"messages"`
	CreatedAt     types.Timestamp  `json:"created_at"`
	UpdatedAt     *types.Timestamp `json:"updated_at,omitempty"`
}

func SessionsFromDomain(sessions []domain.Session) []SessionResponse {
	out := make([]SessionResponse, len(sessions))
	for i, s := range sessions {
		out[i] = SessionFromDomain(s)
	}
	return out
}

type StreamingResponse struct {
	StreamID  string `json:"stream_id"`
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
}

func SessionFromDomain(s domain.Session) SessionResponse {
	ts := s.Timestamps()

	var sourceIDs []string
	if ids := s.SourceLeafIDs(); len(ids) > 0 {
		sourceIDs = make([]string, len(ids))
		for i, id := range ids {
			sourceIDs[i] = id.String()
		}
	}

	return SessionResponse{
		ID:            s.ID().String(),
		WorkspaceID:   s.WorkspaceID().String(),
		UserID:        s.UserID().String(),
		SeedID:        s.SeedID().String(),
		ParentLeafID:  s.ParentLeafID().String(),
		SourceLeafIDs: sourceIDs,
		Type:          string(s.Type()),
		Status:        string(s.Status()),
		Messages:      s.Messages(),
		CreatedAt:     ts.CreatedAt,
		UpdatedAt:     ts.UpdatedAt,
	}
}
