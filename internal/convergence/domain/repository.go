package domain

import (
	"context"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// SignalRepository defines the persistence port for leaf signals.
type SignalRepository interface {
	// Create persists a new signal.
	Create(ctx context.Context, signal Signal) error

	// Delete removes a signal (user retracts their signal).
	Delete(ctx context.Context, id types.ID[signalTag]) error

	// FindByLeaf returns all signals on a leaf.
	FindByLeaf(ctx context.Context, leafID types.LeafID) ([]Signal, error)

	// FindByUser returns all signals by a user in a workspace.
	FindByUser(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) ([]Signal, error)

	// FindByLeafAndUser returns a user's signal on a specific leaf, if any.
	FindByLeafAndUser(ctx context.Context, leafID types.LeafID, userID types.UserID) (Signal, error)

	// CountByLeaf returns signal counts grouped by type for a leaf.
	CountByLeaf(ctx context.Context, leafID types.LeafID) (map[SignalType]int, error)
}

// CheckpointRepository defines the persistence port for consensus checkpoints.
type CheckpointRepository interface {
	// Create persists a new checkpoint.
	Create(ctx context.Context, checkpoint Checkpoint) error

	// FindByID returns a checkpoint by ID.
	FindByID(ctx context.Context, id types.CheckpointID) (Checkpoint, error)

	// FindByWorkspace returns all checkpoints in a workspace.
	FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]Checkpoint, error)

	// FindOpenByWorkspace returns only open checkpoints in a workspace.
	FindOpenByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]Checkpoint, error)

	// Update persists changes to a checkpoint (new signals, status change).
	Update(ctx context.Context, checkpoint Checkpoint) error
}
