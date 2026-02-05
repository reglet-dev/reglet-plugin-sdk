package entities

import "encoding/json"

// Manifest contains complete plugin metadata for introspection.
// Used by both reglet and CLI tools to understand plugin capabilities.
type Manifest struct {
	// Registered services and operations
	Services map[string]ServiceManifest `json:"services" yaml:"services"`

	// Identity
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version" yaml:"version"`
	Description string `json:"description" yaml:"description"`

	// Compatibility
	SDKVersion     string `json:"sdk_version" yaml:"sdk_version"`
	MinHostVersion string `json:"min_host_version,omitempty" yaml:"min_host_version,omitempty"`

	// Config schema (JSON Schema)
	ConfigSchema json.RawMessage `json:"config_schema" yaml:"config_schema"`

	// Capabilities (http, dns, file, exec, etc.)
	Capabilities GrantSet `json:"capabilities" yaml:"capabilities"`
}

// ServiceManifest describes a service and its operations.
type ServiceManifest struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Operations  []OperationManifest `json:"operations"`
}

// OperationManifest describes a single operation.
type OperationManifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	// Input fields this operation requires (subset of plugin config)
	InputFields []string `json:"input_fields,omitempty"`

	// JSON Schema for Result.Data structure
	OutputSchema json.RawMessage `json:"output_schema,omitempty"`

	// Examples for documentation and testing
	Examples []OperationExample `json:"examples,omitempty"`
}

// OperationExample provides a sample input/output pair.
type OperationExample struct {
	// Name is a short identifier (e.g., "basic", "mx_records")
	Name string `json:"name"`

	// Description explains what this example demonstrates
	Description string `json:"description,omitempty"`

	// Input is the example input as JSON
	Input json.RawMessage `json:"input"`

	// ExpectedOutput is the expected Result.Data as JSON (optional)
	ExpectedOutput json.RawMessage `json:"expected_output,omitempty"`

	// ExpectedError indicates this is a negative test case
	ExpectedError string `json:"expected_error,omitempty"`
}
