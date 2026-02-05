package entities

import (
	"time"
)

// Config represents SDK configuration settings.
// These settings control SDK behavior across all operations.
type Config struct {
	// LogLevel is the logging verbosity level (e.g., "debug", "info", "warn", "error").
	LogLevel string `json:"log_level,omitempty"`

	// DefaultTimeout is the default timeout for operations.
	DefaultTimeout time.Duration `json:"default_timeout"`

	// MaxRetries is the maximum number of retry attempts for failed operations.
	MaxRetries int `json:"max_retries"`

	// EnableLogging controls whether SDK operations are logged.
	EnableLogging bool `json:"enable_logging"`
}

// DefaultConfig returns the default SDK configuration.
// These defaults align with constitution requirements for secure-by-default.
func DefaultConfig() Config {
	return Config{
		DefaultTimeout: 30 * time.Second,
		MaxRetries:     3,
		EnableLogging:  true,
		LogLevel:       "info",
	}
}

// ConfigOption is a functional option for configuring SDK settings.
type ConfigOption func(*Config)

// WithDefaultTimeout sets the default timeout for operations.
func WithDefaultTimeout(d time.Duration) ConfigOption {
	return func(c *Config) {
		if d > 0 {
			c.DefaultTimeout = d
		}
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(n int) ConfigOption {
	return func(c *Config) {
		if n >= 0 {
			c.MaxRetries = n
		}
	}
}

// WithLogging enables or disables logging.
func WithLogging(enabled bool) ConfigOption {
	return func(c *Config) {
		c.EnableLogging = enabled
	}
}

// WithLogLevel sets the logging verbosity level.
func WithLogLevel(level string) ConfigOption {
	return func(c *Config) {
		c.LogLevel = level
	}
}

// NewConfig creates a new Config with the given options.
func NewConfig(opts ...ConfigOption) Config {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
