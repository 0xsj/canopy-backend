package v1

import (
	"github.com/0xsj/canopy-backend/internal/ledger/domain"
	"github.com/0xsj/canopy-backend/pkg/types"
)

type SystemEntryResponse struct {
	ID            string          `json:"id"`
	EventSubject  string          `json:"event_subject"`
	EventData     map[string]any  `json:"event_data"`
	SourceContext string          `json:"source_context"`
	CreatedAt     types.Timestamp `json:"created_at"`
}

func SystemEntryFromDomain(e domain.SystemEntry) SystemEntryResponse {
	return SystemEntryResponse{
		ID:            e.ID().String(),
		EventSubject:  e.EventSubject(),
		EventData:     e.EventData(),
		SourceContext: e.SourceContext(),
		CreatedAt:     e.CreatedAt(),
	}
}

func SystemEntriesFromDomain(entries []domain.SystemEntry) []SystemEntryResponse {
	out := make([]SystemEntryResponse, len(entries))
	for i, e := range entries {
		out[i] = SystemEntryFromDomain(e)
	}
	return out
}

type DomainEntryResponse struct {
	ID           string          `json:"id"`
	ActorID      string          `json:"actor_id"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	OrgID        string          `json:"org_id,omitempty"`
	WorkspaceID  string          `json:"workspace_id,omitempty"`
	Metadata     map[string]any  `json:"metadata,omitempty"`
	CreatedAt    types.Timestamp `json:"created_at"`
}

func DomainEntryFromDomain(e domain.DomainEntry) DomainEntryResponse {
	return DomainEntryResponse{
		ID:           e.ID().String(),
		ActorID:      e.ActorID(),
		Action:       string(e.Action()),
		ResourceType: e.ResourceType(),
		ResourceID:   e.ResourceID(),
		OrgID:        e.OrgID(),
		WorkspaceID:  e.WorkspaceID(),
		Metadata:     e.Metadata(),
		CreatedAt:    e.CreatedAt(),
	}
}

func DomainEntriesFromDomain(entries []domain.DomainEntry) []DomainEntryResponse {
	out := make([]DomainEntryResponse, len(entries))
	for i, e := range entries {
		out[i] = DomainEntryFromDomain(e)
	}
	return out
}
