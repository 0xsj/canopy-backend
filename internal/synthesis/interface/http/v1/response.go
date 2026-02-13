package v1

import (
	"github.com/0xsj/canopy-backend/internal/synthesis/domain"
	"github.com/0xsj/canopy-backend/pkg/types"
)

type SynthesisResponse struct {
	ID            string           `json:"id"`
	WorkspaceID   string           `json:"workspace_id"`
	InitiatorID   string           `json:"initiator_id"`
	SourceLeafIDs []string         `json:"source_leaf_ids"`
	ResultLeafID  string           `json:"result_leaf_id,omitempty"`
	Status        string           `json:"status"`
	FailureReason string           `json:"failure_reason,omitempty"`
	CreatedAt     types.Timestamp  `json:"created_at"`
	UpdatedAt     *types.Timestamp `json:"updated_at,omitempty"`
}

func SynthesisFromDomain(sw domain.SynthesisWorkflow) SynthesisResponse {
	leafIDs := make([]string, len(sw.SourceLeafIDs()))
	for i, id := range sw.SourceLeafIDs() {
		leafIDs[i] = id.String()
	}
	ts := sw.Timestamps()
	return SynthesisResponse{
		ID:            sw.ID().String(),
		WorkspaceID:   sw.WorkspaceID().String(),
		InitiatorID:   sw.InitiatorID().String(),
		SourceLeafIDs: leafIDs,
		ResultLeafID:  sw.ResultLeafID().String(),
		Status:        string(sw.Status()),
		FailureReason: sw.FailureReason(),
		CreatedAt:     ts.CreatedAt,
		UpdatedAt:     ts.UpdatedAt,
	}
}

func SynthesesFromDomain(workflows []domain.SynthesisWorkflow) []SynthesisResponse {
	out := make([]SynthesisResponse, len(workflows))
	for i, sw := range workflows {
		out[i] = SynthesisFromDomain(sw)
	}
	return out
}
