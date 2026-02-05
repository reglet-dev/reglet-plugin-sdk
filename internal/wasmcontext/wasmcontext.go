// Package wasmcontext provides context propagation utilities for the Reglet SDK.
// It handles converting between Go contexts and the wire format used for
// WASM host function calls.
package wasmcontext

import (
	stdcontext "context"
	"sync"
	"time"

	"github.com/reglet-dev/reglet-sdk/domain/entities"
)

// contextKey is a type alias for context value keys to avoid collisions.
type contextKey string

// RequestIDKey is the context key for request ID.
const RequestIDKey contextKey = "request_id"

// contextStore holds the current context for the plugin execution.
// Since WASM is single-threaded, we can use a simple global variable.
// The host sets this when calling into the plugin via SetCurrentContext.
var contextStore = struct {
	ctx stdcontext.Context
	sync.RWMutex
}{
	ctx: stdcontext.Background(),
}

// SetCurrentContext sets the current execution context.
//
// This is called by the plugin infrastructure when the host invokes
// an exported function (describe, schema, observe). It updates the global
// context used for the duration of the execution.
func SetCurrentContext(ctx stdcontext.Context) {
	contextStore.Lock()
	defer contextStore.Unlock()
	contextStore.ctx = ctx
}

// GetCurrentContext returns the current execution context.
//
// This is used by SDK functions (exec, net, log) to get the context
// for host function calls. If no context has been set, it returns
// context.Background().
func GetCurrentContext() stdcontext.Context {
	contextStore.RLock()
	defer contextStore.RUnlock()
	if contextStore.ctx == nil {
		return stdcontext.Background()
	}
	return contextStore.ctx
}

// ResetContext resets the global context to background.
//
// This helper function simplifies the cleanup pattern in the plugin lifecyle.
// It should be called (usually via defer) after an operation completes.
func ResetContext() {
	SetCurrentContext(stdcontext.Background())
}

// ContextToWire converts a stdcontext.Context to ContextWireFormat for
// sending to the host via host functions.
//
// It extracts:
// - Deadline (timeout)
// - Cancellation status
// - Request ID (key: RequestIDKey)
func ContextToWire(ctx stdcontext.Context) entities.ContextWire {
	wire := entities.ContextWire{}

	// Extract deadline
	if deadline, ok := ctx.Deadline(); ok {
		wire.Deadline = &deadline
		// Calculate timeout in milliseconds
		timeout := time.Until(deadline)
		if timeout > 0 {
			wire.TimeoutMs = timeout.Milliseconds()
		}
	}

	// Check if context is Canceled
	select {
	case <-ctx.Done():
		wire.Canceled = true
	default:
		wire.Canceled = false
	}

	// Extract request ID from context if available
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		if id, ok := requestID.(string); ok {
			wire.RequestID = id
		}
	}

	return wire
}

// WireToContext converts a ContextWireFormat to a stdcontext.Context.
// This is used when the host provides context information to the plugin.
//
// If parent is nil, context.Background() is used.
// Returns the new context and its CancelFunc.
func WireToContext(parent stdcontext.Context, wire entities.ContextWire) (stdcontext.Context, stdcontext.CancelFunc) {
	if parent == nil {
		parent = stdcontext.Background()
	}

	ctx := parent

	// Apply deadline if present
	var cancel stdcontext.CancelFunc
	switch {
	case wire.Deadline != nil:
		ctx, cancel = stdcontext.WithDeadline(ctx, *wire.Deadline)
	case wire.TimeoutMs > 0:
		ctx, cancel = stdcontext.WithTimeout(ctx, time.Duration(wire.TimeoutMs)*time.Millisecond)
	default:
		ctx, cancel = stdcontext.WithCancel(ctx)
	}

	// Add request ID to context if present
	if wire.RequestID != "" {
		ctx = stdcontext.WithValue(ctx, RequestIDKey, wire.RequestID)
	}

	// If context is already Canceled, cancel immediately
	if wire.Canceled {
		cancel()
	}

	return ctx, cancel
}
