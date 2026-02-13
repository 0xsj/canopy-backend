package v1

import "github.com/0xsj/canopy-backend/internal/session/domain"

type StartSessionRequest struct {
	SeedID       string `json:"seed_id"`
	ParentLeafID string `json:"parent_leaf_id,omitempty"`
	SessionType  string `json:"session_type"`
}

type AddMessageRequest struct {
	Message domain.Message `json:"message"`
}

type CompleteSessionRequest struct {
	LeafID string `json:"leaf_id"`
}
