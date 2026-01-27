//go:build wasip1

package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/reglet-dev/reglet-sdk/go/domain/ports"
	"github.com/reglet-dev/reglet-sdk/go/internal/abi"
	wasmcontext "github.com/reglet-dev/reglet-sdk/go/internal/wasmcontext"
)

// Compile-time interface compliance check
var _ ports.SMTPClient = (*SMTPAdapter)(nil)

// SMTPAdapter implements ports.SMTPClient for the WASM environment.
type SMTPAdapter struct{}

// NewSMTPAdapter creates a new SMTP adapter.
func NewSMTPAdapter() *SMTPAdapter {
	return &SMTPAdapter{}
}

// Connect establishes an SMTP connection to the given host and port.
func (a *SMTPAdapter) Connect(ctx context.Context, host, port string, timeout time.Duration, useTLS, useStartTLS bool) (*ports.SMTPConnectResult, error) {
	request := entities.SMTPRequest{
		Context:   wasmcontext.ContextToWire(ctx),
		Host:      host,
		Port:      port,
		TimeoutMs: int(timeout.Milliseconds()),
		TLS:       useTLS,
		StartTLS:  useStartTLS,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SMTP request: %w", err)
	}

	requestPacked := abi.PtrFromBytes(requestBytes)
	defer abi.DeallocatePacked(requestPacked)

	responsePacked := host_smtp_connect(requestPacked)

	responseBytes := abi.BytesFromPtr(responsePacked)
	defer abi.DeallocatePacked(responsePacked)

	var response entities.SMTPResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SMTP response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("%s: %s", response.Error.Type, response.Error.Message)
	}

	return &ports.SMTPConnectResult{
		Connected:      response.Connected,
		Banner:         response.Banner,
		TLSEnabled:     response.TLS,
		TLSVersion:     response.TLSVersion,
		TLSCipherSuite: response.TLSCipherSuite,
		TLSServerName:  response.TLSServerName,
		ResponseTime:   time.Duration(response.ResponseTimeMs) * time.Millisecond,
	}, nil
}
