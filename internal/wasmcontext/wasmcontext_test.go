package wasmcontext

import (
	"context"
	"testing"
	"time"

	"github.com/reglet-dev/reglet-plugin-sdk/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetAndGetCurrentContext(t *testing.T) {
	// Reset store before test
	ResetContext()
	assert.Equal(t, context.Background(), GetCurrentContext(), "should default to background")

	expectedCtx := context.WithValue(context.Background(), contextKey("key"), "value")
	SetCurrentContext(expectedCtx)

	actualCtx := GetCurrentContext()
	assert.Equal(t, expectedCtx, actualCtx, "context mismatch")
	assert.Equal(t, "value", actualCtx.Value(contextKey("key")))

	// Cleanup
	ResetContext()
	assert.Equal(t, context.Background(), GetCurrentContext())
}

func TestContextToWire(t *testing.T) {
	// 1. Basic context
	ctx := context.WithValue(context.Background(), RequestIDKey, "req-123")
	wire := ContextToWire(ctx)
	assert.Equal(t, "req-123", wire.RequestID)
	assert.False(t, wire.Canceled)
	assert.Nil(t, wire.Deadline)
	assert.Zero(t, wire.TimeoutMs)

	// 2. Canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	wire = ContextToWire(ctx)
	assert.True(t, wire.Canceled)

	// 3. Deadline context
	deadline := time.Now().Add(1 * time.Hour)
	ctx, cancel = context.WithDeadline(context.Background(), deadline)
	defer cancel()
	wire = ContextToWire(ctx)
	require.NotNil(t, wire.Deadline)
	// Truncate to millisecond precision for comparison as time.Now() has monotonic clock
	assert.WithinDuration(t, deadline, *wire.Deadline, time.Millisecond)
	assert.Positive(t, wire.TimeoutMs)
}

func TestWireToContext(t *testing.T) {
	// 1. Basic properties
	wire := entities.ContextWire{
		RequestID: "req-456",
	}
	ctx, cancel := WireToContext(context.Background(), wire)
	defer cancel()

	assert.Equal(t, "req-456", ctx.Value(RequestIDKey))
	assert.NoError(t, ctx.Err())

	// 2. Deadline from wire
	deadline := time.Now().Add(1 * time.Hour)
	wire = entities.ContextWire{
		Deadline: &deadline,
	}
	ctx, cancel = WireToContext(context.Background(), wire)
	defer cancel()

	d, ok := ctx.Deadline()
	assert.True(t, ok)
	assert.WithinDuration(t, deadline, d, time.Millisecond)

	// 3. TimeoutMs from wire
	wire = entities.ContextWire{
		TimeoutMs: 1000,
	}
	ctx, cancel = WireToContext(context.Background(), wire)
	defer cancel()
	d, ok = ctx.Deadline()
	assert.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(1*time.Second), d, 100*time.Millisecond)

	// 4. Pre-canceled
	wire = entities.ContextWire{
		Canceled: true,
	}
	ctx, cancel = WireToContext(context.Background(), wire)
	defer cancel()
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
}
