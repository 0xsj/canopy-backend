package domain

import (
	"context"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// LeafRepository defines the persistence port for leaf entities.
// Leaves are immutable — there is no Update method.
type LeafRepository interface {
	// Create persists a new leaf.
	Create(ctx context.Context, leaf Leaf) error

	// FindByID returns a leaf by ID.
	FindByID(ctx context.Context, id types.LeafID) (Leaf, error)

	// FindByIDs returns leaves matching the given IDs.
	FindByIDs(ctx context.Context, ids []types.LeafID) ([]Leaf, error)

	// FindByWorkspace returns leaves in a workspace with optional filters.
	FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID, filter LeafFilter) ([]Leaf, error)

	// FindByBranch returns all leaves on a branch, ordered by creation time.
	FindByBranch(ctx context.Context, branchID types.BranchID) ([]Leaf, error)

	// FindBySeed returns all leaves grown from a seed.
	FindBySeed(ctx context.Context, seedID types.SeedID) ([]Leaf, error)

	// UpdateLayer changes a leaf's layer (understory → canopy promotion).
	// This is the only mutation allowed on a leaf, driven by the
	// convergence context through an event.
	UpdateLayer(ctx context.Context, id types.LeafID, layer Layer) error
}

// LeafFilter provides optional filtering for workspace leaf queries.
type LeafFilter struct {
	AuthorID *types.UserID
	SeedID   *types.SeedID
	Layer    *Layer
	Tags     []string // leaves matching any of these tags
}

// BranchRepository defines the persistence port for branch entities.
type BranchRepository interface {
	// Create persists a new branch.
	Create(ctx context.Context, branch Branch) error

	// FindByID returns a branch by ID.
	FindByID(ctx context.Context, id types.BranchID) (Branch, error)

	// FindByWorkspace returns all branches in a workspace.
	FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]Branch, error)

	// FindBySeed returns all branches grown from a seed.
	FindBySeed(ctx context.Context, seedID types.SeedID) ([]Branch, error)

	// FindByAuthor returns all branches created by a user in a workspace.
	FindByAuthor(ctx context.Context, workspaceID types.WorkspaceID, authorID types.UserID) ([]Branch, error)

	// Update persists changes to a branch (e.g., setting root leaf).
	Update(ctx context.Context, branch Branch) error
}

// ConnectionRepository defines the persistence port for connection entities.
type ConnectionRepository interface {
	// Create persists a new connection.
	Create(ctx context.Context, connection Connection) error

	// FindByID returns a connection by ID.
	FindByID(ctx context.Context, id types.ConnectionID) (Connection, error)

	// FindByWorkspace returns all connections in a workspace.
	FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]Connection, error)

	// FindByLeaf returns all connections that include a given leaf.
	FindByLeaf(ctx context.Context, leafID types.LeafID) ([]Connection, error)
}

// GraphQueryEngine defines the port for graph traversal queries.
// The initial implementation uses Postgres recursive CTEs. A graph
// database adapter can be introduced later for performance.
type GraphQueryEngine interface {
	// Ancestors returns all ancestor leaves back to the seed root.
	Ancestors(ctx context.Context, leafID types.LeafID) ([]Leaf, error)

	// Descendants returns all descendant leaves from a given leaf.
	Descendants(ctx context.Context, leafID types.LeafID) ([]Leaf, error)

	// Neighborhood returns all leaves within N connections of a given leaf.
	Neighborhood(ctx context.Context, leafID types.LeafID, depth int) ([]Leaf, error)

	// SynthesisLineage traces a synthesis leaf back through its source chain.
	SynthesisLineage(ctx context.Context, leafID types.LeafID) ([]Leaf, error)
}
