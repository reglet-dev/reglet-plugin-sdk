//go:build wasip1

package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/reglet-dev/reglet-sdk/domain/entities"
	"github.com/reglet-dev/reglet-sdk/domain/ports"
	"github.com/reglet-dev/reglet-sdk/internal/abi"
	_ "github.com/reglet-dev/reglet-sdk/log" // Initialize WASM logging handler
)

// Compile-time interface compliance check
var _ ports.DNSResolver = (*DNSAdapter)(nil)

// DNSAdapter implements ports.DNSResolver for the WASM environment.
type DNSAdapter struct {
	// Nameserver is the address of the nameserver to use for resolution (e.g. "8.8.8.8:53").
	// If empty, the host's default resolver is used.
	Nameserver string

	// Timeout is the timeout for DNS queries.
	Timeout time.Duration
}

// NewDNSAdapter creates a new DNS adapter.
func NewDNSAdapter(nameserver string, timeout time.Duration) *DNSAdapter {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &DNSAdapter{
		Nameserver: nameserver,
		Timeout:    timeout,
	}
}

// LookupHost resolves IP addresses for a given host using the host function.
func (r *DNSAdapter) LookupHost(ctx context.Context, host string) ([]string, error) {
	resp, err := r.Lookup(ctx, host, "A")
	if err != nil {
		return nil, err
	}
	recordsA := resp.Records

	resp, err = r.Lookup(ctx, host, "AAAA")
	if err != nil {
		return nil, err
	}
	recordsAAAA := resp.Records

	return append(recordsA, recordsAAAA...), nil
}

// LookupCNAME returns the canonical name for the given host.
func (r *DNSAdapter) LookupCNAME(ctx context.Context, host string) (string, error) {
	resp, err := r.Lookup(ctx, host, "CNAME")
	if err != nil {
		return "", err
	}
	if len(resp.Records) == 0 {
		return "", fmt.Errorf("no CNAME record found")
	}
	return resp.Records[0], nil
}

// LookupMX returns MX records for the given domain.
func (r *DNSAdapter) LookupMX(ctx context.Context, domain string) ([]ports.MXRecord, error) {
	resp, err := r.Lookup(ctx, domain, "MX")
	if err != nil {
		return nil, err
	}

	var records []ports.MXRecord
	for _, mx := range resp.MXRecords {
		records = append(records, ports.MXRecord{
			Host: mx.Host,
			Pref: mx.Pref,
		})
	}
	return records, nil
}

// LookupTXT returns TXT records for the given domain.
func (r *DNSAdapter) LookupTXT(ctx context.Context, domain string) ([]string, error) {
	resp, err := r.Lookup(ctx, domain, "TXT")
	if err != nil {
		return nil, err
	}
	return resp.Records, nil
}

// LookupNS returns NS records for the given domain.
func (r *DNSAdapter) LookupNS(ctx context.Context, domain string) ([]string, error) {
	resp, err := r.Lookup(ctx, domain, "NS")
	if err != nil {
		return nil, err
	}
	return resp.Records, nil
}

// Lookup performs the actual DNS query via the host function.
func (r *DNSAdapter) Lookup(ctx context.Context, hostname, recordType string) (*entities.DNSResponse, error) {
	request := entities.DNSRequest{
		Context:    entities.ContextWire{}, // Zero value for now
		Hostname:   hostname,
		Type:       recordType,
		Nameserver: r.Nameserver,
	}

	if d, ok := ctx.Deadline(); ok {
		_ = d
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("sdk: failed to marshal DNS request: %w", err)
	}

	// Call the host function
	responsePacked := host_dns_lookup(abi.PtrFromBytes(requestBytes))

	// Read and unmarshal the response
	responseBytes := abi.BytesFromPtr(responsePacked)
	abi.DeallocatePacked(responsePacked) // Free memory on Guest side (allocated by Host for result)

	var response entities.DNSResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return nil, fmt.Errorf("sdk: failed to unmarshal DNS response: %w", err)
	}

	if response.Error != nil {
		return nil, response.Error // Convert structured error to Go error
	}

	return &response, nil
}
