package entities

import "github.com/reglet-dev/reglet-abi/hostfunc"

// NetworkCapability defines permitted network access.
type NetworkCapability = hostfunc.NetworkCapability

// NetworkRule defines a single network access rule.
type NetworkRule = hostfunc.NetworkRule

// FileSystemCapability defines permitted filesystem access.
type FileSystemCapability = hostfunc.FileSystemCapability

// FileSystemRule defines a single filesystem access rule.
type FileSystemRule = hostfunc.FileSystemRule

// EnvironmentCapability defines permitted environment variables.
type EnvironmentCapability = hostfunc.EnvironmentCapability

// ExecCapability defines permitted command execution.
type ExecCapability = hostfunc.ExecCapability

// KeyValueCapability defines permitted key-value store access.
type KeyValueCapability = hostfunc.KeyValueCapability

// KeyValueRule defines a single key-value access rule.
type KeyValueRule = hostfunc.KeyValueRule
