package v1

import (
	"github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/internal/workspace/service"
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

type WorkspaceMemberResponse struct {
	WorkspaceID string          `json:"workspace_id"`
	UserID      string          `json:"user_id"`
	Role        string          `json:"role"`
	JoinedAt    types.Timestamp `json:"joined_at"`
}

func MemberFromDomain(m domain.WorkspaceMember) WorkspaceMemberResponse {
	return WorkspaceMemberResponse{
		WorkspaceID: m.WorkspaceID().String(),
		UserID:      m.UserID().String(),
		Role:        string(m.Role()),
		JoinedAt:    m.JoinedAt(),
	}
}

func MembersFromDomain(members []domain.WorkspaceMember) []WorkspaceMemberResponse {
	out := make([]WorkspaceMemberResponse, len(members))
	for i, m := range members {
		out[i] = MemberFromDomain(m)
	}
	return out
}

type LLMConfigResponse struct {
	Provider  string           `json:"provider"`
	Model     string           `json:"model"`
	APIKey    string           `json:"api_key"`
	CreatedAt types.Timestamp  `json:"created_at"`
	UpdatedAt *types.Timestamp `json:"updated_at,omitempty"`
}

// maskAPIKey returns a masked version of an API key.
// Shows the first 3 and last 4 characters: "sk-...xxxx".
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:3] + "..." + key[len(key)-4:]
}

func LLMConfigFromResult(r service.LLMConfigResult) LLMConfigResponse {
	return LLMConfigResponse{
		Provider:  r.Provider,
		Model:     r.Model,
		APIKey:    maskAPIKey(r.APIKey),
		CreatedAt: r.Timestamps.CreatedAt,
		UpdatedAt: r.Timestamps.UpdatedAt,
	}
}
