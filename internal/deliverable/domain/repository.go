package domain

import (
	"context"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// DeliverableRepository defines the persistence port for deliverable entities.
type DeliverableRepository interface {
	// Create persists a new deliverable.
	Create(ctx context.Context, deliverable Deliverable) error

	// FindByID returns a deliverable by ID.
	FindByID(ctx context.Context, id types.DeliverableID) (Deliverable, error)

	// FindByWorkspace returns all deliverables in a workspace.
	FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]Deliverable, error)

	// Update persists changes to a deliverable (content, format, sources).
	Update(ctx context.Context, deliverable Deliverable) error
}

// ExportEngine defines the port for exporting deliverables to different formats.
type ExportEngine interface {
	// Export renders the deliverable content into the target format.
	Export(ctx context.Context, deliverable Deliverable, targetFormat Format) ([]byte, error)
}
