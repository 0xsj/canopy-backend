package errors

import (
	stderrors "errors"
	"maps"
)

// Wrap adds an operation and optional metadata to an existing error.
// If the cause is already a canopyError, its kind, code, and severity
// are preserved in the new wrapper — the original stays in the chain via Unwrap.
func Wrap(cause error, operation string) *canopyError {
	if cause == nil {
		return nil
	}

	wrapped := &canopyError{
		operation: operation,
		metadata:  make(Metadata),
		stack:     Capture(1),
		cause:     cause,
	}

	// Inherit classification from the cause if it's a canopy error.
	var ce Error
	if stderrors.As(cause, &ce) {
		wrapped.kind = ce.ErrorKind()
		wrapped.code = ce.ErrorCode()
		wrapped.severity = ce.ErrorSeverity()
	}

	return wrapped
}

// Is reports whether any error in the chain matches target.
// Delegates to the standard library.
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// As finds the first error in the chain that matches target
// and sets target to that value. Delegates to the standard library.
func As(err error, target any) bool {
	return stderrors.As(err, target)
}

// GetKind extracts the Kind from any error in the chain.
// Returns KindUnknown if no canopy error is found.
func GetKind(err error) Kind {
	var ce Error
	if stderrors.As(err, &ce) {
		return ce.ErrorKind()
	}
	return KindUnknown
}

// GetCode extracts the Code from any error in the chain.
// Returns an empty Code if no canopy error is found.
func GetCode(err error) Code {
	var ce Error
	if stderrors.As(err, &ce) {
		return ce.ErrorCode()
	}
	return ""
}

// GetSeverity extracts the Severity from any error in the chain.
// Returns SeverityLow if no canopy error is found.
func GetSeverity(err error) Severity {
	var ce Error
	if stderrors.As(err, &ce) {
		return ce.ErrorSeverity()
	}
	return SeverityLow
}

// Origin walks to the deepest canopy error in the chain — the point
// where the error was first created. Returns nil if no canopy error is found.
func Origin(err error) Error {
	var origin Error
	current := err
	for current != nil {
		var ce Error
		if stderrors.As(current, &ce) {
			origin = ce
			current = ce.Unwrap()
		} else {
			break
		}
	}
	return origin
}

// OriginFrame returns the call site (file:line) where the error was first
// created. Returns an empty Frame if no canopy error or stack is found.
func OriginFrame(err error) Frame {
	o := Origin(err)
	if o == nil {
		return Frame{}
	}
	stack := o.ErrorStack()
	if len(stack) == 0 {
		return Frame{}
	}
	return stack[0]
}

// CollectMetadata walks the error chain and merges all metadata.
// Values from outer (more recent) errors take precedence.
func CollectMetadata(err error) Metadata {
	collected := make(Metadata)

	// Walk the chain from outermost to innermost, collecting metadata.
	// Inner values are set first, outer values overwrite — giving precedence to the outer layer.
	var chain []Metadata
	current := err
	for current != nil {
		var ce Error
		if stderrors.As(current, &ce) {
			chain = append(chain, ce.ErrorMetadata())
			current = ce.Unwrap()
		} else {
			break
		}
	}

	// Apply inner-first so outer takes precedence.
	for i := len(chain) - 1; i >= 0; i-- {
		maps.Copy(collected, chain[i])
	}

	return collected
}
