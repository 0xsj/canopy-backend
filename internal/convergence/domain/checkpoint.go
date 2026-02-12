package domain

import (
	"fmt"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// CheckpointStatus represents the state of a consensus checkpoint.
type CheckpointStatus string

const (
	CheckpointOpen     CheckpointStatus = "open"
	CheckpointResolved CheckpointStatus = "resolved"
)

// IsValid reports whether the status is a recognized value.
func (s CheckpointStatus) IsValid() bool {
	switch s {
	case CheckpointOpen, CheckpointResolved:
		return true
	}
	return false
}

// Position represents a participant's stance in a consensus checkpoint.
type Position string

const (
	PositionAlign   Position = "align"
	PositionConcern Position = "concern"
	PositionBlock   Position = "block"
)

// IsValid reports whether the position is a recognized value.
func (p Position) IsValid() bool {
	switch p {
	case PositionAlign, PositionConcern, PositionBlock:
		return true
	}
	return false
}

// ConsensusSignal records a participant's position on a checkpoint.
type ConsensusSignal struct {
	checkpointID types.CheckpointID
	userID       types.UserID
	position     Position
	explanation  string // optional
	createdAt    types.Timestamp
}

// NewConsensusSignal creates a new consensus position.
func NewConsensusSignal(checkpointID types.CheckpointID, userID types.UserID, position Position, explanation string) (ConsensusSignal, error) {
	if checkpointID.IsZero() {
		return ConsensusSignal{}, fmt.Errorf("convergence: checkpoint ID is required")
	}
	if userID.IsZero() {
		return ConsensusSignal{}, fmt.Errorf("convergence: user ID is required")
	}
	if !position.IsValid() {
		return ConsensusSignal{}, fmt.Errorf("convergence: invalid position %q", position)
	}

	return ConsensusSignal{
		checkpointID: checkpointID,
		userID:       userID,
		position:     position,
		explanation:  explanation,
		createdAt:    types.Now(),
	}, nil
}

// ReconstructConsensusSignal builds a ConsensusSignal from trusted data.
func ReconstructConsensusSignal(
	checkpointID types.CheckpointID,
	userID types.UserID,
	position Position,
	explanation string,
	createdAt types.Timestamp,
) ConsensusSignal {
	return ConsensusSignal{
		checkpointID: checkpointID,
		userID:       userID,
		position:     position,
		explanation:  explanation,
		createdAt:    createdAt,
	}
}

func (cs ConsensusSignal) CheckpointID() types.CheckpointID { return cs.checkpointID }
func (cs ConsensusSignal) UserID() types.UserID             { return cs.userID }
func (cs ConsensusSignal) Position() Position               { return cs.position }
func (cs ConsensusSignal) Explanation() string              { return cs.explanation }
func (cs ConsensusSignal) CreatedAt() types.Timestamp       { return cs.createdAt }

// Checkpoint represents a consensus checkpoint — a moment where the
// workspace pauses to assess alignment on a set of promoted leaves.
type Checkpoint struct {
	id          types.CheckpointID
	workspaceID types.WorkspaceID
	leafIDs     []types.LeafID
	status      CheckpointStatus
	signals     []ConsensusSignal
	timestamps  types.Timestamps
}

// NewCheckpoint creates a new open consensus checkpoint for a set of leaves.
func NewCheckpoint(workspaceID types.WorkspaceID, leafIDs []types.LeafID) (Checkpoint, error) {
	if workspaceID.IsZero() {
		return Checkpoint{}, fmt.Errorf("convergence: workspace ID is required")
	}
	if len(leafIDs) == 0 {
		return Checkpoint{}, fmt.Errorf("convergence: at least one leaf is required")
	}

	return Checkpoint{
		id:          types.NewCheckpointID(),
		workspaceID: workspaceID,
		leafIDs:     leafIDs,
		status:      CheckpointOpen,
		timestamps:  types.NewMutableTimestamps(),
	}, nil
}

// ReconstructCheckpoint builds a Checkpoint from trusted data.
func ReconstructCheckpoint(
	id types.CheckpointID,
	workspaceID types.WorkspaceID,
	leafIDs []types.LeafID,
	status CheckpointStatus,
	signals []ConsensusSignal,
	timestamps types.Timestamps,
) Checkpoint {
	return Checkpoint{
		id:          id,
		workspaceID: workspaceID,
		leafIDs:     leafIDs,
		status:      status,
		signals:     signals,
		timestamps:  timestamps,
	}
}

// RecordSignal adds a consensus signal to the checkpoint.
// Only open checkpoints accept signals.
func (c *Checkpoint) RecordSignal(signal ConsensusSignal) error {
	if c.status != CheckpointOpen {
		return fmt.Errorf("convergence: checkpoint is %s, cannot record signals", c.status)
	}

	// Check for duplicate signal from same user.
	for _, existing := range c.signals {
		if existing.userID == signal.userID {
			return fmt.Errorf("convergence: user already signaled on this checkpoint")
		}
	}

	c.signals = append(c.signals, signal)
	c.timestamps.Touch()
	return nil
}

// Resolve marks the checkpoint as resolved. Cannot be undone.
func (c *Checkpoint) Resolve() error {
	if c.status != CheckpointOpen {
		return fmt.Errorf("convergence: checkpoint already resolved")
	}
	c.status = CheckpointResolved
	c.timestamps.Touch()
	return nil
}

// HasConsensus reports whether all signals are aligned (no blocks or concerns).
func (c Checkpoint) HasConsensus() bool {
	if len(c.signals) == 0 {
		return false
	}
	for _, s := range c.signals {
		if s.position != PositionAlign {
			return false
		}
	}
	return true
}

func (c Checkpoint) ID() types.CheckpointID         { return c.id }
func (c Checkpoint) WorkspaceID() types.WorkspaceID { return c.workspaceID }
func (c Checkpoint) LeafIDs() []types.LeafID        { return c.leafIDs }
func (c Checkpoint) Status() CheckpointStatus       { return c.status }
func (c Checkpoint) Signals() []ConsensusSignal     { return c.signals }
func (c Checkpoint) Timestamps() types.Timestamps   { return c.timestamps }
