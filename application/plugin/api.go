package plugin

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/reglet-dev/reglet-plugin-sdk/domain/entities"
)

// Plugin is the interface every Reglet plugin must implement.
type Plugin interface {
	// Manifest returns complete metadata about the plugin.
	Manifest(ctx context.Context) (*entities.Manifest, error)
	// Check executes the plugin's main logic with the given configuration.
	Check(ctx context.Context, config []byte) (*entities.Result, error)
}

// Internal variable to hold the user's plugin implementation.
var userPlugin Plugin

// Register initializes the WASM exports and handles the plugin lifecycle.
// Plugin authors call this in their `main()` function.
//
// Version Checking:
// The SDK automatically reports its version (Version) in the Manifest metadata.
// The host is responsible for validating compatibility before loading the plugin.
func Register(p Plugin) {
	if userPlugin != nil {
		slog.Warn("sdk: plugin already registered, ignoring second call", "userPlugin_addr", fmt.Sprintf("%p", &userPlugin))
		return
	}
	userPlugin = p
	slog.Info("sdk: plugin registered successfully", "userPlugin_addr", fmt.Sprintf("%p", &userPlugin))
}
