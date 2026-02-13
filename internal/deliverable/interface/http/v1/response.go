package v1

import (
	"github.com/0xsj/canopy-backend/internal/deliverable/domain"
	"github.com/0xsj/canopy-backend/pkg/types"
)

type DeliverableResponse struct {
	ID            string           `json:"id"`
	WorkspaceID   string           `json:"workspace_id"`
	Format        string           `json:"format"`
	Content       string           `json:"content"`
	SourceLeafIDs []string         `json:"source_leaf_ids"`
	Version       int              `json:"version"`
	CreatedAt     types.Timestamp  `json:"created_at"`
	UpdatedAt     *types.Timestamp `json:"updated_at,omitempty"`
}

func DeliverableFromDomain(d domain.Deliverable) DeliverableResponse {
	leafIDs := make([]string, len(d.SourceLeafIDs()))
	for i, id := range d.SourceLeafIDs() {
		leafIDs[i] = id.String()
	}
	ts := d.Timestamps()
	return DeliverableResponse{
		ID:            d.ID().String(),
		WorkspaceID:   d.WorkspaceID().String(),
		Format:        string(d.Format()),
		Content:       d.Content(),
		SourceLeafIDs: leafIDs,
		Version:       d.Version(),
		CreatedAt:     ts.CreatedAt,
		UpdatedAt:     ts.UpdatedAt,
	}
}

func DeliverablesFromDomain(deliverables []domain.Deliverable) []DeliverableResponse {
	out := make([]DeliverableResponse, len(deliverables))
	for i, d := range deliverables {
		out[i] = DeliverableFromDomain(d)
	}
	return out
}
