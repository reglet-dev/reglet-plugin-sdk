// Package entities provides core domain entities for the SDK.
// These are general-purpose types used across all SDK operations.
// Domain-specific types like Evidence belong in consuming applications (e.g., Reglet).
package entities

import (
	abi "github.com/reglet-dev/reglet-abi"
)

// ResultStatus represents the outcome status of an SDK operation.
type ResultStatus = abi.ResultStatus

const (
	ResultStatusSuccess = abi.ResultStatusSuccess
	ResultStatusFailure = abi.ResultStatusFailure
	ResultStatusError   = abi.ResultStatusError
)

// Result represents the general-purpose outcome of an SDK operation.
// This is the SDK's return type for check functions - consuming applications
// like Reglet map Result to their domain-specific types (e.g., Evidence).
type Result = abi.Result

// Forwarding constructors
var (
	ResultSuccess    = abi.ResultSuccess
	ResultFailure    = abi.ResultFailure
	ResultError      = abi.ResultError
	ResultSuccessPtr = abi.ResultSuccessPtr
	ResultFailurePtr = abi.ResultFailurePtr
	ResultErrorPtr   = abi.ResultErrorPtr
)
