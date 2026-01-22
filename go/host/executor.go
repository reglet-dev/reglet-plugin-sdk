package host

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/reglet-dev/reglet-sdk/go/hostfuncs"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Executor manages the lifecycle of a WASM plugin.
type Executor struct {
	runtime  wazero.Runtime
	registry *hostfuncs.HandlerRegistry
}

// NewExecutor creates a new executor with the given options.
func NewExecutor(ctx context.Context, opts ...Option) (*Executor, error) {
	e := &Executor{}
	for _, opt := range opts {
		opt(e)
	}

	// Default registry if not provided
	if e.registry == nil {
		reg, err := hostfuncs.NewRegistry()
		if err != nil {
			return nil, fmt.Errorf("failed to create default registry: %w", err)
		}
		e.registry = reg
	}

	rt := wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)
	e.runtime = rt

	if err := e.registerHostFunctions(ctx); err != nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("failed to register host functions: %w", err)
	}

	// ... (skipping some methods) ...
	return e, nil
}

// Close releases resources held by the executor.
func (e *Executor) Close(ctx context.Context) error {
	return e.runtime.Close(ctx)
}

// PluginInstance represents an instantiated WASM plugin.
type PluginInstance struct {
	module api.Module
	// Ensure we have access to helper methods like callRaw and unmarshalPacked
	// which are methods on PluginInstance
}

// LoadPlugin instantiates a WASM module.
func (e *Executor) LoadPlugin(ctx context.Context, wasmBytes []byte) (*PluginInstance, error) {
	mod, err := e.runtime.Instantiate(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate module: %w", err)
	}

	// Initialize if needed (though Instantiate usually handles start)
	if init := mod.ExportedFunction("_initialize"); init != nil {
		if _, err := init.Call(ctx); err != nil {
			return nil, fmt.Errorf("failed to call _initialize: %w", err)
		}
	}

	return &PluginInstance{module: mod}, nil
}

// Describe calls the "describe" export of the plugin.
func (p *PluginInstance) Describe(ctx context.Context) (entities.Metadata, error) {
	var metadata entities.Metadata
	packed, err := p.callRaw(ctx, "describe", nil)
	if err != nil {
		return metadata, err
	}
	err = p.unmarshalPacked(packed, &metadata)
	return metadata, err
}

// Schema calls the "schema" export of the plugin.
func (p *PluginInstance) Schema(ctx context.Context) ([]byte, error) {
	packed, err := p.callRaw(ctx, "schema", nil)
	if err != nil {
		return nil, err
	}
	//nolint:gosec // WASM pointers are always 32-bit
	ptr := uint32(packed >> 32)
	//nolint:gosec // WASM lengths are always 32-bit
	length := uint32(packed)
	data, ok := p.module.Memory().Read(ptr, length)
	if !ok {
		return nil, fmt.Errorf("failed to read schema from memory")
	}
	// Copy data to avoid memory issues if wasm memory changes, though here it's read immediately
	schemaCopy := make([]byte, length)
	copy(schemaCopy, data)
	return schemaCopy, nil
}

// Check calls the "observe" export of the plugin.
func (p *PluginInstance) Check(ctx context.Context, config map[string]any) (entities.Result, error) {
	configBytes, err := json.Marshal(config)
	if err != nil {
		return entities.Result{}, err
	}

	packed, err := p.callRaw(ctx, "observe", configBytes)
	if err != nil {
		return entities.Result{}, err
	}

	var result entities.Result
	err = p.unmarshalPacked(packed, &result)
	return result, err
}
