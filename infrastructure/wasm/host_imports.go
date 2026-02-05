//go:build wasip1

// Package wasm provides infrastructure adapters that interface with the WASM host environment.
package wasm

// Define the host function signature for HTTP requests.
//
//go:wasmimport reglet_host http_request
func host_http_request(requestPacked uint64) uint64

// Define the host function signature for DNS lookups.
//
//go:wasmimport reglet_host dns_lookup
func host_dns_lookup(requestPacked uint64) uint64

// Define the host function signature for TCP connections.
//
//go:wasmimport reglet_host tcp_connect
func host_tcp_connect(requestPacked uint64) uint64

// Define the host function signature for SMTP connections.
//
//go:wasmimport reglet_host smtp_connect
func host_smtp_connect(requestPacked uint64) uint64

// Define the host function signature for command execution.
//
//go:wasmimport reglet_host exec_command
func host_exec_command(reqPacked uint64) uint64
