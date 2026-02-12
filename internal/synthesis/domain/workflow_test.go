package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func TestNewSynthesisWorkflow_ValidInput(t *testing.T) {
	wf, err := NewSynthesisWorkflow(
		types.NewWorkspaceID(),
		types.NewUserID(),
		[]types.LeafID{types.NewLeafID(), types.NewLeafID()},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if wf.Status() != WorkflowPending {
		t.Errorf("status = %q, want %q", wf.Status(), WorkflowPending)
	}
}

func TestNewSynthesisWorkflow_RejectsFewerThanTwoSources(t *testing.T) {
	_, err := NewSynthesisWorkflow(
		types.NewWorkspaceID(),
		types.NewUserID(),
		[]types.LeafID{types.NewLeafID()},
	)
	if err == nil {
		t.Fatal("expected error for < 2 source leaves")
	}
}

func TestWorkflow_StateMachine_HappyPath(t *testing.T) {
	wf, _ := NewSynthesisWorkflow(
		types.NewWorkspaceID(), types.NewUserID(),
		[]types.LeafID{types.NewLeafID(), types.NewLeafID()},
	)

	if err := wf.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if wf.Status() != WorkflowProcessing {
		t.Errorf("status = %q, want %q", wf.Status(), WorkflowProcessing)
	}

	resultID := types.NewLeafID()
	if err := wf.Complete(resultID); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if wf.Status() != WorkflowCompleted {
		t.Errorf("status = %q, want %q", wf.Status(), WorkflowCompleted)
	}
	if wf.ResultLeafID() != resultID {
		t.Errorf("result leaf ID = %v, want %v", wf.ResultLeafID(), resultID)
	}
}

func TestWorkflow_StateMachine_Failure(t *testing.T) {
	wf, _ := NewSynthesisWorkflow(
		types.NewWorkspaceID(), types.NewUserID(),
		[]types.LeafID{types.NewLeafID(), types.NewLeafID()},
	)

	_ = wf.Start()
	if err := wf.Fail("LLM timeout"); err != nil {
		t.Fatalf("fail: %v", err)
	}
	if wf.Status() != WorkflowFailed {
		t.Errorf("status = %q, want %q", wf.Status(), WorkflowFailed)
	}
	if wf.FailureReason() != "LLM timeout" {
		t.Errorf("failure reason = %q, want %q", wf.FailureReason(), "LLM timeout")
	}
}

func TestWorkflow_RejectsStartOnNonPending(t *testing.T) {
	wf, _ := NewSynthesisWorkflow(
		types.NewWorkspaceID(), types.NewUserID(),
		[]types.LeafID{types.NewLeafID(), types.NewLeafID()},
	)
	_ = wf.Start()

	err := wf.Start()
	if err == nil {
		t.Fatal("expected error for double start")
	}
}

func TestWorkflow_RejectsCompleteOnPending(t *testing.T) {
	wf, _ := NewSynthesisWorkflow(
		types.NewWorkspaceID(), types.NewUserID(),
		[]types.LeafID{types.NewLeafID(), types.NewLeafID()},
	)

	err := wf.Complete(types.NewLeafID())
	if err == nil {
		t.Fatal("expected error for completing a pending workflow")
	}
}
