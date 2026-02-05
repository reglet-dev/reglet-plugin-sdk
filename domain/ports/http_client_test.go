package ports

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockHTTPClient is a mock implementation of HTTPClient for testing.
type MockHTTPClient struct {
	DoFunc   func(ctx context.Context, req HTTPRequest) (*HTTPResponse, error)
	GetFunc  func(ctx context.Context, url string) (*HTTPResponse, error)
	PostFunc func(ctx context.Context, url string, contentType string, body []byte) (*HTTPResponse, error)
}

func (m *MockHTTPClient) Do(ctx context.Context, req HTTPRequest) (*HTTPResponse, error) {
	if m.DoFunc != nil {
		return m.DoFunc(ctx, req)
	}
	return &HTTPResponse{
		StatusCode: 200,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       []byte(`{"status":"ok"}`),
	}, nil
}

func (m *MockHTTPClient) Get(ctx context.Context, url string) (*HTTPResponse, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, url)
	}
	return m.Do(ctx, HTTPRequest{Method: "GET", URL: url})
}

func (m *MockHTTPClient) Post(ctx context.Context, url string, contentType string, body []byte) (*HTTPResponse, error) {
	if m.PostFunc != nil {
		return m.PostFunc(ctx, url, contentType, body)
	}
	return m.Do(ctx, HTTPRequest{
		Method:  "POST",
		URL:     url,
		Headers: map[string]string{"Content-Type": contentType},
		Body:    body,
	})
}

// Compile-time interface check
var _ HTTPClient = (*MockHTTPClient)(nil)

func TestMockHTTPClient_ImplementsInterface(t *testing.T) {
	// Verify that MockHTTPClient implements HTTPClient interface
	var client HTTPClient = &MockHTTPClient{}
	require.NotNil(t, client)
}

func TestMockHTTPClient_Do(t *testing.T) {
	ctx := context.Background()

	t.Run("default behavior", func(t *testing.T) {
		mock := &MockHTTPClient{}
		resp, err := mock.Do(ctx, HTTPRequest{
			Method: "GET",
			URL:    "https://api.example.com/status",
		})

		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Contains(t, string(resp.Body), "status")
	})

	t.Run("custom behavior", func(t *testing.T) {
		mock := &MockHTTPClient{
			DoFunc: func(ctx context.Context, req HTTPRequest) (*HTTPResponse, error) {
				return &HTTPResponse{
					StatusCode: 404,
					Body:       []byte(`{"error":"not found"}`),
				}, nil
			},
		}

		resp, err := mock.Do(ctx, HTTPRequest{
			Method: "GET",
			URL:    "https://api.example.com/missing",
		})

		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode)
		assert.Contains(t, string(resp.Body), "not found")
	})

	t.Run("error behavior", func(t *testing.T) {
		expectedErr := errors.New("connection refused")
		mock := &MockHTTPClient{
			DoFunc: func(ctx context.Context, req HTTPRequest) (*HTTPResponse, error) {
				return nil, expectedErr
			},
		}

		resp, err := mock.Do(ctx, HTTPRequest{Method: "GET", URL: "https://unreachable.example.com"})

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, expectedErr, err)
	})
}

func TestMockHTTPClient_Get(t *testing.T) {
	ctx := context.Background()
	mock := &MockHTTPClient{}

	resp, err := mock.Get(ctx, "https://api.example.com/data")

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestMockHTTPClient_Post(t *testing.T) {
	ctx := context.Background()

	t.Run("default behavior", func(t *testing.T) {
		mock := &MockHTTPClient{}
		resp, err := mock.Post(ctx, "https://api.example.com/data", "application/json", []byte(`{"key":"value"}`))

		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("custom behavior", func(t *testing.T) {
		mock := &MockHTTPClient{
			PostFunc: func(ctx context.Context, url string, contentType string, body []byte) (*HTTPResponse, error) {
				return &HTTPResponse{
					StatusCode: 201,
					Body:       []byte(`{"id":123}`),
				}, nil
			},
		}

		resp, err := mock.Post(ctx, "https://api.example.com/create", "application/json", []byte(`{"name":"test"}`))

		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)
		assert.Contains(t, string(resp.Body), "123")
	})
}
