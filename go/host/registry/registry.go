package registry

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/invopop/jsonschema"
	"github.com/reglet-dev/reglet-sdk/go/domain/ports"
)

// registryConfig holds configuration for the Registry.
type registryConfig struct {
	strictMode bool // Fail on duplicate registrations
}

func defaultRegistryConfig() registryConfig {
	return registryConfig{
		strictMode: true, // Secure default: prevent accidental overwrites
	}
}

// RegistryOption configures a Registry instance.
type RegistryOption func(*registryConfig)

// WithStrictMode enables/disables strict mode for duplicate registrations.
// Default is true (fail on duplicates). Disable only for testing or hot-reloading.
func WithStrictMode(enabled bool) RegistryOption {
	return func(c *registryConfig) {
		c.strictMode = enabled
	}
}

// Registry implements CapabilityRegistry.
type Registry struct {
	schemas sync.Map // map[string]string (json schema)
	models  sync.Map // map[string]interface{}
	config  registryConfig
}

// NewRegistry creates a new Registry with the given options.
func NewRegistry(opts ...RegistryOption) ports.CapabilityRegistry {
	cfg := defaultRegistryConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &Registry{config: cfg}
}

// Register adds a schema generated from a Go struct.
func (r *Registry) Register(kind string, model interface{}) error {
	if r.config.strictMode {
		if _, exists := r.schemas.Load(kind); exists {
			return fmt.Errorf("capability %q already registered", kind)
		}
	}

	r.models.Store(kind, model)

	// Generate schema using invopop/jsonschema
	s := jsonschema.Reflect(model)
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal schema for %s: %w", kind, err)
	}
	r.schemas.Store(kind, string(data))
	return nil
}

// GetSchema retrieves the JSON Schema for a capability type.
func (r *Registry) GetSchema(kind string) (string, bool) {
	v, ok := r.schemas.Load(kind)
	if !ok {
		return "", false
	}
	return v.(string), true
}

// List returns all registered capability type names.
func (r *Registry) List() []string {
	var keys []string
	r.schemas.Range(func(k, v interface{}) bool {
		keys = append(keys, k.(string))
		return true
	})
	return keys
}
