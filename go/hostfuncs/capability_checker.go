package hostfuncs

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/reglet-dev/reglet-sdk/go/domain/policy"
	"github.com/reglet-dev/reglet-sdk/go/domain/ports"
)

// CapabilityChecker checks if operations are allowed based on granted capabilities.
// It uses the SDK's typed Policy for capability enforcement.
type CapabilityChecker struct {
	policy              ports.Policy
	grantedCapabilities map[string]*entities.GrantSet
	cwd                 string // Current working directory for resolving relative paths
}

// CapabilityCheckerOption configures a CapabilityChecker.
type CapabilityCheckerOption func(*capabilityCheckerConfig)

type capabilityCheckerConfig struct {
	cwd               string
	symlinkResolution bool
}

// WithCapabilityWorkingDirectory sets the working directory for path resolution.
func WithCapabilityWorkingDirectory(cwd string) CapabilityCheckerOption {
	return func(c *capabilityCheckerConfig) {
		c.cwd = cwd
	}
}

// WithCapabilitySymlinkResolution enables or disables symlink resolution.
func WithCapabilitySymlinkResolution(enabled bool) CapabilityCheckerOption {
	return func(c *capabilityCheckerConfig) {
		c.symlinkResolution = enabled
	}
}

// NewCapabilityChecker creates a new capability checker with the given capabilities.
// The cwd is obtained at construction time to avoid side-effects during capability checks.
func NewCapabilityChecker(caps map[string]*entities.GrantSet, opts ...CapabilityCheckerOption) *CapabilityChecker {
	cfg := capabilityCheckerConfig{
		symlinkResolution: true,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	// Get cwd if not provided
	if cfg.cwd == "" {
		cfg.cwd, _ = os.Getwd() // Best effort - empty string will cause relative paths to fail safely
	}

	return &CapabilityChecker{
		policy: policy.NewPolicy(
			policy.WithWorkingDirectory(cfg.cwd),
			policy.WithSymlinkResolution(cfg.symlinkResolution),
		),
		grantedCapabilities: caps,
		cwd:                 cfg.cwd,
	}
}

// Check verifies if a requested capability is granted for a specific plugin.
// This method parses the WASM protocol pattern format used by plugins:
//   - network: "outbound:<port>" or "outbound:<host>"
//   - fs: "read:<path>" or "write:<path>"
//   - env: "<variable>"
//   - exec: "<command>"
func (c *CapabilityChecker) Check(pluginName, kind, pattern string) error {
	grants, ok := c.grantedCapabilities[pluginName]
	if !ok || grants == nil {
		return fmt.Errorf("no capabilities granted to plugin %s", pluginName)
	}

	var allowed bool
	switch kind {
	case "network":
		allowed = c.parseNetworkPattern(pattern, grants)
	case "fs":
		allowed = c.parseFileSystemPattern(pattern, grants)
	case "env":
		allowed = c.parseEnvironmentPattern(pattern, grants)
	case "exec":
		allowed = c.parseExecPattern(pattern, grants)
	default:
		return fmt.Errorf("unknown capability kind: %s", kind)
	}

	if allowed {
		return nil
	}

	return fmt.Errorf("capability denied: %s:%s", kind, pattern)
}

// parseNetworkPattern parses WASM protocol network patterns like "outbound:443" or "outbound:hostname".
func (c *CapabilityChecker) parseNetworkPattern(pattern string, grants *entities.GrantSet) bool {
	parts := strings.SplitN(pattern, ":", 2)
	if len(parts) != 2 {
		return false
	}

	// Legacy format: "outbound:<port_or_host>"
	portOrHost := parts[1]

	// Check if it's a port number
	if port, err := strconv.Atoi(portOrHost); err == nil {
		// It's a port - check with wildcard host
		req := entities.NetworkRequest{Host: "*", Port: port}
		return c.policy.CheckNetwork(req, grants)
	}

	// It's a hostname - check with any port
	// For private network check, we use a special case
	if portOrHost == "private" {
		// Special case for private network access
		req := entities.NetworkRequest{Host: "127.0.0.1", Port: 0}
		return c.policy.CheckNetwork(req, grants)
	}

	// Try as a host pattern - check with common ports
	// The legacy pattern "outbound:hostname" means allow any port to that host
	req := entities.NetworkRequest{Host: portOrHost, Port: 0}
	if c.policy.CheckNetwork(req, grants) {
		return true
	}

	// Also try common ports (80, 443) as fallback
	for _, port := range []int{80, 443, 0} {
		req := entities.NetworkRequest{Host: portOrHost, Port: port}
		if c.policy.CheckNetwork(req, grants) {
			return true
		}
	}

	return false
}

// parseFileSystemPattern parses WASM protocol filesystem patterns like "read:/path" or "write:/path".
func (c *CapabilityChecker) parseFileSystemPattern(pattern string, grants *entities.GrantSet) bool {
	parts := strings.SplitN(pattern, ":", 2)
	if len(parts) != 2 {
		return false
	}

	operation := parts[0]
	path := parts[1]

	req := entities.FileSystemRequest{
		Operation: operation,
		Path:      path,
	}

	return c.policy.CheckFileSystem(req, grants)
}

// parseEnvironmentPattern parses WASM protocol environment variable patterns.
func (c *CapabilityChecker) parseEnvironmentPattern(pattern string, grants *entities.GrantSet) bool {
	req := entities.EnvironmentRequest{
		Variable: pattern,
	}

	return c.policy.CheckEnvironment(req, grants)
}

// parseExecPattern parses WASM protocol exec command patterns.
func (c *CapabilityChecker) parseExecPattern(pattern string, grants *entities.GrantSet) bool {
	req := entities.ExecRequest{
		Command: pattern,
	}

	return c.policy.CheckExec(req, grants)
}

// CheckNetwork performs typed network capability check.
func (c *CapabilityChecker) CheckNetwork(pluginName string, req entities.NetworkRequest) error {
	grants, ok := c.grantedCapabilities[pluginName]
	if !ok || grants == nil {
		return fmt.Errorf("no capabilities granted to plugin %s", pluginName)
	}

	if c.policy.CheckNetwork(req, grants) {
		return nil
	}

	return fmt.Errorf("network capability denied: %s:%d", req.Host, req.Port)
}

// CheckFileSystem performs typed filesystem capability check.
func (c *CapabilityChecker) CheckFileSystem(pluginName string, req entities.FileSystemRequest) error {
	grants, ok := c.grantedCapabilities[pluginName]
	if !ok || grants == nil {
		return fmt.Errorf("no capabilities granted to plugin %s", pluginName)
	}

	if c.policy.CheckFileSystem(req, grants) {
		return nil
	}

	return fmt.Errorf("filesystem capability denied: %s %s", req.Operation, req.Path)
}

// CheckEnvironment performs typed environment capability check.
func (c *CapabilityChecker) CheckEnvironment(pluginName string, req entities.EnvironmentRequest) error {
	grants, ok := c.grantedCapabilities[pluginName]
	if !ok || grants == nil {
		return fmt.Errorf("no capabilities granted to plugin %s", pluginName)
	}

	if c.policy.CheckEnvironment(req, grants) {
		return nil
	}

	return fmt.Errorf("environment capability denied: %s", req.Variable)
}

// CheckExec performs typed exec capability check.
func (c *CapabilityChecker) CheckExec(pluginName string, req entities.ExecRequest) error {
	grants, ok := c.grantedCapabilities[pluginName]
	if !ok || grants == nil {
		return fmt.Errorf("no capabilities granted to plugin %s", pluginName)
	}

	if c.policy.CheckExec(req, grants) {
		return nil
	}

	return fmt.Errorf("exec capability denied: %s", req.Command)
}

// AllowsPrivateNetwork checks if the plugin is allowed to access private network addresses.
func (c *CapabilityChecker) AllowsPrivateNetwork(pluginName string) bool {
	grants, ok := c.grantedCapabilities[pluginName]
	if !ok || grants == nil {
		return false
	}

	// Create a dummy request for private access.
	req := entities.NetworkRequest{Host: "127.0.0.1", Port: 0}
	return c.policy.CheckNetwork(req, grants)
}

// ToCapabilityGetter returns a CapabilityGetter function that uses this checker.
// This allows integration with the exec security features.
func (c *CapabilityChecker) ToCapabilityGetter(pluginName string) CapabilityGetter {
	return func(plugin, capability string) bool {
		// The capability pattern for exec env vars is "env:VARNAME"
		// This can be stored in two places:
		// 1. Exec.Commands as "env:PATH" (Reglet pattern)
		// 2. Env.Variables as "PATH" (SDK pattern)
		if varName, found := strings.CutPrefix(capability, "env:"); found {
			// First try Exec.Commands with "env:VARNAME" pattern
			if err := c.Check(pluginName, "exec", capability); err == nil {
				return true
			}
			// Then try Env.Variables with just "VARNAME"
			if err := c.CheckEnvironment(pluginName, entities.EnvironmentRequest{Variable: varName}); err == nil {
				return true
			}
			return false
		}
		// For other capabilities, use Check with exec kind
		err := c.Check(pluginName, "exec", capability)
		return err == nil
	}
}

// Context helpers for plugin name propagation

type capabilityContextKey struct {
	name string
}

var pluginNameContextKey = &capabilityContextKey{name: "plugin_name"}

// WithCapabilityPluginName adds the plugin name to the context.
func WithCapabilityPluginName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, pluginNameContextKey, name)
}

// CapabilityPluginNameFromContext retrieves the plugin name from the context.
func CapabilityPluginNameFromContext(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(pluginNameContextKey).(string)
	return name, ok
}
