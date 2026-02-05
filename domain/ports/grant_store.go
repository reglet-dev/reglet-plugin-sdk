package ports

import "github.com/reglet-dev/reglet-sdk/domain/entities"

// GrantStore provides persistence for capability grants.
type GrantStore interface {
	// Load retrieves all granted capabilities.
	// Returns empty GrantSet (not error) if no grants exist.
	Load() (*entities.GrantSet, error)

	// Save persists the granted capabilities.
	Save(grants *entities.GrantSet) error

	// ConfigPath returns the path to the backing store (for user messaging).
	ConfigPath() string
}
