// Package parser provides functionality for parsing plugin manifests.
package parser

import (
	"github.com/reglet-dev/reglet-sdk/domain/entities"
	"github.com/reglet-dev/reglet-sdk/domain/ports"
	"gopkg.in/yaml.v3"
)

// YamlManifestParser implements ManifestParser for YAML.
type YamlManifestParser struct{}

// NewYamlManifestParser creates a new YamlManifestParser.
func NewYamlManifestParser() ports.ManifestParser {
	return &YamlManifestParser{}
}

// Parse unmarshals YAML bytes into a Manifest struct.
func (p *YamlManifestParser) Parse(data []byte) (*entities.Manifest, error) {
	var manifest entities.Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}
