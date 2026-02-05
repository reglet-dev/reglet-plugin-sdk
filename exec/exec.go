package exec

import (
	"context"
	"time"

	"github.com/reglet-dev/reglet-sdk/domain/ports"
	"github.com/reglet-dev/reglet-sdk/infrastructure/wasm"
)

// Re-export types from ports for API compatibility
type (
	CommandRequest  = ports.CommandRequest
	CommandResponse = ports.CommandResult
)

// runConfig holds the configuration for command execution.
// This struct is unexported to enforce the functional options pattern.
type runConfig struct {
	runner  ports.CommandRunner
	workdir string        // Working directory for command (default: inherit)
	env     []string      // Environment variables (default: inherit)
	timeout time.Duration // Execution timeout (default: 30s)
}

// defaultRunConfig returns secure defaults for command execution.
// These defaults align with constitution requirements for secure-by-default.
func defaultRunConfig() runConfig {
	return runConfig{
		runner:  wasm.NewExecAdapter(),
		workdir: "",               // Empty = inherit from host
		env:     nil,              // nil = inherit from host
		timeout: 30 * time.Second, // 30s default per spec
	}
}

// RunOption is a functional option for configuring command execution.
// Use With* functions to create options.
type RunOption func(*runConfig)

// WithRunner sets the command runner to use.
// This is useful for injecting mocks during testing.
func WithRunner(r ports.CommandRunner) RunOption {
	return func(c *runConfig) {
		if r != nil {
			c.runner = r
		}
	}
}

// WithWorkdir sets the working directory for the command.
// If not specified, the host's current working directory is used.
func WithWorkdir(dir string) RunOption {
	return func(c *runConfig) {
		c.workdir = dir
	}
}

// WithEnv sets the environment variables for the command.
// Each entry should be in KEY=VALUE format.
// If not specified, the host's environment is inherited.
func WithEnv(env []string) RunOption {
	return func(c *runConfig) {
		c.env = env
	}
}

// WithExecTimeout sets the execution timeout for the command.
// Default is 30 seconds per constitution specification.
// A zero or negative duration is ignored (uses default).
func WithExecTimeout(d time.Duration) RunOption {
	return func(c *runConfig) {
		if d > 0 {
			c.timeout = d
		}
	}
}

// applyRunOptions applies functional options and returns the configuration.
// This is used by the Run function to process variadic options.
func applyRunOptions(opts ...RunOption) runConfig {
	cfg := defaultRunConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Run executes a command on the host system.
// Requires "exec:<command>" capability.
//
// Options:
//   - WithWorkdir(dir): Set working directory (default: inherit from host)
//   - WithEnv(env): Set environment variables (default: inherit from host)
//   - WithExecTimeout(d): Set execution timeout (default: 30s)
//   - WithRunner(r): Inject custom runner (for testing)
//
// Example:
//
//	resp, err := exec.Run(ctx, exec.CommandRequest{
//	    Command: "ls",
//	    Args:    []string{"-la"},
//	}, exec.WithWorkdir("/tmp"), exec.WithExecTimeout(10*time.Second))
func Run(ctx context.Context, req CommandRequest, opts ...RunOption) (*CommandResponse, error) {
	// Apply functional options
	cfg := applyRunOptions(opts...)

	// Override request fields from options if not already set
	if req.Dir == "" && cfg.workdir != "" {
		req.Dir = cfg.workdir
	}
	if req.Env == nil && cfg.env != nil {
		req.Env = cfg.env
	}
	if req.Timeout == 0 {
		req.Timeout = int(cfg.timeout.Seconds())
	}

	return cfg.runner.Run(ctx, req)
}
