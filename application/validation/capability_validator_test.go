package validation_test

import (
	"testing"

	"github.com/reglet-dev/reglet-sdk/application/validation"
	"github.com/reglet-dev/reglet-sdk/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRegistry struct {
	schemas map[string]string
}

func (m *mockRegistry) Register(name string, capability interface{}) error { return nil }
func (m *mockRegistry) GetSchema(name string) (string, bool) {
	s, ok := m.schemas[name]
	return s, ok
}
func (m *mockRegistry) List() []string { return nil }

func TestCapabilityValidator_Validate(t *testing.T) {
	registry := &mockRegistry{
		schemas: map[string]string{
			"network": `{"type": "object", "properties": {"rules": {"type": "array"}}}`,
			"fs":      `{"type": "object", "required": ["rules"], "properties": {"rules": {"type": "array"}}}`,
		},
	}
	validator := validation.NewCapabilityValidator(registry)

	t.Run("Valid Manifest with Network", func(t *testing.T) {
		manifest := &entities.Manifest{
			Name:    "test-plugin",
			Version: "1.0.0",
			Capabilities: entities.GrantSet{
				Network: &entities.NetworkCapability{
					Rules: []entities.NetworkRule{
						{Hosts: []string{"example.com"}, Ports: []string{"443"}},
					},
				},
			},
		}
		res, err := validator.Validate(manifest)
		require.NoError(t, err)
		assert.True(t, res.Valid)
		assert.Empty(t, res.Errors)
	})

	t.Run("Valid Manifest with FS", func(t *testing.T) {
		manifest := &entities.Manifest{
			Version: "1.0.0",
			Capabilities: entities.GrantSet{
				FS: &entities.FileSystemCapability{
					Rules: []entities.FileSystemRule{
						{Read: []string{"/tmp"}},
					},
				},
			},
		}
		res, err := validator.Validate(manifest)
		require.NoError(t, err)
		assert.True(t, res.Valid)
		assert.Empty(t, res.Errors)
	})

	t.Run("Empty GrantSet", func(t *testing.T) {
		manifest := &entities.Manifest{
			Version:      "1.0.0",
			Capabilities: entities.GrantSet{},
		}
		res, err := validator.Validate(manifest)
		require.NoError(t, err)
		assert.True(t, res.Valid)
	})
}
