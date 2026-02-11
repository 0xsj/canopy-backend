package errors

import (
	"strings"
	"testing"
)

func TestCapture_ReturnsFrames(t *testing.T) {
	stack := Capture(0)
	if len(stack) == 0 {
		t.Fatal("Capture returned no frames")
	}

	// First frame should be this test function.
	first := stack[0]
	if !strings.Contains(first.Function, "TestCapture_ReturnsFrames") {
		t.Errorf("first frame function = %q, want it to contain TestCapture_ReturnsFrames", first.Function)
	}

	if !strings.HasSuffix(first.File, "stack_test.go") {
		t.Errorf("first frame file = %q, want it to end with stack_test.go", first.File)
	}

	if first.Line == 0 {
		t.Error("first frame line = 0, want non-zero")
	}
}

func TestCapture_SkipsFrames(t *testing.T) {
	stack := helperCapture()
	if len(stack) == 0 {
		t.Fatal("Capture returned no frames")
	}

	// With skip=1 from helperCapture, first frame should be this test, not helperCapture.
	first := stack[0]
	if strings.Contains(first.Function, "helperCapture") {
		t.Errorf("first frame should not be helperCapture, got %q", first.Function)
	}
}

func helperCapture() StackTrace {
	return Capture(1)
}

func TestFrame_String(t *testing.T) {
	f := Frame{
		Function: "main.doSomething",
		File:     "/app/main.go",
		Line:     42,
	}

	want := "main.doSomething\n\t/app/main.go:42"
	if got := f.String(); got != want {
		t.Errorf("Frame.String() = %q, want %q", got, want)
	}
}

func TestStackTrace_String(t *testing.T) {
	st := StackTrace{
		{Function: "a.First", File: "/a.go", Line: 1},
		{Function: "b.Second", File: "/b.go", Line: 2},
	}

	got := st.String()
	if !strings.Contains(got, "a.First") {
		t.Error("StackTrace.String() missing first frame")
	}
	if !strings.Contains(got, "b.Second") {
		t.Error("StackTrace.String() missing second frame")
	}
}

func TestStackTrace_String_Empty(t *testing.T) {
	var st StackTrace
	if got := st.String(); got != "" {
		t.Errorf("empty StackTrace.String() = %q, want empty", got)
	}
}
