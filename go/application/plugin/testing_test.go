package plugin

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/stretchr/testify/assert"
)

type exampleTestInput struct {
	Value string `json:"value"`
}

type exampleTestOutput struct {
	Result string `json:"result"`
	Count  int    `json:"count"`
}

func TestGenerateExampleTests(t *testing.T) {
	t.Run("GenerateExampleTests", func(t *testing.T) {
		plugin := DefinePlugin(PluginDef{Name: "test", Version: "1.0.0"})

		input := exampleTestInput{Value: "test"}
		output := exampleTestOutput{Result: "ok", Count: 1}

		// Mock examples
		examples := []any{
			Example[exampleTestInput, exampleTestOutput]{
				Name:           "success",
				Description:    "Successful case",
				Input:          input,
				ExpectedOutput: &output,
			},
			Example[exampleTestInput, exampleTestOutput]{
				Name:          "error",
				Description:   "Error case",
				Input:         exampleTestInput{Value: "fail"},
				ExpectedError: "intentional failure",
			},
		}

		// Register handler
		plugin.RegisterHandler(
			"test", "Test service",
			"test_op", "Test operation",
			func(ctx context.Context, req *Request) (*entities.Result, error) {
				var input exampleTestInput
				if err := json.Unmarshal(req.Raw, &input); err != nil {
					return nil, err
				}

				if input.Value == "fail" {
					return entities.ResultErrorPtr("test", "intentional failure"), nil
				}

				return entities.ResultSuccessPtr("ok", map[string]any{
					"result": "ok",
					"count":  1,
				}), nil
			},
			reflect.TypeOf(exampleTestInput{}),
			reflect.TypeOf(exampleTestOutput{}),
			examples,
		)

		GenerateExampleTests(t, plugin, nil)
	})

	t.Run("GenerateExampleTestsWithConfig", func(t *testing.T) {
		plugin := DefinePlugin(PluginDef{Name: "test", Version: "1.0.0"})

		input := exampleTestInput{Value: "test"}
		output := exampleTestOutput{Result: "ok", Count: 1}

		examples := []any{
			Example[exampleTestInput, exampleTestOutput]{
				Name:           "success", // Should be run
				Input:          input,
				ExpectedOutput: &output,
			},
			Example[exampleTestInput, exampleTestOutput]{
				Name:          "skipped", // Should be skipped
				Input:         exampleTestInput{Value: "fail"},
				ExpectedError: "intentional failure",
			},
		}

		plugin.RegisterHandler(
			"test", "Test service",
			"test_op_config", "Test operation with config",
			func(ctx context.Context, req *Request) (*entities.Result, error) {
				var input exampleTestInput
				if err := json.Unmarshal(req.Raw, &input); err != nil {
					return nil, err
				}

				return entities.ResultSuccessPtr("ok", map[string]any{
					"result": "ok",
					"count":  1,
				}), nil
			},
			reflect.TypeOf(exampleTestInput{}),
			reflect.TypeOf(exampleTestOutput{}),
			examples,
		)

		// Verification variable
		clientCalled := false

		config := ExampleTestConfig{
			SkipExamples: []string{"skipped"},
			MockClientFactory: func(name string) any {
				clientCalled = true
				return "mock_client"
			},
		}

		// Run with config
		GenerateExampleTestsWithConfig(t, plugin, nil, config)

		assert.True(t, clientCalled, "MockClientFactory should be called for non-skipped tests")
	})
}

func TestDeepEqual(t *testing.T) {
	tests := []struct {
		name     string
		expected any
		actual   any
		want     bool
	}{
		{"equal strings", "a", "a", true},
		{"different strings", "a", "b", false},
		{"equal ints as float64", 1, float64(1), true},
		{"equal slices", []any{"a", "b"}, []any{"a", "b"}, true},
		{"different slices", []any{"a"}, []any{"b"}, false},
		{"equal maps", map[string]any{"k": "v"}, map[string]any{"k": "v"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deepEqual(tt.expected, tt.actual)
			assert.Equal(t, tt.want, got)
		})
	}
}
