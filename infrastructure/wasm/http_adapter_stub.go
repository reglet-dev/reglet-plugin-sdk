//go:build !wasip1

package wasm

import (
	"context"
	"time"

	"github.com/reglet-dev/reglet-sdk/domain/ports"
)

// HTTPAdapter stub for native builds.
type HTTPAdapter struct{}

func NewHTTPAdapter(defaultTimeout time.Duration) *HTTPAdapter {
	return &HTTPAdapter{}
}

func (c *HTTPAdapter) Do(ctx context.Context, req ports.HTTPRequest) (*ports.HTTPResponse, error) {
	panic("WASM HTTP adapter not available in native build")
}

func (c *HTTPAdapter) Get(ctx context.Context, url string) (*ports.HTTPResponse, error) {
	panic("WASM HTTP adapter not available in native build")
}

func (c *HTTPAdapter) Post(ctx context.Context, url string, contentType string, body []byte) (*ports.HTTPResponse, error) {
	panic("WASM HTTP adapter not available in native build")
}
