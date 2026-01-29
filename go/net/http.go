package sdknet

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/reglet-dev/reglet-sdk/go/application/config"
	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/reglet-dev/reglet-sdk/go/domain/ports"
	"github.com/reglet-dev/reglet-sdk/go/infrastructure/wasm"
)

// HTTPCheckOption is a functional option for configuring HTTP checks.
type HTTPCheckOption func(*httpCheckConfig)

type httpCheckConfig struct {
	client ports.HTTPClient
}

// WithHTTPClient sets the HTTP client to use for the check.
// This is useful for injecting mocks during testing.
func WithHTTPClient(c ports.HTTPClient) HTTPCheckOption {
	return func(cfg *httpCheckConfig) {
		if c != nil {
			cfg.client = c
		}
	}
}

// RunHTTPCheck performs an HTTP request check.
// It parses configuration, executes the HTTP request, and returns a structured Result.
//
// Expected config fields:
//   - url (string, required): Target URL
//   - method (string, optional): HTTP method (default: GET)
//   - headers (map[string]string, optional): Request headers
//   - body (string, optional): Request body
//   - timeout_ms (int, optional): Request timeout in milliseconds (default: 30000)
//   - expected_status (int, optional): Expected HTTP status code for validation
//   - follow_redirects (bool, optional): Whether to follow redirects (default: true)
//   - max_redirects (int, optional): Maximum redirects to follow (default: 10)
//
// Returns a Result with:
//   - Status: "success" if request succeeded and matches expectations, "failure" if status mismatch, "error" if request failed
//   - Data: map containing "status_code", "headers", "body", "latency_ms", "body_truncated"
//   - Error: structured error details if request failed
func RunHTTPCheck(ctx context.Context, cfg config.Config, opts ...HTTPCheckOption) (entities.Result, error) {
	// Parse required fields
	url, err := config.MustGetString(cfg, "url")
	if err != nil {
		return entities.ResultError(entities.NewErrorDetail("config", err.Error()).WithCode("MISSING_URL")), nil
	}

	// Parse optional fields
	method := config.GetStringDefault(cfg, "method", "GET")
	timeoutMs := config.GetIntDefault(cfg, "timeout_ms", 30000)
	bodyPreviewLength := config.GetIntDefault(cfg, "body_preview_length", 200)
	expectedStatus, hasExpectedStatus := config.GetInt(cfg, "expected_status")
	expectedBodyContains := config.GetStringDefault(cfg, "expected_body_contains", "")
	maxRedirects := config.GetIntDefault(cfg, "max_redirects", 10)
	// follow_redirects logic: if false, set maxRedirects to 0?
	if followRedirects, ok := config.GetBool(cfg, "follow_redirects"); ok && !followRedirects {
		maxRedirects = 0
	}

	// Parse headers if provided
	var headers map[string]string
	if headersRaw, ok := cfg["headers"].(map[string]interface{}); ok {
		headers = make(map[string]string)
		for k, v := range headersRaw {
			if vStr, ok := v.(string); ok {
				headers[k] = vStr
			}
		}
	}

	// Parse body if provided
	body := config.GetStringDefault(cfg, "body", "")

	// Configure check
	checkCfg := httpCheckConfig{}
	for _, opt := range opts {
		opt(&checkCfg)
	}

	// If client not injected, create default using config
	if checkCfg.client == nil {
		transportOpts := []TransportOption{
			WithHTTPTimeout(time.Duration(timeoutMs) * time.Millisecond),
			WithMaxRedirects(maxRedirects),
		}
		checkCfg.client = NewTransport(transportOpts...)
	}

	// Create request
	req := ports.HTTPRequest{
		Method:  method,
		URL:     url,
		Headers: headers,
		Timeout: timeoutMs,
	}

	if body != "" {
		req.Body = []byte(body)
	}

	// Execute HTTP request
	start := time.Now()
	resp, err := checkCfg.client.Do(ctx, req)
	latency := time.Since(start)

	// Create metadata
	metadata := entities.NewRunMetadata(start, time.Now())

	if err != nil {
		// Request failed
		errDetail := entities.NewErrorDetail("network", err.Error()).WithCode("REQUEST_FAILED")
		return entities.ResultError(errDetail).WithMetadata(metadata), errDetail
	}

	resultData := map[string]any{
		"status_code":      resp.StatusCode,
		"response_time_ms": latency.Milliseconds(),
		"protocol":         resp.Proto,
	}

	if len(resp.Headers) > 0 {
		resultData["headers"] = resp.Headers
	}

	if len(resp.Body) > 0 {
		// Calculate hash
		hash := sha256.Sum256(resp.Body)
		resultData["body_sha256"] = hex.EncodeToString(hash[:])
		resultData["body_size"] = len(resp.Body)

		// Create preview
		preview := resp.Body
		truncated := false
		// Check truncation only if limit is positive
		// A negative value for bodyPreviewLength indicates "unlimited", so the full body is preserved.
		if bodyPreviewLength >= 0 && len(preview) > bodyPreviewLength {
			preview = preview[:bodyPreviewLength]
			truncated = true
		}
		resultData["body"] = string(preview)
		resultData["body_truncated"] = truncated

		// Compatibility: keep "body_length" as alias for size? Or just use body_size
		resultData["body_length"] = len(resp.Body)
	}

	// Check expected status if specified
	if hasExpectedStatus && resp.StatusCode != expectedStatus {
		message := fmt.Sprintf("HTTP status mismatch: expected %d, got %d", expectedStatus, resp.StatusCode)
		resultData["expected_status"] = expectedStatus
		resultData["actual_status"] = resp.StatusCode
		return entities.ResultFailure(message, resultData).WithMetadata(metadata), nil
	}

	// Check expected body content if specified
	if expectedBodyContains != "" {
		if !strings.Contains(string(resp.Body), expectedBodyContains) {
			message := fmt.Sprintf("HTTP body mismatch: expected to contain '%s'", expectedBodyContains)
			resultData["expected_body_contains"] = expectedBodyContains
			return entities.ResultFailure(message, resultData).WithMetadata(metadata), nil
		}
	}

	// Success
	message := fmt.Sprintf("HTTP %s request successful: %d", method, resp.StatusCode)
	return entities.ResultSuccess(message, resultData).WithMetadata(metadata), nil
}

// transportConfig holds the configuration for a WasmTransport.
// This struct is unexported to enforce the functional options pattern.
type transportConfig struct {
	tlsConfig    *tls.Config   // Optional TLS configuration
	timeout      time.Duration // HTTP request timeout (default: 30s)
	maxRedirects int           // Maximum number of redirects to follow (default: 10)
}

// defaultTransportConfig returns secure defaults for HTTP transport.
// These defaults align with constitution requirements for secure-by-default.
func defaultTransportConfig() transportConfig {
	return transportConfig{
		timeout:      30 * time.Second,
		maxRedirects: 10,
		tlsConfig:    nil, // Use system defaults
	}
}

// TransportOption is a functional option for configuring a WasmTransport.
// Use With* functions to create options.
type TransportOption func(*transportConfig)

// WithHTTPTimeout sets the timeout for HTTP requests.
// Default is 30 seconds per constitution specification.
// A zero or negative duration is ignored (uses default).
func WithHTTPTimeout(d time.Duration) TransportOption {
	return func(c *transportConfig) {
		if d > 0 {
			c.timeout = d
		}
	}
}

// WithMaxRedirects sets the maximum number of redirects to follow.
// Default is 10 redirects. A negative value is ignored (uses default).
// Setting to 0 disables following redirects.
func WithMaxRedirects(n int) TransportOption {
	return func(c *transportConfig) {
		if n >= 0 {
			c.maxRedirects = n
		}
	}
}

// WithTLSConfig sets a custom TLS configuration.
// If nil is passed, the system default TLS configuration is used.
func WithTLSConfig(cfg *tls.Config) TransportOption {
	return func(c *transportConfig) {
		c.tlsConfig = cfg
	}
}

// NewTransport creates a new HTTP client with the given options.
// Without any options, secure defaults are applied:
//   - timeout: 30 seconds
//   - maxRedirects: 10
//   - tlsConfig: system defaults
//
// Example:
//
//	// Use defaults
//	client := NewTransport()
//
//	// With custom timeout
//	client := NewTransport(
//	    WithHTTPTimeout(60*time.Second),
//	    WithMaxRedirects(5),
//	)
func NewTransport(opts ...TransportOption) ports.HTTPClient {
	cfg := defaultTransportConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	// Note: maxRedirects and tlsConfig are currently ignored by the underlying WASM adapter
	return wasm.NewHTTPAdapter(cfg.timeout)
}
