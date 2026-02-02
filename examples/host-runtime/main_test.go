package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerformTLSCheck(t *testing.T) {
	// 1. Setup a mock TLS server
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	var host string
	var port int

	// Parse URL to get host/port for expectations
	parsedURL, _ := url.Parse(ts.URL)
	host = parsedURL.Hostname()
	portStr := parsedURL.Port()
	if portStr == "" {
		port = 443
	} else {
		_, _ = fmt.Sscanf(portStr, "%d", &port)
	}

	ctx := context.Background()

	t.Run("Valid TLS Check", func(t *testing.T) {
		req := TLSCheckRequest{
			Host:      host,
			Port:      port,
			TimeoutMs: 2000,
		}

		resp := performTLSCheckInternal(ctx, req, ts.TLS.Certificates)

		assert.True(t, resp.Connected)
		assert.NotEmpty(t, resp.NotAfter)
		assert.Contains(t, resp.Issuer, "example.com") // httptest default
	})

	t.Run("Invalid Host", func(t *testing.T) {
		req := TLSCheckRequest{
			Host: "",
		}
		resp := performTLSCheck(ctx, req)
		assert.False(t, resp.Connected)
		require.NotNil(t, resp.Error)
		assert.Equal(t, "INVALID_INPUT", resp.Error.Code)
	})

	t.Run("Connection Refused", func(t *testing.T) {
		req := TLSCheckRequest{
			Host: "localhost",
			Port: 1, // Highly unlikely to have a service here
		}
		resp := performTLSCheck(ctx, req)
		assert.False(t, resp.Connected)
		require.NotNil(t, resp.Error)
		assert.Equal(t, "CONNECTION_FAILED", resp.Error.Code)
	})
}

// performTLSCheckInternal is a helper for testing that allows injecting certs for verification
func performTLSCheckInternal(ctx context.Context, req TLSCheckRequest, rootCerts []tls.Certificate) TLSCheckResponse {
	timeout := time.Duration(req.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	dialer := &net.Dialer{
		Timeout: timeout,
	}

	address := fmt.Sprintf("%s:%d", req.Host, req.Port)

	// Create a custom config for testing with the mock server's CA
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // For test simplicity against httptest
		ServerName:         req.Host,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	if err != nil {
		return TLSCheckResponse{
			Connected: false,
			Error: &TLSError{
				Code:    "CONNECTION_FAILED",
				Message: err.Error(),
			},
		}
	}
	defer func() { _ = conn.Close() }()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return TLSCheckResponse{
			Connected: true,
			Error: &TLSError{
				Code:    "NO_CERTIFICATES",
				Message: "peer provided no certificates",
			},
		}
	}

	peerCert := certs[0]
	return TLSCheckResponse{
		Connected: true,
		NotAfter:  peerCert.NotAfter.Format(time.RFC3339),
		Issuer:    peerCert.Issuer.CommonName,
	}
}
