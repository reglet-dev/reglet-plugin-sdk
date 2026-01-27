package config

import (
	"encoding/json"
	"fmt"
)

// Validate maps the config map to the target struct using JSON tags.
// Note: This implementation currently only performs type mapping via JSON.
// Field validation (required, etc.) is not yet enforced at this level
// unless a validator library is integrated.
func Validate(cfg map[string]any, target any) error {
	b, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := json.Unmarshal(b, target); err != nil {
		return fmt.Errorf("failed to unmarshal config to struct: %w", err)
	}
	return nil
}
