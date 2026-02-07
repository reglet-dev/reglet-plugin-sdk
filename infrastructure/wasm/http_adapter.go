//go:build wasip1

package wasm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/reglet-dev/reglet-plugin-sdk/domain/entities"
	"github.com/reglet-dev/reglet-plugin-sdk/domain/ports"
	"github.com/reglet-dev/reglet-plugin-sdk/internal/abi"
	wasmcontext "github.com/reglet-dev/reglet-plugin-sdk/internal/wasmcontext"
	_ "github.com/reglet-dev/reglet-plugin-sdk/log"
)

// MaxHTTPBodySize definition if needed, or import from somewhere.
const MaxHTTPBodySize = 10 * 1024 * 1024 // 10 MB

// Compile-time interface compliance check
var _ ports.HTTPClient = (*HTTPAdapter)(nil)

// HTTPAdapter implements ports.HTTPClient for the WASM environment.
type HTTPAdapter struct {
	DefaultTimeout time.Duration
}

// NewHTTPAdapter creates a new HTTP adapter.
func NewHTTPAdapter(defaultTimeout time.Duration) *HTTPAdapter {
	if defaultTimeout == 0 {
		defaultTimeout = 30 * time.Second
	}
	return &HTTPAdapter{
		DefaultTimeout: defaultTimeout,
	}
}

// Do executes an HTTP request.
func (c *HTTPAdapter) Do(ctx context.Context, req ports.HTTPRequest) (*ports.HTTPResponse, error) {
	// Prepare wire request
	wireCtx := wasmcontext.ContextToWire(ctx)

	headers := make(map[string][]string)
	for k, v := range req.Headers {
		headers[k] = []string{v}
	}

	rawBody := ""
	if len(req.Body) > 0 {
		rawBody = base64.StdEncoding.EncodeToString(req.Body)
	}

	wireReq := entities.HTTPRequest{
		Context: wireCtx,
		Method:  req.Method,
		URL:     req.URL,
		Headers: headers, // map[string][]string
		Body:    rawBody,
	}

	// Marshal and call host
	reqBytes, err := json.Marshal(wireReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	respPacked := host_http_request(abi.PtrFromBytes(reqBytes))
	respBytes := abi.BytesFromPtr(respPacked)
	abi.DeallocatePacked(respPacked)

	var wireResp entities.HTTPResponse
	if err := json.Unmarshal(respBytes, &wireResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if wireResp.Error != nil {
		return nil, wireResp.Error
	}

	if wireResp.BodyTruncated {
		return nil, fmt.Errorf("response body too large")
	}

	// Decode body
	var body []byte
	if wireResp.Body != "" {
		body, err = base64.StdEncoding.DecodeString(wireResp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to decode response body: %w", err)
		}
	}

	return &ports.HTTPResponse{
		StatusCode: wireResp.StatusCode,
		Headers:    wireResp.Headers,
		Body:       body,
		Proto:      wireResp.Proto,
	}, nil
}

// Get performs a GET request.
func (c *HTTPAdapter) Get(ctx context.Context, url string) (*ports.HTTPResponse, error) {
	return c.Do(ctx, ports.HTTPRequest{
		Method:  http.MethodGet,
		URL:     url,
		Timeout: int(c.DefaultTimeout.Milliseconds()),
	})
}

// Post performs a POST request.
func (c *HTTPAdapter) Post(ctx context.Context, url string, contentType string, body []byte) (*ports.HTTPResponse, error) {
	return c.Do(ctx, ports.HTTPRequest{
		Method: http.MethodPost,
		URL:    url,
		Headers: map[string]string{
			"Content-Type": contentType,
		},
		Body:    body,
		Timeout: int(c.DefaultTimeout.Milliseconds()),
	})
}
