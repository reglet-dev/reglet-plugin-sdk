package ports

import "github.com/reglet-dev/reglet-sdk/domain/entities"

// Prompter handles interactive capability authorization.
type Prompter interface {
	// IsInteractive returns true if running in an interactive terminal.
	IsInteractive() bool

	// PromptForCapability asks the user to grant a capability.
	// Returns: granted (allow this time), always (persist to store), error.
	PromptForCapability(req entities.CapabilityRequest) (granted bool, always bool, err error)

	// PromptForCapabilities prompts for multiple capabilities at once.
	// Returns the GrantSet of approved capabilities.
	PromptForCapabilities(reqs []entities.CapabilityRequest) (*entities.GrantSet, error)

	// FormatNonInteractiveError creates a helpful error for non-interactive mode.
	FormatNonInteractiveError(missing *entities.GrantSet) error
}
