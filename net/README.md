# Net Package

The `net` package provides network operations for Reglet WASM plugins, implementing a Hexagonal Architecture that separates domain logic from infrastructure adapters.

## Overview

This package offers high-level "Check" functions for verifying network connectivity (TCP, DNS, HTTP, SMTP). These functions are designed to be:
- **Secure by Default**: Using safe defaults for timeouts and configurations.
- **Testable**: Supporting dependency injection via functional options for mock-based testing.
- **WASM-Optimized**: Routing traffic through the host environment when running in WASM.

## Check Functions

### RunTCPCheck

Performs a TCP connection check.

```go
cfg := config.Config{
    "host":       "example.com",
    "port":       443,
    "timeout_ms": 5000,
}
result, err := sdknet.RunTCPCheck(ctx, cfg)
```

### RunDNSCheck

Performs a DNS lookup.

```go
cfg := config.Config{
    "hostname":    "example.com",
    "record_type": "A", // A, AAAA, CNAME, MX, TXT, NS
}
result, err := sdknet.RunDNSCheck(ctx, cfg)
```

### RunHTTPCheck

Performs an HTTP request.

```go
cfg := config.Config{
    "url":            "https://example.com",
    "method":         "GET",
    "expected_status": 200,
}
result, err := sdknet.RunHTTPCheck(ctx, cfg)
```

### RunSMTPCheck

Performs an SMTP connection check.

```go
cfg := config.Config{
    "host":         "smtp.example.com",
    "port":         587,
    "use_starttls": true,
}
result, err := sdknet.RunSMTPCheck(ctx, cfg)
```

## Advanced Usage & Testing

The package exposes functional options to inject custom adapters (ports), enabling mock-based unit testing without a WASM runtime.

### Mocking TCP

```go
mockDialer := &MyMockTCPDialer{} // Implements ports.TCPDialer
result, err := sdknet.RunTCPCheck(ctx, cfg, sdknet.WithTCPDialer(mockDialer))
```

### Mocking HTTP

```go
mockClient := &MyMockHTTPClient{} // Implements ports.HTTPClient
result, err := sdknet.RunHTTPCheck(ctx, cfg, sdknet.WithHTTPClient(mockClient))
```

### Custom Resolver/Transport

For use cases outside of the check functions, you can create configured clients that use the underlying WASM adapters:

```go
// Create a DNS resolver with custom timeout
resolver := sdknet.NewResolver(
    sdknet.WithDNSTimeout(10 * time.Second),
    sdknet.WithNameserver("1.1.1.1:53"),
)

// Create an HTTP client
client := sdknet.NewTransport(
    sdknet.WithHTTPTimeout(60 * time.Second),
)
```

## Architecture

- **Domain/Ports**: Interfaces defined in `go/domain/ports` (e.g., `TCPDialer`, `HTTPClient`).
- **Infrastructure/WASM**: Adapters in `go/infrastructure/wasm` implement these ports using `//go:wasmimport`.
- **Public API**: `go/net` functions orchestrate logic using ports, defaulting to WASM adapters.

This design allows the same code to be tested natively (using mocks) and run in WASM (using host functions).