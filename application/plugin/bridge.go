package plugin

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

// CallHost invokes a host function that uses the SDK's packed uint64 ABI.
func CallHost[Req any, Resp any](hostFunc func(uint64) uint64, req Req) (Resp, error) {
	var resp Resp
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return resp, fmt.Errorf("failed to marshal request: %w", err)
	}

	packedReq := PackBytes(reqBytes)
	packedResp := hostFunc(packedReq)

	respBytes := UnpackBytes(packedResp)
	if respBytes == nil {
		return resp, fmt.Errorf("host returned no data")
	}

	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return resp, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return resp, nil
}

// PackBytes packs a byte slice into a uint64 (ptr << 32 | len).
func PackBytes(data []byte) uint64 {
	if len(data) == 0 {
		return 0
	}
	//nolint:gosec // Safe pointer arithmetic for ABI packing
	ptr := uint32(uintptr(unsafe.Pointer(&data[0])))
	return (uint64(ptr) << 32) | uint64(len(data))
}

// UnpackBytes unpacks a uint64 into a byte slice.
func UnpackBytes(packed uint64) []byte {
	//nolint:gosec // WASM pointers are 32-bit
	ptr := uint32(packed >> 32)
	//nolint:gosec // WASM lengths are 32-bit
	length := uint32(packed)
	if ptr == 0 || length == 0 {
		return nil
	}
	//nolint:gosec,govet // Safe pointer arithmetic for ABI unpacking
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
}
