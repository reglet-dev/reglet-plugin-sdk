//go:build wasip1

// Package plugin provides the core Plugin interface and WASM export lifecycle.
package plugin

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug" // For stack traces in panic recovery
	"time"          // For time.Now()

	"github.com/reglet-dev/reglet-plugin-sdk/domain/entities"
	"github.com/reglet-dev/reglet-plugin-sdk/domain/errors"
	"github.com/reglet-dev/reglet-plugin-sdk/internal/abi"
	wasmcontext "github.com/reglet-dev/reglet-plugin-sdk/internal/wasmcontext"
	_ "github.com/reglet-dev/reglet-plugin-sdk/log" // Initialize WASM logging handler
)

// Define the functions that will be exported to the WASM host.
// These functions perform panic recovery and ABI translation.

//go:wasmexport _manifest
func _manifest() uint64 {
	return handleExportedCall(func() (interface{}, error) {
		if userPlugin == nil {
			return nil, fmt.Errorf("plugin not registered")
		}
		// Use current context or create one with timeout
		ctx := wasmcontext.GetCurrentContext()
		manifest, err := userPlugin.Manifest(ctx)
		if err != nil {
			return nil, err
		}
		// Auto-populate SDK version for manifest
		manifest.SDKVersion = Version
		return manifest, nil
	})
}

//go:wasmexport _observe
func _observe(configPtr uint32, configLen uint32) uint64 {
	return handleExportedCall(func() (interface{}, error) {
		slog.Debug("sdk: _observe called", "userPlugin_addr", fmt.Sprintf("%p", &userPlugin), "userPlugin_nil", userPlugin == nil)
		if userPlugin == nil {
			return nil, fmt.Errorf("plugin not registered")
		}

		// Read config from WASM memory
		configBytes := abi.BytesFromPtr(abi.PackPtrLen(configPtr, configLen))

		// Use current context or create one with timeout
		ctx := wasmcontext.GetCurrentContext()

		// Store context for SDK functions to use
		wasmcontext.SetCurrentContext(ctx)
		defer wasmcontext.ResetContext() // Reset after execution

		// Pass raw bytes to Check
		evidence, err := userPlugin.Check(ctx, configBytes)
		if err != nil {
			// If user's check returns an error, create a Failure Evidence from it.
			// This ensures all returned values are of type Evidence.
			failResult := entities.ResultFailure(err.Error(), nil)
			evidence = &failResult
		}

		// Ensure Timestamp is always set, even if plugin didn't explicitly set it.
		if evidence.Timestamp.IsZero() {
			evidence.Timestamp = time.Now()
		}
		return evidence, nil
	})
}

// handleExportedCall is a generic wrapper for WASM exported functions.
// It provides panic recovery, error handling, and JSON serialization.
// It ensures that on any error or panic, a structured Evidence with ErrorDetail is returned.
func handleExportedCall(f func() (interface{}, error)) (packedResult uint64) {
	// Use a named return parameter to ensure it's set before `panic` is propagated.
	defer func() {
		if r := recover(); r != nil {
			// Free all tracked allocations on panic to prevent leaks.
			abi.FreeAllTracked()

			errDetail := &entities.ErrorDetail{
				Message: fmt.Sprintf("plugin panic: %v", r),
				Type:    "panic",
				Stack:   debug.Stack(), // Capture stack trace for panics
			}
			slog.Error("sdk: plugin panic recovered", "error", errDetail.Message)
			packedResult = packEvidenceWithError(entities.Result{Status: entities.ResultStatusError, Error: errDetail, Timestamp: time.Now()})
		}
	}()

	result, err := f()
	if err != nil {
		slog.Error("sdk: plugin function returned error", "error", err.Error())
		packedResult = packEvidenceWithError(entities.Result{Status: entities.ResultStatusError, Error: errors.ToErrorDetail(err), Timestamp: time.Now()})
		return
	}

	var dataToMarshal []byte
	switch v := result.(type) {
	case []byte: // For returning raw JSON bytes if needed
		dataToMarshal = v
	default:
		var marshalErr error
		dataToMarshal, marshalErr = json.Marshal(v)
		if marshalErr != nil {
			slog.Error("sdk: failed to marshal result", "error", marshalErr.Error())
			packedResult = packEvidenceWithError(entities.Result{Status: entities.ResultStatusError, Error: errors.ToErrorDetail(marshalErr), Timestamp: time.Now()})
			return
		}
	}

	packedResult = abi.PtrFromBytes(dataToMarshal)
	return
}

// packEvidenceWithError marshals an Evidence struct (containing an error) to JSON
// and returns the packed pointer/length. Used for internal SDK errors/panics.
func packEvidenceWithError(ev entities.Result) uint64 {
	data, err := json.Marshal(ev)
	if err != nil {
		// Fallback if even marshaling the error fails
		slog.Error("sdk: critical - failed to marshal error evidence", "original_error", ev.Error.Message, "marshal_error", err.Error())
		fallbackErr := &entities.ErrorDetail{Message: "sdk: critical error during error marshalling", Type: "internal"}
		data, _ = json.Marshal(entities.Result{Status: entities.ResultStatusError, Error: fallbackErr, Timestamp: time.Now()}) // Try to marshal a generic error
	}
	return abi.PtrFromBytes(data)
}
