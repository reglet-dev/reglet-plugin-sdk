package sdknet

import (
	"context"
	"errors"
	"testing"

	"github.com/reglet-dev/reglet-sdk/application/config"
	"github.com/reglet-dev/reglet-sdk/domain/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDNSResolver
type MockDNSResolver struct {
	mock.Mock
}

func (m *MockDNSResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	args := m.Called(ctx, host)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockDNSResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	args := m.Called(ctx, host)
	return args.String(0), args.Error(1)
}

func (m *MockDNSResolver) LookupMX(ctx context.Context, domain string) ([]ports.MXRecord, error) {
	args := m.Called(ctx, domain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.MXRecord), args.Error(1)
}

func (m *MockDNSResolver) LookupTXT(ctx context.Context, domain string) ([]string, error) {
	args := m.Called(ctx, domain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockDNSResolver) LookupNS(ctx context.Context, domain string) ([]string, error) {
	args := m.Called(ctx, domain)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func TestRunDNSCheck_Validation(t *testing.T) {
	tests := []struct {
		name      string
		cfg       config.Config
		errCode   string
		errDetail string
	}{
		{
			name:    "Missing Hostname",
			cfg:     config.Config{},
			errCode: "MISSING_HOSTNAME",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RunDNSCheck(context.Background(), tt.cfg)
			require.NoError(t, err)
			assert.True(t, result.IsError())
			assert.Equal(t, tt.errCode, result.Error.Code)
		})
	}
}

func TestRunDNSCheck_Mock_A_Record(t *testing.T) {
	mockResolver := new(MockDNSResolver)

	// Mock LookupHost returning IPs
	mockResolver.On("LookupHost", mock.Anything, "example.com").Return([]string{"1.2.3.4", "2001:db8::1"}, nil)

	cfg := config.Config{
		"hostname":    "example.com",
		"record_type": "A",
	}

	result, err := RunDNSCheck(context.Background(), cfg, WithDNSResolver(mockResolver))

	require.NoError(t, err)
	assert.True(t, result.IsSuccess())
	assert.NotNil(t, result.Data["records"])
	records := result.Data["records"].([]string)
	// Should be filtered to only IPv4
	assert.Len(t, records, 1)
	assert.Equal(t, "1.2.3.4", records[0])

	mockResolver.AssertExpectations(t)
}

func TestRunDNSCheck_Mock_MX_Record(t *testing.T) {
	mockResolver := new(MockDNSResolver)

	mxRecords := []ports.MXRecord{
		{Host: "mail.example.com", Pref: 10},
	}
	mockResolver.On("LookupMX", mock.Anything, "example.com").Return(mxRecords, nil)

	cfg := config.Config{
		"hostname":    "example.com",
		"record_type": "MX",
	}

	result, err := RunDNSCheck(context.Background(), cfg, WithDNSResolver(mockResolver))

	require.NoError(t, err)
	assert.True(t, result.IsSuccess())
	assert.NotNil(t, result.Data["mx_records"])
	mxData := result.Data["mx_records"].([]map[string]any)
	assert.Len(t, mxData, 1)
	assert.Equal(t, "mail.example.com", mxData[0]["host"])
	// Check type of pref, json unmarshal usually makes numbers float64, but here it's uint16 in struct
	// Converted to any in map.
	assert.Equal(t, uint16(10), mxData[0]["pref"])

	mockResolver.AssertExpectations(t)
}

func TestRunDNSCheck_Mock_LookupFailed(t *testing.T) {
	mockResolver := new(MockDNSResolver)

	mockResolver.On("LookupHost", mock.Anything, "example.com").Return(nil, errors.New("no such host"))

	cfg := config.Config{
		"hostname":    "example.com",
		"record_type": "A",
	}

	result, err := RunDNSCheck(context.Background(), cfg, WithDNSResolver(mockResolver))

	require.NoError(t, err)
	assert.True(t, result.IsError())
	assert.Equal(t, "LOOKUP_FAILED", result.Error.Code)
	mockResolver.AssertExpectations(t)
}

func TestRunDNSCheck_DefaultResolver_PanicsOnNative(t *testing.T) {
	cfg := config.Config{"hostname": "example.com"}
	assert.PanicsWithValue(t, "WASM DNS adapter not available in native build", func() {
		_, _ = RunDNSCheck(context.Background(), cfg)
	})
}
