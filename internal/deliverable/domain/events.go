package domain

import "time"

// Event subjects published by the deliverable context.
const (
	SubjectDeliverableDraftCreated = "deliverable.draft.created"
	SubjectDeliverableUpdated      = "deliverable.updated"
	SubjectDeliverableFinalized    = "deliverable.finalized"
)

// DeliverableDraftCreatedData is published when a new deliverable draft
// is created from consensus-backed leaves.
type DeliverableDraftCreatedData struct {
	DeliverableID string    `json:"deliverable_id"`
	WorkspaceID   string    `json:"workspace_id"`
	Format        string    `json:"format"`
	SourceLeafIDs []string  `json:"source_leaf_ids"`
	Timestamp     time.Time `json:"timestamp"`
}

// DeliverableUpdatedData is published when a deliverable's content is edited.
type DeliverableUpdatedData struct {
	DeliverableID string    `json:"deliverable_id"`
	WorkspaceID   string    `json:"workspace_id"`
	Version       int       `json:"version"`
	Timestamp     time.Time `json:"timestamp"`
}

// DeliverableFinalizedData is published when a deliverable is marked as final.
type DeliverableFinalizedData struct {
	DeliverableID string    `json:"deliverable_id"`
	WorkspaceID   string    `json:"workspace_id"`
	Format        string    `json:"format"`
	Version       int       `json:"version"`
	Timestamp     time.Time `json:"timestamp"`
}
