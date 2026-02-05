package host

import (
	"github.com/reglet-dev/reglet-sdk/hostfuncs"
)

// Option defines a functional option for configuring the Executor.
type Option func(*Executor)

// WithHostFunctions configures the executor with a host function registry.
func WithHostFunctions(registry *hostfuncs.HandlerRegistry) Option {
	return func(e *Executor) {
		e.registry = registry
	}
}
