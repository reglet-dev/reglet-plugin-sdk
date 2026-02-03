//go:build wasip1

package sdknet

import (
	"testing"
	"time"

	"github.com/reglet-dev/reglet-sdk/go/infrastructure/wasm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolverOption_DefaultConfig(t *testing.T) {
	cfg := defaultResolverConfig()

	assert.Empty(t, cfg.nameserver, "default nameserver should be empty (use host default)")
	assert.Equal(t, 5*time.Second, cfg.timeout, "default timeout should be 5s")
	assert.Equal(t, 3, cfg.retries, "default retries should be 3")
}

func TestNewResolver_WithDefaults(t *testing.T) {
	resolver := NewResolver()

	require.NotNil(t, resolver, "NewResolver should return a non-nil resolver")

	// Assert it's a WASM adapter
	adapter, ok := resolver.(*wasm.DNSAdapter)
	require.True(t, ok, "NewResolver should return *wasm.DNSAdapter in wasip1 build")

	assert.Empty(t, adapter.Nameserver, "default nameserver should be empty")
	assert.Equal(t, 5*time.Second, adapter.Timeout, "default timeout should be 5s")
}

func TestNewResolver_WithNameserver(t *testing.T) {
	resolver := NewResolver(WithNameserver("8.8.8.8:53"))

	adapter, ok := resolver.(*wasm.DNSAdapter)
	require.True(t, ok)

	assert.Equal(t, "8.8.8.8:53", adapter.Nameserver)
	// Other defaults should still apply
	assert.Equal(t, 5*time.Second, adapter.Timeout)
}

func TestNewResolver_WithDNSTimeout(t *testing.T) {
	resolver := NewResolver(WithDNSTimeout(10 * time.Second))

	adapter, ok := resolver.(*wasm.DNSAdapter)
	require.True(t, ok)

	assert.Equal(t, 10*time.Second, adapter.Timeout)
}

func TestNewResolver_WithDNSTimeout_IgnoresInvalid(t *testing.T) {
	tests := []struct {
		name     string
		timeout  time.Duration
		expected time.Duration
	}{
		{"zero should use default", 0, 5 * time.Second},
		{"negative should use default", -1 * time.Second, 5 * time.Second},
		{"positive should be applied", 15 * time.Second, 15 * time.Second},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolver := NewResolver(WithDNSTimeout(tc.timeout))
			adapter, ok := resolver.(*wasm.DNSAdapter)
			require.True(t, ok)
			assert.Equal(t, tc.expected, adapter.Timeout)
		})
	}
}

func TestNewResolver_WithRetries(t *testing.T) {
	resolver := NewResolver(WithRetries(5))
	require.NotNil(t, resolver)
}

func TestNewResolver_MultipleOptions(t *testing.T) {
	resolver := NewResolver(
		WithNameserver("1.1.1.1:53"),
		WithDNSTimeout(8*time.Second),
		WithRetries(2),
	)

	adapter, ok := resolver.(*wasm.DNSAdapter)
	require.True(t, ok)

	assert.Equal(t, "1.1.1.1:53", adapter.Nameserver)
	assert.Equal(t, 8*time.Second, adapter.Timeout)
}

func TestNewResolver_OptionsApplyInOrder(t *testing.T) {
	// Last option wins for the same setting
	resolver := NewResolver(
		WithNameserver("8.8.8.8:53"),
		WithNameserver("1.1.1.1:53"), // This should override
	)

	adapter, ok := resolver.(*wasm.DNSAdapter)
	require.True(t, ok)

	assert.Equal(t, "1.1.1.1:53", adapter.Nameserver, "last option should win")
}
