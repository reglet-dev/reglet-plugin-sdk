package entities

import "github.com/reglet-dev/reglet-abi/hostfunc"

// NetworkRequest represents a runtime request to access the network.
type NetworkRequest = hostfunc.NetworkRequest

// FileSystemRequest represents a runtime request to access the filesystem.
type FileSystemRequest = hostfunc.FileSystemRequest

// EnvironmentRequest represents a runtime request to access environment variables.
type EnvironmentRequest = hostfunc.EnvironmentRequest

// ExecCapabilityRequest represents a runtime request to execute a command.
type ExecCapabilityRequest = hostfunc.ExecCapabilityRequest

// KeyValueRequest represents a runtime request to access the key-value store.
type KeyValueRequest = hostfunc.KeyValueRequest

// CapabilityRequest represents a request for a capability to be granted (e.g. via prompt).
type CapabilityRequest struct {
	Rule        interface{}
	Kind        string
	Description string
	RiskLevel   RiskLevel
	IsBroad     bool
}
