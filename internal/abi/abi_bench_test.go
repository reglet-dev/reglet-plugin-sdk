//go:build wasip1

// Package abi provides benchmark tests for SDK baseline performance.
// These benchmarks establish a baseline for hot path validation.
package abi

import (
	"encoding/json"
	"testing"
)

// BenchmarkPackPtrLen measures pointer packing performance.
// This is a minimal operation but in the hot path of all host function calls.
func BenchmarkPackPtrLen(b *testing.B) {
	ptr := uint32(0x12345678)
	length := uint32(256)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		packed := PackPtrLen(ptr, length)
		_ = packed
	}
}

// BenchmarkUnpackPtrLen measures pointer unpacking performance.
func BenchmarkUnpackPtrLen(b *testing.B) {
	packed := PackPtrLen(0x12345678, 256)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ptr, length := UnpackPtrLen(packed)
		_, _ = ptr, length
	}
}

// BenchmarkPackUnpackRoundtrip measures complete pack/unpack cycle.
func BenchmarkPackUnpackRoundtrip(b *testing.B) {
	ptr := uint32(0x12345678)
	length := uint32(256)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		packed := PackPtrLen(ptr, length)
		p, l := UnpackPtrLen(packed)
		_, _ = p, l
	}
}

// BenchmarkJSONMarshalDNSRequest measures wire format encoding performance.
// Wire format encoding is a hot path for all host function communication.
func BenchmarkJSONMarshalDNSRequest(b *testing.B) {
	type DNSRequest struct {
		Hostname   string `json:"hostname"`
		RecordType string `json:"type"`
		Nameserver string `json:"nameserver,omitempty"`
	}

	req := DNSRequest{
		Hostname:   "example.com",
		RecordType: "A",
		Nameserver: "8.8.8.8",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(req)
	}
}

// BenchmarkJSONUnmarshalDNSResponse measures wire format decoding performance.
// Wire format decoding is a hot path for all host function responses.
func BenchmarkJSONUnmarshalDNSResponse(b *testing.B) {
	type DNSResponse struct {
		Records []string `json:"records"`
		Error   *string  `json:"error"`
	}

	data := []byte(`{"records":["192.0.2.1","192.0.2.2"],"error":null}`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var resp DNSResponse
		_ = json.Unmarshal(data, &resp)
	}
}

// BenchmarkJSONMarshalHTTPRequest measures HTTP wire format encoding.
func BenchmarkJSONMarshalHTTPRequest(b *testing.B) {
	type HTTPRequest struct {
		URL     string            `json:"url"`
		Method  string            `json:"method"`
		Headers map[string]string `json:"headers,omitempty"`
		Body    string            `json:"body,omitempty"`
		Timeout int               `json:"timeout_ms"`
	}

	req := HTTPRequest{
		URL:     "https://example.com/api/v1/status",
		Method:  "GET",
		Headers: map[string]string{"Content-Type": "application/json"},
		Timeout: 30000,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(req)
	}
}

// BenchmarkJSONMarshalTCPRequest measures TCP wire format encoding.
func BenchmarkJSONMarshalTCPRequest(b *testing.B) {
	type TCPRequest struct {
		Host    string `json:"host"`
		Port    int    `json:"port"`
		Timeout int    `json:"timeout_ms"`
	}

	req := TCPRequest{Host: "example.com", Port: 443, Timeout: 5000}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(req)
	}
}

// BenchmarkAllocation measures memory allocation at various sizes.
func BenchmarkAllocation(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"64B", 64},
		{"256B", 256},
		{"1KB", 1024},
		{"4KB", 4096},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				buf := make([]byte, tc.size)
				_ = buf
			}
		})
	}
}
