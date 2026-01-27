package wazero

import (
	"context"
	"fmt"

	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/reglet-dev/reglet-sdk/go/hostfuncs"
	"github.com/tetratelabs/wazero/api"
)

// CapabilityChecker validates operations against granted capabilities.
// It can be used as middleware or directly in handler implementations.
type CapabilityChecker interface {
	// Check verifies if a capability is granted.
	// kind is the capability type (e.g., "network", "fs", "exec", "env").
	// pattern is the capability pattern to check.
	Check(pluginName, kind, pattern string) error

	// CheckNetwork validates network operations.
	CheckNetwork(pluginName string, req entities.NetworkRequest) error

	// CheckFileSystem validates filesystem operations.
	CheckFileSystem(pluginName string, req entities.FileSystemRequest) error

	// CheckEnvironment validates environment variable access.
	CheckEnvironment(pluginName string, req entities.EnvironmentRequest) error

	// CheckExec validates command execution.
	CheckExec(pluginName string, req entities.ExecRequest) error
}

// CapabilityMiddlewareConfig configures capability checking for host functions.
type CapabilityMiddlewareConfig struct {
	// Checker is the capability checker to use.
	Checker CapabilityChecker

	// FunctionCapabilities maps function names to required capabilities.
	// Each entry specifies the capability kind and a function to extract
	// the pattern from the request.
	FunctionCapabilities map[string]CapabilityRequirement
}

// CapabilityRequirement specifies the capability needed for a function.
type CapabilityRequirement struct {
	// Kind is the capability type (e.g., "network", "exec").
	Kind string

	// PatternExtractor extracts the capability pattern from the request.
	// For example, for http_request, it might extract the target host.
	PatternExtractor func(request []byte) (string, error)
}

// WithCapabilityMiddleware creates a middleware that checks capabilities
// before invoking handlers.
func WithCapabilityMiddleware(checker CapabilityChecker) hostfuncs.Middleware {
	return func(next hostfuncs.ByteHandler) hostfuncs.ByteHandler {
		return func(ctx context.Context, payload []byte) ([]byte, error) {
			// Get function name from context
			hctx, ok := ctx.(hostfuncs.HostContext)
			if !ok {
				return next(ctx, payload)
			}

			funcName := hctx.FunctionName()
			if funcName == "" {
				return next(ctx, payload)
			}

			// Get plugin name - try context first, then look for module name
			pluginName, _ := PluginNameFromContext(ctx)
			if pluginName == "" {
				// Fallback to any plugin name set in HostContext
				if name, ok := ctx.Value(pluginNameKey).(string); ok {
					pluginName = name
				}
			}

			// For now, pass through to handler
			// Specific capability checks should be done in the handlers
			// based on the request content
			_ = pluginName // Will be used when specific capability checks are added
			return next(ctx, payload)
		}
	}
}

// WazeroCapabilityHandler creates a wazero GoModuleFunc that wraps a handler
// with capability checking.
func WazeroCapabilityHandler(
	handler func(ctx context.Context, mod api.Module, stack []uint64),
	checker CapabilityChecker,
) api.GoModuleFunc {
	return func(ctx context.Context, mod api.Module, stack []uint64) {
		// Add plugin name to context
		pluginName := GetPluginName(ctx, mod)
		ctx = WithPluginName(ctx, pluginName)

		// Call the wrapped handler
		handler(ctx, mod, stack)
	}
}

// NewCapabilityGetterFromChecker creates a CapabilityGetter function from a CapabilityChecker.
// This allows the SDK's exec security features to use the capability checker.
func NewCapabilityGetterFromChecker(checker CapabilityChecker) hostfuncs.CapabilityGetter {
	return func(pluginName, capability string) bool {
		err := checker.Check(pluginName, "exec", capability)
		return err == nil
	}
}

// CapabilityDeniedError represents a capability check failure.
type CapabilityDeniedError struct {
	PluginName string
	Kind       string
	Pattern    string
}

func (e *CapabilityDeniedError) Error() string {
	return fmt.Sprintf("capability denied: plugin %q requires %s:%s", e.PluginName, e.Kind, e.Pattern)
}
