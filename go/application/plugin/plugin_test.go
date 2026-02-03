//go:build !wasip1

package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/reglet-dev/reglet-sdk/go/domain/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Type aliases for convenience in tests
type (
	Result          = entities.Result
	ErrorDetail     = entities.ErrorDetail
	Config          = map[string]any
	NetworkError    = errors.NetworkError
	DNSError        = errors.DNSError
	HTTPError       = errors.HTTPError
	TCPError        = errors.TCPError
	TimeoutError    = errors.TimeoutError
	CapabilityError = errors.CapabilityError
	ConfigError     = errors.ConfigError
	ExecError       = errors.ExecError
	SchemaError     = errors.SchemaError
	MemoryError     = errors.MemoryError
	WireFormatError = errors.WireFormatError
)

// Function aliases
var ToErrorDetail = errors.ToErrorDetail

func TestManifest_Capabilities(t *testing.T) {
	tests := []struct {
		name     string
		manifest entities.Manifest
	}{
		{
			name: "single fs capability",
			manifest: entities.Manifest{
				Name:    "test",
				Version: "1.0.0",
				Capabilities: entities.GrantSet{
					FS: &entities.FileSystemCapability{
						Rules: []entities.FileSystemRule{
							{Read: []string{"/etc/**"}},
						},
					},
				},
			},
		},
		{
			name: "multiple capabilities",
			manifest: entities.Manifest{
				Name:    "test",
				Version: "1.0.0",
				Capabilities: entities.GrantSet{
					Network: &entities.NetworkCapability{
						Rules: []entities.NetworkRule{
							{Hosts: []string{"*"}, Ports: []string{"80", "443"}},
						},
					},
					Exec: &entities.ExecCapability{
						Commands: []string{"systemctl"},
					},
				},
			},
		},
		{
			name: "no capabilities",
			manifest: entities.Manifest{
				Name:         "test",
				Version:      "1.0.0",
				Capabilities: entities.GrantSet{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON serialization
			data, err := json.Marshal(tt.manifest)
			require.NoError(t, err)

			var decoded entities.Manifest
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)
			assert.Equal(t, tt.manifest.Name, decoded.Name)
			assert.Equal(t, tt.manifest.Version, decoded.Version)
			// Verify capabilities can be serialized/deserialized
			if !tt.manifest.Capabilities.IsEmpty() {
				assert.False(t, decoded.Capabilities.IsEmpty())
			}
		})
	}
}

// MockPlugin for testing Manifest
type MockPlugin struct{}

func (m *MockPlugin) Manifest(ctx context.Context) (*entities.Manifest, error) {
	return &entities.Manifest{
		Name:    "stub",
		Version: "1.0.0",
		Capabilities: entities.GrantSet{
			KV: &entities.KeyValueCapability{
				Rules: []entities.KeyValueRule{
					{Operation: "read", Keys: []string{"test:resource"}},
				},
			},
		},
	}, nil
}

func (m *MockPlugin) Execute(ctx context.Context, config map[string]any) (*entities.Result, error) {
	res := entities.ResultSuccess("mock executed", nil)
	return &res, nil
}

func TestPluginManifest(t *testing.T) {
	userPlugin := &MockPlugin{}

	manifest, err := userPlugin.Manifest(context.Background())
	if err != nil {
		t.Fatalf("Manifest failed: %v", err)
	}
	if manifest.Name != "stub" {
		t.Errorf("expected name stub, got %s", manifest.Name)
	}
	assert.Equal(t, "1.0.0", manifest.Version)
	// Verify capabilities are present
	assert.False(t, manifest.Capabilities.IsEmpty(), "manifest should have capabilities")
	assert.NotNil(t, manifest.Capabilities.KV, "KV capability should be set")
}

func TestResult_Serialization(t *testing.T) {
	tests := []struct {
		name   string
		result Result
	}{
		{
			name: "success result",
			result: entities.ResultSuccess("operation completed", map[string]interface{}{
				"stdout":    "hello world",
				"exit_code": 0,
			}),
		},
		{
			name:   "failure result with data",
			result: entities.ResultFailure("check failed", map[string]interface{}{"attempted": true}),
		},
		{
			name: "error result",
			result: entities.ResultError(&entities.ErrorDetail{
				Message: "connection refused",
				Type:    "network",
				Code:    "ECONNREFUSED",
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON marshal/unmarshal round-trip
			data, err := json.Marshal(tt.result)
			require.NoError(t, err)

			var decoded Result
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.result.Status, decoded.Status)
			assert.Equal(t, tt.result.Message, decoded.Message)

			if tt.result.Error != nil {
				require.NotNil(t, decoded.Error)
				assert.Equal(t, tt.result.Error.Message, decoded.Error.Message)
				assert.Equal(t, tt.result.Error.Type, decoded.Error.Type)
				assert.Equal(t, tt.result.Error.Code, decoded.Error.Code)
			} else {
				assert.Nil(t, decoded.Error)
			}
		})
	}
}

func TestConfig_Handling(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		wantJSON string
	}{
		{
			name: "simple config",
			config: Config{
				"host": "example.com",
				"port": 443,
			},
			wantJSON: `{"host":"example.com","port":443}`,
		},
		{
			name: "nested config",
			config: Config{
				"server": map[string]interface{}{
					"host": "example.com",
					"port": 443,
				},
				"timeout": 30,
			},
		},
		{
			name:     "empty config",
			config:   Config{},
			wantJSON: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON serialization
			data, err := json.Marshal(tt.config)
			require.NoError(t, err)

			// Test JSON deserialization
			var decoded Config
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			// For simple configs, check exact JSON match
			if tt.wantJSON != "" {
				assert.JSONEq(t, tt.wantJSON, string(data))
			}
		})
	}
}

func TestToErrorDetail(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantMessage string
		wantType    string
	}{
		{
			name:        "simple error",
			err:         fmt.Errorf("connection failed"),
			wantMessage: "connection failed",
			wantType:    "internal",
		},
		{
			name:        "wrapped error",
			err:         fmt.Errorf("failed to connect: %w", fmt.Errorf("timeout")),
			wantMessage: "failed to connect: timeout",
			wantType:    "internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detail := ToErrorDetail(tt.err)
			require.NotNil(t, detail)
			assert.Equal(t, tt.wantMessage, detail.Message)
			assert.Equal(t, tt.wantType, detail.Type)
		})
	}
}

func TestResultSuccess(t *testing.T) {
	data := map[string]interface{}{
		"result": "ok",
		"count":  42,
	}

	result := entities.ResultSuccess("operation succeeded", data)
	assert.Equal(t, entities.ResultStatusSuccess, result.Status)
	assert.Equal(t, "operation succeeded", result.Message)
	assert.Equal(t, data, result.Data)
	assert.Nil(t, result.Error)
}

func TestResultFailure(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		data        map[string]any
		wantMessage string
	}{
		{
			name:        "failure with data",
			message:     "connection refused",
			data:        map[string]any{"attempted": true},
			wantMessage: "connection refused",
		},
		{
			name:        "failure without data",
			message:     "validation failed",
			data:        nil,
			wantMessage: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := entities.ResultFailure(tt.message, tt.data)
			assert.Equal(t, entities.ResultStatusFailure, result.Status)
			assert.Equal(t, tt.wantMessage, result.Message)
			assert.Equal(t, tt.data, result.Data)
		})
	}
}

func TestResultError(t *testing.T) {
	errDetail := &entities.ErrorDetail{
		Message: "network error",
		Type:    "network",
		Code:    "ECONNREFUSED",
	}

	result := entities.ResultError(errDetail)
	assert.Equal(t, entities.ResultStatusError, result.Status)
	assert.Equal(t, errDetail.Message, result.Message)
	require.NotNil(t, result.Error)
	assert.Equal(t, "network error", result.Error.Message)
	assert.Equal(t, "network", result.Error.Type)
}

func TestConfigErrorConstruction(t *testing.T) {
	err := fmt.Errorf("missing required field 'host'")
	result := entities.ResultError(ToErrorDetail(&ConfigError{
		Field: "host",
		Err:   err,
	}))

	assert.Equal(t, entities.ResultStatusError, result.Status)
	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Message, "config validation failed for field 'host'")
	assert.Equal(t, "config", result.Error.Type)
	assert.Equal(t, "host", result.Error.Code)
}

func TestNetworkErrorConstruction(t *testing.T) {
	err := fmt.Errorf("connection timeout")
	result := entities.ResultError(ToErrorDetail(&NetworkError{
		Operation: "connect",
		Target:    "api.example.com:443",
		Err:       err,
	}))

	assert.Equal(t, entities.ResultStatusError, result.Status)
	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Message, "network connect failed for api.example.com:443")
	assert.Equal(t, "network", result.Error.Type)
	assert.Equal(t, "connect", result.Error.Code)
}

// Test ToErrorDetail with custom error types
func TestToErrorDetail_CustomErrorTypes(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType string
		expectedCode string
	}{
		{
			name: "NetworkError",
			err: &NetworkError{
				Operation: "http_request",
				Target:    "api.example.com",
				Err:       fmt.Errorf("connection refused"),
			},
			expectedType: "network",
			expectedCode: "http_request",
		},
		{
			name: "DNSError",
			err: &DNSError{
				Hostname:   "example.com",
				RecordType: "A",
				Err:        fmt.Errorf("no such host"),
			},
			expectedType: "network",
			expectedCode: "dns_A",
		},
		{
			name: "HTTPError",
			err: &HTTPError{
				Method:     "GET",
				URL:        "https://api.example.com",
				StatusCode: 500,
				Err:        fmt.Errorf("internal server error"),
			},
			expectedType: "network",
			expectedCode: "http_500",
		},
		{
			name: "TCPError",
			err: &TCPError{
				Network: "tcp",
				Address: "example.com:443",
				Err:     fmt.Errorf("connection refused"),
			},
			expectedType: "network",
			expectedCode: "tcp_connect",
		},
		{
			name: "TimeoutError",
			err: &TimeoutError{
				Operation: "dns_lookup",
				Duration:  5 * time.Second,
			},
			expectedType: "timeout",
			expectedCode: "dns_lookup",
		},
		{
			name: "CapabilityError",
			err: &CapabilityError{
				Required: "network:outbound",
				Pattern:  "api.example.com:443",
			},
			expectedType: "capability",
			expectedCode: "network:outbound",
		},
		{
			name: "ConfigError",
			err: &ConfigError{
				Field: "hostname",
				Err:   fmt.Errorf("invalid format"),
			},
			expectedType: "config",
			expectedCode: "hostname",
		},
		{
			name: "ExecError",
			err: &ExecError{
				Command:  "grep",
				ExitCode: 1,
				Stderr:   "pattern not found",
			},
			expectedType: "exec",
			expectedCode: "exit_1",
		},
		{
			name: "SchemaError",
			err: &SchemaError{
				Type: "MyStruct",
				Err:  fmt.Errorf("unsupported type"),
			},
			expectedType: "validation",
			expectedCode: "schema",
		},
		{
			name: "MemoryError",
			err: &MemoryError{
				Requested: 10 * 1024 * 1024,
				Current:   95 * 1024 * 1024,
				Limit:     100 * 1024 * 1024,
			},
			expectedType: "internal",
			expectedCode: "memory_limit",
		},
		{
			name: "WireFormatError",
			err: &WireFormatError{
				Operation: "unmarshal",
				Type:      "DNSResponseWire",
				Err:       fmt.Errorf("invalid json"),
			},
			expectedType: "internal",
			expectedCode: "wire_format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detail := ToErrorDetail(tt.err)

			require.NotNil(t, detail)
			assert.Equal(t, tt.expectedType, detail.Type, "Error type mismatch")
			assert.Equal(t, tt.expectedCode, detail.Code, "Error code mismatch")
			assert.NotEmpty(t, detail.Message)
		})
	}
}

func TestToErrorDetail_TimeoutPropagation(t *testing.T) {
	// Test that timeout is properly detected through wrapped errors
	timeoutErr := &TimeoutError{Operation: "http", Duration: 10 * time.Second}

	tests := []struct {
		name         string
		err          error
		expectedType string
	}{
		{
			name: "DNSError with timeout",
			err: &DNSError{
				Hostname:   "example.com",
				RecordType: "A",
				Err:        timeoutErr,
			},
			expectedType: "timeout",
		},
		{
			name: "HTTPError with timeout",
			err: &HTTPError{
				Method: "GET",
				URL:    "https://example.com",
				Err:    timeoutErr,
			},
			expectedType: "timeout",
		},
		{
			name: "TCPError with timeout",
			err: &TCPError{
				Network: "tcp",
				Address: "example.com:443",
				Err:     timeoutErr,
			},
			expectedType: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detail := ToErrorDetail(tt.err)

			require.NotNil(t, detail)
			assert.Equal(t, tt.expectedType, detail.Type, "Should detect timeout")
		})
	}
}
