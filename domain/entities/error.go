package entities

import "github.com/reglet-dev/reglet-abi/hostfunc"

// ErrorDetail provides structured error information.
type ErrorDetail = hostfunc.ErrorDetail

// NewErrorDetail creates a new ErrorDetail with the given type and message.
func NewErrorDetail(errorType, message string) *ErrorDetail {
	return hostfunc.NewErrorDetail(errorType, message)
}
