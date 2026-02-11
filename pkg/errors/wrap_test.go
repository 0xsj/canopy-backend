package errors

import (
	stderrors "errors"
	"testing"
)

func TestWrap_InheritsClassification(t *testing.T) {
	cause := New("leaf not found").
		WithKind(KindNotFound).
		WithCode("exploration_leaf_not_found").
		WithSeverity(SeverityLow)

	wrapped := Wrap(cause, "exploration: get leaf")

	if wrapped.ErrorKind() != KindNotFound {
		t.Errorf("kind = %v, want KindNotFound", wrapped.ErrorKind())
	}
	if wrapped.ErrorCode() != "exploration_leaf_not_found" {
		t.Errorf("code = %v, want exploration_leaf_not_found", wrapped.ErrorCode())
	}
	if wrapped.ErrorSeverity() != SeverityLow {
		t.Errorf("severity = %v, want SeverityLow", wrapped.ErrorSeverity())
	}
}

func TestWrap_PreservesChain(t *testing.T) {
	cause := New("leaf not found").WithKind(KindNotFound)
	wrapped := Wrap(cause, "exploration: get leaf")

	if wrapped.Unwrap() != cause {
		t.Error("Unwrap() did not return the original cause")
	}
}

func TestWrap_NilReturnsNil(t *testing.T) {
	if got := Wrap(nil, "some operation"); got != nil {
		t.Errorf("Wrap(nil) = %v, want nil", got)
	}
}

func TestWrap_NonCanopyError(t *testing.T) {
	cause := stderrors.New("raw database error")
	wrapped := Wrap(cause, "exploration: query")

	if wrapped.ErrorKind() != KindUnknown {
		t.Errorf("kind = %v, want KindUnknown for non-canopy cause", wrapped.ErrorKind())
	}

	want := "exploration: query: raw database error"
	if got := wrapped.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestWrap_ErrorString(t *testing.T) {
	cause := New("leaf not found")
	wrapped := Wrap(cause, "exploration: get leaf")

	want := "exploration: get leaf: leaf not found"
	if got := wrapped.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestWrap_MetadataIsIndependent(t *testing.T) {
	cause := New("fail").WithMetadata("leaf_id", "abc")
	wrapped := Wrap(cause, "service").WithMetadata("workspace_id", "xyz")

	if _, ok := wrapped.ErrorMetadata()["leaf_id"]; ok {
		t.Error("wrapper should not have cause's metadata directly")
	}
	if wrapped.ErrorMetadata()["workspace_id"] != "xyz" {
		t.Error("wrapper should have its own metadata")
	}
}

func TestWrap_CapturesStack(t *testing.T) {
	cause := New("fail")
	wrapped := Wrap(cause, "service")

	if wrapped.ErrorStack() == nil {
		t.Error("wrapped error should have its own stack trace")
	}
}

// --- Is / As ---

func TestIs_MatchesThroughChain(t *testing.T) {
	sentinel := New("not found").WithKind(KindNotFound)
	wrapped := Wrap(sentinel, "service layer")
	double := Wrap(wrapped, "handler layer")

	if !Is(double, sentinel) {
		t.Error("Is should find sentinel through wrapped chain")
	}
}

func TestIs_NoMatchOnDifferentError(t *testing.T) {
	err := New("not found")
	other := New("conflict")

	if Is(err, other) {
		t.Error("Is should not match different errors")
	}
}

func TestAs_ExtractsCanopyError(t *testing.T) {
	cause := New("fail").WithKind(KindInternal)
	wrapped := Wrap(cause, "service")

	var ce Error
	if !As(wrapped, &ce) {
		t.Fatal("As should find a canopy Error in the chain")
	}
	if ce.ErrorKind() != KindNotFound && ce.ErrorKind() != KindInternal {
		// The outermost canopy error (wrapped) inherited KindInternal.
		t.Errorf("kind = %v, want KindInternal", ce.ErrorKind())
	}
}

// --- GetKind / GetCode / GetSeverity ---

func TestGetKind_FromCanopyError(t *testing.T) {
	err := New("fail").WithKind(KindConflict)
	if got := GetKind(err); got != KindConflict {
		t.Errorf("GetKind = %v, want KindConflict", got)
	}
}

func TestGetKind_FromWrapped(t *testing.T) {
	cause := New("fail").WithKind(KindNotFound)
	wrapped := Wrap(cause, "service")

	if got := GetKind(wrapped); got != KindNotFound {
		t.Errorf("GetKind = %v, want KindNotFound", got)
	}
}

func TestGetKind_FromStdError(t *testing.T) {
	err := stderrors.New("plain error")
	if got := GetKind(err); got != KindUnknown {
		t.Errorf("GetKind = %v, want KindUnknown", got)
	}
}

func TestGetCode_FromCanopyError(t *testing.T) {
	err := New("fail").WithCode("test_code")
	if got := GetCode(err); got != "test_code" {
		t.Errorf("GetCode = %v, want test_code", got)
	}
}

func TestGetCode_FromStdError(t *testing.T) {
	err := stderrors.New("plain")
	if got := GetCode(err); got != "" {
		t.Errorf("GetCode = %q, want empty", got)
	}
}

func TestGetSeverity_FromCanopyError(t *testing.T) {
	err := New("fail").WithSeverity(SeverityCritical)
	if got := GetSeverity(err); got != SeverityCritical {
		t.Errorf("GetSeverity = %v, want SeverityCritical", got)
	}
}

func TestGetSeverity_FromStdError(t *testing.T) {
	err := stderrors.New("plain")
	if got := GetSeverity(err); got != SeverityLow {
		t.Errorf("GetSeverity = %v, want SeverityLow", got)
	}
}

// --- CollectMetadata ---

func TestCollectMetadata_MergesChain(t *testing.T) {
	cause := New("fail").
		WithMetadata("leaf_id", "abc").
		WithMetadata("user_id", "u1")

	wrapped := Wrap(cause, "service").
		WithMetadata("workspace_id", "ws1")

	meta := CollectMetadata(wrapped)

	if meta["leaf_id"] != "abc" {
		t.Errorf("metadata[leaf_id] = %v, want abc", meta["leaf_id"])
	}
	if meta["user_id"] != "u1" {
		t.Errorf("metadata[user_id] = %v, want u1", meta["user_id"])
	}
	if meta["workspace_id"] != "ws1" {
		t.Errorf("metadata[workspace_id] = %v, want ws1", meta["workspace_id"])
	}
}

func TestCollectMetadata_OuterTakesPrecedence(t *testing.T) {
	cause := New("fail").WithMetadata("key", "inner")
	wrapped := Wrap(cause, "service").WithMetadata("key", "outer")

	meta := CollectMetadata(wrapped)

	if meta["key"] != "outer" {
		t.Errorf("metadata[key] = %v, want outer (outer takes precedence)", meta["key"])
	}
}

func TestCollectMetadata_StdErrorReturnsEmpty(t *testing.T) {
	err := stderrors.New("plain")
	meta := CollectMetadata(err)

	if len(meta) != 0 {
		t.Errorf("metadata should be empty for std error, got %v", meta)
	}
}

func TestCollectMetadata_NilReturnsEmpty(t *testing.T) {
	meta := CollectMetadata(nil)
	if len(meta) != 0 {
		t.Errorf("metadata should be empty for nil, got %v", meta)
	}
}
