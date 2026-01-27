package ports

import (
	"context"
	"time"
)

// TCPDialer defines the interface for TCP connection operations.
// Infrastructure adapters implement this to provide TCP functionality.
type TCPDialer interface {
	// Dial establishes a TCP connection to the given address.
	Dial(ctx context.Context, address string) (TCPConnection, error)

	// DialWithTimeout establishes a TCP connection with a timeout.
	DialWithTimeout(ctx context.Context, address string, timeoutMs int) (TCPConnection, error)

	// DialSecure establishes a TCP connection with timeout and optional TLS.
	DialSecure(ctx context.Context, address string, timeoutMs int, tls bool) (TCPConnection, error)
}

// TCPConnection represents an established TCP connection.
type TCPConnection interface {
	// Close closes the connection.
	Close() error

	// RemoteAddr returns the remote address.
	RemoteAddr() string

	// IsConnected returns true if the connection is established.
	IsConnected() bool

	// LocalAddr returns the local address.
	LocalAddr() string

	// IsTLS returns true if the connection is TLS-secured.
	IsTLS() bool

	// TLSVersion returns the TLS version string (e.g. "TLS 1.3").
	TLSVersion() string

	// TLSCipherSuite returns the TLS cipher suite name.
	TLSCipherSuite() string

	// TLSServerName returns the TLS server name (SNI).
	TLSServerName() string

	// TLSCertSubject returns the subject of the peer certificate.
	TLSCertSubject() string

	// TLSCertIssuer returns the issuer of the peer certificate.
	TLSCertIssuer() string

	// TLSCertNotAfter returns the expiration time of the peer certificate.
	TLSCertNotAfter() *time.Time
}
