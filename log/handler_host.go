//go:build !wasip1

// Package log provides structure logging (slog) adapted for Reglet SDK's WASM environment.
package log

import (
	"context"
	"fmt"
	"log/slog"
)

// Handle for non-WASM builds (e.g., host tests).
// This is a stub to allow the code to compile and basic tests to run.
func (h *WasmLogHandler) Handle(_ context.Context, record slog.Record) error {
	// In a real host environment (not test), we might want to fallback to stdout.
	// For now, this is just to satisfy the interface for tests.
	fmt.Printf("[HOST-STUB] Level=%s Msg=%q\n", record.Level, record.Message)
	return nil
}
