package domain

import (
	"context"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// SeedRepository defines the persistence port for seed entities.
type SeedRepository interface {
	// Create persists a new seed.
	Create(ctx context.Context, seed Seed) error

	// FindByID returns a seed by ID.
	// Returns a NotFound error if no seed exists with that ID.
	FindByID(ctx context.Context, id types.SeedID) (Seed, error)

	// FindByWorkspace returns all seeds in a workspace.
	FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]Seed, error)

	// Update persists changes to an existing seed (constraints, tags).
	Update(ctx context.Context, seed Seed) error

	// UpdatePosition persists a position change for a seed on the canvas.
	UpdatePosition(ctx context.Context, id types.SeedID, x, y float64) error
}
