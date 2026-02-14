package domain

import "time"

// Event subjects published by the workspace context.
const (
	SubjectWorkspaceCreated       = "workspace.created"
	SubjectMemberJoined           = "workspace.member.joined"
	SubjectMemberLeft             = "workspace.member.left"
	SubjectWorkspaceConfigUpdated = "workspace.config.updated"
	SubjectPhaseTransitioned      = "workspace.phase.transitioned"
	SubjectLLMConfigUpdated       = "workspace.llm_config.updated"
	SubjectLLMConfigDeleted       = "workspace.llm_config.deleted"
)

// WorkspaceCreatedData is published when a new workspace is created.
type WorkspaceCreatedData struct {
	WorkspaceID string    `json:"workspace_id"`
	OrgID       string    `json:"org_id"`
	TeamID      string    `json:"team_id,omitempty"`
	Name        string    `json:"name"`
	Timestamp   time.Time `json:"timestamp"`
}

// MemberJoinedData is published when a user joins a workspace.
type MemberJoinedData struct {
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	Role        string    `json:"role"`
	Timestamp   time.Time `json:"timestamp"`
}

// MemberLeftData is published when a user leaves a workspace.
type MemberLeftData struct {
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	Timestamp   time.Time `json:"timestamp"`
}

// WorkspaceConfigUpdatedData is published when workspace configuration changes.
type WorkspaceConfigUpdatedData struct {
	WorkspaceID    string    `json:"workspace_id"`
	LoreKeeperMode string    `json:"lore_keeper_mode"`
	LoreKeeperID   string    `json:"lore_keeper_id,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// PhaseTransitionedData is published when a workspace advances to a new phase.
type PhaseTransitionedData struct {
	WorkspaceID   string    `json:"workspace_id"`
	PreviousPhase string    `json:"previous_phase"`
	NewPhase      string    `json:"new_phase"`
	Timestamp     time.Time `json:"timestamp"`
}

// LLMConfigUpdatedData is published when a workspace's LLM config is set or changed.
type LLMConfigUpdatedData struct {
	WorkspaceID string    `json:"workspace_id"`
	Provider    string    `json:"provider"`
	Model       string    `json:"model"`
	Timestamp   time.Time `json:"timestamp"`
}

// LLMConfigDeletedData is published when a workspace's LLM config is removed.
type LLMConfigDeletedData struct {
	WorkspaceID string    `json:"workspace_id"`
	Timestamp   time.Time `json:"timestamp"`
}
