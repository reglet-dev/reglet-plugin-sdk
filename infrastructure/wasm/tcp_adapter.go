//go:build wasip1

package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/reglet-dev/reglet-plugin-sdk/domain/entities"
	"github.com/reglet-dev/reglet-plugin-sdk/domain/ports"
	"github.com/reglet-dev/reglet-plugin-sdk/internal/abi"
	wasmcontext "github.com/reglet-dev/reglet-plugin-sdk/internal/wasmcontext"
)

// Compile-time interface compliance check
var _ ports.TCPDialer = (*TCPAdapter)(nil)

// TCPAdapter implements ports.TCPDialer for the WASM environment.
type TCPAdapter struct{}

// NewTCPAdapter creates a new TCP adapter.
func NewTCPAdapter() *TCPAdapter {
	return &TCPAdapter{}
}

// Dial establishes a TCP connection to the given address.
func (a *TCPAdapter) Dial(ctx context.Context, address string) (ports.TCPConnection, error) {
	return a.DialSecure(ctx, address, 5000, false) // Default 5s timeout, no TLS
}

// DialWithTimeout establishes a TCP connection with a timeout.
func (a *TCPAdapter) DialWithTimeout(ctx context.Context, address string, timeoutMs int) (ports.TCPConnection, error) {
	return a.DialSecure(ctx, address, timeoutMs, false)
}

// DialSecure establishes a TCP connection with timeout and optional TLS.
func (a *TCPAdapter) DialSecure(ctx context.Context, address string, timeoutMs int, tls bool) (ports.TCPConnection, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}

	request := entities.TCPRequest{
		Context:   wasmcontext.ContextToWire(ctx),
		Host:      host,
		Port:      port,
		TimeoutMs: timeoutMs,
		TLS:       tls,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TCP request: %w", err)
	}

	requestPacked := abi.PtrFromBytes(requestBytes)
	defer abi.DeallocatePacked(requestPacked)

	responsePacked := host_tcp_connect(requestPacked)

	responseBytes := abi.BytesFromPtr(responsePacked)
	defer abi.DeallocatePacked(responsePacked)

	var response entities.TCPResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal TCP response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("%s: %s", response.Error.Type, response.Error.Message)
	}

	return &WasmTCPConnection{
		response: response,
	}, nil
}

// WasmTCPConnection adapts the WASM response to the TCPConnection interface.
type WasmTCPConnection struct {
	response entities.TCPResponse
}

func (c *WasmTCPConnection) Close() error {
	// Connection is ephemeral in WASM check context, nothing to close
	return nil
}

func (c *WasmTCPConnection) RemoteAddr() string {
	return c.response.RemoteAddr
}

func (c *WasmTCPConnection) IsConnected() bool {
	return c.response.Connected
}

func (c *WasmTCPConnection) LocalAddr() string {
	return c.response.LocalAddr
}

func (c *WasmTCPConnection) IsTLS() bool {
	return c.response.TLS
}

func (c *WasmTCPConnection) TLSVersion() string {
	return c.response.TLSVersion
}

func (c *WasmTCPConnection) TLSCipherSuite() string {
	return c.response.TLSCipherSuite
}

func (c *WasmTCPConnection) TLSServerName() string {
	return c.response.TLSServerName
}

func (c *WasmTCPConnection) TLSCertSubject() string {
	return c.response.TLSCertSubject
}

func (c *WasmTCPConnection) TLSCertIssuer() string {
	return c.response.TLSCertIssuer
}

func (c *WasmTCPConnection) TLSCertNotAfter() *time.Time {
	return c.response.TLSCertNotAfter
}
