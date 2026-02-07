// Package plugintest provides a test harness for Reglet plugins.
package plugintest

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/reglet-dev/reglet-abi/hostfunc"
	"github.com/reglet-dev/reglet-plugin-sdk/application/plugin"
	"github.com/reglet-dev/reglet-plugin-sdk/domain/entities"
)

// TestCase defines a test case for a plugin.
type TestCase struct {
	Name     string
	Config   map[string]any
	Validate func(t *testing.T, r *entities.Result)
}

// RunPluginTests runs a suite of tests against a plugin.
func RunPluginTests(t *testing.T, p plugin.Plugin, tests []TestCase) {
	t.Helper()

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			var configBytes []byte
			var err error
			if tc.Config != nil {
				configBytes, err = json.Marshal(tc.Config)
				if err != nil {
					t.Fatalf("failed to marshal config: %v", err)
				}
			}

			// Invoke plugin Check
			result, err := p.Check(context.Background(), configBytes)
			// Handle execution error (e.g. panic or unhandled error) by treating it as an Error Result
			if err != nil {
				errDetail := &hostfunc.ErrorDetail{
					Message: err.Error(),
					Type:    "execution_error",
				}
				res := entities.ResultError(errDetail)
				result = &res
			}

			if tc.Validate != nil {
				tc.Validate(t, result)
			}
		})
	}
}

// AssertSuccess asserts the result is a success.
func AssertSuccess(t *testing.T, r *entities.Result) {
	t.Helper()
	if r.Status != entities.ResultStatusSuccess {
		t.Errorf("expected success, got %s: %s", r.Status, r.Message)
	}
}

// AssertFailure asserts the result is a failure.
func AssertFailure(t *testing.T, r *entities.Result) {
	t.Helper()
	if r.Status != entities.ResultStatusFailure {
		t.Errorf("expected failure, got %s: %s", r.Status, r.Message)
	}
}

// AssertDataField asserts a specific field in Data matches expected value.
func AssertDataField(t *testing.T, r *entities.Result, key string, expected any) {
	t.Helper()
	val, ok := r.Data[key]
	if !ok {
		t.Errorf("missing data field %q", key)
		return
	}

	// Handle basic numeric conversion for JSON unmarshaled data
	if expectedNum, ok := toFloat64(expected); ok {
		if actualNum, ok := toFloat64(val); ok {
			if expectedNum != actualNum {
				t.Errorf("field %q: expected %v, got %v", key, expected, val)
			}
			return
		}
	}

	if !reflect.DeepEqual(val, expected) {
		t.Errorf("field %q: expected %v, got %v", key, expected, val)
	}
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	case float32:
		return float64(n), true
	default:
		return 0, false
	}
}
