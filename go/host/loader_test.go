package host_test

import (
	"testing"

	"github.com/reglet-dev/reglet-sdk/go/host"
	"github.com/reglet-dev/reglet-sdk/go/host/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// LoaderIntegrationSuite tests the Loader with full integration.
type LoaderIntegrationSuite struct {
	suite.Suite
	registry *registry.Registry
	loader   *host.Loader
}

func (s *LoaderIntegrationSuite) SetupTest() {
	// Create and configure registry
	reg := registry.NewRegistry(registry.WithStrictMode(false))
	err := reg.Register("network", map[string]any{"type": "string"})
	s.Require().NoError(err)
	err = reg.Register("fs", map[string]any{"type": "string"})
	s.Require().NoError(err)
	err = reg.Register("env", map[string]any{"type": "string"})
	s.Require().NoError(err)
	err = reg.Register("exec", map[string]any{"type": "string"})
	s.Require().NoError(err)
	err = reg.Register("kv", map[string]any{"type": "string"})
	s.Require().NoError(err)

	s.registry = reg.(*registry.Registry)
	s.loader = host.NewLoader(host.WithRegistry(reg))
}

func (s *LoaderIntegrationSuite) TestValidManifest() {
	yaml := `
name: "test-plugin"
version: "1.0.0"
capabilities:
  - kind: network
    pattern: "outbound:example.com:80"
  - kind: fs
    pattern: "read:/data/**"
`
	manifest, err := s.loader.LoadManifest([]byte(yaml), nil)
	s.Require().NoError(err)
	s.Equal("test-plugin", manifest.Name)
	s.Len(manifest.Capabilities, 2)

	// Check capabilities
	// Note: YAML unmarshal might not preserve order if it was a map, but here it's a list.
	// Order should be preserved.
	s.Equal("network", manifest.Capabilities[0].Category)
	s.Equal("outbound:example.com:80", manifest.Capabilities[0].Resource)

	s.Equal("fs", manifest.Capabilities[1].Category)
	s.Equal("read:/data/**", manifest.Capabilities[1].Resource)
}

func (s *LoaderIntegrationSuite) TestManifestWithMultipleRules() {
	yaml := `
name: "multi-rule-plugin"
version: "1.0.0"
capabilities:
  - kind: network
    pattern: "outbound:api.internal:80"
  - kind: network
    pattern: "outbound:*.external.com:443"
  - kind: kv
    pattern: "read:config/*"
  - kind: kv
    pattern: "read-write:cache/*"
`
	manifest, err := s.loader.LoadManifest([]byte(yaml), nil)
	s.Require().NoError(err)
	s.Len(manifest.Capabilities, 4)

	s.Equal("network", manifest.Capabilities[0].Category)
	s.Equal("outbound:api.internal:80", manifest.Capabilities[0].Resource)

	s.Equal("network", manifest.Capabilities[1].Category)
	s.Equal("outbound:*.external.com:443", manifest.Capabilities[1].Resource)
}

func (s *LoaderIntegrationSuite) TestInvalidYAML() {
	yaml := `
name: "test-plugin"
version: "1.0.0"
capabilities:
  network: "should be a list of objects"
`
	_, err := s.loader.LoadManifest([]byte(yaml), nil)
	s.Require().Error(err)
}

func (s *LoaderIntegrationSuite) TestMissingSchemaRegistration() {
	// Create loader with empty registry
	emptyReg := registry.NewRegistry()
	loaderEmpty := host.NewLoader(host.WithRegistry(emptyReg))

	yaml := `
name: "test-plugin"
version: "1.0.0"
capabilities:
  - kind: network
    pattern: "outbound:example.com:443"
`
	_, err := loaderEmpty.LoadManifest([]byte(yaml), nil)
	s.Require().Error(err)
	s.Contains(err.Error(), "no schema registered for capability network") // If validation is enabled
}

func (s *LoaderIntegrationSuite) TestEnvCapability() {
	yaml := `
name: "env-plugin"
version: "1.0.0"
capabilities:
  - kind: env
    pattern: "APP_*,DEBUG"
`
	manifest, err := s.loader.LoadManifest([]byte(yaml), nil)
	s.Require().NoError(err)
	s.Len(manifest.Capabilities, 1)
	s.Equal("env", manifest.Capabilities[0].Category)
	s.Equal("APP_*,DEBUG", manifest.Capabilities[0].Resource)
}

func (s *LoaderIntegrationSuite) TestExecCapability() {
	yaml := `
name: "exec-plugin"
version: "1.0.0"
capabilities:
  - kind: exec
    pattern: "/usr/bin/ls,/usr/bin/cat"
`
	manifest, err := s.loader.LoadManifest([]byte(yaml), nil)
	s.Require().NoError(err)
	s.Len(manifest.Capabilities, 1)
	s.Equal("exec", manifest.Capabilities[0].Category)
	s.Equal("/usr/bin/ls,/usr/bin/cat", manifest.Capabilities[0].Resource)
}

func TestLoaderIntegrationSuite(t *testing.T) {
	suite.Run(t, new(LoaderIntegrationSuite))
}

// Additional standalone tests for backwards compatibility
func TestLoader_Integration(t *testing.T) {
	// 1. Setup Registry
	reg := registry.NewRegistry(registry.WithStrictMode(false))
	err := reg.Register("network", map[string]any{"type": "string"})
	require.NoError(t, err)
	err = reg.Register("fs", map[string]any{"type": "string"})
	require.NoError(t, err)

	// 2. Setup Loader
	loader := host.NewLoader(
		host.WithRegistry(reg),
	)

	t.Run("Valid Manifest", func(t *testing.T) {
		yaml := `
name: "test-plugin"
version: "1.0.0"
capabilities:
  - kind: network
    pattern: "outbound:example.com:80"
`
		manifest, err := loader.LoadManifest([]byte(yaml), nil)
		require.NoError(t, err)
		assert.Equal(t, "test-plugin", manifest.Name)
		assert.Len(t, manifest.Capabilities, 1)
		assert.Equal(t, "network", manifest.Capabilities[0].Category)
	})

	t.Run("Missing Capability Registration", func(t *testing.T) {
		emptyReg := registry.NewRegistry()
		loaderEmpty := host.NewLoader(host.WithRegistry(emptyReg))

		yaml2 := `
name: "test-plugin"
version: "1.0.0"
capabilities:
  - kind: network
    pattern: "outbound:example.com:443"
`
		_, err := loaderEmpty.LoadManifest([]byte(yaml2), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no schema registered for capability network")
	})
}
