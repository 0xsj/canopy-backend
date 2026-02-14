package domain

import (
	"fmt"
	"strings"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// LLMProvider identifies a supported LLM provider.
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

// LLMConfig holds a workspace's custom LLM provider configuration.
// The API key is stored encrypted; the domain treats it as opaque bytes.
type LLMConfig struct {
	workspaceID types.WorkspaceID
	provider    LLMProvider
	model       string
	apiKeyEnc   []byte // encrypted API key (opaque to domain)
	timestamps  types.Timestamps
}

// NewLLMConfig creates a new LLM configuration for a workspace.
func NewLLMConfig(wsID types.WorkspaceID, provider, model string, apiKeyEnc []byte) (LLMConfig, error) {
	if wsID.IsZero() {
		return LLMConfig{}, fmt.Errorf("workspace: llm config: workspace ID is required")
	}

	p := LLMProvider(strings.ToLower(strings.TrimSpace(provider)))
	if !p.IsValid() {
		return LLMConfig{}, fmt.Errorf("workspace: llm config: invalid provider %q", provider)
	}

	model = strings.TrimSpace(model)
	if model == "" {
		return LLMConfig{}, fmt.Errorf("workspace: llm config: model is required")
	}

	if len(apiKeyEnc) == 0 {
		return LLMConfig{}, fmt.Errorf("workspace: llm config: encrypted API key is required")
	}

	return LLMConfig{
		workspaceID: wsID,
		provider:    p,
		model:       model,
		apiKeyEnc:   apiKeyEnc,
		timestamps:  types.NewMutableTimestamps(),
	}, nil
}

// ReconstructLLMConfig builds an LLMConfig from trusted persistence data.
func ReconstructLLMConfig(
	wsID types.WorkspaceID,
	provider string,
	model string,
	apiKeyEnc []byte,
	timestamps types.Timestamps,
) LLMConfig {
	return LLMConfig{
		workspaceID: wsID,
		provider:    LLMProvider(provider),
		model:       model,
		apiKeyEnc:   apiKeyEnc,
		timestamps:  timestamps,
	}
}

func (c LLMConfig) WorkspaceID() types.WorkspaceID { return c.workspaceID }
func (c LLMConfig) Provider() LLMProvider          { return c.provider }
func (c LLMConfig) Model() string                  { return c.model }
func (c LLMConfig) APIKeyEnc() []byte              { return c.apiKeyEnc }
func (c LLMConfig) Timestamps() types.Timestamps   { return c.timestamps }
