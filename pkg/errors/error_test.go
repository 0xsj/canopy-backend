package errors

import (
	stderrors "errors"
	"testing"
)

func TestNew_Defaults(t *testing.T) {
	err := New("something failed")

	if err.ErrorKind() != KindUnknown {
		t.Errorf("kind = %v, want KindUnknown", err.ErrorKind())
	}
	if err.ErrorSeverity() != SeverityLow {
		t.Errorf("severity = %v, want SeverityLow", err.ErrorSeverity())
	}
	if err.ErrorCode() != "" {
		t.Errorf("code = %q, want empty", err.ErrorCode())
	}
	if err.ErrorOperation() != "" {
		t.Errorf("operation = %q, want empty", err.ErrorOperation())
	}
	if err.ErrorMetadata() == nil {
		t.Error("metadata is nil, want initialized map")
	}
	if err.ErrorStack() == nil {
		t.Error("stack is nil, want captured stack")
	}
	if err.Unwrap() != nil {
		t.Error("cause should be nil")
	}
}

func TestNewf_FormatsMessage(t *testing.T) {
	err := Newf("leaf %s not found in workspace %d", "abc", 42)

	want := "leaf abc not found in workspace 42"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestBuilder_WithKind(t *testing.T) {
	err := New("fail").WithKind(KindNotFound)
	if err.ErrorKind() != KindNotFound {
		t.Errorf("kind = %v, want KindNotFound", err.ErrorKind())
	}
}

func TestBuilder_WithCode(t *testing.T) {
	err := New("fail").WithCode("exploration_leaf_not_found")
	if err.ErrorCode() != "exploration_leaf_not_found" {
		t.Errorf("code = %q, want exploration_leaf_not_found", err.ErrorCode())
	}
}

func TestBuilder_WithSeverity(t *testing.T) {
	err := New("fail").WithSeverity(SeverityCritical)
	if err.ErrorSeverity() != SeverityCritical {
		t.Errorf("severity = %v, want SeverityCritical", err.ErrorSeverity())
	}
}

func TestBuilder_WithOperation(t *testing.T) {
	err := New("fail").WithOperation("exploration: create leaf")
	if err.ErrorOperation() != "exploration: create leaf" {
		t.Errorf("operation = %q, want exploration: create leaf", err.ErrorOperation())
	}
}

func TestBuilder_WithMetadata(t *testing.T) {
	err := New("fail").
		WithMetadata("leaf_id", "abc").
		WithMetadata("workspace_id", "xyz")

	meta := err.ErrorMetadata()
	if meta["leaf_id"] != "abc" {
		t.Errorf("metadata[leaf_id] = %v, want abc", meta["leaf_id"])
	}
	if meta["workspace_id"] != "xyz" {
		t.Errorf("metadata[workspace_id] = %v, want xyz", meta["workspace_id"])
	}
}

func TestBuilder_WithCause(t *testing.T) {
	cause := stderrors.New("db connection failed")
	err := New("query failed").WithCause(cause)

	if err.Unwrap() != cause {
		t.Error("Unwrap() did not return the cause")
	}
}

func TestBuilder_Chaining(t *testing.T) {
	err := New("something broke").
		WithKind(KindInternal).
		WithCode("exploration_internal").
		WithSeverity(SeverityHigh).
		WithOperation("exploration: process").
		WithMetadata("request_id", "req-123")

	if err.ErrorKind() != KindInternal {
		t.Errorf("kind = %v, want KindInternal", err.ErrorKind())
	}
	if err.ErrorCode() != "exploration_internal" {
		t.Errorf("code = %q, want exploration_internal", err.ErrorCode())
	}
	if err.ErrorSeverity() != SeverityHigh {
		t.Errorf("severity = %v, want SeverityHigh", err.ErrorSeverity())
	}
	if err.ErrorOperation() != "exploration: process" {
		t.Errorf("operation = %q, want exploration: process", err.ErrorOperation())
	}
	if err.ErrorMetadata()["request_id"] != "req-123" {
		t.Errorf("metadata[request_id] = %v, want req-123", err.ErrorMetadata()["request_id"])
	}
}

func TestError_SatisfiesInterface(t *testing.T) {
	var _ Error = New("test")
}

func TestError_SatisfiesStdError(t *testing.T) {
	var _ error = New("test")
}

func TestError_String_MessageOnly(t *testing.T) {
	err := New("something failed")
	if got := err.Error(); got != "something failed" {
		t.Errorf("Error() = %q, want %q", got, "something failed")
	}
}

func TestError_String_OperationAndMessage(t *testing.T) {
	err := New("leaf not found").WithOperation("exploration: get leaf")
	want := "exploration: get leaf: leaf not found"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestError_String_OperationAndCause(t *testing.T) {
	cause := stderrors.New("connection refused")
	err := &canopyError{
		operation: "exploration: get leaf",
		cause:     cause,
		metadata:  make(Metadata),
	}
	want := "exploration: get leaf: connection refused"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestError_String_AllParts(t *testing.T) {
	cause := stderrors.New("connection refused")
	err := New("query failed").
		WithOperation("exploration: get leaf").
		WithCause(cause)

	want := "exploration: get leaf: query failed: connection refused"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestError_String_CauseOnly(t *testing.T) {
	cause := stderrors.New("something")
	err := &canopyError{
		cause:    cause,
		metadata: make(Metadata),
	}
	if got := err.Error(); got != "something" {
		t.Errorf("Error() = %q, want %q", got, "something")
	}
}

func TestError_String_Empty(t *testing.T) {
	err := &canopyError{metadata: make(Metadata)}
	if got := err.Error(); got != "" {
		t.Errorf("Error() = %q, want empty", got)
	}
}
