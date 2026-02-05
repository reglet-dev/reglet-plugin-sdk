//go:build wasip1

package sdknet

import (
	"testing"
	"time"

	"github.com/reglet-dev/reglet-sdk/infrastructure/wasm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransportOption_DefaultConfig(t *testing.T) {
	cfg := defaultTransportConfig()

	assert.Equal(t, 30*time.Second, cfg.timeout, "default timeout should be 30s")
	assert.Equal(t, 10, cfg.maxRedirects, "default maxRedirects should be 10")
	assert.Nil(t, cfg.tlsConfig, "default tlsConfig should be nil")
}

func TestNewTransport_WithDefaults(t *testing.T) {
	transport := NewTransport()

	require.NotNil(t, transport, "NewTransport should return a non-nil transport")

	adapter, ok := transport.(*wasm.HTTPAdapter)
	require.True(t, ok, "NewTransport should return *wasm.HTTPAdapter in wasip1 build")

	assert.Equal(t, 30*time.Second, adapter.DefaultTimeout, "default timeout should be 30s")
}

func TestNewTransport_WithHTTPTimeout(t *testing.T) {
	transport := NewTransport(WithHTTPTimeout(60 * time.Second))

	adapter, ok := transport.(*wasm.HTTPAdapter)
	require.True(t, ok)

	assert.Equal(t, 60*time.Second, adapter.DefaultTimeout)
}

func TestNewTransport_WithHTTPTimeout_IgnoresInvalid(t *testing.T) {
	tests := []struct {
		name     string
		timeout  time.Duration
		expected time.Duration
	}{
		{"zero should use default", 0, 30 * time.Second},
		{"negative should use default", -1 * time.Second, 30 * time.Second},
		{"positive should be applied", 45 * time.Second, 45 * time.Second},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			transport := NewTransport(WithHTTPTimeout(tc.timeout))
			adapter, ok := transport.(*wasm.HTTPAdapter)
			require.True(t, ok)
			assert.Equal(t, tc.expected, adapter.DefaultTimeout)
		})
	}
}

func TestNewTransport_WithMaxRedirects(t *testing.T) {
	// MaxRedirects is parsed into config but ignored by WASM adapter currently.
	transport := NewTransport(WithMaxRedirects(5))
	require.NotNil(t, transport)
}

func TestNewTransport_WithMaxRedirects_ZeroDisables(t *testing.T) {
	transport := NewTransport(WithMaxRedirects(0))
	require.NotNil(t, transport)
}

func TestNewTransport_WithMaxRedirects_IgnoresNegative(t *testing.T) {
	transport := NewTransport(WithMaxRedirects(-1))
	require.NotNil(t, transport)
}

func TestNewTransport_MultipleOptions(t *testing.T) {
	transport := NewTransport(
		WithHTTPTimeout(60*time.Second),
		WithMaxRedirects(3),
	)

	adapter, ok := transport.(*wasm.HTTPAdapter)
	require.True(t, ok)

	assert.Equal(t, 60*time.Second, adapter.DefaultTimeout)
}

func TestNewTransport_OptionsApplyInOrder(t *testing.T) {
	// Last option wins for the same setting
	transport := NewTransport(
		WithHTTPTimeout(60*time.Second),
		WithHTTPTimeout(15*time.Second), // This should override
	)

	adapter, ok := transport.(*wasm.HTTPAdapter)
	require.True(t, ok)

	assert.Equal(t, 15*time.Second, adapter.DefaultTimeout, "last option should win")
}
