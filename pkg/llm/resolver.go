package llm

import (
	"context"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// ProviderResolver resolves an LLM provider for a workspace.
// When a workspace has a custom LLM config, that provider is returned.
// Otherwise the server-wide default provider is used as fallback.
type ProviderResolver interface {
	Resolve(ctx context.Context, workspaceID types.WorkspaceID) (Provider, error)
}
