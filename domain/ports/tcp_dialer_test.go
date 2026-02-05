package ports

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTCPDialer is a mock implementation of TCPDialer for testing.
type MockTCPDialer struct {
	DialFunc            func(ctx context.Context, address string) (TCPConnection, error)
	DialWithTimeoutFunc func(ctx context.Context, address string, timeoutMs int) (TCPConnection, error)
	DialSecureFunc      func(ctx context.Context, address string, timeoutMs int, tls bool) (TCPConnection, error)
}

func (m *MockTCPDialer) Dial(ctx context.Context, address string) (TCPConnection, error) {
	if m.DialFunc != nil {
		return m.DialFunc(ctx, address)
	}
	return &mockTCPConn{connected: true, addr: address}, nil
}

func (m *MockTCPDialer) DialWithTimeout(ctx context.Context, address string, timeoutMs int) (TCPConnection, error) {
	if m.DialWithTimeoutFunc != nil {
		return m.DialWithTimeoutFunc(ctx, address, timeoutMs)
	}
	return &mockTCPConn{connected: true, addr: address}, nil
}

func (m *MockTCPDialer) DialSecure(ctx context.Context, address string, timeoutMs int, tls bool) (TCPConnection, error) {
	if m.DialSecureFunc != nil {
		return m.DialSecureFunc(ctx, address, timeoutMs, tls)
	}
	return &mockTCPConn{connected: true, addr: address}, nil
}

// mockTCPConn is a minimal TCPConnection implementation for testing
type mockTCPConn struct {
	connected bool
	addr      string
	closed    bool
}

func (m *mockTCPConn) Close() error {
	m.closed = true
	m.connected = false
	return nil
}

func (m *mockTCPConn) RemoteAddr() string {
	return m.addr
}

func (m *mockTCPConn) IsConnected() bool {
	return m.connected && !m.closed
}

func (m *mockTCPConn) LocalAddr() string {
	return "127.0.0.1:0"
}

func (m *mockTCPConn) IsTLS() bool {
	return false
}

func (m *mockTCPConn) TLSVersion() string {
	return ""
}

func (m *mockTCPConn) TLSCipherSuite() string {
	return ""
}

func (m *mockTCPConn) TLSServerName() string {
	return ""
}

func (m *mockTCPConn) TLSCertSubject() string {
	return ""
}

func (m *mockTCPConn) TLSCertIssuer() string {
	return ""
}

func (m *mockTCPConn) TLSCertNotAfter() *time.Time {
	return nil
}

// Compile-time interface check
var _ TCPDialer = (*MockTCPDialer)(nil)

func TestMockTCPDialer_ImplementsInterface(t *testing.T) {
	// Verify that MockTCPDialer implements TCPDialer interface
	var dialer TCPDialer = &MockTCPDialer{}
	require.NotNil(t, dialer)
}

func TestMockTCPDialer_Dial(t *testing.T) {
	ctx := context.Background()

	t.Run("successful dial", func(t *testing.T) {
		mock := &MockTCPDialer{}
		conn, err := mock.Dial(ctx, "example.com:443")

		require.NoError(t, err)
		require.NotNil(t, conn)
		assert.True(t, conn.IsConnected())
		assert.Equal(t, "example.com:443", conn.RemoteAddr())

		// Clean up
		conn.Close()
		assert.False(t, conn.IsConnected())
	})

	t.Run("custom behavior - connection refused", func(t *testing.T) {
		expectedErr := errors.New("connection refused")
		mock := &MockTCPDialer{
			DialFunc: func(ctx context.Context, address string) (TCPConnection, error) {
				return nil, expectedErr
			},
		}

		conn, err := mock.Dial(ctx, "unreachable.example.com:443")

		assert.Error(t, err)
		assert.Nil(t, conn)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		mock := &MockTCPDialer{
			DialFunc: func(ctx context.Context, address string) (TCPConnection, error) {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				return &mockTCPConn{connected: true, addr: address}, nil
			},
		}

		conn, err := mock.Dial(ctx, "example.com:443")

		assert.Error(t, err)
		assert.Nil(t, conn)
	})
}

func TestMockTCPDialer_DialWithTimeout(t *testing.T) {
	ctx := context.Background()

	t.Run("successful dial with timeout", func(t *testing.T) {
		mock := &MockTCPDialer{}
		conn, err := mock.DialWithTimeout(ctx, "example.com:443", 5000)

		require.NoError(t, err)
		require.NotNil(t, conn)
		assert.True(t, conn.IsConnected())

		// Clean up
		conn.Close()
	})

	t.Run("timeout error", func(t *testing.T) {
		expectedErr := errors.New("i/o timeout")
		mock := &MockTCPDialer{
			DialWithTimeoutFunc: func(ctx context.Context, address string, timeoutMs int) (TCPConnection, error) {
				return nil, expectedErr
			},
		}

		conn, err := mock.DialWithTimeout(ctx, "slow.example.com:443", 100)

		assert.Error(t, err)
		assert.Nil(t, conn)
		assert.Contains(t, err.Error(), "timeout")
	})
}

func TestMockTCPConnection(t *testing.T) {
	conn := &mockTCPConn{connected: true, addr: "example.com:443"}

	t.Run("initial state", func(t *testing.T) {
		assert.True(t, conn.IsConnected())
		assert.Equal(t, "example.com:443", conn.RemoteAddr())
	})

	t.Run("close", func(t *testing.T) {
		err := conn.Close()
		assert.NoError(t, err)
		assert.False(t, conn.IsConnected())
	})
}
