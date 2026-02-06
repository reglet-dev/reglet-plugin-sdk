package entities

import abi "github.com/reglet-dev/reglet-abi"

// Manifest contains complete plugin metadata for introspection.
type Manifest = abi.Manifest

// ServiceManifest describes a service and its operations.
type ServiceManifest = abi.ServiceManifest

// OperationManifest describes a single operation.
type OperationManifest = abi.OperationManifest

// OperationExample provides a sample input/output pair.
type OperationExample = abi.OperationExample
