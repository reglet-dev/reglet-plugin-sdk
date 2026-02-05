//go:build !wasip1

package wasm

import (
	"context"
	"time"

	"github.com/reglet-dev/reglet-sdk/domain/entities"
	"github.com/reglet-dev/reglet-sdk/domain/ports"
)

// DNSAdapter stub for native builds.
type DNSAdapter struct{}

func NewDNSAdapter(nameserver string, timeout time.Duration) *DNSAdapter {
	return &DNSAdapter{}
}

func (r *DNSAdapter) LookupHost(ctx context.Context, host string) ([]string, error) {
	panic("WASM DNS adapter not available in native build")
}

func (r *DNSAdapter) LookupCNAME(ctx context.Context, host string) (string, error) {
	panic("WASM DNS adapter not available in native build")
}

func (r *DNSAdapter) LookupMX(ctx context.Context, domain string) ([]ports.MXRecord, error) {
	panic("WASM DNS adapter not available in native build")
}

func (r *DNSAdapter) LookupTXT(ctx context.Context, domain string) ([]string, error) {
	panic("WASM DNS adapter not available in native build")
}

func (r *DNSAdapter) LookupNS(ctx context.Context, domain string) ([]string, error) {
	panic("WASM DNS adapter not available in native build")
}

func (r *DNSAdapter) Lookup(ctx context.Context, hostname, recordType string) (*entities.DNSResponse, error) {
	panic("WASM DNS adapter not available in native build")
}
