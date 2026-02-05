package plugin

import (
	"reflect"
	"sync"
)

// operation is a marker interface for identifying Op fields via reflection.
type operation interface {
	isOp()
}

// Op defines a typed operation with explicit input and output types.
// I is the input type (parsed from config JSON)
// O is the output type (serialized to Result.Data)
type Op[I, O any] struct{}

func (Op[I, O]) isOp() {}

// Example defines a sample input/output pair for documentation and testing.
type Example[I, O any] struct {
	// Name is a short identifier (e.g., "basic", "with_tls", "error_case")
	Name string

	// Description explains what this example demonstrates
	Description string

	// Input is the example input data
	Input I

	// ExpectedOutput is the expected output (nil if not verifying output)
	ExpectedOutput *O

	// ExpectedError is set if this example should produce an error
	// Used for negative test cases
	ExpectedError string
}

// opTypeInfo stores type information captured during registration.
// Since Go's reflect cannot extract type parameters from instantiated generics,
// we capture them at registration time using the generic RegisterOp function.
type opTypeInfo struct {
	inputType  reflect.Type
	outputType reflect.Type
	examples   []any // []Example[I,O] stored as any for type erasure
}

var (
	opRegistry   = make(map[string]opTypeInfo)
	opRegistryMu sync.RWMutex
)

// RegisterOp captures type parameters and examples for an operation.
// Must be called in init() BEFORE MustRegisterService for each Op field.
//
// Example:
//
//	func init() {
//	    plugin.RegisterOp[ResolveInput, ResolveOutput]("Resolve",
//	        plugin.Example[ResolveInput, ResolveOutput]{
//	            Name:  "basic",
//	            Input: ResolveInput{Hostname: "example.com"},
//	        },
//	    )
//	    plugin.MustRegisterService(core.Plugin, &DNSService{})
//	}
func RegisterOp[I, O any](fieldName string, examples ...Example[I, O]) {
	opRegistryMu.Lock()
	defer opRegistryMu.Unlock()

	var zeroIn I
	var zeroOut O

	// Convert typed examples to []any for storage
	examplesAny := make([]any, len(examples))
	for i, ex := range examples {
		examplesAny[i] = ex
	}

	opRegistry[fieldName] = opTypeInfo{
		inputType:  reflect.TypeOf(zeroIn),
		outputType: reflect.TypeOf(zeroOut),
		examples:   examplesAny,
	}
}

// getOpTypeInfo retrieves registered type info for an operation field.
// Returns false if the field was not registered.
func getOpTypeInfo(fieldName string) (opTypeInfo, bool) {
	opRegistryMu.RLock()
	defer opRegistryMu.RUnlock()

	info, ok := opRegistry[fieldName]
	return info, ok
}

// clearOpRegistry clears the registry (for testing only).
func clearOpRegistry() {
	opRegistryMu.Lock()
	defer opRegistryMu.Unlock()
	opRegistry = make(map[string]opTypeInfo)
}
