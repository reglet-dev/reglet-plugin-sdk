package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResultSuccess(t *testing.T) {
	data := map[string]any{"ip": "192.0.2.1"}
	result := ResultSuccess("DNS lookup successful", data)

	assert.Equal(t, ResultStatusSuccess, result.Status)
	assert.Equal(t, "DNS lookup successful", result.Message)
	assert.Equal(t, data, result.Data)
	assert.True(t, result.IsSuccess())
	assert.False(t, result.IsFailure())
	assert.False(t, result.IsError())
}

func TestResultFailure(t *testing.T) {
	data := map[string]any{"expected": "200", "actual": "404"}
	result := ResultFailure("HTTP status mismatch", data)

	assert.Equal(t, ResultStatusFailure, result.Status)
	assert.Equal(t, "HTTP status mismatch", result.Message)
	assert.Equal(t, data, result.Data)
	assert.False(t, result.IsSuccess())
	assert.True(t, result.IsFailure())
	assert.False(t, result.IsError())
}

func TestResultError(t *testing.T) {
	err := NewErrorDetail("timeout", "DNS lookup timed out").WithCode("DNS_TIMEOUT")
	result := ResultError(err)

	assert.Equal(t, ResultStatusError, result.Status)
	assert.Equal(t, "DNS lookup timed out", result.Message)
	require.NotNil(t, result.Error)
	assert.Equal(t, "DNS_TIMEOUT", result.Error.Code)
	assert.Equal(t, "timeout", result.Error.Type)
	assert.False(t, result.IsSuccess())
	assert.False(t, result.IsFailure())
	assert.True(t, result.IsError())
}

func TestResult_WithMetadata(t *testing.T) {
	start := time.Now()
	end := start.Add(100 * time.Millisecond)
	meta := NewRunMetadata(start, end)

	result := ResultSuccess("test", nil).WithMetadata(meta)

	require.NotNil(t, result.Metadata)
	assert.Equal(t, start, result.Metadata.StartTime)
	assert.Equal(t, end, result.Metadata.EndTime)
	assert.Equal(t, 100*time.Millisecond, result.Metadata.Duration)
}

func TestNewRunMetadata(t *testing.T) {
	start := time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 20, 10, 0, 5, 0, time.UTC)

	meta := NewRunMetadata(start, end)

	assert.Equal(t, start, meta.StartTime)
	assert.Equal(t, end, meta.EndTime)
	assert.Equal(t, 5*time.Second, meta.Duration)
}

func TestErrorDetail_Error(t *testing.T) {
	err := NewErrorDetail("test_error", "something went wrong")

	// Error() method formats as "type: message [code]" (or just "message" for type "internal")
	assert.Equal(t, "test_error: something went wrong", err.Error())
}

func TestErrorDetail_WithDetails(t *testing.T) {
	details := map[string]any{"retry_count": 3}
	err := NewErrorDetail("RETRY_EXCEEDED", "max retries").WithDetails(details)

	assert.Equal(t, details, err.Details)
}
