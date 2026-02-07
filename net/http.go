package sdknet

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/reglet-dev/reglet-plugin-sdk/application/config"
	"github.com/reglet-dev/reglet-plugin-sdk/domain/entities"
	"github.com/reglet-dev/reglet-plugin-sdk/domain/ports"
	"github.com/reglet-dev/reglet-plugin-sdk/infrastructure/wasm"
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
//
// RunHTTPCheck performs an HTTP request check.
// It parses configuration, executes the HTTP request, and returns a structured Result.
func RunHTTPCheck(ctx context.Context, cfg config.Config, opts ...HTTPCheckOption) (entities.Result, error) {
	parsedCfg, err := parseHTTPCheckConfig(cfg)
	if err != nil {
		return entities.ResultError(entities.NewErrorDetail("config", err.Error()).WithCode("MISSING_URL")), nil
	}

	// Configure check dependencies
	checkCfg := httpCheckConfig{}
	for _, opt := range opts {
		opt(&checkCfg)
	}

	// If client not injected, create default using config
	if checkCfg.client == nil {
		transportOpts := []TransportOption{
			WithHTTPTimeout(time.Duration(parsedCfg.TimeoutMs) * time.Millisecond),
			WithMaxRedirects(parsedCfg.MaxRedirects),
		}
		checkCfg.client = NewTransport(transportOpts...)
	}

	// Execute HTTP request
	start := time.Now()
	resp, err := checkCfg.client.Do(ctx, parsedCfg.Request)
	latency := time.Since(start)

	// Create metadata
	metadata := entities.NewRunMetadata(start, time.Now())

	if err != nil {
		errDetail := entities.NewErrorDetail("network", err.Error()).WithCode("REQUEST_FAILED")
		return entities.ResultError(errDetail).WithMetadata(metadata), errDetail
	}

	return buildHTTPResult(resp, latency, parsedCfg, metadata), nil
}

type parsedHTTPConfig struct {
	ExpectedBodyContains string
	Request              ports.HTTPRequest
	TimeoutMs            int
	BodyPreviewLength    int
	ExpectedStatus       int
	MaxRedirects         int
	HasExpectedStatus    bool
}

func parseHTTPCheckConfig(cfg config.Config) (*parsedHTTPConfig, error) {
	url, err := config.MustGetString(cfg, "url")
	if err != nil {
		return nil, err
	}

	pc := &parsedHTTPConfig{
		TimeoutMs:            config.GetIntDefault(cfg, "timeout_ms", 30000),
		BodyPreviewLength:    config.GetIntDefault(cfg, "body_preview_length", 200),
		ExpectedBodyContains: config.GetStringDefault(cfg, "expected_body_contains", ""),
		MaxRedirects:         config.GetIntDefault(cfg, "max_redirects", 10),
	}

	if es, ok := config.GetInt(cfg, "expected_status"); ok {
		pc.ExpectedStatus = es
		pc.HasExpectedStatus = true
	}

	if followRedirects, ok := config.GetBool(cfg, "follow_redirects"); ok && !followRedirects {
		pc.MaxRedirects = 0
	}

	// Parse headers
	headers := parseHeaders(cfg)
	body := config.GetStringDefault(cfg, "body", "")

	pc.Request = ports.HTTPRequest{
		Method:  config.GetStringDefault(cfg, "method", "GET"),
		URL:     url,
		Headers: headers,
		Timeout: pc.TimeoutMs,
	}
	if body != "" {
		pc.Request.Body = []byte(body)
	}

	return pc, nil
}

func parseHeaders(cfg config.Config) map[string]string {
	var headers map[string]string
	if headersRaw, ok := cfg["headers"].(map[string]interface{}); ok {
		headers = make(map[string]string)
		for k, v := range headersRaw {
			if vStr, ok := v.(string); ok {
				headers[k] = vStr
			}
		}
	}
	return headers
}

func buildHTTPResult(resp *ports.HTTPResponse, latency time.Duration, cfg *parsedHTTPConfig, metadata *entities.RunMetadata) entities.Result {
	resultData := map[string]any{
		"status_code":      resp.StatusCode,
		"response_time_ms": latency.Milliseconds(),
		"protocol":         resp.Proto,
	}

	if len(resp.Headers) > 0 {
		resultData["headers"] = resp.Headers
	}

	if len(resp.Body) > 0 {
		addBodyInfo(resultData, resp.Body, cfg.BodyPreviewLength)
	}

	// Validations
	if cfg.HasExpectedStatus && resp.StatusCode != cfg.ExpectedStatus {
		message := fmt.Sprintf("HTTP status mismatch: expected %d, got %d", cfg.ExpectedStatus, resp.StatusCode)
		resultData["expected_status"] = cfg.ExpectedStatus
		resultData["actual_status"] = resp.StatusCode
		return entities.ResultFailure(message, resultData).WithMetadata(metadata)
	}

	if cfg.ExpectedBodyContains != "" {
		if !strings.Contains(string(resp.Body), cfg.ExpectedBodyContains) {
			message := fmt.Sprintf("HTTP body mismatch: expected to contain '%s'", cfg.ExpectedBodyContains)
			resultData["expected_body_contains"] = cfg.ExpectedBodyContains
			return entities.ResultFailure(message, resultData).WithMetadata(metadata)
		}
	}

	message := fmt.Sprintf("HTTP %s request successful: %d", cfg.Request.Method, resp.StatusCode)
	return entities.ResultSuccess(message, resultData).WithMetadata(metadata)
}

func addBodyInfo(data map[string]any, body []byte, previewLen int) {
	hash := sha256.Sum256(body)
	data["body_sha256"] = hex.EncodeToString(hash[:])
	data["body_size"] = len(body)
	data["body_length"] = len(body) // Compat alias

	preview := body
	truncated := false
	if previewLen >= 0 && len(preview) > previewLen {
		preview = preview[:previewLen]
		truncated = true
	}
	data["body"] = string(preview)
	data["body_truncated"] = truncated
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
