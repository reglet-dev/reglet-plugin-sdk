//go:build wasip1

package sdknet

import (
	"testing"
	"time"
)

// BenchmarkNewResolver benchmarks the overhead of creating a resolver with default options
func BenchmarkNewResolver(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = NewResolver()
	}
}

// BenchmarkNewResolverWithOptions benchmarks resolver creation with multiple options
func BenchmarkNewResolverWithOptions(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = NewResolver(
			WithNameserver("8.8.8.8:53"),
			WithDNSTimeout(10*time.Second),
			WithRetries(5),
		)
	}
}

// BenchmarkResolverOptionApplication benchmarks the performance of applying options
func BenchmarkResolverOptionApplication(b *testing.B) {
	b.ReportAllocs()

	opts := []ResolverOption{
		WithNameserver("1.1.1.1:53"),
		WithDNSTimeout(5 * time.Second),
		WithRetries(3),
	}

	for i := 0; i < b.N; i++ {
		cfg := defaultResolverConfig()
		for _, opt := range opts {
			opt(&cfg)
		}
	}
}

// BenchmarkDefaultResolverConfig benchmarks the creation of default configuration
func BenchmarkDefaultResolverConfig(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = defaultResolverConfig()
	}
}
