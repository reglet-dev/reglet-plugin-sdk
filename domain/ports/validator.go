package ports

import "github.com/reglet-dev/reglet-sdk/domain/entities"

// CapabilityValidator validates capability configurations against schemas.
type CapabilityValidator interface {
	// Validate checks the manifest capabilities against registered schemas.
	Validate(manifest *entities.Manifest) (*entities.ValidationResult, error)
}
