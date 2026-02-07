package sdknet

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/reglet-dev/reglet-plugin-sdk/application/config"
	"github.com/reglet-dev/reglet-plugin-sdk/domain/entities"
	"github.com/reglet-dev/reglet-plugin-sdk/domain/ports"
	"github.com/reglet-dev/reglet-plugin-sdk/infrastructure/wasm"
)

// DNSCheckOption is a functional option for configuring DNS checks.
type DNSCheckOption func(*dnsCheckConfig)

type dnsCheckConfig struct {
	resolver ports.DNSResolver
}

// WithDNSResolver sets the DNS resolver to use for the check.
// This is useful for injecting mocks during testing.
func WithDNSResolver(r ports.DNSResolver) DNSCheckOption {
	return func(c *dnsCheckConfig) {
		if r != nil {
			c.resolver = r
		}
	}
}

// RunDNSCheck performs a DNS lookup check.
// It parses configuration, executes the DNS lookup, and returns a structured Result.
//
// Expected config fields:
//   - hostname (string, required): Domain name to resolve
//   - record_type (string, optional): DNS record type (A, AAAA, CNAME, MX, TXT, NS). Default: A
//   - nameserver (string, optional): Custom nameserver (e.g., "8.8.8.8")
//   - timeout_ms (int, optional): Lookup timeout in milliseconds (default: 5000)
//
// Returns a Result with:
//   - Status: "success" if lookup succeeded, "error" if failed
//   - Data: map containing "records" ([]string) or "mx_records" (for MX queries)
//   - Error: structured error details if lookup failed
func RunDNSCheck(ctx context.Context, cfg config.Config, opts ...DNSCheckOption) (entities.Result, error) {
	// Parse required fields
	hostname, err := config.MustGetString(cfg, "hostname")
	if err != nil {
		return entities.ResultError(entities.NewErrorDetail("config", err.Error()).WithCode("MISSING_HOSTNAME")), nil
	}

	// Parse optional fields
	recordType := config.GetStringDefault(cfg, "record_type", "A")
	nameserver := config.GetStringDefault(cfg, "nameserver", "")
	timeoutMs := config.GetIntDefault(cfg, "timeout_ms", 5000)

	// Configure check
	// Create default resolver based on config
	resolverOpts := []ResolverOption{}
	if nameserver != "" {
		resolverOpts = append(resolverOpts, WithNameserver(nameserver))
	}
	if timeoutMs > 0 {
		resolverOpts = append(resolverOpts, WithDNSTimeout(time.Duration(timeoutMs)*time.Millisecond))
	}
	defaultResolver := NewResolver(resolverOpts...)

	checkCfg := dnsCheckConfig{
		resolver: defaultResolver,
	}

	for _, opt := range opts {
		opt(&checkCfg)
	}

	// Execute DNS lookup based on record type
	start := time.Now()
	records, mxRecords, lookupErr := performDNSLookup(ctx, checkCfg.resolver, hostname, recordType)
	latency := time.Since(start)
	metadata := entities.NewRunMetadata(start, time.Now())

	// Build result data
	resultData := make(map[string]any)
	resultData["query_time_ms"] = latency.Milliseconds()

	if len(records) > 0 {
		resultData["records"] = records
	}
	if len(mxRecords) > 0 {
		// Convert MXRecords to map format for result
		mxMap := make([]map[string]any, len(mxRecords))
		for i, mx := range mxRecords {
			mxMap[i] = map[string]any{
				"host": mx.Host,
				"pref": mx.Pref,
			}
		}
		resultData["mx_records"] = mxMap
	}
	resultData["record_count"] = len(records) + len(mxRecords)
	resultData["record_type"] = recordType
	resultData["hostname"] = hostname

	// Return result based on lookup status
	if lookupErr == nil {
		message := fmt.Sprintf("DNS lookup successful for %s (%s)", hostname, recordType)
		return entities.ResultSuccess(message, resultData).WithMetadata(metadata), nil
	}

	// Lookup failed
	// specific flags for timeout and not found
	isTimeout := strings.Contains(lookupErr.Error(), "timeout") || strings.Contains(lookupErr.Error(), "deadline exceeded")
	isNotFound := strings.Contains(lookupErr.Error(), "no such host") || strings.Contains(lookupErr.Error(), "not found")

	resultData["is_timeout"] = isTimeout
	resultData["is_not_found"] = isNotFound

	errDetail := entities.NewErrorDetail("network", lookupErr.Error()).WithCode("LOOKUP_FAILED")
	res := entities.ResultError(errDetail).WithMetadata(metadata)
	res.Data = resultData
	return res, nil
}

func performDNSLookup(ctx context.Context, resolver ports.DNSResolver, hostname, recordType string) ([]string, []ports.MXRecord, error) {
	var records []string
	var mxRecords []ports.MXRecord
	var err error

	switch recordType {
	case "A":
		// Filter for IPv4
		allIPs, lookupErr := resolver.LookupHost(ctx, hostname)
		err = lookupErr
		if err == nil {
			for _, ip := range allIPs {
				if net.ParseIP(ip).To4() != nil {
					records = append(records, ip)
				}
			}
		}
	case "AAAA":
		// Filter for IPv6
		allIPs, lookupErr := resolver.LookupHost(ctx, hostname)
		err = lookupErr
		if err == nil {
			for _, ip := range allIPs {
				if net.ParseIP(ip).To4() == nil && net.ParseIP(ip) != nil {
					records = append(records, ip)
				}
			}
		}
	case "CNAME":
		cname, lookupErr := resolver.LookupCNAME(ctx, hostname)
		err = lookupErr
		if err == nil {
			records = []string{cname}
		}
	case "MX":
		mxRecords, err = resolver.LookupMX(ctx, hostname)
	case "TXT":
		records, err = resolver.LookupTXT(ctx, hostname)
	case "NS":
		records, err = resolver.LookupNS(ctx, hostname)
	default:
		err = fmt.Errorf("unsupported record type: %s", recordType)
	}

	return records, mxRecords, err
}

// resolverConfig holds the configuration for a WasmResolver.
// This struct is unexported to enforce the functional options pattern.
type resolverConfig struct {
	nameserver string        // DNS nameserver address (e.g., "8.8.8.8:53")
	timeout    time.Duration // Timeout for DNS queries (default: 5s)
	retries    int           // Number of retry attempts (default: 3)
}

// defaultResolverConfig returns secure defaults for DNS resolution.
// These defaults align with constitution requirements for secure-by-default.
func defaultResolverConfig() resolverConfig {
	return resolverConfig{
		nameserver: "", // Empty = use host's default resolver
		timeout:    5 * time.Second,
		retries:    3,
	}
}

// ResolverOption is a functional option for configuring a WasmResolver.
// Use With* functions to create options.
type ResolverOption func(*resolverConfig)

// WithNameserver sets a custom DNS nameserver address.
// The address should be in "host:port" format (e.g., "8.8.8.8:53").
// If port is omitted, ":53" is assumed.
// If not specified, the host's default resolver is used.
func WithNameserver(ns string) ResolverOption {
	return func(c *resolverConfig) {
		c.nameserver = ns
	}
}

// WithDNSTimeout sets the timeout for DNS queries.
// Default is 5 seconds per constitution specification.
// A zero or negative duration is ignored (uses default).
func WithDNSTimeout(d time.Duration) ResolverOption {
	return func(c *resolverConfig) {
		if d > 0 {
			c.timeout = d
		}
	}
}

// WithRetries sets the number of retry attempts for failed DNS queries.
// Default is 3 retries. A negative value is ignored (uses default).
// Setting to 0 disables retries (single attempt only).
func WithRetries(n int) ResolverOption {
	return func(c *resolverConfig) {
		if n >= 0 {
			c.retries = n
		}
	}
}

// NewResolver creates a new DNS resolver with the given options.
// Without any options, secure defaults are applied:
//   - timeout: 5 seconds
//   - retries: 3
//   - nameserver: host's default resolver
//
// Example:
//
//	// Use defaults
//	resolver := NewResolver()
//
//	// With custom nameserver and timeout
//	resolver := NewResolver(
//	    WithNameserver("1.1.1.1:53"),
//	    WithDNSTimeout(10*time.Second),
//	)
func NewResolver(opts ...ResolverOption) ports.DNSResolver {
	cfg := defaultResolverConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	// Note: retries are currently ignored by the underlying WASM adapter
	return wasm.NewDNSAdapter(cfg.nameserver, cfg.timeout)
}
