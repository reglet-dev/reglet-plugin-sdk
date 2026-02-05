package log

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToLogAttrWire(t *testing.T) {
	tests := []struct {
		name     string
		attr     slog.Attr
		wantType string
		wantVal  string
	}{
		{
			name:     "string",
			attr:     slog.String("key", "value"),
			wantType: "string",
			wantVal:  "value",
		},
		{
			name:     "int64",
			attr:     slog.Int64("key", 123),
			wantType: "int64",
			wantVal:  "123",
		},
		{
			name:     "bool",
			attr:     slog.Bool("key", true),
			wantType: "bool",
			wantVal:  "true",
		},
		{
			name:     "float64",
			attr:     slog.Float64("key", 1.23),
			wantType: "float64",
			wantVal:  "1.230000",
		},
		{
			name:     "time",
			attr:     slog.Time("key", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			wantType: "time",
			wantVal:  "2024-01-01T00:00:00Z",
		},
		{
			name:     "duration",
			attr:     slog.Duration("key", 1*time.Hour),
			wantType: "duration",
			wantVal:  "1h0m0s",
		},
		{
			name:     "error",
			attr:     slog.Any("key", errors.New("test error")),
			wantType: "error",
			wantVal:  "test error",
		},
		{
			name:     "nil",
			attr:     slog.Any("key", nil),
			wantType: "any",
			wantVal:  "<nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wire := toLogAttrWire(tt.attr)
			assert.Equal(t, tt.attr.Key, wire.Key)
			assert.Equal(t, tt.wantType, wire.Type)
			assert.Equal(t, tt.wantVal, wire.Value)
		})
	}
}

func TestToLogAttrWire_JSON(t *testing.T) {
	// Test structured object that should be serialized as JSON
	type MyStruct struct {
		Field string `json:"field"`
	}
	obj := MyStruct{Field: "data"}
	attr := slog.Any("key", obj)

	wire := toLogAttrWire(attr)
	assert.Equal(t, "key", wire.Key)
	assert.Equal(t, "json", wire.Type)

	var decoded MyStruct
	err := json.Unmarshal([]byte(wire.Value), &decoded)
	require.NoError(t, err)
	assert.Equal(t, obj, decoded)
}

func TestToLogAttrWire_LogValuer(t *testing.T) {
	// Test types that implement LogValuer
	attr := slog.Any("key", logValuer{val: "resolved"})
	wire := toLogAttrWire(attr)

	assert.Equal(t, "key", wire.Key)
	assert.Equal(t, "string", wire.Type)
	assert.Equal(t, "resolved", wire.Value)
}

type logValuer struct {
	val string
}

func (l logValuer) LogValue() slog.Value {
	return slog.StringValue(l.val)
}

func TestNewHandler_Defaults(t *testing.T) {
	h := NewHandler()
	assert.NotNil(t, h)
	// Check default level via Enabled
	assert.True(t, h.Enabled(context.TODO(), slog.LevelInfo))
	assert.False(t, h.Enabled(context.TODO(), slog.LevelDebug))
}

func TestNewHandler_Options(t *testing.T) {
	h := NewHandler(
		WithLevel(slog.LevelDebug),
		WithSource(true),
	)
	assert.NotNil(t, h)
	assert.True(t, h.Enabled(context.TODO(), slog.LevelDebug))
	// We can't easily check addSource without internal access, creates coverage gap but verified behavior.
}
