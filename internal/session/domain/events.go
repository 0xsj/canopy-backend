package domain

import "time"

// Event subjects published by the session context.
const (
	SubjectSessionStarted      = "session.started"
	SubjectSessionCheckpointed = "session.checkpointed"
	SubjectSessionCompleted    = "session.completed"
)

// SessionStartedData is published when a new session begins.
type SessionStartedData struct {
	SessionID   string    `json:"session_id"`
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	SeedID      string    `json:"seed_id"`
	SessionType string    `json:"session_type"`
	Timestamp   time.Time `json:"timestamp"`
}

// SessionCheckpointedData is published when a session enters the shaping phase.
type SessionCheckpointedData struct {
	SessionID   string    `json:"session_id"`
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	Timestamp   time.Time `json:"timestamp"`
}

// SessionCompletedData is published when a session completes and a leaf
// is confirmed.
type SessionCompletedData struct {
	SessionID   string    `json:"session_id"`
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	LeafID      string    `json:"leaf_id"`
	Timestamp   time.Time `json:"timestamp"`
}
