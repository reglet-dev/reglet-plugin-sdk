//go:build !wasip1

package wasm

import (
	"context"
	"time"

	"github.com/reglet-dev/reglet-plugin-sdk/domain/ports"
)

// SMTPAdapter stub for native builds.
type SMTPAdapter struct{}

func NewSMTPAdapter() *SMTPAdapter {
	return &SMTPAdapter{}
}

func (a *SMTPAdapter) Connect(ctx context.Context, host, port string, timeout time.Duration, useTLS, useStartTLS bool) (*ports.SMTPConnectResult, error) {
	panic("WASM SMTP adapter not available in native build. Use WithSMTPClient() to inject a mock.")
}
