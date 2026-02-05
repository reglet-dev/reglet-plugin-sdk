package sdknet

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/reglet-dev/reglet-sdk/application/config"
	"github.com/reglet-dev/reglet-sdk/domain/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockSMTPClient
type MockSMTPClient struct {
	mock.Mock
}

func (m *MockSMTPClient) Connect(ctx context.Context, host, port string, timeout time.Duration, useTLS, useStartTLS bool) (*ports.SMTPConnectResult, error) {
	args := m.Called(ctx, host, port, timeout, useTLS, useStartTLS)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.SMTPConnectResult), args.Error(1)
}

func TestRunSMTPCheck_Validation(t *testing.T) {
	tests := []struct {
		name      string
		cfg       config.Config
		errCode   string
		errDetail string
	}{
		{
			name:    "Missing Host",
			cfg:     config.Config{"port": 25},
			errCode: "MISSING_HOST",
		},
		{
			name:    "Missing Port",
			cfg:     config.Config{"host": "smtp.example.com"},
			errCode: "MISSING_PORT",
		},
		{
			name:    "Invalid Port Low",
			cfg:     config.Config{"host": "smtp.example.com", "port": 0},
			errCode: "INVALID_PORT",
		},
		{
			name:    "Invalid Port High",
			cfg:     config.Config{"host": "smtp.example.com", "port": 65536},
			errCode: "INVALID_PORT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RunSMTPCheck(context.Background(), tt.cfg)
			require.NoError(t, err)
			assert.True(t, result.IsError())
			assert.Equal(t, tt.errCode, result.Error.Code)
		})
	}
}

func TestRunSMTPCheck_WithMockClient_Success(t *testing.T) {
	mockClient := new(MockSMTPClient)

	expectedResult := &ports.SMTPConnectResult{
		Connected:    true,
		Banner:       "220 smtp.example.com ESMTP",
		TLSEnabled:   true,
		TLSVersion:   "TLS 1.3",
		ResponseTime: 50 * time.Millisecond,
	}

	mockClient.On("Connect", mock.Anything, "smtp.example.com", "587", 30*time.Second, false, true).Return(expectedResult, nil)

	cfg := config.Config{
		"host":         "smtp.example.com",
		"port":         587,
		"use_starttls": true,
		"timeout_ms":   30000,
	}

	result, err := RunSMTPCheck(context.Background(), cfg, WithSMTPClient(mockClient))

	require.NoError(t, err)
	assert.True(t, result.IsSuccess())
	assert.Equal(t, true, result.Data["connected"])
	assert.Equal(t, "220 smtp.example.com ESMTP", result.Data["banner"])
	assert.Equal(t, "TLS 1.3", result.Data["tls_version"])
	assert.Greater(t, result.Data["latency_ms"], int64(-1))

	mockClient.AssertExpectations(t)
}

func TestRunSMTPCheck_DefaultClient_PanicsOnNative(t *testing.T) {
	cfg := config.Config{"host": "smtp.example.com", "port": 25}
	assert.PanicsWithValue(t, "WASM SMTP adapter not available in native build. Use WithSMTPClient() to inject a mock.", func() {
		_, _ = RunSMTPCheck(context.Background(), cfg)
	})
}

func TestRunSMTPCheck_WithMockClient_ConnectionFailed(t *testing.T) {
	mockClient := new(MockSMTPClient)

	mockClient.On("Connect", mock.Anything, "smtp.example.com", "25", 30*time.Second, false, false).Return(nil, errors.New("timeout"))

	cfg := config.Config{
		"host": "smtp.example.com",
		"port": 25,
	}

	result, err := RunSMTPCheck(context.Background(), cfg, WithSMTPClient(mockClient))

	require.Error(t, err)
	assert.True(t, result.IsError())
	assert.Equal(t, "CONNECTION_FAILED", result.Error.Code)
	assert.Contains(t, result.Error.Message, "timeout")

	mockClient.AssertExpectations(t)
}
