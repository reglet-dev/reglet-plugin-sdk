# ABI Package

The `abi` package implements the Application Binary Interface (ABI) for WASM plugins. It handles memory management, pointer packing/unpacking, and data transfer across the WASM boundary between the plugin (guest) and the host.

## Overview

This package is **internal** and should not be used directly by plugin authors. It provides low-level primitives used by other SDK packages (`net`, `exec`, `log`) to communicate with the host.

## Key Concepts

### WASM Linear Memory

WASM modules have a single contiguous block of memory called **linear memory**. All data exchange between the plugin and host happens through this memory:

- **Guest (Plugin)**: Allocates memory and writes data
- **Host**: Reads from guest-allocated memory or writes to guest-allocated memory

### Pointer Packing

Since WASM functions can only pass/return integers, we pack pointers and lengths into a single `uint64`:

```
High 32 bits: Memory pointer (uint32)
Low 32 bits:  Data length (uint32)
```

This allows a single return value to convey both the location and size of data.

## Memory Management

### Configuration

The memory manager can be configured using functional options:

```go
import "github.com/reglet-dev/reglet-sdk/go/internal/abi"

// Configure with custom memory limit (default: 100 MB)
abi.Configure(abi.WithMaxTotalAllocations(50 * 1024 * 1024)) // 50 MB
```

### Memory Limit

To prevent unbounded memory growth, the ABI enforces a configurable limit:

```go
const DefaultMaxTotalAllocations = 100 * 1024 * 1024 // 100 MB
```

If allocations exceed this limit, the plugin panics with a clear error message.

### Observability

Check current memory usage with the `Stats()` function:

```go
allocations, totalBytes := abi.Stats()
slog.Info("Memory stats", "allocations", allocations, "bytes", totalBytes)
```

## Exported WASM Functions

These functions are exported to the host and implement the WASM memory protocol:

### allocate

```go
//go:wasmexport allocate
func allocate(size uint32) uint32
```

Allocates memory in WASM linear memory and returns a pointer:

- **Called by**: Host (when passing data to plugin)
- **Returns**: Pointer to allocated memory, or 0 for zero-size
- **Tracking**: Allocation is tracked to prevent GC

### deallocate

```go
//go:wasmexport deallocate
func deallocate(ptr uint32, size uint32)
```

Frees previously allocated memory:

- **Called by**: Host (after reading data from plugin)
- **Effect**: Removes tracking, allows GC to collect
- **Safety**: Idempotent; untracked pointers are silently ignored

## SDK Functions

### PtrFromBytes

```go
func PtrFromBytes(data []byte) uint64
```

Allocates WASM memory, copies data into it, and returns packed pointer/length:

```go
jsonData := []byte(`{"key": "value"}`)
packed := abi.PtrFromBytes(jsonData)
// Returns: packed uint64 containing pointer and length
```

### BytesFromPtr

```go
func BytesFromPtr(packed uint64) []byte
```

Unpacks a uint64 and reads the corresponding data from WASM memory:

```go
packed := hostFunction()
data := abi.BytesFromPtr(packed)
// Returns: []byte copy of the data
```

### DeallocatePacked

```go
func DeallocatePacked(packed uint64)
```

Unpacks a uint64 and deallocates the memory:

```go
packed := pluginFunction()
data := abi.BytesFromPtr(packed)
abi.DeallocatePacked(packed) // Free after use
```

### PackPtrLen / UnpackPtrLen

```go
func PackPtrLen(ptr, length uint32) uint64
func UnpackPtrLen(packed uint64) (ptr, length uint32)
```

Pack and unpack pointer/length pairs. Uses the `PtrHighBits` constant (32) for bit shifting.

**Panics**: If `ptr == 0` and `length > 0` (invalid state)

### FreeAllTracked

```go
func FreeAllTracked()
```

Frees all currently tracked allocations. Called during panic recovery:

```go
defer func() {
    if r := recover(); r != nil {
        abi.FreeAllTracked()
        panic(r)
    }
}()
```

### Stats

```go
func Stats() (allocations int, totalBytes int)
```

Returns current memory allocation statistics for monitoring and debugging.

## Memory Protocol

### Guest → Host

```go
// 1. Plugin prepares data
data := []byte(`{"result": "success"}`)

// 2. Allocate and copy to WASM memory
packed := abi.PtrFromBytes(data)

// 3. Return packed pointer to host
return packed

// 4. Host reads data, then calls deallocate(ptr, len)
```

### Host → Guest

```go
// Host calls guest's allocate(size), writes data, passes packed ptr/len
func guestFunction(packed uint64) {
    data := abi.BytesFromPtr(packed)
    defer abi.DeallocatePacked(packed)
    // Process data...
}
```

## Memory Safety

### Pointer Validation

```go
if ptr == 0 && length > 0 {
    panic("abi: invalid pack - null pointer with non-zero length")
}
```

### Double-Free Protection

Deallocation uses actual stored slice length, not the caller's size argument:

```go
actualSize := len(storedSlice)
globalMemoryManager.totalAllocated -= actualSize
if globalMemoryManager.totalAllocated < 0 {
    globalMemoryManager.totalAllocated = 0
}
```

## Unsafe Operations

The ABI uses `unsafe.Pointer` for direct memory access:

```go
//nolint:gosec // G103: Valid unsafe.Pointer use for WASM linear memory access
dest := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), len(data))
```

**Why Safe**: WASM linear memory is a contiguous array where uint32 offsets map directly to valid memory addresses.

## Limitations

1. **100 MB Default Limit**: Configurable via `WithMaxTotalAllocations`
2. **Single-Threaded**: WASM is single-threaded; mutex is defensive
3. **Copy Overhead**: `BytesFromPtr` always copies data for safety

## See Also

- [Main SDK Documentation](../../README.md)
