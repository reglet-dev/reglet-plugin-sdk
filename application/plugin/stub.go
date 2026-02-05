//go:build !wasip1

package plugin

import (
	"context"

	"github.com/reglet-dev/reglet-sdk/domain/entities"
)

// Plugin is the interface every Reglet plugin must implement.
type Plugin interface {
	Manifest(ctx context.Context) (*entities.Manifest, error)
	Check(ctx context.Context, config []byte) (*entities.Result, error)
}

// StubPlugin is a no-op implementation of the Plugin interface for testing or non-WASM environments.
type StubPlugin struct{}

// Manifest returns a default manifest for the StubPlugin.
func (s *StubPlugin) Manifest(ctx context.Context) (*entities.Manifest, error) {
	return &entities.Manifest{
		Name:    "stub",
		Version: "0.0.1",
	}, nil
}

// Check performs a no-op check for the StubPlugin.
func (s *StubPlugin) Check(ctx context.Context, config []byte) (*entities.Result, error) {
	return &entities.Result{
		Status:  entities.ResultStatusSuccess,
		Message: "Stub check successful",
	}, nil
}

// Register is a stub for non-WASM platforms.
func Register(p Plugin) {
	// No-op on non-WASM platforms
}
