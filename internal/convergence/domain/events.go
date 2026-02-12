package domain

import "time"

// Event subjects published by the convergence context.
const (
	SubjectLeafPromotedToCanopy    = "convergence.leaf.promoted"
	SubjectCheckpointCreated       = "convergence.checkpoint.created"
	SubjectConsensusSignalRecorded = "convergence.signal.recorded"
	SubjectConsensusReached        = "convergence.consensus.reached"
)

// LeafPromotedToCanopyData is published when a leaf is promoted from
// understory to canopy layer.
type LeafPromotedToCanopyData struct {
	LeafID      string    `json:"leaf_id"`
	WorkspaceID string    `json:"workspace_id"`
	Timestamp   time.Time `json:"timestamp"`
}

// CheckpointCreatedData is published when a new consensus checkpoint is opened.
type CheckpointCreatedData struct {
	CheckpointID string    `json:"checkpoint_id"`
	WorkspaceID  string    `json:"workspace_id"`
	LeafIDs      []string  `json:"leaf_ids"`
	Timestamp    time.Time `json:"timestamp"`
}

// ConsensusSignalRecordedData is published when a participant records
// their position on a checkpoint.
type ConsensusSignalRecordedData struct {
	CheckpointID string    `json:"checkpoint_id"`
	WorkspaceID  string    `json:"workspace_id"`
	UserID       string    `json:"user_id"`
	Position     string    `json:"position"`
	Timestamp    time.Time `json:"timestamp"`
}

// ConsensusReachedData is published when all participants align on a checkpoint.
type ConsensusReachedData struct {
	CheckpointID string    `json:"checkpoint_id"`
	WorkspaceID  string    `json:"workspace_id"`
	LeafIDs      []string  `json:"leaf_ids"`
	Timestamp    time.Time `json:"timestamp"`
}
