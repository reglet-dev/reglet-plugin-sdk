package entities

import "fmt"

// ErrorDetail provides structured error information.
// Used across SDK operations and as wire protocol error format.
// Error Types: "network", "timeout", "config", "panic", "capability", "validation", "internal"
type ErrorDetail struct {
	// Wrapped contains a wrapped error for error chains.
	Wrapped *ErrorDetail `json:"wrapped,omitempty"`

	// Details contains additional error context.
	Details map[string]any `json:"details,omitempty"`

	// Message is a human-readable error description.
	Message string `json:"message"`

	// Type categorizes the error.
	Type string `json:"type"`

	// Code is a machine-readable error code.
	Code string `json:"code"`

	// Stack contains the stack trace for panic errors.
	Stack []byte `json:"stack,omitempty"`

	// IsTimeout indicates if this was a timeout error.
	IsTimeout bool `json:"is_timeout,omitempty"`

	// IsNotFound indicates if this was a "not found" error.
	IsNotFound bool `json:"is_not_found,omitempty"`
}

// Error implements the error interface.
func (e *ErrorDetail) Error() string {
	if e == nil {
		return ""
	}
	msg := e.Message
	if e.Type != "" && e.Type != "internal" {
		msg = fmt.Sprintf("%s: %s", e.Type, msg)
	}
	if e.Code != "" {
		msg = fmt.Sprintf("%s [%s]", msg, e.Code)
	}
	if e.Wrapped != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.Wrapped.Error())
	}
	return msg
}

// NewErrorDetail creates a new ErrorDetail with the given type and message.
func NewErrorDetail(errorType, message string) *ErrorDetail {
	return &ErrorDetail{
		Type:    errorType,
		Message: message,
	}
}

// WithDetails returns a copy of the ErrorDetail with the given details attached.
func (e *ErrorDetail) WithDetails(details map[string]any) *ErrorDetail {
	e.Details = details
	return e
}

// WithCode returns a copy of the ErrorDetail with the given code attached.
func (e *ErrorDetail) WithCode(code string) *ErrorDetail {
	e.Code = code
	return e
}
