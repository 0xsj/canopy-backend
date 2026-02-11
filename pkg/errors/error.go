package errors

import (
	"fmt"
	"strings"
)

// Error is the base error interface for Canopy.
// All domain, service, and adapter errors satisfy this contract.
type Error interface {
	error
	ErrorKind() Kind
	ErrorCode() Code
	ErrorSeverity() Severity
	ErrorOperation() string
	ErrorMetadata() map[string]any
	ErrorStack() StackTrace
	Unwrap() error
}

// Metadata is a key-value bag attached to an error for structured logging.
type Metadata map[string]any

// canopyError is the concrete implementation of Error.
type canopyError struct {
	message   string
	kind      Kind
	code      Code
	severity  Severity
	operation string
	metadata  Metadata
	stack     StackTrace
	cause     error
}

// New creates a new error with the given message.
// A stack trace is captured at the call site.
func New(message string) *canopyError {
	return &canopyError{
		message:  message,
		kind:     KindUnknown,
		severity: SeverityLow,
		metadata: make(Metadata),
		stack:    Capture(1),
	}
}

// Newf creates a new error with a formatted message.
func Newf(format string, args ...any) *canopyError {
	return &canopyError{
		message:  fmt.Sprintf(format, args...),
		kind:     KindUnknown,
		severity: SeverityLow,
		metadata: make(Metadata),
		stack:    Capture(1),
	}
}

// Builder methods — each returns *canopyError so calls can be chained.

func (e *canopyError) WithKind(k Kind) *canopyError {
	e.kind = k
	return e
}

func (e *canopyError) WithCode(c Code) *canopyError {
	e.code = c
	return e
}

func (e *canopyError) WithSeverity(s Severity) *canopyError {
	e.severity = s
	return e
}

func (e *canopyError) WithOperation(op string) *canopyError {
	e.operation = op
	return e
}

func (e *canopyError) WithMetadata(key string, value any) *canopyError {
	e.metadata[key] = value
	return e
}

func (e *canopyError) WithCause(err error) *canopyError {
	e.cause = err
	return e
}

// Error satisfies the error interface.
func (e *canopyError) Error() string {
	var b strings.Builder

	if e.operation != "" {
		b.WriteString(e.operation)
	}

	if e.message != "" {
		if b.Len() > 0 {
			b.WriteString(": ")
		}
		b.WriteString(e.message)
	}

	if e.cause != nil {
		if b.Len() > 0 {
			b.WriteString(": ")
		}
		b.WriteString(e.cause.Error())
	}

	return b.String()
}

// Interface implementation.

func (e *canopyError) ErrorKind() Kind               { return e.kind }
func (e *canopyError) ErrorCode() Code               { return e.code }
func (e *canopyError) ErrorSeverity() Severity       { return e.severity }
func (e *canopyError) ErrorOperation() string        { return e.operation }
func (e *canopyError) ErrorMetadata() map[string]any { return e.metadata }
func (e *canopyError) ErrorStack() StackTrace        { return e.stack }
func (e *canopyError) Unwrap() error                 { return e.cause }
