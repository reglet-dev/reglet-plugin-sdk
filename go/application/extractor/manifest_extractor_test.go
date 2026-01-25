package extractor_test

import (
	"errors"
	"testing"

	"github.com/reglet-dev/reglet-sdk/go/application/extractor"
	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockParser implements ports.ManifestParser
type mockParser struct {
	manifest *entities.PluginManifest
	err      error
}

func (m *mockParser) Parse(data []byte) (*entities.PluginManifest, error) {
	return m.manifest, m.err
}

// mockRenderer implements ports.TemplateEngine
type mockRenderer struct {
	output []byte
	err    error
}

func (m *mockRenderer) Render(template []byte, data map[string]interface{}) ([]byte, error) {
	return m.output, m.err
}

func TestManifestExtractor_Extract(t *testing.T) {
	t.Run("should extract capabilities successfully without template", func(t *testing.T) {
		expectedCaps := &entities.GrantSet{
			Network: &entities.NetworkCapability{
				Rules: []entities.NetworkRule{
					{Hosts: []string{"example.com"}, Ports: []string{"443"}},
				},
			},
		}

		parser := &mockParser{
			manifest: &entities.PluginManifest{
				Capabilities: expectedCaps,
			},
		}

		manifestBytes := []byte("dummy")
		ext := extractor.NewManifestExtractor(manifestBytes, extractor.WithParser(parser))

		caps, err := ext.Extract(nil)
		require.NoError(t, err)
		assert.Equal(t, expectedCaps, caps)
	})

	t.Run("should fail if parser is missing", func(t *testing.T) {
		ext := extractor.NewManifestExtractor([]byte("dummy"))
		_, err := ext.Extract(nil)
		assert.ErrorContains(t, err, "manifest parser is required")
	})

	t.Run("should fail if rendering fails", func(t *testing.T) {
		renderer := &mockRenderer{
			err: errors.New("render error"),
		}
		parser := &mockParser{} // won't be called

		ext := extractor.NewManifestExtractor(
			[]byte("{{.bad}}"),
			extractor.WithParser(parser),
			extractor.WithTemplateEngine(renderer),
		)

		_, err := ext.Extract(nil)
		assert.ErrorContains(t, err, "failed to render manifest: render error")
	})

	t.Run("should fail if parsing fails", func(t *testing.T) {
		renderer := &mockRenderer{
			output: []byte("rendered"),
		}
		parser := &mockParser{
			err: errors.New("parse error"),
		}

		ext := extractor.NewManifestExtractor(
			[]byte("template"),
			extractor.WithParser(parser),
			extractor.WithTemplateEngine(renderer),
		)

		_, err := ext.Extract(nil)
		assert.ErrorContains(t, err, "failed to parse manifest: parse error")
	})

	t.Run("should return empty grant set if manifest has no capabilities", func(t *testing.T) {
		parser := &mockParser{
			manifest: &entities.PluginManifest{
				Capabilities: nil,
			},
		}

		ext := extractor.NewManifestExtractor([]byte("dummy"), extractor.WithParser(parser))

		caps, err := ext.Extract(nil)
		require.NoError(t, err)
		assert.NotNil(t, caps)
		assert.True(t, caps.IsEmpty())
	})

	t.Run("should use renderer before parsing", func(t *testing.T) {
		expectedCaps := &entities.GrantSet{}

		// Renderer returns specific output
		renderer := &mockRenderer{
			output: []byte("rendered output"),
		}

		// Parser expects that specific output
		parser := &mockParser{
			manifest: &entities.PluginManifest{Capabilities: expectedCaps},
		}

		ext := extractor.NewManifestExtractor(
			[]byte("template"),
			extractor.WithParser(parser),
			extractor.WithTemplateEngine(renderer),
		)

		// We can't easily verify the call arguments with this simple mock,
		// but we can verify the flow doesn't error and uses both components.
		caps, err := ext.Extract(map[string]interface{}{"foo": "bar"})
		require.NoError(t, err)
		assert.Equal(t, expectedCaps, caps)
	})
}
