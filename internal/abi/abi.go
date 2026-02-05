//go:build wasip1

// Package abi provides memory management for the WASM linear memory.
//
// This package handles the low-level allocation and deallocation of memory
// in the WASM linear memory space. It provides functions for safely transferring
// data between the Guest (plugin) and Host runtime by tracking allocations and
// preventing memory leaks through GC pinning.
//
// # Memory Model
//
// The WASM guest uses a linear memory model where all allocations exist in a
// contiguous address space. This package tracks all allocations to:
//   - Prevent Go GC from collecting memory still in use by the host
//   - Enforce memory limits to prevent unbounded growth
//   - Enable bulk deallocation during panic recovery
//
// # Usage
//
// Plugin authors typically interact with this package indirectly through the
// plugin.Register() lifecycle. Direct usage is for advanced scenarios:
//
//	// Send data to host
//	packed := abi.PtrFromBytes(myData)
//	// ... pass packed to host function ...
//	abi.DeallocatePacked(packed) // cleanup after host is done
//
//	// Receive data from host
//	data := abi.BytesFromPtr(packedFromHost)
package abi

import (
	"fmt"
	"sync"
	"unsafe"
)

// Memory limit constants.
const (
	// DefaultMaxTotalAllocations is the maximum total memory that can be allocated
	// by the SDK. This prevents unbounded memory growth in WASM linear memory.
	// Value: 100 MB
	DefaultMaxTotalAllocations = 100 * 1024 * 1024

	// PtrHighBits is the bit shift for storing pointer in the high 32 bits.
	PtrHighBits = 32
)

// ManagerOption configures the memory manager.
type ManagerOption func(*managerConfig)

// managerConfig holds configuration for the memory manager.
type managerConfig struct {
	maxTotalAllocations int
}

// defaultManagerConfig returns the default memory manager configuration.
func defaultManagerConfig() managerConfig {
	return managerConfig{
		maxTotalAllocations: DefaultMaxTotalAllocations,
	}
}

// WithMaxTotalAllocations sets the maximum total memory allocation limit.
// The limit must be positive; values <= 0 are ignored.
func WithMaxTotalAllocations(limit int) ManagerOption {
	return func(c *managerConfig) {
		if limit > 0 {
			c.maxTotalAllocations = limit
		}
	}
}

// memoryManager tracks all allocations made by the SDK in WASM linear memory.
// It keeps a reference to allocated slices to prevent the Go GC from collecting them,
// effectively "pinning" the memory until explicitly freed or during panic recovery.
type memoryManagerState struct {
	ptrs           map[uint32][]byte // ptr -> slice reference (16 bytes)
	totalAllocated int               // Total bytes currently allocated (8 bytes)
	config         managerConfig     // Configuration (8 bytes)
	sync.Mutex                       // Embedded (8 bytes) - placed last for alignment
}

// globalMemoryManager is the singleton instance.
var globalMemoryManager = &memoryManagerState{
	ptrs:   make(map[uint32][]byte),
	config: defaultManagerConfig(),
}

// Configure applies options to the global memory manager.
// This should be called early in initialization, before any allocations.
// Thread-safe but not recommended to call after allocations have started.
func Configure(opts ...ManagerOption) {
	globalMemoryManager.Lock()
	defer globalMemoryManager.Unlock()

	for _, opt := range opts {
		opt(&globalMemoryManager.config)
	}
}

// allocate reserves memory in the WASM linear memory and returns a pointer.
// The host can read from this pointer. It tracks the allocation to prevent GC.
//
// Returns 0 for zero-size allocations. Panics if allocation would exceed
// the configured memory limit (default: DefaultMaxTotalAllocations).
//
//go:wasmexport allocate
func allocate(size uint32) uint32 {
	if size == 0 {
		return 0
	}

	globalMemoryManager.Lock()
	defer globalMemoryManager.Unlock()

	maxAlloc := globalMemoryManager.config.maxTotalAllocations
	if globalMemoryManager.totalAllocated+int(size) > maxAlloc {
		panic(fmt.Sprintf(
			"abi: memory allocation limit exceeded (requested: %d bytes, current: %d bytes, limit: %d bytes)",
			size, globalMemoryManager.totalAllocated, maxAlloc,
		))
	}

	buf := make([]byte, size)
	//nolint:gosec // G103: Valid unsafe.Pointer for WASM linear memory address extraction
	ptr := uint32(uintptr(unsafe.Pointer(&buf[0])))

	// PIN THE MEMORY: Store the slice to prevent GC
	globalMemoryManager.ptrs[ptr] = buf
	globalMemoryManager.totalAllocated += int(size)

	return ptr
}

// deallocate frees memory by removing the reference from the memory manager,
// allowing the Go GC to collect it.
//
// The actual freed size is determined by the stored slice length, not the passed
// size parameter. This prevents counter corruption from mismatched size arguments.
// Untracked pointers are silently ignored (idempotent behavior).
//
//go:wasmexport deallocate
func deallocate(ptr uint32, size uint32) {
	globalMemoryManager.Lock()
	defer globalMemoryManager.Unlock()

	storedSlice, exists := globalMemoryManager.ptrs[ptr]
	if !exists {
		return // Ignore untracked pointers (idempotent)
	}

	actualSize := len(storedSlice)
	delete(globalMemoryManager.ptrs, ptr)
	globalMemoryManager.totalAllocated -= actualSize

	// Prevent negative totalAllocated due to double-free or other bugs
	if globalMemoryManager.totalAllocated < 0 {
		globalMemoryManager.totalAllocated = 0
	}
}

// FreeAllTracked frees all memory currently tracked by the SDK.
//
// This is typically called during panic recovery or module shutdown to prevent
// memory leaks. After this call, all previously allocated memory becomes eligible
// for garbage collection.
func FreeAllTracked() {
	globalMemoryManager.Lock()
	defer globalMemoryManager.Unlock()

	clear(globalMemoryManager.ptrs)
	globalMemoryManager.totalAllocated = 0
}

// Stats returns current memory allocation statistics.
// Useful for debugging and monitoring memory usage.
func Stats() (allocations int, totalBytes int) {
	globalMemoryManager.Lock()
	defer globalMemoryManager.Unlock()

	return len(globalMemoryManager.ptrs), globalMemoryManager.totalAllocated
}

// PtrFromBytes allocates WASM memory, copies the given data into it,
// and returns the packed pointer and length as a uint64.
//
// The allocated memory is tracked by the SDK for later deallocation.
// This is used when the Guest (plugin) sends data to the Host.
//
// Returns 0 for empty/nil input.
func PtrFromBytes(data []byte) uint64 {
	if len(data) == 0 {
		return 0
	}

	//nolint:gosec // G115: len(data) is bounded by slice capacity, safe for uint32
	size := uint32(len(data))
	ptr := allocate(size)
	copyToMemory(ptr, data)

	return PackPtrLen(ptr, size)
}

// BytesFromPtr unpacks a uint64 into a pointer and length, then reads
// the corresponding data from WASM linear memory.
//
// The memory must have been allocated by the Host for the Guest to read.
// This is used when the Guest receives data from the Host.
//
// Returns nil for zero pointer or zero length.
func BytesFromPtr(packed uint64) []byte {
	ptr, length := UnpackPtrLen(packed)
	if ptr == 0 || length == 0 {
		return nil
	}

	return readFromMemory(ptr, length)
}

// DeallocatePacked unpacks a uint64 pointer/length and deallocates the memory.
//
// This is used to free memory allocated by the Guest after passing it to the Host.
// The Guest should call this after the Host is done with the data.
//
// Safe to call with zero values (no-op).
func DeallocatePacked(packed uint64) {
	ptr, length := UnpackPtrLen(packed)
	if ptr != 0 && length > 0 {
		deallocate(ptr, length)
	}
}

// PackPtrLen packs a pointer and length into a single uint64.
//
// The pointer is stored in the high 32 bits, length in the low 32 bits.
// This encoding allows efficient transfer of pointer+length pairs across
// the WASM boundary using a single register.
//
// Panics if ptr is 0 and length > 0, indicating an invalid state.
func PackPtrLen(ptr, length uint32) uint64 {
	if ptr == 0 && length > 0 {
		panic(fmt.Sprintf("abi: invalid pack - null pointer (0x0) with non-zero length (%d)", length))
	}

	return (uint64(ptr) << PtrHighBits) | uint64(length)
}

// UnpackPtrLen unpacks a uint64 into its original pointer and length.
//
// Panics if the packed value contains ptr == 0 with length > 0,
// indicating a corrupted or invalid packed value.
func UnpackPtrLen(packed uint64) (ptr, length uint32) {
	//nolint:gosec // G115: Intentional truncation to extract high/low 32-bit halves
	ptr = uint32(packed >> PtrHighBits)
	//nolint:gosec // G115: Intentional truncation to extract low 32-bit half
	length = uint32(packed)

	if ptr == 0 && length > 0 {
		panic(fmt.Sprintf("abi: invalid unpack - null pointer (0x0) with non-zero length (%d)", length))
	}

	return ptr, length
}

// copyToMemory copies data to WASM linear memory at the given pointer.
func copyToMemory(ptr uint32, data []byte) {
	// WASM linear memory: uint32 offset -> pointer conversion is safe and necessary
	//nolint:gosec,govet // G103, unsafeptr: Valid unsafe.Pointer use for WASM linear memory access
	dest := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), len(data))
	copy(dest, data)
}

// readFromMemory reads data from WASM linear memory.
// Returns a copy of the data, not a reference to the original memory.
func readFromMemory(ptr uint32, length uint32) []byte {
	// WASM linear memory: uint32 offset -> pointer conversion is safe and necessary
	//nolint:gosec,govet // G103, unsafeptr: Valid unsafe.Pointer use for WASM linear memory access
	src := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
	data := make([]byte, length)
	copy(data, src)

	return data
}
