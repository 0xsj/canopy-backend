package domain

import (
	"context"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// ThreadRepository defines the persistence port for discussion threads.
type ThreadRepository interface {
	// Save persists a thread (create or update with new comments).
	// Threads are upserted — if no thread exists for the leaf, one is created.
	Save(ctx context.Context, thread Thread) error

	// FindByLeaf returns the thread for a leaf.
	// Returns a NotFound error if no thread exists.
	FindByLeaf(ctx context.Context, leafID types.LeafID) (Thread, error)

	// Exists reports whether a thread exists for a leaf.
	Exists(ctx context.Context, leafID types.LeafID) (bool, error)
}
