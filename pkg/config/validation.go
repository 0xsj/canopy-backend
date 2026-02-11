package config

import (
	"fmt"
	"slices"
	"strings"
)

// Errors collects multiple validation errors and reports them together.
type Errors struct {
	errs []string
}

// Add records a validation error.
func (e *Errors) Add(format string, args ...any) {
	e.errs = append(e.errs, fmt.Sprintf(format, args...))
}

// Required checks that a string field is not empty.
func (e *Errors) Required(name, value string) {
	if value == "" {
		e.Add("%s is required", name)
	}
}

// OneOf checks that a value is one of the allowed options.
func (e *Errors) OneOf(name, value string, allowed []string) {
	if !slices.Contains(allowed, value) {
		e.Add("%s must be one of [%s], got %q", name, strings.Join(allowed, ", "), value)
	}
}

// PortRange checks that a port number is in the valid range.
func (e *Errors) PortRange(name string, port int) {
	if port < 1 || port > 65535 {
		e.Add("%s must be between 1 and 65535, got %d", name, port)
	}
}

// Positive checks that an integer is greater than zero.
func (e *Errors) Positive(name string, value int) {
	if value <= 0 {
		e.Add("%s must be positive, got %d", name, value)
	}
}

// Err returns a combined error if any validations failed, nil otherwise.
func (e *Errors) Err() error {
	if len(e.errs) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(e.errs, "; "))
}
