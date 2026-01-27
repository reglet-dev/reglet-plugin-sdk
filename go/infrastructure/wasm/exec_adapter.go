//go:build wasip1

package wasm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/reglet-dev/reglet-sdk/go/domain/ports"
	"github.com/reglet-dev/reglet-sdk/go/internal/abi"
	wasmcontext "github.com/reglet-dev/reglet-sdk/go/internal/wasmcontext"
)

// Compile-time interface compliance check
var _ ports.CommandRunner = (*ExecAdapter)(nil)

// ExecAdapter implements ports.CommandRunner for the WASM environment.
type ExecAdapter struct{}

// NewExecAdapter creates a new ExecAdapter.
func NewExecAdapter() *ExecAdapter {
	return &ExecAdapter{}
}

// Run executes a command on the host system.
func (a *ExecAdapter) Run(ctx context.Context, req ports.CommandRequest) (*ports.CommandResult, error) {
	// 1. Prepare wire request with context
	wireReq := entities.ExecRequest{
		Context: wasmcontext.ContextToWire(ctx),
		Command: req.Command,
		Args:    req.Args,
		Dir:     req.Dir,
		Env:     req.Env,
	}

	reqData, err := json.Marshal(wireReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 2. Send to host
	reqPacked := abi.PtrFromBytes(reqData)
	defer abi.DeallocatePacked(reqPacked)

	resPacked := host_exec_command(reqPacked)

	// 3. Read response
	resBytes := abi.BytesFromPtr(resPacked)
	if resBytes == nil {
		return nil, fmt.Errorf("host returned null response")
	}
	defer abi.DeallocatePacked(resPacked) // Free host-allocated response memory

	var wireRes entities.ExecResponse
	if err := json.Unmarshal(resBytes, &wireRes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 4. Handle errors
	if wireRes.Error != nil {
		return nil, wireRes.Error
	}

	return &ports.CommandResult{
		Stdout:     wireRes.Stdout,
		Stderr:     wireRes.Stderr,
		ExitCode:   wireRes.ExitCode,
		DurationMs: wireRes.DurationMs,
		IsTimeout:  wireRes.IsTimeout,
	}, nil
}
