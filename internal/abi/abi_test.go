//go:build wasip1

package abi

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetMemoryManager clears all tracked allocations for test isolation.
func resetMemoryManager() {
	FreeAllTracked()
}

func TestPackPtrLen(t *testing.T) {
	tests := []struct {
		name   string
		ptr    uint32
		length uint32
		want   uint64
	}{
		{
			name:   "typical values",
			ptr:    0x12345678,
			length: 0xABCDEF00,
			want:   (uint64(0x12345678) << PtrHighBits) | uint64(0xABCDEF00),
		},
		{
			name:   "zero pointer zero length",
			ptr:    0,
			length: 0,
			want:   0,
		},
		{
			name:   "max pointer",
			ptr:    0xFFFFFFFF,
			length: 1,
			want:   (uint64(0xFFFFFFFF) << PtrHighBits) | 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packed := PackPtrLen(tt.ptr, tt.length)
			assert.Equal(t, tt.want, packed, "packed value mismatch")

			gotPtr, gotLen := UnpackPtrLen(packed)
			assert.Equal(t, tt.ptr, gotPtr, "unpacked pointer mismatch")
			assert.Equal(t, tt.length, gotLen, "unpacked length mismatch")
		})
	}
}

func TestPackPtrLen_PanicsOnNullPointerWithLength(t *testing.T) {
	assert.Panics(t, func() {
		PackPtrLen(0, 100)
	}, "expected panic for null pointer with non-zero length")
}

func TestUnpackPtrLen_PanicsOnInvalidPacked(t *testing.T) {
	assert.Panics(t, func() {
		// Invalid packed: ptr=0, len=1
		UnpackPtrLen(uint64(1))
	}, "expected panic for invalid packed value")
}

func TestAllocateDeallocate(t *testing.T) {
	resetMemoryManager()

	size := uint32(1024)
	ptr := allocate(size)
	require.NotZero(t, ptr, "allocate returned 0")

	// Verify allocation is tracked
	allocCount, totalBytes := Stats()
	assert.Equal(t, 1, allocCount, "expected 1 tracked allocation")
	assert.Equal(t, int(size), totalBytes, "total bytes mismatch")

	// Write and read data
	data := []byte("hello world")
	copyToMemory(ptr, data)
	readData := readFromMemory(ptr, uint32(len(data)))
	assert.Equal(t, data, readData, "memory read mismatch")

	// Deallocate
	deallocate(ptr, size)

	allocCount, totalBytes = Stats()
	assert.Equal(t, 0, allocCount, "expected 0 tracked allocations after deallocate")
	assert.Equal(t, 0, totalBytes, "expected 0 total bytes after deallocate")
}

func TestAllocate_ZeroSize(t *testing.T) {
	ptr := allocate(0)
	assert.Zero(t, ptr, "allocate(0) should return 0")
}

func TestDeallocate_Idempotent(t *testing.T) {
	resetMemoryManager()

	ptr := allocate(100)
	deallocate(ptr, 100)
	// Second deallocate should not panic or corrupt state
	deallocate(ptr, 100)

	_, totalBytes := Stats()
	assert.GreaterOrEqual(t, totalBytes, 0, "total bytes should not be negative")
}

func TestFreeAllTracked(t *testing.T) {
	resetMemoryManager()

	// Allocate multiple buffers
	allocate(100)
	allocate(200)

	allocCount, _ := Stats()
	require.Equal(t, 2, allocCount, "expected 2 tracked allocations")

	FreeAllTracked()

	allocCount, totalBytes := Stats()
	assert.Equal(t, 0, allocCount, "expected 0 allocations after FreeAllTracked")
	assert.Equal(t, 0, totalBytes, "expected 0 bytes after FreeAllTracked")
}

func TestStats(t *testing.T) {
	resetMemoryManager()

	allocCount, totalBytes := Stats()
	assert.Equal(t, 0, allocCount)
	assert.Equal(t, 0, totalBytes)

	allocate(256)
	allocate(512)

	allocCount, totalBytes = Stats()
	assert.Equal(t, 2, allocCount)
	assert.Equal(t, 768, totalBytes)

	FreeAllTracked()
}

func TestPtrFromBytes(t *testing.T) {
	resetMemoryManager()

	data := []byte("test data")
	packed := PtrFromBytes(data)

	ptr, length := UnpackPtrLen(packed)
	assert.NotZero(t, ptr, "expected non-zero pointer")
	assert.Equal(t, uint32(len(data)), length, "length mismatch")

	// Verify content
	readData := BytesFromPtr(packed)
	assert.Equal(t, data, readData, "data mismatch")

	DeallocatePacked(packed)

	allocCount, _ := Stats()
	assert.Equal(t, 0, allocCount, "expected 0 allocations after cleanup")
}

func TestPtrFromBytes_Empty(t *testing.T) {
	packed := PtrFromBytes(nil)
	assert.Zero(t, packed, "PtrFromBytes(nil) should return 0")

	packed = PtrFromBytes([]byte{})
	assert.Zero(t, packed, "PtrFromBytes([]) should return 0")
}

func TestBytesFromPtr_ZeroValues(t *testing.T) {
	data := BytesFromPtr(0)
	assert.Nil(t, data, "BytesFromPtr(0) should return nil")
}

func TestDeallocatePacked_ZeroValue(t *testing.T) {
	// Should not panic
	DeallocatePacked(0)
}

func TestConcurrency(t *testing.T) {
	resetMemoryManager()

	var wg sync.WaitGroup
	iterations := 100

	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			packed := PtrFromBytes([]byte("concurrent test data"))
			_ = BytesFromPtr(packed)
			DeallocatePacked(packed)
		}()
	}
	wg.Wait()

	allocCount, _ := Stats()
	assert.Equal(t, 0, allocCount, "expected 0 allocations after concurrent operations")
}

func TestConfigure_WithMaxTotalAllocations(t *testing.T) {
	resetMemoryManager()

	// Configure with a small limit
	Configure(WithMaxTotalAllocations(1024))

	// This should succeed
	ptr := allocate(512)
	require.NotZero(t, ptr)
	deallocate(ptr, 512)

	// This should panic (exceeds limit)
	assert.Panics(t, func() {
		allocate(2048)
	}, "expected panic when exceeding allocation limit")

	// Reset to default for other tests
	Configure(WithMaxTotalAllocations(DefaultMaxTotalAllocations))
	FreeAllTracked()
}

func TestConfigure_InvalidLimit(t *testing.T) {
	resetMemoryManager()

	// Zero or negative limits should be ignored
	Configure(WithMaxTotalAllocations(0))
	Configure(WithMaxTotalAllocations(-100))

	// Should still work with default limit
	ptr := allocate(1024)
	require.NotZero(t, ptr)
	deallocate(ptr, 1024)
}
