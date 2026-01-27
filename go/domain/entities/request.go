package entities

// NetworkRequest represents a runtime request to access the network.
type NetworkRequest struct {
	Host string
	Port int
}

// FileSystemRequest represents a runtime request to access the filesystem.
type FileSystemRequest struct {
	Path      string
	Operation string // "read", "write"
}

// EnvironmentRequest represents a runtime request to access environment variables.
type EnvironmentRequest struct {
	Variable string
}

// ExecRequest represents a runtime request to execute a command.
type ExecRequest struct {
	Command string
}

// KeyValueRequest represents a runtime request to access the key-value store.
type KeyValueRequest struct {
	Key       string
	Operation string // "read", "write"
}

// CapabilityRequest represents a request for a capability to be granted (e.g. via prompt).
type CapabilityRequest struct {
	Rule        interface{}
	Kind        string
	Description string
	RiskLevel   RiskLevel
	IsBroad     bool
}
