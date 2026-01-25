package host

import (
	"fmt"

	apptemplate "github.com/reglet-dev/reglet-sdk/go/application/template"
	"github.com/reglet-dev/reglet-sdk/go/application/validation"
	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/reglet-dev/reglet-sdk/go/domain/ports"
	"github.com/reglet-dev/reglet-sdk/go/infrastructure/parser"
)

// loaderConfig holds configuration for the Loader.
type loaderConfig struct {
	registry        ports.CapabilityRegistry
	templateEngine  ports.TemplateEngine
	parser          ports.ManifestParser
	strictTemplates bool // Fail on missing template keys
}

func defaultLoaderConfig() loaderConfig {
	return loaderConfig{
		parser:          parser.NewYamlManifestParser(),
		strictTemplates: true, // Secure default: fail on missing keys
	}
}

// Loader orchestrates the manifest loading pipeline.
type Loader struct {
	validator ports.CapabilityValidator
	config    loaderConfig
}

// LoaderOption configures the Loader.
type LoaderOption func(*loaderConfig)

// WithRegistry configures the loader with a capability registry for validation.
func WithRegistry(r ports.CapabilityRegistry) LoaderOption {
	return func(c *loaderConfig) {
		c.registry = r
	}
}

// WithParser sets a custom manifest parser.
func WithParser(p ports.ManifestParser) LoaderOption {
	return func(c *loaderConfig) {
		c.parser = p
	}
}

// WithTemplateEngine sets a template engine.
func WithTemplateEngine(t ports.TemplateEngine) LoaderOption {
	return func(c *loaderConfig) {
		c.templateEngine = t
	}
}

// WithStrictTemplates enables/disables strict template mode.
// When enabled (default), template rendering fails if a referenced key is missing.
// Disable only for development or when missing keys should become empty strings.
func WithStrictTemplates(enabled bool) LoaderOption {
	return func(c *loaderConfig) {
		c.strictTemplates = enabled
	}
}

// NewLoader creates a new Loader with defaults.
func NewLoader(opts ...LoaderOption) *Loader {
	cfg := defaultLoaderConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Create default template engine if not provided
	if cfg.templateEngine == nil {
		cfg.templateEngine = apptemplate.NewGoTemplateEngine(
			apptemplate.WithStrict(cfg.strictTemplates),
		)
	}

	l := &Loader{config: cfg}
	if cfg.registry != nil {
		l.validator = validation.NewCapabilityValidator(cfg.registry)
	}
	return l
}

// LoadManifest loads, parses, and validates a plugin manifest.
func (l *Loader) LoadManifest(raw []byte, config map[string]interface{}) (*entities.PluginManifest, error) {
	var data = raw

	if l.config.templateEngine != nil {
		var err error
		data, err = l.config.templateEngine.Render(raw, config)
		if err != nil {
			return nil, fmt.Errorf("failed to render manifest: %w", err)
		}
	}

	manifest, err := l.config.parser.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if l.validator != nil {
		res, err := l.validator.Validate(manifest)
		if err != nil {
			return nil, fmt.Errorf("validation error: %w", err)
		}
		if !res.Valid {
			msg := "manifest validation failed:"
			for _, e := range res.Errors {
				msg += fmt.Sprintf("\n- %s: %s", e.Field, e.Message)
			}
			return nil, fmt.Errorf("%s", msg)
		}
	}

	return manifest, nil
}
