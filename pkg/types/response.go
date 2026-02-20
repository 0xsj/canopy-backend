package types

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// FieldError describes a validation failure on a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// APIError is the error variant of the response envelope.
type APIError struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

// Response is a discriminated union: either Data is set (success)
// or Error is set (failure), never both.
//
// Use the constructors OK, Fail, and ValidationFail to create responses.
// Direct struct construction is discouraged — use Validate() to check
// the invariant if you receive a Response from untrusted code.
type Response[T any] struct {
	Data  *T        `json:"data,omitempty"`
	Error *APIError `json:"error,omitempty"`
}

// OK creates a success response with the given data.
func OK[T any](data T) Response[T] {
	return Response[T]{Data: &data}
}

// Fail creates an error response with the given code and message.
func Fail[T any](code, message string) Response[T] {
	return Response[T]{
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	}
}

// FailWithDetails creates an error response with field-level validation details.
func FailWithDetails[T any](code, message string, details []FieldError) Response[T] {
	return Response[T]{
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// Validate checks the discriminated union invariant.
// Returns an error if both Data and Error are set, or if neither is set.
func (r Response[T]) Validate() error {
	hasData := r.Data != nil
	hasError := r.Error != nil

	if hasData && hasError {
		return fmt.Errorf("types: invalid response: both data and error are set")
	}
	if !hasData && !hasError {
		return fmt.Errorf("types: invalid response: neither data nor error is set")
	}
	return nil
}

// IsError reports whether this is an error response.
func (r Response[T]) IsError() bool {
	return r.Error != nil
}

// StatusCode maps the response to an HTTP status code.
// Success responses return the provided success code.
// Error responses derive the status from the error code pattern,
// defaulting to 500 if no mapping exists.
func (r Response[T]) StatusCode(successCode int) int {
	if !r.IsError() {
		return successCode
	}
	return http.StatusInternalServerError
}

// Write serializes the response as JSON and writes it to the http.ResponseWriter.
func Write[T any](w http.ResponseWriter, statusCode int, resp Response[T]) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

// WriteOK is a shorthand for writing a 200 success response.
func WriteOK[T any](w http.ResponseWriter, data T) {
	Write(w, http.StatusOK, OK(data))
}

// WriteCreated is a shorthand for writing a 201 success response.
func WriteCreated[T any](w http.ResponseWriter, data T) {
	Write(w, http.StatusCreated, OK(data))
}

// WriteAccepted is a shorthand for writing a 202 accepted response.
func WriteAccepted[T any](w http.ResponseWriter, data T) {
	Write(w, http.StatusAccepted, OK(data))
}

// WriteError writes an error response with the given HTTP status code.
func WriteError(w http.ResponseWriter, statusCode int, code, message string) {
	Write(w, statusCode, Fail[struct{}](code, message))
}

// WriteValidationError writes a 400 response with field-level details.
func WriteValidationError(w http.ResponseWriter, details []FieldError) {
	Write(w, http.StatusBadRequest, FailWithDetails[struct{}](
		"validation_failed",
		"one or more fields are invalid",
		details,
	))
}
