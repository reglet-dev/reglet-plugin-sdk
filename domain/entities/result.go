// Package entities provides core domain entities for the SDK.
// These are general-purpose types used across all SDK operations.
// Domain-specific types like Evidence belong in consuming applications (e.g., Reglet).
package entities

import (
	"time"
)

// ResultStatus represents the outcome status of an SDK operation.
type ResultStatus string

const (
	// ResultStatusSuccess indicates the operation completed successfully.
	ResultStatusSuccess ResultStatus = "success"

	// ResultStatusFailure indicates the operation failed.
	ResultStatusFailure ResultStatus = "failure"

	// ResultStatusError indicates an error occurred during the operation.
	ResultStatusError ResultStatus = "error"
)

// Result represents the general-purpose outcome of an SDK operation.
// This is the SDK's return type for check functions - consuming applications
// like Reglet map Result to their domain-specific types (e.g., Evidence).
type Result struct {
	// Timestamp is when this result was created.
	// This is automatically set by the SDK when the result is returned.
	Timestamp time.Time `json:"timestamp"`

	// Data contains operation-specific result data.
	// The structure depends on the operation type (DNS, HTTP, TCP, etc.).
	Data map[string]any `json:"data,omitempty"`

	// Metadata contains execution metadata (timing, versions, etc.).
	Metadata *RunMetadata `json:"metadata,omitempty"`

	// Error contains structured error information if Status is Error.
	Error *ErrorDetail `json:"error,omitempty"`

	// Status indicates whether the operation succeeded, failed, or errored.
	Status ResultStatus `json:"status"`

	// Message provides a human-readable description of the result.
	Message string `json:"message,omitempty"`
}

// ResultSuccess creates a successful Result with the given message and data.
func ResultSuccess(message string, data map[string]any) Result {
	return Result{
		Status:  ResultStatusSuccess,
		Message: message,
		Data:    data,
	}
}

// ResultFailure creates a failure Result with the given message and data.
// Use this when the operation completed but the check did not pass.
func ResultFailure(message string, data map[string]any) Result {
	return Result{
		Status:  ResultStatusFailure,
		Message: message,
		Data:    data,
	}
}

// ResultError creates an error Result with the given error details.
// Use this when the operation could not complete due to an error.
func ResultError(err *ErrorDetail) Result {
	return Result{
		Status:  ResultStatusError,
		Message: err.Message,
		Error:   err,
	}
}

// WithMetadata returns a copy of the Result with the given metadata attached.
func (r Result) WithMetadata(m *RunMetadata) Result {
	r.Metadata = m
	return r
}

// IsSuccess returns true if the result indicates success.
func (r Result) IsSuccess() bool {
	return r.Status == ResultStatusSuccess
}

// IsFailure returns true if the result indicates failure.
func (r Result) IsFailure() bool {
	return r.Status == ResultStatusFailure
}

// IsError returns true if the result indicates an error.
func (r Result) IsError() bool {
	return r.Status == ResultStatusError
}

// Pointer-returning helpers for cleaner handler code

// ResultSuccessPtr creates a successful Result and returns a pointer to it.
func ResultSuccessPtr(message string, data map[string]any) *Result {
	r := ResultSuccess(message, data)
	return &r
}

// ResultFailurePtr creates a failure Result and returns a pointer to it.
func ResultFailurePtr(message string, data map[string]any) *Result {
	r := ResultFailure(message, data)
	return &r
}

// ResultErrorPtr creates an error Result and returns a pointer to it.
func ResultErrorPtr(errType, message string) *Result {
	r := ResultError(&ErrorDetail{Type: errType, Message: message})
	return &r
}
