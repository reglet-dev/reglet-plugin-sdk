package hostfuncs

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

// TCPConnectRequest contains parameters for a TCP connection test.
type TCPConnectRequest struct {
	// Host is the target hostname or IP address.
	Host string `json:"host"`

	// Port is the target port number.
	Port int `json:"port"`

	// Timeout is the connection timeout in milliseconds. Default is 5000 (5s).
	Timeout int `json:"timeout_ms,omitempty"`

	// UseTLS indicates whether to use TLS for the connection.
	UseTLS bool `json:"use_tls,omitempty"`

	// TLSConfig is an optional custom TLS configuration.
	// This field is not marshaled to JSON and is intended for internal use
	// or when calling PerformTCPConnect directly from Go code.
	TLSConfig *tls.Config `json:"-"`
}

// TCPConnectResponse contains the result of a TCP connection test.
type TCPConnectResponse struct {
	// Error contains error information if the connection failed.
	Error *TCPError `json:"error,omitempty"`

	// RemoteAddr is the resolved remote address if connected.
	RemoteAddr string `json:"remote_addr,omitempty"`

	// LatencyMs is the connection latency in milliseconds.
	LatencyMs int64 `json:"latency_ms,omitempty"`

	// Connected indicates whether the connection was successful.
	Connected bool `json:"connected"`

	// TLSVersion is the TLS version used (e.g. "TLS 1.2").
	TLSVersion string `json:"tls_version,omitempty"`

	// TLSCipherSuite is the cipher suite used.
	TLSCipherSuite string `json:"tls_cipher_suite,omitempty"`

	// TLSServerName is the server name from the TLS handshake.
	TLSServerName string `json:"tls_server_name,omitempty"`

	// TLSCertSubject is the subject of the peer certificate.
	TLSCertSubject string `json:"tls_cert_subject,omitempty"`

	// TLSCertIssuer is the issuer of the peer certificate.
	TLSCertIssuer string `json:"tls_cert_issuer,omitempty"`

	// TLSCertExpiry is the expiration time of the peer certificate.
	TLSCertExpiry string `json:"tls_cert_expiry,omitempty"`
}

// TCPError represents a TCP connection error.
type TCPError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *TCPError) Error() string {
	return e.Message
}

// TCPOption is a functional option for configuring TCP connection behavior.
type TCPOption func(*tcpConfig)

type tcpConfig struct {
	timeout        time.Duration
	ssrfProtection bool
	allowPrivate   bool
}

func defaultTCPConfig() tcpConfig {
	return tcpConfig{
		timeout: 5 * time.Second,
	}
}

// WithTCPTimeout sets the TCP connection timeout.
func WithTCPTimeout(d time.Duration) TCPOption {
	return func(c *tcpConfig) {
		if d > 0 {
			c.timeout = d
		}
	}
}

// WithTCPSSRFProtection enables SSRF protection.
// When enabled, private/reserved IPs are blocked unless allowPrivate is true.
// DNS is resolved once and the resolved IP is used for the connection (prevents DNS rebinding).
func WithTCPSSRFProtection(allowPrivate bool) TCPOption {
	return func(c *tcpConfig) {
		c.ssrfProtection = true
		c.allowPrivate = allowPrivate
	}
}

// PerformTCPConnect tests TCP connectivity to the specified host and port.
// This is a pure Go implementation with no WASM runtime dependencies.
//
// Example usage from a WASM host:
//
//	func handleTCPConnect(req hostfuncs.TCPConnectRequest) hostfuncs.TCPConnectResponse {
//	    return hostfuncs.PerformTCPConnect(ctx, req)
//	}
func PerformTCPConnect(ctx context.Context, req TCPConnectRequest, opts ...TCPOption) TCPConnectResponse {
	cfg := defaultTCPConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Override config from request if specified
	if req.Timeout > 0 {
		cfg.timeout = time.Duration(req.Timeout) * time.Millisecond
	}

	// Validate request
	if req.Host == "" {
		return TCPConnectResponse{
			Connected: false,
			Error: &TCPError{
				Code:    "INVALID_REQUEST",
				Message: "host is required",
			},
		}
	}
	if req.Port <= 0 || req.Port > 65535 {
		return TCPConnectResponse{
			Connected: false,
			Error: &TCPError{
				Code:    "INVALID_REQUEST",
				Message: fmt.Sprintf("invalid port: %d", req.Port),
			},
		}
	}

	// SSRF Protection: validate and resolve address
	originalHost := req.Host
	if cfg.ssrfProtection {
		addr := fmt.Sprintf("%s:%d", req.Host, req.Port)
		var opts []NetfilterOption
		if cfg.allowPrivate {
			opts = append(opts, WithBlockPrivate(false), WithBlockLocalhost(false))
		}
		result := ValidateAddress(addr, opts...)

		if !result.Allowed {
			return TCPConnectResponse{
				Connected: false,
				Error: &TCPError{
					Code:    "SSRF_BLOCKED",
					Message: result.Reason,
				},
			}
		}

		// Use resolved IP for connection to prevent DNS rebinding
		if result.ResolvedIP != "" {
			req.Host = result.ResolvedIP
		}
	}

	// Build address
	address := fmt.Sprintf("%s:%d", req.Host, req.Port)

	// Apply timeout to context
	ctx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()

	// Attempt connection
	start := time.Now()
	dialer := &net.Dialer{
		Timeout: cfg.timeout,
	}

	var conn net.Conn
	var err error

	if req.UseTLS {
		// Prepare TLS config
		tlsConfig := req.TLSConfig
		if tlsConfig == nil {
			// Use original hostname for SNI, not resolved IP
			serverName := req.Host
			if originalHost != "" && originalHost != req.Host {
				serverName = originalHost
			}
			tlsConfig = &tls.Config{
				ServerName: serverName,
				MinVersion: tls.VersionTLS12,
			}
		} else if tlsConfig.ServerName == "" {
			// Ensure ServerName is set for SNI if not provided
			// Use original hostname for SNI, not resolved IP
			serverName := req.Host
			if originalHost != "" && originalHost != req.Host {
				serverName = originalHost
			}
			tlsConfig.ServerName = serverName
		}

		conn, err = tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}

	latency := time.Since(start)

	if err != nil {
		code := "CONNECTION_FAILED"
		switch {
		case strings.Contains(err.Error(), "timeout"), ctx.Err() == context.DeadlineExceeded:
			code = "TIMEOUT"
		case strings.Contains(err.Error(), "refused"):
			code = "CONNECTION_REFUSED"
		case strings.Contains(err.Error(), "no such host"):
			code = "HOST_NOT_FOUND"
		case strings.Contains(err.Error(), "certificate"):
			code = "TLS_ERROR"
		}

		return TCPConnectResponse{
			Connected: false,
			LatencyMs: latency.Milliseconds(),
			Error: &TCPError{
				Code:    code,
				Message: err.Error(),
			},
		}
	}
	defer func() { _ = conn.Close() }()

	resp := TCPConnectResponse{
		Connected:  true,
		RemoteAddr: conn.RemoteAddr().String(),
		LatencyMs:  latency.Milliseconds(),
	}

	// Extract TLS info if applicable
	if tlsConn, ok := conn.(*tls.Conn); ok {
		state := tlsConn.ConnectionState()
		resp.TLSVersion = tlsVersionString(state.Version)
		resp.TLSCipherSuite = tls.CipherSuiteName(state.CipherSuite)
		resp.TLSServerName = state.ServerName

		if len(state.PeerCertificates) > 0 {
			cert := state.PeerCertificates[0]
			resp.TLSCertSubject = cert.Subject.String()
			resp.TLSCertIssuer = cert.Issuer.String()
			resp.TLSCertExpiry = cert.NotAfter.Format(time.RFC3339)
		}
	}

	return resp
}
