package domain

import (
	"fmt"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// Connection links two or more leaves across branches, representing a
// relationship discovered by a participant. Connections are immutable
// after creation.
type Connection struct {
	id          types.ConnectionID
	workspaceID types.WorkspaceID
	authorID    types.UserID
	leafIDs     []types.LeafID
	timestamps  types.Timestamps
}

// NewConnection creates a connection between leaves. At least two leaves
// are required.
func NewConnection(workspaceID types.WorkspaceID, authorID types.UserID, leafIDs []types.LeafID) (Connection, error) {
	if workspaceID.IsZero() {
		return Connection{}, fmt.Errorf("exploration: workspace ID is required for connection")
	}
	if authorID.IsZero() {
		return Connection{}, fmt.Errorf("exploration: author ID is required for connection")
	}
	if len(leafIDs) < 2 {
		return Connection{}, fmt.Errorf("exploration: connection requires at least 2 leaves")
	}

	// Check for duplicates.
	seen := make(map[string]bool, len(leafIDs))
	for _, id := range leafIDs {
		if id.IsZero() {
			return Connection{}, fmt.Errorf("exploration: leaf ID must not be empty")
		}
		key := id.String()
		if seen[key] {
			return Connection{}, fmt.Errorf("exploration: duplicate leaf ID %q in connection", key)
		}
		seen[key] = true
	}

	return Connection{
		id:          types.NewConnectionID(),
		workspaceID: workspaceID,
		authorID:    authorID,
		leafIDs:     leafIDs,
		timestamps:  types.NewTimestamps(),
	}, nil
}

// ReconstructConnection builds a Connection from trusted data.
func ReconstructConnection(
	id types.ConnectionID,
	workspaceID types.WorkspaceID,
	authorID types.UserID,
	leafIDs []types.LeafID,
	timestamps types.Timestamps,
) Connection {
	return Connection{
		id:          id,
		workspaceID: workspaceID,
		authorID:    authorID,
		leafIDs:     leafIDs,
		timestamps:  timestamps,
	}
}

func (c Connection) ID() types.ConnectionID         { return c.id }
func (c Connection) WorkspaceID() types.WorkspaceID { return c.workspaceID }
func (c Connection) AuthorID() types.UserID         { return c.authorID }
func (c Connection) LeafIDs() []types.LeafID        { return c.leafIDs }
func (c Connection) Timestamps() types.Timestamps   { return c.timestamps }
