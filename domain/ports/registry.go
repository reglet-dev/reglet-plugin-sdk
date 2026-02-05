package ports

// CapabilityRegistry manages JSON schemas for capability types.
type CapabilityRegistry interface {
	// Register adds a schema generated from a Go struct.
	Register(kind string, model interface{}) error

	// GetSchema retrieves the JSON Schema for a capability type.
	GetSchema(kind string) (string, bool)

	// List returns all registered capability type names.
	List() []string
}
