package extractor_test

import (
	"fmt"
	"testing"

	"github.com/reglet-dev/reglet-sdk/go/application/extractor"
	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockManifestParser is a mock implementation of ManifestParser
type MockManifestParser struct {
	mock.Mock
}

func (m *MockManifestParser) Parse(data []byte) (*entities.Manifest, error) {
	args := m.Called(data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Manifest), args.Error(1)
}

// mockRenderer implements ports.TemplateEngine
type mockRenderer struct {
	mock.Mock
}

func (m *mockRenderer) Render(template []byte, data map[string]interface{}) ([]byte, error) {
	args := m.Called(template, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func TestManifestExtractor_Extract(t *testing.T) {
	t.Run("should extract capabilities successfully without template", func(t *testing.T) {
		expectedCaps := entities.GrantSet{
			Network: &entities.NetworkCapability{
				Rules: []entities.NetworkRule{
					{Hosts: []string{"google.com"}, Ports: []string{"*"}},
				},
			},
			FS: &entities.FileSystemCapability{
				Rules: []entities.FileSystemRule{
					{Read: []string{"/tmp"}},
				},
			},
		}

		mockParser := new(MockManifestParser)
		manifestBytes := []byte("manifest-data")
		mockParser.On("Parse", manifestBytes).Return(&entities.Manifest{
			Capabilities: expectedCaps,
		}, nil)

		ext := extractor.NewManifestExtractor(manifestBytes, extractor.WithParser(mockParser))

		caps, err := ext.Extract(nil)
		require.NoError(t, err)

		// Verify capabilities present
		assert.NotNil(t, caps)
		assert.Equal(t, 1, len(caps.Network.Rules))
		assert.Equal(t, 1, len(caps.FS.Rules))

		mockParser.AssertExpectations(t)
	})

	t.Run("should fail if parser is missing", func(t *testing.T) {
		ext := extractor.NewManifestExtractor([]byte("dummy"))
		_, err := ext.Extract(nil)
		assert.Error(t, err)
	})

	t.Run("should fail if parser fails", func(t *testing.T) {
		mockParser := new(MockManifestParser)
		mockParser.On("Parse", mock.Anything).Return((*entities.Manifest)(nil), fmt.Errorf("parse error"))

		ext := extractor.NewManifestExtractor([]byte("dummy"), extractor.WithParser(mockParser))
		_, err := ext.Extract(nil)
		assert.Error(t, err)
	})

	t.Run("should render template if engine provided", func(t *testing.T) {
		mockParser := new(MockManifestParser)
		mockRenderer := new(mockRenderer)

		manifestBytes := []byte("{{ .val }}")
		renderedBytes := []byte("rendered")
		config := map[string]interface{}{"val": "rendered"}

		mockRenderer.On("Render", manifestBytes, config).Return(renderedBytes, nil)
		mockParser.On("Parse", renderedBytes).Return(&entities.Manifest{}, nil)

		ext := extractor.NewManifestExtractor(manifestBytes,
			extractor.WithParser(mockParser),
			extractor.WithTemplateEngine(mockRenderer),
		)

		_, err := ext.Extract(config)
		require.NoError(t, err)

		mockRenderer.AssertExpectations(t)
		mockParser.AssertExpectations(t)
	})

	t.Run("should fail if rendering fails", func(t *testing.T) {
		mockParser := new(MockManifestParser)
		mockRenderer := new(mockRenderer)

		mockRenderer.On("Render", mock.Anything, mock.Anything).Return(([]byte)(nil), fmt.Errorf("render error"))

		ext := extractor.NewManifestExtractor([]byte("dummy"),
			extractor.WithParser(mockParser),
			extractor.WithTemplateEngine(mockRenderer),
		)

		_, err := ext.Extract(nil)
		assert.Error(t, err)
	})

	t.Run("should return empty grant set if manifest has no capabilities", func(t *testing.T) {
		mockParser := new(MockManifestParser)
		mockParser.On("Parse", mock.Anything).Return(&entities.Manifest{
			Capabilities: entities.GrantSet{},
		}, nil)

		ext := extractor.NewManifestExtractor([]byte("dummy"), extractor.WithParser(mockParser))

		caps, err := ext.Extract(nil)
		require.NoError(t, err)
		assert.NotNil(t, caps)
		assert.True(t, caps.IsEmpty())
	})

	t.Run("should use renderer before parsing", func(t *testing.T) {
		expectedCaps := &entities.GrantSet{}

		mockRenderer := new(mockRenderer)
		mockRenderer.On("Render", mock.Anything, mock.Anything).Return([]byte("rendered output"), nil)

		mockParser := new(MockManifestParser)
		mockParser.On("Parse", []byte("rendered output")).Return(&entities.Manifest{Capabilities: entities.GrantSet{}}, nil)

		ext := extractor.NewManifestExtractor(
			[]byte("template"),
			extractor.WithParser(mockParser),
			extractor.WithTemplateEngine(mockRenderer),
		)

		caps, err := ext.Extract(map[string]interface{}{"foo": "bar"})
		require.NoError(t, err)
		assert.Equal(t, expectedCaps, caps)

		mockRenderer.AssertExpectations(t)
		mockParser.AssertExpectations(t)
	})
}
