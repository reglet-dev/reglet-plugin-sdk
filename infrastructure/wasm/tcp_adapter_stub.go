//go:build !wasip1

package wasm

import (
	"context"

	"github.com/reglet-dev/reglet-sdk/domain/ports"
)

// Compile-time interface compliance check
var _ ports.TCPDialer = (*TCPAdapter)(nil)

// TCPAdapter implements ports.TCPDialer for the native environment (stub).
// This allows compiling the SDK on non-WASM targets (e.g. for running tests).
type TCPAdapter struct{}

// NewTCPAdapter creates a new TCP adapter stub.
func NewTCPAdapter() *TCPAdapter {
	return &TCPAdapter{}
}

// Dial panics because real WASM calls are not supported natively.
func (a *TCPAdapter) Dial(ctx context.Context, address string) (ports.TCPConnection, error) {
	panic("WASM TCP adapter not available in native build. Use WithTCPDialer() to inject a mock.")
}

// DialWithTimeout panics because real WASM calls are not supported natively.
func (a *TCPAdapter) DialWithTimeout(ctx context.Context, address string, timeoutMs int) (ports.TCPConnection, error) {
	panic("WASM TCP adapter not available in native build. Use WithTCPDialer() to inject a mock.")
}

// DialSecure panics because real WASM calls are not supported natively.
func (a *TCPAdapter) DialSecure(ctx context.Context, address string, timeoutMs int, tls bool) (ports.TCPConnection, error) {
	panic("WASM TCP adapter not available in native build. Use WithTCPDialer() to inject a mock.")
}
