package llm

import (
	"context"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// ProviderResolver resolves an LLM provider and model for a workspace and task type.
// The task type drives model selection — different tasks route to different
// model tiers (flagship, balanced, fast). When a workspace or user has a
// custom LLM config, that provider is returned; otherwise the server-wide
// default is used as fallback.
//
// Returns the provider, the model name to use (set on ChatRequest.Model),
// and any error. Callers must set the returned model on their ChatRequest.
type ProviderResolver interface {
	Resolve(ctx context.Context, workspaceID types.WorkspaceID, task TaskType) (Provider, string, error)
}
