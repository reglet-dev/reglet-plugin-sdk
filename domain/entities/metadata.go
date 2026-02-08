package entities

import (
	abi "github.com/reglet-dev/reglet-abi"
)

// RunMetadata contains execution metadata for SDK operations.
type RunMetadata = abi.RunMetadata

// NewRunMetadata creates a new RunMetadata with the given start and end times.
var NewRunMetadata = abi.NewRunMetadata
