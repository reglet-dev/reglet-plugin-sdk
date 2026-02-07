// Package sdknet provides high-level network check functions for SDK plugins.
package sdknet

import (
	"context"
	"fmt"
	"time"

	"github.com/reglet-dev/reglet-plugin-sdk/application/config"
	"github.com/reglet-dev/reglet-plugin-sdk/domain/entities"
	"github.com/reglet-dev/reglet-plugin-sdk/domain/ports"
	"github.com/reglet-dev/reglet-plugin-sdk/infrastructure/wasm"
)

// TCPCheckOption is a functional option for configuring TCP checks.
type TCPCheckOption func(*tcpCheckConfig)

type tcpCheckConfig struct {
	dialer ports.TCPDialer
}

func defaultTCPCheckConfig() tcpCheckConfig {
	return tcpCheckConfig{
		dialer: wasm.NewTCPAdapter(),
	}
}

// WithTCPDialer sets the TCP dialer to use for the check.
// This is useful for injecting mocks during testing.
func WithTCPDialer(d ports.TCPDialer) TCPCheckOption {
	return func(c *tcpCheckConfig) {
		if d != nil {
			c.dialer = d
		}
	}
}

// RunTCPCheck performs a TCP connectivity check.
// It parses configuration, executes the connection test, and returns a structured Result.
//
// Expected config fields:
//   - host (string, required): Target hostname or IP address
//   - port (int, required): Target port number (1-65535)
//   - timeout_ms (int, optional): Connection timeout in milliseconds (default: 5000)
//
// Returns a Result with:
//   - Status: "success" if connected, "error" if failed
//   - Data: map containing "connected", "remote_addr", "latency_ms"
//   - Error: structured error details if connection failed
func RunTCPCheck(ctx context.Context, cfg config.Config, opts ...TCPCheckOption) (entities.Result, error) {
	// Parse required fields
	host, err := config.MustGetString(cfg, "host")
	if err != nil {
		return entities.ResultError(entities.NewErrorDetail("config", err.Error()).WithCode("MISSING_HOST")), nil
	}

	port, err := config.MustGetInt(cfg, "port")
	if err != nil {
		return entities.ResultError(entities.NewErrorDetail("config", err.Error()).WithCode("MISSING_PORT")), nil
	}

	// Validate port range
	if port < 1 || port > 65535 {
		return entities.ResultError(entities.NewErrorDetail("config", fmt.Sprintf("invalid port: %d (must be 1-65535)", port)).WithCode("INVALID_PORT")), nil
	}

	// Parse optional timeout
	timeoutMs := config.GetIntDefault(cfg, "timeout_ms", 5000)

	// Parse TLS config
	tls := config.GetBoolDefault(cfg, "tls", false)
	_ = config.GetStringDefault(cfg, "expected_tls_version", "") // Parsed but unused in core check, plugins may use it

	// Configure check
	checkCfg := defaultTCPCheckConfig()
	for _, opt := range opts {
		opt(&checkCfg)
	}

	address := fmt.Sprintf("%s:%d", host, port)

	// Execute TCP connect
	start := time.Now()
	conn, err := checkCfg.dialer.DialSecure(ctx, address, timeoutMs, tls)
	latency := time.Since(start)

	// Create metadata
	metadata := entities.NewRunMetadata(start, time.Now())

	if err != nil {
		// Connection failed - convert error to ErrorDetail
		// Note: We lost granular error codes from hostfuncs unless we parse error string
		// or if err satisfies an interface. For now we use generic "CONNECTION_FAILED"
		// or try to match common strings if critical.
		errDetail := entities.NewErrorDetail("network", err.Error()).WithCode("CONNECTION_FAILED")
		return entities.ResultError(errDetail).WithMetadata(metadata), errDetail
	}

	defer func() { _ = conn.Close() }()

	// Build result data
	resultData := map[string]any{
		"connected":        conn.IsConnected(),
		"response_time_ms": latency.Milliseconds(),
		"address":          address,
	}

	if conn.RemoteAddr() != "" {
		resultData["remote_addr"] = conn.RemoteAddr()
	}

	if conn.LocalAddr() != "" {
		resultData["local_addr"] = conn.LocalAddr()
	}

	if conn.IsTLS() {
		resultData["tls"] = true
		resultData["tls_version"] = conn.TLSVersion()
		resultData["tls_cipher_suite"] = conn.TLSCipherSuite()
		resultData["tls_server_name"] = conn.TLSServerName()
		if conn.TLSCertSubject() != "" {
			resultData["tls_cert_subject"] = conn.TLSCertSubject()
			resultData["tls_cert_issuer"] = conn.TLSCertIssuer()
		}
		if notAfter := conn.TLSCertNotAfter(); notAfter != nil {
			resultData["tls_cert_not_after"] = notAfter.Format(time.RFC3339)
			resultData["tls_cert_days_remaining"] = int(time.Until(*notAfter).Hours() / 24)
		}
	}

	if conn.IsConnected() {
		return entities.ResultSuccess("TCP connection successful", resultData).WithMetadata(metadata), nil
	}

	// Connected is false but no error? Should not happen normally if Dial returns nil err.
	return entities.ResultError(entities.NewErrorDetail("network", "TCP connection failed").WithCode("CONNECTION_FAILED")).WithMetadata(metadata), nil
}
