package sdknet

import (
	"context"
	"fmt"
	"time"

	"github.com/reglet-dev/reglet-sdk/application/config"
	"github.com/reglet-dev/reglet-sdk/domain/entities"
	"github.com/reglet-dev/reglet-sdk/domain/ports"
	"github.com/reglet-dev/reglet-sdk/infrastructure/wasm"
)

// SMTPCheckOption is a functional option for configuring SMTP checks.
type SMTPCheckOption func(*smtpCheckConfig)

type smtpCheckConfig struct {
	client ports.SMTPClient
}

func defaultSMTPCheckConfig() smtpCheckConfig {
	return smtpCheckConfig{
		client: wasm.NewSMTPAdapter(),
	}
}

// WithSMTPClient sets the SMTP client to use for the check.
// This is useful for injecting mocks during testing.
func WithSMTPClient(c ports.SMTPClient) SMTPCheckOption {
	return func(cfg *smtpCheckConfig) {
		if c != nil {
			cfg.client = c
		}
	}
}

// RunSMTPCheck performs an SMTP connectivity check.
// It parses configuration, executes the SMTP connection test, and returns a structured Result.
//
// Expected config fields:
//   - host (string, required): SMTP server hostname
//   - port (int, required): SMTP server port (typically 25, 465, or 587)
//   - use_tls (bool, optional): Use implicit TLS (port 465). Default: false
//   - use_starttls (bool, optional): Upgrade to TLS via STARTTLS (port 587). Default: false
//   - timeout_ms (int, optional): Connection timeout in milliseconds (default: 30000)
//
// Returns a Result with:
//   - Status: "success" if connected, "error" if failed
//   - Data: map containing "connected", "banner", "tls_version", "latency_ms"
//   - Error: structured error details if connection failed
func RunSMTPCheck(ctx context.Context, cfg config.Config, opts ...SMTPCheckOption) (entities.Result, error) {
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

	// Parse optional fields
	useTLS := config.GetBoolDefault(cfg, "use_tls", false)
	useSTARTTLS := config.GetBoolDefault(cfg, "use_starttls", false)
	timeoutMs := config.GetIntDefault(cfg, "timeout_ms", 30000)

	// Configure check
	checkCfg := defaultSMTPCheckConfig()
	for _, opt := range opts {
		opt(&checkCfg)
	}

	// Execute SMTP connect
	start := time.Now()
	resp, err := checkCfg.client.Connect(ctx, host, fmt.Sprintf("%d", port), time.Duration(timeoutMs)*time.Millisecond, useTLS, useSTARTTLS)
	latency := time.Since(start)

	// Create metadata
	metadata := entities.NewRunMetadata(start, time.Now())

	if err != nil {
		// Connection failed
		errDetail := entities.NewErrorDetail("network", err.Error()).WithCode("CONNECTION_FAILED")
		return entities.ResultError(errDetail).WithMetadata(metadata), errDetail
	}

	// Build result data
	resultData := map[string]any{
		"connected":  resp.Connected,
		"latency_ms": latency.Milliseconds(),
	}

	if resp.Banner != "" {
		resultData["banner"] = resp.Banner
	}
	if resp.TLSVersion != "" {
		resultData["tls_version"] = resp.TLSVersion
	}
	if resp.TLSCipherSuite != "" {
		resultData["tls_cipher_suite"] = resp.TLSCipherSuite
	}
	if resp.TLSServerName != "" {
		resultData["tls_server_name"] = resp.TLSServerName
	}

	resultData["address"] = fmt.Sprintf("%s:%d", host, port)

	// Return result based on connection status
	if resp.Connected {
		message := fmt.Sprintf("SMTP connection successful to %s:%d", host, port)
		return entities.ResultSuccess(message, resultData).WithMetadata(metadata), nil
	}

	// Should not reach here if err is nil and Connected is false, but safe fallback
	return entities.ResultError(entities.NewErrorDetail("network", "SMTP connection failed").WithCode("CONNECTION_FAILED")).WithMetadata(metadata), nil
}
