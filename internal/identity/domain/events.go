package domain

import "time"

// Event subjects published by the identity context.
const (
	SubjectUserRegistered       = "identity.user.registered"
	SubjectUserProfileUpdated   = "identity.user.profile_updated"
	SubjectUserLLMConfigUpdated = "identity.user.llm_config.updated"
	SubjectUserLLMConfigDeleted = "identity.user.llm_config.deleted"
)

// UserRegisteredData is the event payload published when a new user
// registers. All fields are primitives per ADR-006.
type UserRegisteredData struct {
	UserID      string    `json:"user_id"`
	ExternalID  string    `json:"external_id"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	Timestamp   time.Time `json:"timestamp"`
}

// UserProfileUpdatedData is the event payload published when a user
// updates their profile. Only the current (post-update) values are included.
type UserProfileUpdatedData struct {
	UserID      string    `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	AvatarURL   string    `json:"avatar_url"`
	Timestamp   time.Time `json:"timestamp"`
}

// UserLLMConfigUpdatedData is the event payload published when a user
// sets or updates their personal LLM configuration.
type UserLLMConfigUpdatedData struct {
	UserID    string    `json:"user_id"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	Timestamp time.Time `json:"timestamp"`
}

// UserLLMConfigDeletedData is the event payload published when a user
// deletes their personal LLM configuration.
type UserLLMConfigDeletedData struct {
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}
