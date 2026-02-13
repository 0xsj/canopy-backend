package v1

import (
	"github.com/0xsj/canopy-backend/internal/convergence/domain"
	"github.com/0xsj/canopy-backend/pkg/types"
)

type SignalResponse struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	LeafID      string          `json:"leaf_id"`
	UserID      string          `json:"user_id"`
	Type        string          `json:"type"`
	Annotation  string          `json:"annotation,omitempty"`
	CreatedAt   types.Timestamp `json:"created_at"`
}

func SignalFromDomain(s domain.Signal) SignalResponse {
	ts := s.Timestamps()
	return SignalResponse{
		ID:          s.ID().String(),
		WorkspaceID: s.WorkspaceID().String(),
		LeafID:      s.LeafID().String(),
		UserID:      s.UserID().String(),
		Type:        string(s.Type()),
		Annotation:  s.Annotation(),
		CreatedAt:   ts.CreatedAt,
	}
}

func SignalsFromDomain(signals []domain.Signal) []SignalResponse {
	out := make([]SignalResponse, len(signals))
	for i, s := range signals {
		out[i] = SignalFromDomain(s)
	}
	return out
}

type SignalCountsResponse struct {
	Counts map[string]int `json:"counts"`
}

type ConsensusSignalResponse struct {
	CheckpointID string          `json:"checkpoint_id"`
	UserID       string          `json:"user_id"`
	Position     string          `json:"position"`
	Explanation  string          `json:"explanation"`
	CreatedAt    types.Timestamp `json:"created_at"`
}

type CheckpointResponse struct {
	ID           string                    `json:"id"`
	WorkspaceID  string                    `json:"workspace_id"`
	LeafIDs      []string                  `json:"leaf_ids"`
	Status       string                    `json:"status"`
	Signals      []ConsensusSignalResponse `json:"signals,omitempty"`
	HasConsensus bool                      `json:"has_consensus"`
	CreatedAt    types.Timestamp           `json:"created_at"`
	UpdatedAt    *types.Timestamp          `json:"updated_at,omitempty"`
}

func CheckpointFromDomain(c domain.Checkpoint) CheckpointResponse {
	leafIDs := make([]string, len(c.LeafIDs()))
	for i, id := range c.LeafIDs() {
		leafIDs[i] = id.String()
	}

	signals := make([]ConsensusSignalResponse, len(c.Signals()))
	for i, s := range c.Signals() {
		signals[i] = ConsensusSignalResponse{
			CheckpointID: s.CheckpointID().String(),
			UserID:       s.UserID().String(),
			Position:     string(s.Position()),
			Explanation:  s.Explanation(),
			CreatedAt:    s.CreatedAt(),
		}
	}

	ts := c.Timestamps()
	return CheckpointResponse{
		ID:           c.ID().String(),
		WorkspaceID:  c.WorkspaceID().String(),
		LeafIDs:      leafIDs,
		Status:       string(c.Status()),
		Signals:      signals,
		HasConsensus: c.HasConsensus(),
		CreatedAt:    ts.CreatedAt,
		UpdatedAt:    ts.UpdatedAt,
	}
}
