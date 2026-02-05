package ports

import "github.com/reglet-dev/reglet-sdk/domain/entities"

// Extractor analyzes plugin configuration and returns required capabilities.
type Extractor interface {
	// Extract returns capabilities needed based on the plugin's config.
	// Returns a GrantSet representing what the plugin needs.
	Extract(config map[string]interface{}) (*entities.GrantSet, error)
}

// ExtractorRegistry manages extractors by plugin name.
type ExtractorRegistry interface {
	Register(pluginName string, extractor Extractor)
	Get(pluginName string) (Extractor, bool)
}
