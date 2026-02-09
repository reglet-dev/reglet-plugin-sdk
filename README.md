# Reglet Plugin SDK

Go SDK for writing WASM plugins that run on the [Reglet](https://github.com/reglet-dev/reglet) host. Handles host communication, typed operations, schema generation, and capability declarations.

## Installation

```bash
go get github.com/reglet-dev/reglet-plugin-sdk
```

Requires Go 1.25+ and targets `GOOS=wasip1 GOARCH=wasm`.

## Quick Start

A plugin defines its metadata, registers services with typed operations, and implements handlers.

### 1. Define the plugin

```go
// core/plugin.go
package core

import (
	"github.com/reglet-dev/reglet-plugin-sdk/application/plugin"
	"github.com/reglet-dev/reglet-plugin-sdk/domain/entities"
)

var Plugin = plugin.DefinePlugin(plugin.PluginDef{
	Name:        "http",
	Version:     "1.0.0",
	Description: "HTTP/HTTPS request checking and validation",
	Config:      &HTTPConfig{},
	Capabilities: entities.GrantSet{
		Network: &entities.NetworkCapability{
			Rules: []entities.NetworkRule{
				{Hosts: []string{"*"}, Ports: []string{"80", "443"}},
			},
		},
	},
})

type HTTPConfig struct {
	URL            string `json:"url" jsonschema:"required,description=URL to request"`
	Method         string `json:"method,omitempty" jsonschema:"enum=GET,enum=POST,default=GET"`
	ExpectedStatus int    `json:"expected_status,omitempty" jsonschema:"default=200"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"default=30"`
}
```

### 2. Register typed operations

```go
// services/check.go
package services

import (
	"context"
	"fmt"

	"github.com/reglet-dev/reglet-plugin-sdk/application/plugin"
	"github.com/reglet-dev/reglet-plugin-sdk/domain/ports"
	"my-plugin/core"
)

type CheckInput struct {
	URL            string `json:"url" jsonschema:"required,description=URL to request"`
	ExpectedStatus int    `json:"expected_status,omitempty" jsonschema:"default=200"`
}

type CheckOutput struct {
	StatusCode int    `json:"status_code"`
	URL        string `json:"url"`
}

type HTTPService struct {
	plugin.Service `name:"http" desc:"HTTP request checking"`

	Check plugin.Op[CheckInput, CheckOutput] `desc:"Check URL" method:"CheckHandler"`
}

func init() {
	plugin.RegisterOp[CheckInput, CheckOutput]("Check",
		plugin.Example[CheckInput, CheckOutput]{
			Name:  "basic",
			Input: CheckInput{URL: "https://example.com", ExpectedStatus: 200},
			ExpectedOutput: &CheckOutput{
				StatusCode: 200,
				URL:        "https://example.com",
			},
		},
	)
	plugin.MustRegisterService(core.Plugin, &HTTPService{})
}

func (s *HTTPService) CheckHandler(ctx context.Context, in *CheckInput) (*CheckOutput, error) {
	client := plugin.GetClient[ports.HTTPClient](ctx)

	resp, err := client.Get(ctx, in.URL)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return &CheckOutput{
		StatusCode: resp.StatusCode,
		URL:        in.URL,
	}, nil
}
```

### 3. Wire up the entry point

```go
// plugin.go
package main

import (
	"github.com/reglet-dev/reglet-plugin-sdk/application/plugin"
	_ "my-plugin/services" // auto-registers via init()
)

func main() {
	plugin.Register(&myPlugin{})
}
```

### 4. Build

```bash
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm .
```

## Plugin Interface

Every plugin implements two methods:

```go
type Plugin interface {
	Manifest(ctx context.Context) (*entities.Manifest, error)
	Check(ctx context.Context, config []byte) (*entities.Result, error)
}
```

`Manifest` returns metadata (name, version, capabilities, services, config schema). `Check` executes the plugin logic. If you use `DefinePlugin` + typed operations, the SDK generates the manifest and routes config to the correct handler automatically.

## Typed Operations

The `Op[I, O]` generic type gives you type-safe handlers with auto-generated JSON schemas.

```go
type MyService struct {
	plugin.Service `name:"dns" desc:"DNS resolution"`

	Resolve plugin.Op[ResolveInput, ResolveOutput] `desc:"Resolve hostname" method:"ResolveHandler"`
}
```

- `I` and `O` are plain structs with `json` and `jsonschema` tags
- The SDK parses input JSON into `*I`, calls your handler, and serializes `*O` into the result
- `RegisterOp` captures type info and examples before `MustRegisterService` wires everything up
- Field names are auto-converted to snake_case for the operation name (`Resolve` -> `resolve`)

### Client injection

Handlers get clients from context, injected by the host runtime:

```go
func (s *DNSService) ResolveHandler(ctx context.Context, in *ResolveInput) (*ResolveOutput, error) {
	resolver := plugin.GetClient[ports.DNSResolver](ctx)
	records, err := resolver.LookupHost(ctx, in.Hostname)
	// ...
}
```

`GetClient[T]` panics if the client is missing or wrong type. Use `TryGetClient[T]` for a safe version.

### Examples

Register examples alongside operations for documentation and the CLI help text:

```go
plugin.RegisterOp[ResolveInput, ResolveOutput]("Resolve",
	plugin.Example[ResolveInput, ResolveOutput]{
		Name:           "basic_a",
		Description:    "Resolve A record",
		Input:          ResolveInput{Hostname: "example.com", RecordType: "A"},
		ExpectedOutput: &ResolveOutput{Hostname: "example.com", Records: []string{"93.184.216.34"}},
	},
	plugin.Example[ResolveInput, ResolveOutput]{
		Name:          "nxdomain",
		Description:   "Non-existent domain",
		Input:         ResolveInput{Hostname: "invalid.test"},
		ExpectedError: "DNS lookup failed",
	},
)
```

## Capabilities

Plugins declare what they need from the host. The host decides what to grant.

```go
Capabilities: entities.GrantSet{
	Network: &entities.NetworkCapability{
		Rules: []entities.NetworkRule{
			{Hosts: []string{"api.example.com"}, Ports: []string{"443"}},
		},
	},
	FS: &entities.FileSystemCapability{
		Rules: []entities.FileSystemRule{
			{Read: []string{"/etc/hosts"}},
		},
	},
	Exec: &entities.ExecCapability{
		Commands: []string{"/usr/bin/systemctl"},
	},
	Env: &entities.EnvCapability{
		Vars: []string{"HOME", "AWS_REGION"},
	},
	KV: &entities.KVCapability{
		Rules: []entities.KVRule{
			{Keys: []string{"cache-*"}, Op: "read-write"},
		},
	},
}
```

Request the minimum you need. Prefer specific hosts over wildcards, specific commands over shells.

## Domain Ports

The SDK defines interfaces for host-provided services. WASM adapters implement these using host function imports.

| Interface | Methods |
|-----------|---------|
| `ports.HTTPClient` | `Do`, `Get`, `Post` |
| `ports.DNSResolver` | `LookupHost`, `LookupCNAME`, `LookupMX`, `LookupTXT`, `LookupNS` |
| `ports.TCPDialer` | `Dial`, `DialWithTimeout`, `DialSecure` |
| `ports.SMTPClient` | `Connect` |
| `ports.CommandRunner` | `Run` |

## High-Level Check Functions

The `net/` and `exec/` packages provide ready-made check functions with functional options:

```go
import sdknet "github.com/reglet-dev/reglet-plugin-sdk/net"

result, err := sdknet.RunHTTPCheck(ctx, configMap)
result, err := sdknet.RunDNSCheck(ctx, configMap)
result, err := sdknet.RunTCPCheck(ctx, configMap)
result, err := sdknet.RunSMTPCheck(ctx, configMap)
```

Inject mocks for testing:

```go
result, err := sdknet.RunHTTPCheck(ctx, cfg, sdknet.WithHTTPClient(mockClient))
```

## Config Helpers

Safe extraction from `map[string]any`:

```go
import "github.com/reglet-dev/reglet-plugin-sdk/application/config"

hostname, err := config.MustGetString(cfg, "hostname")     // error if missing
port := config.GetIntDefault(cfg, "port", 443)              // default if missing
verbose, ok := config.GetBool(cfg, "verbose")               // returns ok=false if missing
```

## Results

```go
entities.ResultSuccess("check passed", map[string]any{"status": "ok"})
entities.ResultFailure("check failed", map[string]any{"reason": "timeout"})
entities.ResultError(entities.NewErrorDetail("network", "connection refused"))
```

Typed handlers return `(*Output, error)` and the SDK wraps it into a Result automatically.

## Logging

Import the log package to route `slog` calls through the host:

```go
import (
	"log/slog"
	_ "github.com/reglet-dev/reglet-plugin-sdk/log"
)

slog.InfoContext(ctx, "resolving", "hostname", hostname)
```

## Testing

```go
import plugintest "github.com/reglet-dev/reglet-plugin-sdk/testing"

func TestPlugin(t *testing.T) {
	plugintest.RunPluginTests(t, myPlugin, []plugintest.TestCase{
		{
			Name:   "basic",
			Config: map[string]any{"url": "https://example.com"},
			Validate: func(t *testing.T, r *entities.Result) {
				plugintest.AssertSuccess(t, r)
				plugintest.AssertDataField(t, r, "status_code", 200)
			},
		},
	})
}
```

For typed handlers, test directly:

```go
func TestCheckHandler(t *testing.T) {
	svc := &HTTPService{}
	ctx := plugin.WithClient(context.Background(), mockHTTPClient)

	out, err := svc.CheckHandler(ctx, &CheckInput{URL: "https://example.com"})
	require.NoError(t, err)
	assert.Equal(t, 200, out.StatusCode)
}
```

## Limitations

- **Single-threaded**: WASI Preview 1; goroutines work for logical concurrency only
- **Buffered I/O**: HTTP responses and command output are fully buffered in memory
- **100 MB memory limit**: The SDK tracks allocations and panics if exceeded
- **No raw sockets or UDP**: Only HTTP, DNS, TCP, and SMTP via host functions

## License

See [LICENSE](LICENSE)
