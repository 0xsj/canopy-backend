package domain

import (
	"fmt"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// WorkflowStatus tracks the lifecycle of a synthesis workflow.
type WorkflowStatus string

const (
	WorkflowPending    WorkflowStatus = "pending"
	WorkflowProcessing WorkflowStatus = "processing"
	WorkflowCompleted  WorkflowStatus = "completed"
	WorkflowFailed     WorkflowStatus = "failed"
)

// IsValid reports whether the status is a recognized value.
func (s WorkflowStatus) IsValid() bool {
	switch s {
	case WorkflowPending, WorkflowProcessing, WorkflowCompleted, WorkflowFailed:
		return true
	}
	return false
}

// IsTerminal reports whether the workflow is in a final state.
func (s WorkflowStatus) IsTerminal() bool {
	return s == WorkflowCompleted || s == WorkflowFailed
}

// SynthesisWorkflow manages the process of combining multiple leaves
// into a new synthesis leaf with source attribution. The workflow
// orchestrates the LLM call and tracks the resulting leaf.
type SynthesisWorkflow struct {
	id            types.ID[synthesisTag]
	workspaceID   types.WorkspaceID
	initiatorID   types.UserID
	sourceLeafIDs []types.LeafID
	resultLeafID  types.LeafID // set on completion
	status        WorkflowStatus
	failureReason string // set on failure
	timestamps    types.Timestamps
}

type synthesisTag struct{}

// SynthesisID is the exported type alias for use in adapter packages.
type SynthesisID = types.ID[synthesisTag]

const prefixSynthesis = "syn"

// NewSynthesisWorkflow creates a new pending synthesis workflow.
// At least two source leaves are required.
func NewSynthesisWorkflow(
	workspaceID types.WorkspaceID,
	initiatorID types.UserID,
	sourceLeafIDs []types.LeafID,
) (SynthesisWorkflow, error) {
	if workspaceID.IsZero() {
		return SynthesisWorkflow{}, fmt.Errorf("synthesis: workspace ID is required")
	}
	if initiatorID.IsZero() {
		return SynthesisWorkflow{}, fmt.Errorf("synthesis: initiator ID is required")
	}
	if len(sourceLeafIDs) < 2 {
		return SynthesisWorkflow{}, fmt.Errorf("synthesis: at least 2 source leaves are required")
	}

	return SynthesisWorkflow{
		id:            types.NewID[synthesisTag](prefixSynthesis),
		workspaceID:   workspaceID,
		initiatorID:   initiatorID,
		sourceLeafIDs: sourceLeafIDs,
		status:        WorkflowPending,
		timestamps:    types.NewMutableTimestamps(),
	}, nil
}

// ReconstructSynthesisWorkflow builds a SynthesisWorkflow from trusted data.
func ReconstructSynthesisWorkflow(
	id types.ID[synthesisTag],
	workspaceID types.WorkspaceID,
	initiatorID types.UserID,
	sourceLeafIDs []types.LeafID,
	resultLeafID types.LeafID,
	status WorkflowStatus,
	failureReason string,
	timestamps types.Timestamps,
) SynthesisWorkflow {
	return SynthesisWorkflow{
		id:            id,
		workspaceID:   workspaceID,
		initiatorID:   initiatorID,
		sourceLeafIDs: sourceLeafIDs,
		resultLeafID:  resultLeafID,
		status:        status,
		failureReason: failureReason,
		timestamps:    timestamps,
	}
}

// Start transitions the workflow from pending to processing.
func (w *SynthesisWorkflow) Start() error {
	if w.status != WorkflowPending {
		return fmt.Errorf("synthesis: can only start a pending workflow, current status: %s", w.status)
	}
	w.status = WorkflowProcessing
	w.timestamps.Touch()
	return nil
}

// Complete marks the workflow as successfully completed with the resulting leaf.
func (w *SynthesisWorkflow) Complete(resultLeafID types.LeafID) error {
	if w.status != WorkflowProcessing {
		return fmt.Errorf("synthesis: can only complete a processing workflow, current status: %s", w.status)
	}
	if resultLeafID.IsZero() {
		return fmt.Errorf("synthesis: result leaf ID is required")
	}
	w.status = WorkflowCompleted
	w.resultLeafID = resultLeafID
	w.timestamps.Touch()
	return nil
}

// Fail marks the workflow as failed with a reason.
func (w *SynthesisWorkflow) Fail(reason string) error {
	if w.status != WorkflowProcessing {
		return fmt.Errorf("synthesis: can only fail a processing workflow, current status: %s", w.status)
	}
	w.status = WorkflowFailed
	w.failureReason = reason
	w.timestamps.Touch()
	return nil
}

func (w SynthesisWorkflow) ID() types.ID[synthesisTag]     { return w.id }
func (w SynthesisWorkflow) WorkspaceID() types.WorkspaceID { return w.workspaceID }
func (w SynthesisWorkflow) InitiatorID() types.UserID      { return w.initiatorID }
func (w SynthesisWorkflow) SourceLeafIDs() []types.LeafID  { return w.sourceLeafIDs }
func (w SynthesisWorkflow) ResultLeafID() types.LeafID     { return w.resultLeafID }
func (w SynthesisWorkflow) Status() WorkflowStatus         { return w.status }
func (w SynthesisWorkflow) FailureReason() string          { return w.failureReason }
func (w SynthesisWorkflow) Timestamps() types.Timestamps   { return w.timestamps }

// SynthesisIDFrom creates a SynthesisID from a trusted database string.
func SynthesisIDFrom(raw string) types.ID[synthesisTag] { return types.IDFrom[synthesisTag](raw) }
