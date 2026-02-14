package domain

import (
	"fmt"
	"strings"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// LLMProvider identifies a supported LLM provider for user-level config.
type LLMProvider string

const (
	LLMProviderAnthropic LLMProvider = "anthropic"
	LLMProviderOpenAI    LLMProvider = "openai"
	LLMProviderGemini    LLMProvider = "gemini"
)

// IsValid reports whether the provider is a recognized value.
func (p LLMProvider) IsValid() bool {
	switch p {
	case LLMProviderAnthropic, LLMProviderOpenAI, LLMProviderGemini:
		return true
	}
	return false
}

// UserLLMConfig holds a user's personal LLM provider configuration.
// The API key is stored encrypted; the domain treats it as opaque bytes.
type UserLLMConfig struct {
	userID     types.UserID
	provider   LLMProvider
	model      string
	apiKeyEnc  []byte // encrypted API key (opaque to domain)
	timestamps types.Timestamps
}

// NewUserLLMConfig creates a new per-user LLM configuration.
func NewUserLLMConfig(userID types.UserID, provider, model string, apiKeyEnc []byte) (UserLLMConfig, error) {
	if userID.IsZero() {
		return UserLLMConfig{}, fmt.Errorf("identity: user llm config: user ID is required")
	}

	p := LLMProvider(strings.ToLower(strings.TrimSpace(provider)))
	if !p.IsValid() {
		return UserLLMConfig{}, fmt.Errorf("identity: user llm config: invalid provider %q", provider)
	}

	model = strings.TrimSpace(model)
	if model == "" {
		return UserLLMConfig{}, fmt.Errorf("identity: user llm config: model is required")
	}

	if len(apiKeyEnc) == 0 {
		return UserLLMConfig{}, fmt.Errorf("identity: user llm config: encrypted API key is required")
	}

	return UserLLMConfig{
		userID:     userID,
		provider:   p,
		model:      model,
		apiKeyEnc:  apiKeyEnc,
		timestamps: types.NewMutableTimestamps(),
	}, nil
}

// ReconstructUserLLMConfig builds a UserLLMConfig from trusted persistence data.
func ReconstructUserLLMConfig(
	userID types.UserID,
	provider string,
	model string,
	apiKeyEnc []byte,
	timestamps types.Timestamps,
) UserLLMConfig {
	return UserLLMConfig{
		userID:     userID,
		provider:   LLMProvider(provider),
		model:      model,
		apiKeyEnc:  apiKeyEnc,
		timestamps: timestamps,
	}
}

func (c UserLLMConfig) UserID() types.UserID         { return c.userID }
func (c UserLLMConfig) Provider() LLMProvider        { return c.provider }
func (c UserLLMConfig) Model() string                { return c.model }
func (c UserLLMConfig) APIKeyEnc() []byte            { return c.apiKeyEnc }
func (c UserLLMConfig) Timestamps() types.Timestamps { return c.timestamps }
