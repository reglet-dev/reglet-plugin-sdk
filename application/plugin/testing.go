package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/reglet-dev/reglet-plugin-sdk/domain/entities"
)

// GenerateExampleTests creates table-driven tests from registered operation examples.
// Call this in plugin test files to ensure examples stay valid.
//
// Example usage:
//
//	func TestExamples(t *testing.T) {
//	    mockClient := &mockDNSResolver{...}
//	    plugin.GenerateExampleTests(t, core.Plugin, mockClient)
//	}
func GenerateExampleTests(t *testing.T, plugin *PluginDefinition, mockClient any) {
	t.Helper()

	manifest := plugin.Manifest()

	for svcName, svc := range manifest.Services {
		for _, op := range svc.Operations {
			for _, ex := range op.Examples {
				testName := fmt.Sprintf("%s/%s/%s", svcName, op.Name, ex.Name)

				t.Run(testName, func(t *testing.T) {
					runExampleTest(t, plugin, svcName, op.Name, ex, mockClient)
				})
			}
		}
	}
}

// runExampleTest executes a single example test.
func runExampleTest(
	t *testing.T,
	plugin *PluginDefinition,
	svcName, opName string,
	ex entities.OperationExample,
	mockClient any,
) {
	t.Helper()

	// Get handler
	handler, ok := plugin.GetHandler(svcName, opName)
	if !ok {
		t.Fatalf("handler not found: %s/%s", svcName, opName)
	}

	// Build request from example input
	req := &Request{
		Client: mockClient,
		Raw:    ex.Input,
	}

	// Execute handler
	result, err := handler(context.Background(), req)

	// Check expected error case
	if ex.ExpectedError != "" {
		if err != nil {
			if !strings.Contains(err.Error(), ex.ExpectedError) {
				t.Errorf("expected error containing %q, got %q", ex.ExpectedError, err.Error())
			}
			return
		}
		// Error might be in Result.Error
		if result != nil && result.Status == entities.ResultStatusError {
			if result.Error != nil && strings.Contains(result.Error.Message, ex.ExpectedError) {
				return // Expected error found
			}
		}
		t.Errorf("expected error containing %q, got success", ex.ExpectedError)
		return
	}

	// Check success case
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.Status == entities.ResultStatusError {
		t.Fatalf("unexpected error result: %s", result.Error.Message)
	}

	// If expected output provided, verify it matches
	if len(ex.ExpectedOutput) > 0 {
		var expected map[string]any
		if err := json.Unmarshal(ex.ExpectedOutput, &expected); err != nil {
			t.Fatalf("failed to parse expected output: %v", err)
		}

		verifyOutput(t, expected, result.Data)
	}
}

// verifyOutput compares expected fields against actual data.
// Only checks fields present in expected (allows extra fields in actual).
func verifyOutput(t *testing.T, expected, actual map[string]any) {
	t.Helper()

	for key, expectedVal := range expected {
		actualVal, ok := actual[key]
		if !ok {
			t.Errorf("missing field %q in output", key)
			continue
		}

		if !deepEqual(expectedVal, actualVal) {
			t.Errorf("field %q: expected %v (%T), got %v (%T)",
				key, expectedVal, expectedVal, actualVal, actualVal)
		}
	}
}

// deepEqual compares two values, handling JSON number conversions.
func deepEqual(expected, actual any) bool {
	// Handle slice comparison
	expectedSlice, expectedIsSlice := expected.([]any)
	actualSlice, actualIsSlice := actual.([]any)
	if expectedIsSlice && actualIsSlice {
		if len(expectedSlice) != len(actualSlice) {
			return false
		}
		for i := range expectedSlice {
			if !deepEqual(expectedSlice[i], actualSlice[i]) {
				return false
			}
		}
		return true
	}

	// Handle map comparison
	expectedMap, expectedIsMap := expected.(map[string]any)
	actualMap, actualIsMap := actual.(map[string]any)
	if expectedIsMap && actualIsMap {
		for k, v := range expectedMap {
			av, ok := actualMap[k]
			if !ok || !deepEqual(v, av) {
				return false
			}
		}
		return true
	}

	// Handle numeric comparison (JSON unmarshals numbers as float64)
	if expectedNum, ok := toFloat64(expected); ok {
		if actualNum, ok := toFloat64(actual); ok {
			return expectedNum == actualNum
		}
	}

	// Direct comparison
	return reflect.DeepEqual(expected, actual)
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	default:
		return 0, false
	}
}

// ExampleTestConfig allows customizing example test behavior.
type ExampleTestConfig struct {
	// SkipExamples lists example names to skip (e.g., "error_case")
	SkipExamples []string

	// MockClientFactory creates a mock client for each test
	// If nil, uses the client passed to GenerateExampleTests
	MockClientFactory func(exampleName string) any
}

// GenerateExampleTestsWithConfig creates tests with custom configuration.
func GenerateExampleTestsWithConfig(
	t *testing.T,
	plugin *PluginDefinition,
	defaultClient any,
	config ExampleTestConfig,
) {
	t.Helper()

	manifest := plugin.Manifest()
	skipSet := make(map[string]bool)
	for _, name := range config.SkipExamples {
		skipSet[name] = true
	}

	for svcName, svc := range manifest.Services {
		for _, op := range svc.Operations {
			for _, ex := range op.Examples {
				if skipSet[ex.Name] {
					continue
				}

				testName := fmt.Sprintf("%s/%s/%s", svcName, op.Name, ex.Name)

				t.Run(testName, func(t *testing.T) {
					client := defaultClient
					if config.MockClientFactory != nil {
						client = config.MockClientFactory(ex.Name)
					}
					runExampleTest(t, plugin, svcName, op.Name, ex, client)
				})
			}
		}
	}
}
