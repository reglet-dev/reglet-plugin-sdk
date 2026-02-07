//go:build !wasip1

package plugin

import (
	"context"

	"github.com/reglet-dev/reglet-plugin-sdk/domain/entities"
)

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
