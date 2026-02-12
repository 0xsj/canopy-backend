package domain

import "time"

// Event subjects published by the synthesis context.
const (
	SubjectSynthesisStarted     = "synthesis.started"
	SubjectSynthesisCompleted   = "synthesis.completed"
	SubjectSynthesisLeafCreated = "synthesis.leaf.created"
)

// SynthesisStartedData is published when a synthesis workflow begins processing.
type SynthesisStartedData struct {
	SynthesisID   string    `json:"synthesis_id"`
	WorkspaceID   string    `json:"workspace_id"`
	InitiatorID   string    `json:"initiator_id"`
	SourceLeafIDs []string  `json:"source_leaf_ids"`
	Timestamp     time.Time `json:"timestamp"`
}

// SynthesisCompletedData is published when a synthesis workflow completes.
type SynthesisCompletedData struct {
	SynthesisID  string    `json:"synthesis_id"`
	WorkspaceID  string    `json:"workspace_id"`
	ResultLeafID string    `json:"result_leaf_id"`
	Timestamp    time.Time `json:"timestamp"`
}

// SynthesisLeafCreatedData is published when the synthesis output leaf
// is persisted. Subscribers can use this to update graph views.
type SynthesisLeafCreatedData struct {
	LeafID        string    `json:"leaf_id"`
	WorkspaceID   string    `json:"workspace_id"`
	SourceLeafIDs []string  `json:"source_leaf_ids"`
	InitiatorID   string    `json:"initiator_id"`
	Timestamp     time.Time `json:"timestamp"`
}
