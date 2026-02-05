package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockClient struct {
	Value string
}

func TestGetClient(t *testing.T) {
	ctx := context.Background()
	client := &mockClient{Value: "test"}

	ctx = WithClient(ctx, client)

	got := GetClient[*mockClient](ctx)
	require.NotNil(t, got)
	assert.Equal(t, "test", got.Value)
}

func TestGetClient_Panic_NoClient(t *testing.T) {
	ctx := context.Background()

	assert.Panics(t, func() {
		GetClient[*mockClient](ctx)
	})
}

func TestTryGetClient(t *testing.T) {
	ctx := context.Background()
	client := &mockClient{Value: "test"}

	// Without client
	got, ok := TryGetClient[*mockClient](ctx)
	assert.False(t, ok)
	assert.Nil(t, got)

	// With client
	ctx = WithClient(ctx, client)
	got, ok = TryGetClient[*mockClient](ctx)
	assert.True(t, ok)
	assert.Equal(t, "test", got.Value)
}
