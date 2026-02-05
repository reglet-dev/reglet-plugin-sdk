# Exec Package

The `exec` package provides command execution capabilities for Reglet WASM plugins. It implements a Hexagonal Architecture that separates domain logic from infrastructure adapters, allowing for easy testing and modularity.

## Overview

This package wraps the host's command execution functionality, translating Go-style command requests into wire format messages that cross the WASM boundary. All command execution happens on the host side with explicit capability grants.

## Security Model

- **Requires Capability**: `exec` or `exec:<pattern>` capability grant.
- **Sandboxed**: Commands run in a host-controlled environment.
- **No Direct Access**: Plugins cannot directly access the host filesystem or processes.
- **Configurable Limits**: The host enforces timeouts, output size limits, and allowed commands.

## Basic Usage

```go
package main

import (
    "context"

    "github.com/reglet-dev/reglet-sdk/go"
    "github.com/reglet-dev/reglet-sdk/go/exec"
)

type MyPlugin struct{}

func (p *MyPlugin) Check(ctx context.Context, config sdk.Config) (sdk.Evidence, error) {
    // Simple command execution
    req := exec.CommandRequest{
        Command: "ls",
        Args:    []string{"-la", "/tmp"},
    }

    result, err := exec.Run(ctx, req)
    if err != nil {
        return sdk.Failure("exec", err.Error()), nil
    }

    return sdk.Success(map[string]interface{}{
        "stdout":      result.Stdout,
        "stderr":      result.Stderr,
        "exit_code":   result.ExitCode,
        "duration_ms": result.DurationMs,
    }), nil
}
```

## Advanced Usage

### Functional Options

The `Run` function supports functional options for configuration:

```go
result, err := exec.Run(ctx, req, 
    exec.WithWorkdir("/var/log"),
    exec.WithEnv([]string{"DEBUG=true"}),
    exec.WithExecTimeout(10 * time.Second),
)
```

### Mocking for Tests

You can inject a mock runner to unit test your plugin logic without a WASM runtime:

```go
import "github.com/reglet-dev/reglet-sdk/go/domain/ports"

// In your test:
mockRunner := &MyMockRunner{} // Implements ports.CommandRunner
result, err := exec.Run(ctx, req, exec.WithRunner(mockRunner))
```

### Timeout Handling

Use Go's context or the `WithExecTimeout` option to enforce timeouts:

```go
// 5 second timeout via context
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

result, err := exec.Run(ctx, req)
```

## API Reference

### CommandRequest

```go
type CommandRequest struct {
    Command string   // Command to execute (required)
    Args    []string // Command arguments (optional)
    Dir     string   // Working directory (optional, defaults to host's choice)
    Env     []string // Environment variables as "KEY=VALUE" pairs (optional)
    Timeout int      // Timeout in seconds (optional)
}
```

### CommandResponse

```go
type CommandResponse struct {
    Stdout     string // Standard output from command
    Stderr     string // Standard error from command
    ExitCode   int    // Exit code (0 = success)
    DurationMs int64  // Execution duration in milliseconds
    IsTimeout  bool   // True if command timed out
}
```

### Functions

#### Run

```go
func Run(ctx context.Context, req CommandRequest, opts ...RunOption) (*CommandResponse, error)
```

Executes a command on the host system. Returns the command output and metadata, or an error if the command cannot be executed.

### Options

- `WithWorkdir(dir string)`: Sets the working directory.
- `WithEnv(env []string)`: Sets environment variables.
- `WithExecTimeout(d time.Duration)`: Sets the execution timeout.
- `WithRunner(r ports.CommandRunner)`: Injects a custom runner (useful for testing).

## Architecture

- **Domain/Ports**: The `CommandRunner` interface is defined in `go/domain/ports`.
- **Infrastructure/WASM**: The `ExecAdapter` in `go/infrastructure/wasm` implements the port using host functions.
- **Public API**: The `go/exec` package provides the `Run` function which orchestrates the call, defaulting to the WASM adapter.

This design ensures the SDK is testable on native environments while providing seamless host integration in WASM.

## See Also

- [Main SDK Documentation](../README.md)
- [Net Package Documentation](../net/README.md)