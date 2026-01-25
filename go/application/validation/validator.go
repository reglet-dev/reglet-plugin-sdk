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
func (v *CapabilityValidator) Validate(manifest *entities.PluginManifest) (*entities.ValidationResult, error) {
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
				// We can check BasicOutput for simple list of errors
				// But we assume the library provides a structured error
				// For now, simple error string
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

	if manifest.Capabilities.Network != nil {
		validateCap("network", manifest.Capabilities.Network)
	}
	if manifest.Capabilities.FS != nil {
		validateCap("fs", manifest.Capabilities.FS)
	}
	if manifest.Capabilities.Env != nil {
		validateCap("env", manifest.Capabilities.Env)
	}
	if manifest.Capabilities.Exec != nil {
		validateCap("exec", manifest.Capabilities.Exec)
	}
	if manifest.Capabilities.KV != nil {
		validateCap("kv", manifest.Capabilities.KV)
	}

	if len(result.Errors) > 0 {
		result.Valid = false
	}

	return result, nil
}
