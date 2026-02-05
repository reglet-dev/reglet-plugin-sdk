package ports

import (
	"context"
)

// HTTPClient defines the interface for HTTP operations.
// Infrastructure adapters implement this to provide HTTP functionality.
type HTTPClient interface {
	// Do executes an HTTP request and returns the response.
	Do(ctx context.Context, req HTTPRequest) (*HTTPResponse, error)

	// Get performs an HTTP GET request.
	Get(ctx context.Context, url string) (*HTTPResponse, error)

	// Post performs an HTTP POST request.
	Post(ctx context.Context, url string, contentType string, body []byte) (*HTTPResponse, error)
}

// HTTPRequest represents an HTTP request.
type HTTPRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
	Timeout int // milliseconds
}

// HTTPResponse represents an HTTP response.
type HTTPResponse struct {
	Headers    map[string][]string
	Proto      string
	Body       []byte
	StatusCode int
}
