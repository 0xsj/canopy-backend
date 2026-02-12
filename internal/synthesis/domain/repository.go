package domain

import (
	"context"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// SynthesisRepository defines the persistence port for synthesis workflows.
type SynthesisRepository interface {
	// Create persists a new synthesis workflow.
	Create(ctx context.Context, workflow SynthesisWorkflow) error

	// FindByID returns a workflow by ID.
	FindByID(ctx context.Context, id types.ID[synthesisTag]) (SynthesisWorkflow, error)

	// FindByWorkspace returns all synthesis workflows in a workspace.
	FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]SynthesisWorkflow, error)

	// Update persists changes to a workflow (status transitions, result leaf).
	Update(ctx context.Context, workflow SynthesisWorkflow) error
}
