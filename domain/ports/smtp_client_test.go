package ports

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSMTPClient is a mock implementation of SMTPClient for testing.
type MockSMTPClient struct {
	ConnectFunc func(ctx context.Context, host, port string, timeout time.Duration, useTLS, useStartTLS bool) (*SMTPConnectResult, error)
}

func (m *MockSMTPClient) Connect(ctx context.Context, host, port string, timeout time.Duration, useTLS, useStartTLS bool) (*SMTPConnectResult, error) {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx, host, port, timeout, useTLS, useStartTLS)
	}
	return &SMTPConnectResult{
		Connected:    true,
		Banner:       "220 smtp.example.com ESMTP",
		TLSEnabled:   useTLS || useStartTLS,
		TLSVersion:   "TLS 1.3",
		ResponseTime: 100 * time.Millisecond,
	}, nil
}

// Compile-time interface check
var _ SMTPClient = (*MockSMTPClient)(nil)

func TestMockSMTPClient_ImplementsInterface(t *testing.T) {
	var client SMTPClient = &MockSMTPClient{}
	require.NotNil(t, client)
}

func TestMockSMTPClient_Connect(t *testing.T) {
	ctx := context.Background()

	t.Run("default behavior", func(t *testing.T) {
		mock := &MockSMTPClient{}
		res, err := mock.Connect(ctx, "smtp.example.com", "587", 5*time.Second, false, true)

		require.NoError(t, err)
		assert.True(t, res.Connected)
		assert.Equal(t, "220 smtp.example.com ESMTP", res.Banner)
		assert.True(t, res.TLSEnabled)
	})

	t.Run("custom behavior", func(t *testing.T) {
		mock := &MockSMTPClient{
			ConnectFunc: func(ctx context.Context, host, port string, timeout time.Duration, useTLS, useStartTLS bool) (*SMTPConnectResult, error) {
				return &SMTPConnectResult{
					Connected: false,
					Banner:    "",
				}, nil
			},
		}

		res, err := mock.Connect(ctx, "smtp.example.com", "25", 5*time.Second, false, false)

		require.NoError(t, err)
		assert.False(t, res.Connected)
	})

	t.Run("error behavior", func(t *testing.T) {
		expectedErr := errors.New("connection refused")
		mock := &MockSMTPClient{
			ConnectFunc: func(ctx context.Context, host, port string, timeout time.Duration, useTLS, useStartTLS bool) (*SMTPConnectResult, error) {
				return nil, expectedErr
			},
		}

		res, err := mock.Connect(ctx, "smtp.example.com", "25", 5*time.Second, false, false)

		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Equal(t, expectedErr, err)
	})
}
