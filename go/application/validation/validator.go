// Package validation provides validation logic for plugin manifests and capabilities.
package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/reglet-dev/reglet-sdk/go/domain/ports"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

// CapabilityValidator implements validation using JSON schemas.
type CapabilityValidator struct {
	registry ports.CapabilityRegistry
	compiler *jsonschema.Compiler
}

// NewCapabilityValidator creates a new validator.
func NewCapabilityValidator(registry ports.CapabilityRegistry) ports.CapabilityValidator {
	return &CapabilityValidator{
		registry: registry,
		compiler: jsonschema.NewCompiler(),
	}
}

// Validate checks the manifest capabilities against registered schemas.
func (v *CapabilityValidator) Validate(manifest *entities.Manifest) (*entities.ValidationResult, error) {
	result := &entities.ValidationResult{Valid: true}

	if manifest.Capabilities == nil {
		return result, nil
	}

	// Helper to validate a specific capability
	validateCap := func(kind string, capData interface{}) {
		schemaStr, ok := v.registry.GetSchema(kind)
		if !ok {
			result.Valid = false
			result.Errors = append(result.Errors, entities.ValidationError{
				Field:   kind,
				Message: fmt.Sprintf("no schema registered for capability %s", kind),
			})
			return
		}

		if err := v.compiler.AddResource(kind, strings.NewReader(schemaStr)); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, entities.ValidationError{
				Field:   kind,
				Message: fmt.Sprintf("failed to add schema resource for %s: %v", kind, err),
			})
			return
		}

		sch, err := v.compiler.Compile(kind)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, entities.ValidationError{
				Field:   kind,
				Message: fmt.Sprintf("invalid schema for %s: %v", kind, err),
			})
			return
		}

		// Marshal capability to JSON (to interface{} for validation)
		b, _ := json.Marshal(capData)
		var obj interface{}
		if err := json.Unmarshal(b, &obj); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, entities.ValidationError{
				Field:   kind,
				Message: fmt.Sprintf("failed to prepare validation object: %v", err),
			})
			return
		}

		if err := sch.Validate(obj); err != nil {
			result.Valid = false
			var ve *jsonschema.ValidationError
			if errors.As(err, &ve) {
				result.Errors = append(result.Errors, entities.ValidationError{
					Field:   kind,
					Message: ve.Error(),
				})
			} else {
				result.Errors = append(result.Errors, entities.ValidationError{
					Field:   kind,
					Message: err.Error(),
				})
			}
		}
	}

	// Check for duplicates and validate each capability in the slice
	seen := make(map[string]bool)
	for _, cap := range manifest.Capabilities {
		// Validate capability structure (e.g., Category must not be empty)
		if cap.Category == "" {
			result.Valid = false
			result.Errors = append(result.Errors, entities.ValidationError{
				Field:   "capabilities",
				Message: "capability missing category",
			})
			continue
		}

		key := cap.String() // Assuming entities.Capability has a String() method for uniqueness
		if seen[key] {
			result.Valid = false
			result.Errors = append(result.Errors, entities.ValidationError{
				Field:   "capabilities",
				Message: fmt.Sprintf("duplicate capability: %s", key),
			})
			continue // Continue to check other capabilities
		}
		seen[key] = true

		// Validate against schema using the helper function
		validateCap(cap.Category, cap)
	}

	if len(result.Errors) > 0 {
		result.Valid = false
	}

	return result, nil
}
