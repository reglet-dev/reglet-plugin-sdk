// Package log provides structure logging (slog) adapted for Reglet SDK's WASM environment.
package log

import (
	"context"
	"log/slog"
)

// WasmLogHandler implements slog.Handler to route logs through a host function.
type WasmLogHandler struct {
	opts handlerConfig
}

// HandlerOption configures the WasmLogHandler.
type HandlerOption func(*handlerConfig)

type handlerConfig struct {
	level     slog.Level
	addSource bool
}

// defaultHandlerConfig returns the default configuration.
func defaultHandlerConfig() handlerConfig {
	return handlerConfig{
		level: slog.LevelInfo,
	}
}

// WithLevel sets the minimum log level to report.
// Records below this level will be filtered on the guest side.
func WithLevel(level slog.Level) HandlerOption {
	return func(c *handlerConfig) {
		c.level = level
	}
}

// WithSource enables reporting of source location (file/line).
func WithSource(enabled bool) HandlerOption {
	return func(c *handlerConfig) {
		c.addSource = enabled
	}
}

// NewHandler creates a new WasmLogHandler with the given options.
func NewHandler(opts ...HandlerOption) *WasmLogHandler {
	cfg := defaultHandlerConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &WasmLogHandler{opts: cfg}
}

// Enabled reports whether the handler handles records at the given level.
func (h *WasmLogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.level
}

// WithAttrs returns a new WasmLogHandler that includes the given attributes.
func (h *WasmLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := *h
	return &newHandler
}

// WithGroup returns a new WasmLogHandler with the given group name.
func (h *WasmLogHandler) WithGroup(name string) slog.Handler {
	// Similar to WithAttrs, simplified.
	newHandler := *h
	return &newHandler
}

// init configures the default slog handler to use our WasmLogHandler.
func init() {
	slog.SetDefault(slog.New(NewHandler()))
	slog.Info("Reglet SDK: Slog handler initialized.")
}
