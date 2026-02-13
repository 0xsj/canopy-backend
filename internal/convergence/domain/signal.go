package domain

import (
	"fmt"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// SignalType represents the kind of signal a user applies to a leaf.
type SignalType string

const (
	SignalUpvote SignalType = "upvote"
	SignalPin    SignalType = "pin"
	SignalFlag   SignalType = "flag"
)

// IsValid reports whether the signal type is a recognized value.
func (s SignalType) IsValid() bool {
	switch s {
	case SignalUpvote, SignalPin, SignalFlag:
		return true
	}
	return false
}

// Signal records a user's reaction to a leaf. Signals are immutable
// after creation — a user can remove a signal but not edit it.
type Signal struct {
	id          types.ID[signalTag]
	workspaceID types.WorkspaceID
	leafID      types.LeafID
	userID      types.UserID
	signalType  SignalType
	annotation  string // optional, used for flags to explain the concern
	timestamps  types.Timestamps
}

type signalTag struct{}

// SignalID is the exported type alias for use in adapter packages.
type SignalID = types.ID[signalTag]

const prefixSignal = "sig"

// NewSignal creates a new signal on a leaf.
func NewSignal(workspaceID types.WorkspaceID, leafID types.LeafID, userID types.UserID, signalType SignalType, annotation string) (Signal, error) {
	if workspaceID.IsZero() {
		return Signal{}, fmt.Errorf("convergence: workspace ID is required")
	}
	if leafID.IsZero() {
		return Signal{}, fmt.Errorf("convergence: leaf ID is required")
	}
	if userID.IsZero() {
		return Signal{}, fmt.Errorf("convergence: user ID is required")
	}
	if !signalType.IsValid() {
		return Signal{}, fmt.Errorf("convergence: invalid signal type %q", signalType)
	}
	if signalType == SignalFlag && annotation == "" {
		return Signal{}, fmt.Errorf("convergence: flag signals require an annotation")
	}

	return Signal{
		id:          types.NewID[signalTag](prefixSignal),
		workspaceID: workspaceID,
		leafID:      leafID,
		userID:      userID,
		signalType:  signalType,
		annotation:  annotation,
		timestamps:  types.NewTimestamps(),
	}, nil
}

// ReconstructSignal builds a Signal from trusted data.
func ReconstructSignal(
	id types.ID[signalTag],
	workspaceID types.WorkspaceID,
	leafID types.LeafID,
	userID types.UserID,
	signalType SignalType,
	annotation string,
	timestamps types.Timestamps,
) Signal {
	return Signal{
		id:          id,
		workspaceID: workspaceID,
		leafID:      leafID,
		userID:      userID,
		signalType:  signalType,
		annotation:  annotation,
		timestamps:  timestamps,
	}
}

func (s Signal) ID() types.ID[signalTag]        { return s.id }
func (s Signal) WorkspaceID() types.WorkspaceID { return s.workspaceID }
func (s Signal) LeafID() types.LeafID           { return s.leafID }
func (s Signal) UserID() types.UserID           { return s.userID }
func (s Signal) Type() SignalType               { return s.signalType }
func (s Signal) Annotation() string             { return s.annotation }
func (s Signal) Timestamps() types.Timestamps   { return s.timestamps }

// SignalIDFrom creates a SignalID from a trusted database string.
func SignalIDFrom(raw string) types.ID[signalTag] { return types.IDFrom[signalTag](raw) }
