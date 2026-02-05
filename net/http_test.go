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

// MockHTTPClient
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(ctx context.Context, req ports.HTTPRequest) (*ports.HTTPResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.HTTPResponse), args.Error(1)
}

func (m *MockHTTPClient) Get(ctx context.Context, url string) (*ports.HTTPResponse, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.HTTPResponse), args.Error(1)
}

func (m *MockHTTPClient) Post(ctx context.Context, url string, contentType string, body []byte) (*ports.HTTPResponse, error) {
	args := m.Called(ctx, url, contentType, body)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.HTTPResponse), args.Error(1)
}

func TestRunHTTPCheck_Validation(t *testing.T) {
	tests := []struct {
		name      string
		cfg       config.Config
		errCode   string
		errDetail string
	}{
		{
			name:    "Missing URL",
			cfg:     config.Config{"method": "GET"},
			errCode: "MISSING_URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RunHTTPCheck(context.Background(), tt.cfg)
			require.NoError(t, err)
			assert.True(t, result.IsError())
			assert.Equal(t, tt.errCode, result.Error.Code)
		})
	}
}

func TestRunHTTPCheck_Mock_Success_GET(t *testing.T) {
	mockClient := new(MockHTTPClient)

	expectedResponse := &ports.HTTPResponse{
		StatusCode: 200,
		Body:       []byte("OK"),
		Headers:    map[string][]string{"Content-Type": {"text/plain"}},
	}

	// Match request
	mockClient.On("Do", mock.Anything, mock.MatchedBy(func(req ports.HTTPRequest) bool {
		return req.Method == "GET" && req.URL == "https://example.com"
	})).Return(expectedResponse, nil)

	cfg := config.Config{
		"url":    "https://example.com",
		"method": "GET",
	}

	result, err := RunHTTPCheck(context.Background(), cfg, WithHTTPClient(mockClient))

	require.NoError(t, err)
	assert.True(t, result.IsSuccess())
	assert.Equal(t, 200, result.Data["status_code"])
	assert.Equal(t, "OK", result.Data["body"])
	assert.Greater(t, result.Data["response_time_ms"].(int64), int64(-1))

	mockClient.AssertExpectations(t)
}

func TestRunHTTPCheck_Mock_StatusMismatch(t *testing.T) {
	mockClient := new(MockHTTPClient)

	expectedResponse := &ports.HTTPResponse{
		StatusCode: 404,
		Body:       []byte("Not Found"),
	}

	mockClient.On("Do", mock.Anything, mock.Anything).Return(expectedResponse, nil)

	cfg := config.Config{
		"url":             "https://example.com",
		"expected_status": 200,
	}

	result, err := RunHTTPCheck(context.Background(), cfg, WithHTTPClient(mockClient))

	require.NoError(t, err)
	assert.True(t, result.IsFailure())
	assert.Contains(t, result.Message, "mismatch")
	assert.Equal(t, 404, result.Data["actual_status"])

	mockClient.AssertExpectations(t)
}

func TestRunHTTPCheck_Mock_RequestFailed(t *testing.T) {
	mockClient := new(MockHTTPClient)

	mockClient.On("Do", mock.Anything, mock.Anything).Return(nil, errors.New("timeout"))

	cfg := config.Config{
		"url": "https://example.com",
	}

	result, err := RunHTTPCheck(context.Background(), cfg, WithHTTPClient(mockClient))

	require.Error(t, err)
	assert.True(t, result.IsError())
	mockClient.AssertExpectations(t)
}

func TestRunHTTPCheck_DefaultClient_PanicsOnNative(t *testing.T) {
	cfg := config.Config{"url": "https://example.com"}
	assert.PanicsWithValue(t, "WASM HTTP adapter not available in native build", func() {
		_, _ = RunHTTPCheck(context.Background(), cfg)
	})
}
