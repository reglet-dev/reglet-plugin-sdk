package ports

import (
	"context"
	"time"
)

// SMTPClient defines the interface for SMTP connection operations.
// Infrastructure adapters implement this to provide SMTP functionality.
type SMTPClient interface {
	// Connect establishes an SMTP connection to the given host and port.
	Connect(ctx context.Context, host, port string, timeout time.Duration, useTLS, useStartTLS bool) (*SMTPConnectResult, error)
}

// SMTPConnectResult represents the result of an SMTP connection attempt.
type SMTPConnectResult struct {
	Banner         string
	TLSVersion     string
	TLSCipherSuite string
	TLSServerName  string
	ResponseTime   time.Duration
	Connected      bool
	TLSEnabled     bool
}
