package entities

// NetworkCapability defines permitted network access.
type NetworkCapability struct {
	Rules []NetworkRule `json:"rules" yaml:"rules" jsonschema:"required"`
}

// NetworkRule defines a single network access rule.
type NetworkRule struct {
	Hosts []string `json:"hosts" yaml:"hosts" jsonschema:"required"`
	Ports []string `json:"ports" yaml:"ports" jsonschema:"required"` // "80", "8000-9000", "*"
}

// FileSystemCapability defines permitted filesystem access.
type FileSystemCapability struct {
	Rules []FileSystemRule `json:"rules" yaml:"rules" jsonschema:"required"`
}

// FileSystemRule defines a single filesystem access rule.
type FileSystemRule struct {
	Read  []string `json:"read,omitempty" yaml:"read,omitempty"`
	Write []string `json:"write,omitempty" yaml:"write,omitempty"`
}

// EnvironmentCapability defines permitted environment variables.
type EnvironmentCapability struct {
	Variables []string `json:"vars" yaml:"vars" jsonschema:"required"`
}

// ExecCapability defines permitted command execution.
type ExecCapability struct {
	Commands []string `json:"commands" yaml:"commands" jsonschema:"required"`
}

// KeyValueCapability defines permitted key-value store access.
type KeyValueCapability struct {
	Rules []KeyValueRule `json:"rules" yaml:"rules" jsonschema:"required"`
}

// KeyValueRule defines a single key-value access rule.
type KeyValueRule struct {
	Operation string   `json:"op" yaml:"op" jsonschema:"required,enum=read,enum=write,enum=read-write"`
	Keys      []string `json:"keys" yaml:"keys" jsonschema:"required"`
}
