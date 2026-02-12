package domain

import (
	"fmt"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// Branch groups a chain of leaves created by a single author from a seed
// or parent leaf. Each branch has a root leaf and belongs to a workspace
// and seed. Branches are immutable after creation.
type Branch struct {
	id          types.BranchID
	workspaceID types.WorkspaceID
	seedID      types.SeedID
	authorID    types.UserID
	rootLeafID  types.LeafID
	timestamps  types.Timestamps
}

// NewBranch creates a new branch. The rootLeafID is set after the first
// leaf on this branch is created — pass a zero LeafID initially and set
// it via SetRootLeaf.
func NewBranch(workspaceID types.WorkspaceID, seedID types.SeedID, authorID types.UserID) (Branch, error) {
	if workspaceID.IsZero() {
		return Branch{}, fmt.Errorf("exploration: workspace ID is required for branch")
	}
	if seedID.IsZero() {
		return Branch{}, fmt.Errorf("exploration: seed ID is required for branch")
	}
	if authorID.IsZero() {
		return Branch{}, fmt.Errorf("exploration: author ID is required for branch")
	}

	return Branch{
		id:          types.NewBranchID(),
		workspaceID: workspaceID,
		seedID:      seedID,
		authorID:    authorID,
		timestamps:  types.NewTimestamps(),
	}, nil
}

// SetRootLeaf assigns the first leaf on this branch. Can only be set once.
func (b *Branch) SetRootLeaf(leafID types.LeafID) error {
	if leafID.IsZero() {
		return fmt.Errorf("exploration: leaf ID is required")
	}
	if !b.rootLeafID.IsZero() {
		return fmt.Errorf("exploration: root leaf already set")
	}
	b.rootLeafID = leafID
	return nil
}

// ReconstructBranch builds a Branch from trusted data.
func ReconstructBranch(
	id types.BranchID,
	workspaceID types.WorkspaceID,
	seedID types.SeedID,
	authorID types.UserID,
	rootLeafID types.LeafID,
	timestamps types.Timestamps,
) Branch {
	return Branch{
		id:          id,
		workspaceID: workspaceID,
		seedID:      seedID,
		authorID:    authorID,
		rootLeafID:  rootLeafID,
		timestamps:  timestamps,
	}
}

func (b Branch) ID() types.BranchID             { return b.id }
func (b Branch) WorkspaceID() types.WorkspaceID { return b.workspaceID }
func (b Branch) SeedID() types.SeedID           { return b.seedID }
func (b Branch) AuthorID() types.UserID         { return b.authorID }
func (b Branch) RootLeafID() types.LeafID       { return b.rootLeafID }
func (b Branch) Timestamps() types.Timestamps   { return b.timestamps }
