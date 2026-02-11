package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// Frame represents a single frame in a stack trace.
type Frame struct {
	Function string
	File     string
	Line     int
}

func (f Frame) String() string {
	return fmt.Sprintf("%s\n\t%s:%d", f.Function, f.File, f.Line)
}

// StackTrace is an ordered list of frames from the call site upward.
type StackTrace []Frame

func (st StackTrace) String() string {
	var b strings.Builder
	for i, frame := range st {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(frame.String())
	}
	return b.String()
}

// Capture collects a stack trace, skipping the specified number of
// frames above the caller. A skip of 0 starts at the caller of Capture.
func Capture(skip int) StackTrace {
	const maxDepth = 32
	var pcs [maxDepth]uintptr

	// +2 accounts for runtime.Callers and Capture itself.
	n := runtime.Callers(skip+2, pcs[:])
	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(pcs[:n])
	stack := make(StackTrace, 0, n)

	for {
		frame, more := frames.Next()
		stack = append(stack, Frame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		})
		if !more {
			break
		}
	}

	return stack
}
