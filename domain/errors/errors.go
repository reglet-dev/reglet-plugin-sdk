// Package errors provides domain-specific error types for the SDK.
// All error types support error unwrapping via errors.As() and errors.Is().
package errors

import (
	stdErrors "errors"
	"fmt"
	"time"

	"github.com/reglet-dev/reglet-sdk/domain/entities"
)

// ErrorDetail is an alias to entities.ErrorDetail for backward compatibility/convenience.
type ErrorDetail = entities.ErrorDetail

// DetailedError is an interface for custom error types that can convert themselves
// to a structured ErrorDetail. This follows the Open/Closed Principle - new error
// types only need to implement this interface without modifying ToErrorDetail.
type DetailedError interface {
	error
	ToErrorDetail() *entities.ErrorDetail
}

// ToErrorDetail converts a Go error to our structured ErrorDetail.
// This function recognizes custom error types and categorizes them appropriately.
func ToErrorDetail(err error) *entities.ErrorDetail {
	if err == nil {
		return nil
	}

	// If the error is already a *ErrorDetail (entity), use it directly.
	var e *entities.ErrorDetail
	if stdErrors.As(err, &e) {
		return e
	}

	// Check if error matches domain errors.DetailedError interface
	var de DetailedError
	if stdErrors.As(err, &de) {
		return de.ToErrorDetail()
	}

	// Generic error - categorize as internal
	return &entities.ErrorDetail{
		Message: err.Error(),
		Type:    "internal",
	}
}

// NetworkError represents a network operation failure.
type NetworkError struct {
	Err       error
	Operation string
	Target    string
}

func (e *NetworkError) Error() string {
	if e.Target != "" {
		return fmt.Sprintf("network %s failed for %s: %v", e.Operation, e.Target, e.Err)
	}
	return fmt.Sprintf("network %s failed: %v", e.Operation, e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

// ToErrorDetail implements DetailedError.
func (e *NetworkError) ToErrorDetail() *entities.ErrorDetail {
	return &entities.ErrorDetail{Message: e.Error(), Type: "network", Code: e.Operation}
}

// TimeoutError represents a timeout during an operation.
type TimeoutError struct {
	Operation string
	Target    string
	Duration  time.Duration
}

func (e *TimeoutError) Error() string {
	if e.Target != "" {
		return fmt.Sprintf("%s timeout after %v (target: %s)", e.Operation, e.Duration, e.Target)
	}
	return fmt.Sprintf("%s timeout after %v", e.Operation, e.Duration)
}

func (e *TimeoutError) Timeout() bool {
	return true
}

// ToErrorDetail implements DetailedError.
func (e *TimeoutError) ToErrorDetail() *entities.ErrorDetail {
	return &entities.ErrorDetail{Message: e.Error(), Type: "timeout", Code: e.Operation, IsTimeout: true}
}

// CapabilityError represents a capability check failure.
type CapabilityError struct {
	Required string // Required capability (e.g., "network:outbound", "exec")
	Pattern  string // Optional: specific pattern that was denied
}

func (e *CapabilityError) Error() string {
	if e.Pattern != "" {
		return fmt.Sprintf("missing capability: %s (pattern: %s)", e.Required, e.Pattern)
	}
	return fmt.Sprintf("missing capability: %s", e.Required)
}

// ToErrorDetail implements DetailedError.
func (e *CapabilityError) ToErrorDetail() *entities.ErrorDetail {
	return &entities.ErrorDetail{Message: e.Error(), Type: "capability", Code: e.Required}
}

// ConfigError represents a configuration validation error.
type ConfigError struct {
	Err   error
	Field string
}

func (e *ConfigError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("config validation failed for field '%s': %v", e.Field, e.Err)
	}
	return fmt.Sprintf("config validation failed: %v", e.Err)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// ToErrorDetail implements DetailedError.
func (e *ConfigError) ToErrorDetail() *entities.ErrorDetail {
	return &entities.ErrorDetail{Message: e.Error(), Type: "config", Code: e.Field}
}

// ExecError represents a command execution error.
type ExecError struct {
	Err      error
	Command  string
	Stderr   string
	ExitCode int
}

func (e *ExecError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("failed to execute '%s': %v", e.Command, e.Err)
	}
	if e.Stderr != "" {
		return fmt.Sprintf("command '%s' exited with code %d: %s", e.Command, e.ExitCode, e.Stderr)
	}
	return fmt.Sprintf("command '%s' exited with code %d", e.Command, e.ExitCode)
}

func (e *ExecError) Unwrap() error {
	return e.Err
}

// ToErrorDetail implements DetailedError.
func (e *ExecError) ToErrorDetail() *entities.ErrorDetail {
	return &entities.ErrorDetail{Message: e.Error(), Type: "exec", Code: fmt.Sprintf("exit_%d", e.ExitCode)}
}

// DNSError represents a DNS lookup failure.
type DNSError struct {
	Err        error
	Hostname   string
	RecordType string
	Nameserver string
}

func (e *DNSError) Error() string {
	if e.Nameserver != "" {
		return fmt.Sprintf("dns lookup for %s (%s) via %s failed: %v",
			e.Hostname, e.RecordType, e.Nameserver, e.Err)
	}
	return fmt.Sprintf("dns lookup for %s (%s) failed: %v", e.Hostname, e.RecordType, e.Err)
}

func (e *DNSError) Unwrap() error {
	return e.Err
}

func (e *DNSError) Timeout() bool {
	if t, ok := e.Err.(interface{ Timeout() bool }); ok {
		return t.Timeout()
	}
	return false
}

// ToErrorDetail implements DetailedError.
func (e *DNSError) ToErrorDetail() *entities.ErrorDetail {
	detail := &entities.ErrorDetail{Message: e.Error(), Type: "network", Code: "dns_" + e.RecordType}
	if e.Timeout() {
		detail.Type = "timeout"
		detail.IsTimeout = true
	}
	return detail
}

// HTTPError represents an HTTP request failure.
type HTTPError struct {
	Err        error
	Method     string
	URL        string
	StatusCode int
}

func (e *HTTPError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("http %s %s failed with status %d: %v", e.Method, e.URL, e.StatusCode, e.Err)
	}
	return fmt.Sprintf("http %s %s failed: %v", e.Method, e.URL, e.Err)
}

func (e *HTTPError) Unwrap() error {
	return e.Err
}

func (e *HTTPError) Timeout() bool {
	if t, ok := e.Err.(interface{ Timeout() bool }); ok {
		return t.Timeout()
	}
	return false
}

// ToErrorDetail implements DetailedError.
func (e *HTTPError) ToErrorDetail() *entities.ErrorDetail {
	detail := &entities.ErrorDetail{Message: e.Error(), Type: "network", Code: fmt.Sprintf("http_%d", e.StatusCode)}
	if e.Timeout() {
		detail.Type = "timeout"
		detail.IsTimeout = true
	}
	return detail
}

// TCPError represents a TCP connection failure.
type TCPError struct {
	Err     error
	Network string
	Address string
}

func (e *TCPError) Error() string {
	return fmt.Sprintf("tcp connect to %s (%s) failed: %v", e.Address, e.Network, e.Err)
}

func (e *TCPError) Unwrap() error {
	return e.Err
}

func (e *TCPError) Timeout() bool {
	if t, ok := e.Err.(interface{ Timeout() bool }); ok {
		return t.Timeout()
	}
	return false
}

// ToErrorDetail implements DetailedError.
func (e *TCPError) ToErrorDetail() *entities.ErrorDetail {
	detail := &entities.ErrorDetail{Message: e.Error(), Type: "network", Code: "tcp_connect"}
	if e.Timeout() {
		detail.Type = "timeout"
		detail.IsTimeout = true
	}
	return detail
}

// SchemaError represents a schema generation or validation error.
type SchemaError struct {
	Err  error
	Type string
}

func (e *SchemaError) Error() string {
	if e.Type != "" {
		return fmt.Sprintf("schema error for type %s: %v", e.Type, e.Err)
	}
	return fmt.Sprintf("schema error: %v", e.Err)
}

func (e *SchemaError) Unwrap() error {
	return e.Err
}

// ToErrorDetail implements DetailedError.
func (e *SchemaError) ToErrorDetail() *entities.ErrorDetail {
	return &entities.ErrorDetail{Message: e.Error(), Type: "validation", Code: "schema"}
}

// MemoryError represents a memory allocation failure.
type MemoryError struct {
	Requested int // Requested allocation size
	Current   int // Current total allocated
	Limit     int // Maximum allowed
}

func (e *MemoryError) Error() string {
	return fmt.Sprintf("memory allocation failed: requested %d bytes, current %d bytes, limit %d bytes",
		e.Requested, e.Current, e.Limit)
}

// ToErrorDetail implements DetailedError.
func (e *MemoryError) ToErrorDetail() *entities.ErrorDetail {
	return &entities.ErrorDetail{Message: e.Error(), Type: "internal", Code: "memory_limit"}
}

// WireFormatError represents a wire format encoding/decoding error.
type WireFormatError struct {
	Err       error
	Operation string
	Type      string
}

func (e *WireFormatError) Error() string {
	return fmt.Sprintf("wire format %s failed for %s: %v", e.Operation, e.Type, e.Err)
}

func (e *WireFormatError) Unwrap() error {
	return e.Err
}

// ToErrorDetail implements DetailedError.
func (e *WireFormatError) ToErrorDetail() *entities.ErrorDetail {
	return &entities.ErrorDetail{Message: e.Error(), Type: "internal", Code: "wire_format"}
}
