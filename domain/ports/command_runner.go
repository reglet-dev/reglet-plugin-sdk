package ports

import (
	"context"
)

// CommandRunner defines the interface for command execution.
// Infrastructure adapters implement this to provide exec functionality.
type CommandRunner interface {
	// Run executes a command and returns the result.
	Run(ctx context.Context, req CommandRequest) (*CommandResult, error)
}

// CommandRequest holds parameters for command execution.
type CommandRequest struct {
	Command string
	Args    []string
	Dir     string
	Env     []string
	Timeout int // milliseconds
}

// CommandResult represents the result of a command execution.
type CommandResult struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int64
	IsTimeout  bool
}
