//go:build !wasip1

package wasm

import (
	"context"

	"github.com/reglet-dev/reglet-sdk/domain/ports"
)

// Compile-time interface compliance check
var _ ports.CommandRunner = (*ExecAdapter)(nil)

// ExecAdapter stub for native builds.
type ExecAdapter struct{}

// NewExecAdapter creates a new ExecAdapter stub.
func NewExecAdapter() *ExecAdapter {
	return &ExecAdapter{}
}

// Run panics because WASM execution is not available natively.
func (a *ExecAdapter) Run(ctx context.Context, req ports.CommandRequest) (*ports.CommandResult, error) {
	panic("WASM Exec adapter not available in native build. Use WithCommandRunner() to inject a mock.")
}
