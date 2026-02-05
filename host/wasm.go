package host

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/tetratelabs/wazero/api"
)

func (e *Executor) registerHostFunctions(ctx context.Context) error {
	builder := e.runtime.NewHostModuleBuilder("reglet_host")

	// 1. Register standard handlers from registry
	for _, name := range e.registry.Names() {
		localName := name
		builder.NewFunctionBuilder().
			WithFunc(func(ctx context.Context, m api.Module, packed uint64) uint64 {
				//nolint:gosec // WASM pointers are 32-bit
				ptr := uint32(packed >> 32)
				//nolint:gosec // WASM lengths are 32-bit
				length := uint32(packed)
				payload, ok := m.Memory().Read(ptr, length)
				if !ok {
					return 0
				}
				resp, _ := e.registry.Invoke(ctx, localName, payload)

				allocate := m.ExportedFunction("allocate")
				results, _ := allocate.Call(ctx, uint64(len(resp)))
				//nolint:gosec // WASM pointers are 32-bit
				respPtr := uint32(results[0])
				m.Memory().Write(respPtr, resp)
				return (uint64(respPtr) << 32) | uint64(len(resp))
			}).
			Export(name)
	}

	// 2. Register mandatory log_message function
	builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, packed uint64) {
			//nolint:gosec // WASM pointers are 32-bit
			ptr := uint32(packed >> 32)
			//nolint:gosec // WASM lengths are 32-bit
			length := uint32(packed)
			payload, ok := m.Memory().Read(ptr, length)
			if !ok {
				return
			}

			var logMsg struct {
				Level   string `json:"level"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(payload, &logMsg); err == nil {
				slog.Info("Plugin Log", "level", logMsg.Level, "msg", logMsg.Message)
			} else {
				slog.Info("Plugin Log (raw)", "payload", string(payload))
			}
		}).
		Export("log_message")

	_, err := builder.Instantiate(ctx)
	return err
}

func (p *PluginInstance) callRaw(ctx context.Context, name string, input []byte) (uint64, error) {
	f := p.module.ExportedFunction(name)
	if f == nil {
		return 0, fmt.Errorf("export %q not found", name)
	}

	var results []uint64
	var err error

	if len(input) == 0 {
		results, err = f.Call(ctx)
	} else {
		allocate := p.module.ExportedFunction("allocate")
		if allocate == nil {
			return 0, fmt.Errorf("guest does not export 'allocate'")
		}
		resAlloc, errAlloc := allocate.Call(ctx, uint64(len(input)))
		if errAlloc != nil {
			return 0, fmt.Errorf("failed to allocate in guest: %w", errAlloc)
		}
		if len(resAlloc) == 0 {
			return 0, fmt.Errorf("allocate returned no results")
		}
		//nolint:gosec // WASM pointers are 32-bit
		ptr := uint32(resAlloc[0])
		if !p.module.Memory().Write(ptr, input) {
			return 0, fmt.Errorf("failed to write input to guest memory")
		}
		results, err = f.Call(ctx, uint64(ptr), uint64(len(input)))
	}

	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	return results[0], nil
}

func (p *PluginInstance) unmarshalPacked(packed uint64, v any) error {
	//nolint:gosec // WASM pointers are 32-bit
	ptr := uint32(packed >> 32)
	//nolint:gosec // WASM lengths are 32-bit
	length := uint32(packed)

	if ptr == 0 || length == 0 {
		return fmt.Errorf("null response from plugin")
	}
	data, ok := p.module.Memory().Read(ptr, length)
	if !ok {
		return fmt.Errorf("failed to read response from memory")
	}
	return json.Unmarshal(data, v)
}
