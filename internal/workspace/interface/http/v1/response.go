package v1

import (
	"github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/types"
)

type WorkspaceResponse struct {
	ID             string           `json:"id"`
	OrgID          string           `json:"org_id"`
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	Phase          string           `json:"phase"`
	LoreKeeperMode string           `json:"lore_keeper_mode"`
	LoreKeeperID   string           `json:"lore_keeper_id,omitempty"`
	Configuration  map[string]any   `json:"configuration,omitempty"`
	CreatedAt      types.Timestamp  `json:"created_at"`
	UpdatedAt      *types.Timestamp `json:"updated_at,omitempty"`
}

func WorkspaceFromDomain(ws domain.Workspace) WorkspaceResponse {
	ts := ws.Timestamps()
	return WorkspaceResponse{
		ID:             ws.ID().String(),
		OrgID:          ws.OrgID().String(),
		Name:           ws.Name(),
		Description:    ws.Description(),
		Phase:          string(ws.Phase()),
		LoreKeeperMode: string(ws.LoreKeeperMode()),
		LoreKeeperID:   ws.LoreKeeperID().String(),
		Configuration:  ws.Configuration(),
		CreatedAt:      ts.CreatedAt,
		UpdatedAt:      ts.UpdatedAt,
	}
}

func WorkspacesFromDomain(workspaces []domain.Workspace) []WorkspaceResponse {
	out := make([]WorkspaceResponse, len(workspaces))
	for i, ws := range workspaces {
		out[i] = WorkspaceFromDomain(ws)
	}
	return out
}
