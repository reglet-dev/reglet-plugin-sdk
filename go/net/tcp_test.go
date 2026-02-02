package sdknet

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/reglet-dev/reglet-sdk/go/application/config"
	"github.com/reglet-dev/reglet-sdk/go/domain/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockTCPConnection
type MockTCPConnection struct {
	mock.Mock
}

func (m *MockTCPConnection) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockTCPConnection) RemoteAddr() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTCPConnection) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockTCPConnection) LocalAddr() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTCPConnection) IsTLS() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockTCPConnection) TLSVersion() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTCPConnection) TLSCipherSuite() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTCPConnection) TLSServerName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTCPConnection) TLSCertSubject() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTCPConnection) TLSCertIssuer() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTCPConnection) TLSCertNotAfter() *time.Time {
	args := m.Called()
	return args.Get(0).(*time.Time)
}

// MockTCPDialer
type MockTCPDialer struct {
	mock.Mock
}

func (m *MockTCPDialer) Dial(ctx context.Context, address string) (ports.TCPConnection, error) {
	args := m.Called(ctx, address)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ports.TCPConnection), args.Error(1)
}

func (m *MockTCPDialer) DialWithTimeout(ctx context.Context, address string, timeoutMs int) (ports.TCPConnection, error) {
	args := m.Called(ctx, address, timeoutMs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ports.TCPConnection), args.Error(1)
}

func (m *MockTCPDialer) DialSecure(ctx context.Context, address string, timeoutMs int, tls bool) (ports.TCPConnection, error) {
	args := m.Called(ctx, address, timeoutMs, tls)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ports.TCPConnection), args.Error(1)
}

func TestRunTCPCheck_Validation(t *testing.T) {
	tests := []struct {
		name      string
		cfg       config.Config
		errCode   string
		errDetail string
	}{
		{
			name:    "Missing Host",
			cfg:     config.Config{"port": 80},
			errCode: "MISSING_HOST",
		},
		{
			name:    "Missing Port",
			cfg:     config.Config{"host": "example.com"},
			errCode: "MISSING_PORT",
		},
		{
			name:    "Invalid Port Low",
			cfg:     config.Config{"host": "example.com", "port": 0},
			errCode: "INVALID_PORT",
		},
		{
			name:    "Invalid Port High",
			cfg:     config.Config{"host": "example.com", "port": 65536},
			errCode: "INVALID_PORT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RunTCPCheck(context.Background(), tt.cfg)
			require.NoError(t, err)
			assert.True(t, result.IsError())
			assert.Equal(t, tt.errCode, result.Error.Code)
		})
	}
}

func TestRunTCPCheck_WithMockDialer_Success(t *testing.T) {
	mockDialer := new(MockTCPDialer)
	mockConn := new(MockTCPConnection)

	// Expect close to be called via defer
	mockConn.On("Close").Return(nil)
	mockConn.On("IsConnected").Return(true)
	mockConn.On("RemoteAddr").Return("1.2.3.4:443")
	mockConn.On("LocalAddr").Return("127.0.0.1:5678")
	mockConn.On("IsTLS").Return(false)

	// Expect DialSecure with correct args (tls=false by default)
	mockDialer.On("DialSecure", mock.Anything, "google.com:443", 5000, false).Return(mockConn, nil)

	cfg := config.Config{"host": "google.com", "port": 443}

	result, err := RunTCPCheck(context.Background(), cfg, WithTCPDialer(mockDialer))

	require.NoError(t, err)
	assert.True(t, result.IsSuccess())
	assert.Equal(t, true, result.Data["connected"])
	assert.Equal(t, "1.2.3.4:443", result.Data["remote_addr"])
	assert.Greater(t, result.Data["response_time_ms"].(int64), int64(-1)) // Should be >= 0

	mockDialer.AssertExpectations(t)
	mockConn.AssertExpectations(t)
}

func TestRunTCPCheck_WithMockDialer_ConnectionFailed(t *testing.T) {
	mockDialer := new(MockTCPDialer)

	// Simulate connection error
	mockDialer.On("DialSecure", mock.Anything, "example.com:80", 5000, false).Return(nil, errors.New("timeout"))

	cfg := config.Config{"host": "example.com", "port": 80}

	result, err := RunTCPCheck(context.Background(), cfg, WithTCPDialer(mockDialer))

	require.Error(t, err)
	assert.True(t, result.IsError())
	assert.Equal(t, "CONNECTION_FAILED", result.Error.Code)
	assert.Contains(t, result.Error.Message, "timeout")

	mockDialer.AssertExpectations(t)
}

func TestRunTCPCheck_DefaultClient_PanicsOnNative(t *testing.T) {
	cfg := config.Config{"host": "example.com", "port": 80}

	assert.PanicsWithValue(t, "WASM TCP adapter not available in native build. Use WithTCPDialer() to inject a mock.", func() {
		_, _ = RunTCPCheck(context.Background(), cfg)
	})
}
