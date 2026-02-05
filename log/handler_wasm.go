//go:build wasip1

package log

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/reglet-dev/reglet-sdk/internal/abi"
	wasmcontext "github.com/reglet-dev/reglet-sdk/internal/wasmcontext"
)

// Define the host function signature for logging messages.
// This matches the signature defined in internal/wasm/hostfuncs/registry.go.
//
//go:wasmimport reglet_host log_message
//nolint:revive // intentional snake_case to match WASM import convention
func host_log_message(messagePacked uint64)

// Handle serializes a slog.Record and sends it to the host via a host function.
func (h *WasmLogHandler) Handle(ctx context.Context, record slog.Record) error {
	logMsg := LogMessageWire{
		Context:   wasmcontext.ContextToWire(ctx),
		Level:     record.Level.String(),
		Message:   record.Message,
		Timestamp: record.Time,
	}

	// Convert slog.Attr to LogAttrWire
	record.Attrs(func(attr slog.Attr) bool {
		logMsg.Attrs = append(logMsg.Attrs, toLogAttrWire(attr))
		return true // Continue iterating
	})

	requestBytes, err := json.Marshal(logMsg)
	if err != nil {
		// Fallback to println if marshaling fails.
		fmt.Printf("sdk: failed to marshal log message for host: %v, original: %s\n", err, record.Message)
		// We still return nil because we printed the error to stdout/stderr
		return nil
	}

	// Call the host function (no return value)
	host_log_message(abi.PtrFromBytes(requestBytes))
	return nil
}
