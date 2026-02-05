package ports

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDNSResolver is a mock implementation of DNSResolver for testing.
type MockDNSResolver struct {
	LookupHostFunc  func(ctx context.Context, host string) ([]string, error)
	LookupCNAMEFunc func(ctx context.Context, host string) (string, error)
	LookupMXFunc    func(ctx context.Context, domain string) ([]MXRecord, error)
	LookupTXTFunc   func(ctx context.Context, domain string) ([]string, error)
	LookupNSFunc    func(ctx context.Context, domain string) ([]string, error)
}

func (m *MockDNSResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	if m.LookupHostFunc != nil {
		return m.LookupHostFunc(ctx, host)
	}
	return []string{"192.0.2.1"}, nil
}

func (m *MockDNSResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	if m.LookupCNAMEFunc != nil {
		return m.LookupCNAMEFunc(ctx, host)
	}
	return "example.com", nil
}

func (m *MockDNSResolver) LookupMX(ctx context.Context, domain string) ([]MXRecord, error) {
	if m.LookupMXFunc != nil {
		return m.LookupMXFunc(ctx, domain)
	}
	return []MXRecord{{Host: "mail.example.com", Pref: 10}}, nil
}

func (m *MockDNSResolver) LookupTXT(ctx context.Context, domain string) ([]string, error) {
	if m.LookupTXTFunc != nil {
		return m.LookupTXTFunc(ctx, domain)
	}
	return []string{"v=spf1 include:_spf.example.com ~all"}, nil
}

func (m *MockDNSResolver) LookupNS(ctx context.Context, domain string) ([]string, error) {
	if m.LookupNSFunc != nil {
		return m.LookupNSFunc(ctx, domain)
	}
	return []string{"ns1.example.com", "ns2.example.com"}, nil
}

// Compile-time interface check
var _ DNSResolver = (*MockDNSResolver)(nil)

func TestMockDNSResolver_ImplementsInterface(t *testing.T) {
	// Verify that MockDNSResolver implements DNSResolver interface
	var resolver DNSResolver = &MockDNSResolver{}
	require.NotNil(t, resolver)
}

func TestMockDNSResolver_LookupHost(t *testing.T) {
	ctx := context.Background()

	t.Run("default behavior", func(t *testing.T) {
		mock := &MockDNSResolver{}
		ips, err := mock.LookupHost(ctx, "example.com")

		require.NoError(t, err)
		assert.Equal(t, []string{"192.0.2.1"}, ips)
	})

	t.Run("custom behavior", func(t *testing.T) {
		mock := &MockDNSResolver{
			LookupHostFunc: func(ctx context.Context, host string) ([]string, error) {
				return []string{"192.0.2.1", "192.0.2.2"}, nil
			},
		}

		ips, err := mock.LookupHost(ctx, "example.com")

		require.NoError(t, err)
		assert.Len(t, ips, 2)
		assert.Contains(t, ips, "192.0.2.1")
		assert.Contains(t, ips, "192.0.2.2")
	})
}

func TestMockDNSResolver_LookupCNAME(t *testing.T) {
	ctx := context.Background()
	mock := &MockDNSResolver{}

	cname, err := mock.LookupCNAME(ctx, "www.example.com")

	require.NoError(t, err)
	assert.Equal(t, "example.com", cname)
}

func TestMockDNSResolver_LookupMX(t *testing.T) {
	ctx := context.Background()
	mock := &MockDNSResolver{}

	mxRecords, err := mock.LookupMX(ctx, "example.com")

	require.NoError(t, err)
	require.Len(t, mxRecords, 1)
	assert.Equal(t, "mail.example.com", mxRecords[0].Host)
	assert.Equal(t, uint16(10), mxRecords[0].Pref)
}

func TestMockDNSResolver_LookupTXT(t *testing.T) {
	ctx := context.Background()
	mock := &MockDNSResolver{}

	txtRecords, err := mock.LookupTXT(ctx, "example.com")

	require.NoError(t, err)
	require.Len(t, txtRecords, 1)
	assert.Contains(t, txtRecords[0], "v=spf1")
}

func TestMockDNSResolver_LookupNS(t *testing.T) {
	ctx := context.Background()
	mock := &MockDNSResolver{}

	nsRecords, err := mock.LookupNS(ctx, "example.com")

	require.NoError(t, err)
	require.Len(t, nsRecords, 2)
	assert.Contains(t, nsRecords, "ns1.example.com")
	assert.Contains(t, nsRecords, "ns2.example.com")
}
