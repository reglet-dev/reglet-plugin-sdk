//go:build wasip1

package sdknet

import (
	"crypto/tls"
	"testing"
	"time"
)

// BenchmarkNewTransport benchmarks the overhead of creating a transport with default options
func BenchmarkNewTransport(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = NewTransport()
	}
}

// BenchmarkNewTransportWithOptions benchmarks transport creation with multiple options
func BenchmarkNewTransportWithOptions(b *testing.B) {
	b.ReportAllocs()

	tlsCfg := &tls.Config{InsecureSkipVerify: false}

	for i := 0; i < b.N; i++ {
		_ = NewTransport(
			WithHTTPTimeout(60*time.Second),
			WithMaxRedirects(5),
			WithTLSConfig(tlsCfg),
		)
	}
}

// BenchmarkTransportOptionApplication benchmarks the performance of applying options
func BenchmarkTransportOptionApplication(b *testing.B) {
	b.ReportAllocs()

	tlsCfg := &tls.Config{InsecureSkipVerify: false}
	opts := []TransportOption{
		WithHTTPTimeout(30 * time.Second),
		WithMaxRedirects(10),
		WithTLSConfig(tlsCfg),
	}

	for i := 0; i < b.N; i++ {
		cfg := defaultTransportConfig()
		for _, opt := range opts {
			opt(&cfg)
		}
	}
}

// BenchmarkDefaultTransportConfig benchmarks the creation of default configuration
func BenchmarkDefaultTransportConfig(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = defaultTransportConfig()
	}
}
