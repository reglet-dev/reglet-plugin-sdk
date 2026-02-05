// Package log provides structure logging (slog) adapted for Reglet SDK's WASM environment.
package log

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/reglet-dev/reglet-sdk/domain/entities"
)

// LogMessageWire is the JSON wire format for a log message from Guest to Host.
type LogMessageWire struct {
	Timestamp time.Time            `json:"timestamp"`       // 24 bytes
	Attrs     []LogAttrWire        `json:"attrs,omitempty"` // 24 bytes (slice)
	Level     string               `json:"level"`           // 16 bytes
	Message   string               `json:"message"`         // 16 bytes
	Context   entities.ContextWire `json:"context"`         // contains pointers, largest struct
}

// LogAttrWire represents a single slog attribute for wire transfer.
type LogAttrWire struct {
	Key   string `json:"key"`
	Type  string `json:"type"`  // "string", "int64", "bool", "float64", "time", "error", "any"
	Value string `json:"value"` // String representation of the value
}

// toLogAttrWire converts a slog.Attr to LogAttrWire.
func toLogAttrWire(attr slog.Attr) LogAttrWire {
	wire := LogAttrWire{
		Key: attr.Key,
	}
	// Resolve the attribute value
	attr.Value = attr.Value.Resolve()

	switch attr.Value.Kind() {
	case slog.KindString:
		wire.Type = "string"
		wire.Value = attr.Value.String()
	case slog.KindInt64:
		wire.Type = "int64"
		wire.Value = fmt.Sprintf("%d", attr.Value.Int64())
	case slog.KindUint64:
		wire.Type = "uint64"
		wire.Value = fmt.Sprintf("%d", attr.Value.Uint64())
	case slog.KindBool:
		wire.Type = "bool"
		wire.Value = fmt.Sprintf("%t", attr.Value.Bool())
	case slog.KindFloat64:
		wire.Type = "float64"
		wire.Value = fmt.Sprintf("%f", attr.Value.Float64())
	case slog.KindTime:
		wire.Type = "time"
		wire.Value = attr.Value.Time().Format(time.RFC3339Nano)
	case slog.KindDuration:
		wire.Type = "duration"
		wire.Value = attr.Value.Duration().String()
	case slog.KindAny:
		if v := attr.Value.Any(); v != nil {
			if err, isErr := v.(error); isErr {
				wire.Type = "error"
				wire.Value = err.Error()
			} else if data, marshalErr := json.Marshal(v); marshalErr == nil {
				wire.Type = "json"
				wire.Value = string(data)
			} else {
				wire.Type = "any"
				wire.Value = fmt.Sprintf("%v", v)
			}
		} else {
			wire.Type = "any"
			wire.Value = "<nil>"
		}
	case slog.KindGroup:
		wire.Type = "group"
		wire.Value = fmt.Sprintf("%v", attr.Value.Any())
	case slog.KindLogValuer:
		return toLogAttrWire(slog.Attr{Key: attr.Key, Value: attr.Value.LogValuer().LogValue()})
	default:
		wire.Type = "any"
		wire.Value = fmt.Sprintf("%v", attr.Value.Any())
	}
	return wire
}
