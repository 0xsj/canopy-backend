package domain

import "time"

// Event subjects published by the exploration context.
const (
	SubjectLeafCreated       = "exploration.leaf.created"
	SubjectBranchCreated     = "exploration.branch.created"
	SubjectConnectionCreated = "exploration.connection.created"
	SubjectLeafPromoted      = "exploration.leaf.promoted"
	SubjectLeafUpvoted       = "exploration.leaf.upvoted"
	SubjectLeafPinned        = "exploration.leaf.pinned"
	SubjectLeafFlagged       = "exploration.leaf.flagged"
)

// LeafCreatedData is published when a new leaf is confirmed.
type LeafCreatedData struct {
	LeafID      string    `json:"leaf_id"`
	WorkspaceID string    `json:"workspace_id"`
	SeedID      string    `json:"seed_id"`
	BranchID    string    `json:"branch_id"`
	AuthorID    string    `json:"author_id"`
	ParentID    string    `json:"parent_id,omitempty"`
	Title       string    `json:"title"`
	Layer       string    `json:"layer"`
	IsSynthesis bool      `json:"is_synthesis"`
	Timestamp   time.Time `json:"timestamp"`
}

// BranchCreatedData is published when a new branch is created.
type BranchCreatedData struct {
	BranchID    string    `json:"branch_id"`
	WorkspaceID string    `json:"workspace_id"`
	SeedID      string    `json:"seed_id"`
	AuthorID    string    `json:"author_id"`
	Timestamp   time.Time `json:"timestamp"`
}

// ConnectionCreatedData is published when leaves are connected.
type ConnectionCreatedData struct {
	ConnectionID string    `json:"connection_id"`
	WorkspaceID  string    `json:"workspace_id"`
	AuthorID     string    `json:"author_id"`
	LeafIDs      []string  `json:"leaf_ids"`
	Timestamp    time.Time `json:"timestamp"`
}

// LeafPromotedData is published when a leaf is promoted to the canopy layer.
type LeafPromotedData struct {
	LeafID      string    `json:"leaf_id"`
	WorkspaceID string    `json:"workspace_id"`
	Timestamp   time.Time `json:"timestamp"`
}

// LeafSignaledData is published when a leaf receives a signal (upvote, pin, flag).
type LeafSignaledData struct {
	LeafID      string    `json:"leaf_id"`
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	SignalType  string    `json:"signal_type"`
	Timestamp   time.Time `json:"timestamp"`
}
