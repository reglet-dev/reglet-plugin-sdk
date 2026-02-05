package ports

import (
	"context"
)

// DNSResolver defines the interface for DNS resolution operations.
// Infrastructure adapters (e.g., WASM host functions) implement this interface.
type DNSResolver interface {
	// LookupHost resolves IP addresses for a given hostname.
	// Returns A and AAAA records as string slices.
	LookupHost(ctx context.Context, host string) ([]string, error)

	// LookupCNAME returns the canonical name for the given host.
	LookupCNAME(ctx context.Context, host string) (string, error)

	// LookupMX returns MX records for the given domain.
	LookupMX(ctx context.Context, domain string) ([]MXRecord, error)

	// LookupTXT returns TXT records for the given domain.
	LookupTXT(ctx context.Context, domain string) ([]string, error)

	// LookupNS returns NS records (nameservers) for the given domain.
	LookupNS(ctx context.Context, domain string) ([]string, error)
}

// MXRecord represents a DNS MX record.
type MXRecord struct {
	Host string
	Pref uint16
}
