package plugin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testConfig struct {
	Name string `json:"name" jsonschema:"required"`
}

func TestManifest_WithTypedOp(t *testing.T) {
	clearOpRegistry()

	// Register op with types and examples
	RegisterOp[testInput, testOutput]("TestOp",
		Example[testInput, testOutput]{
			Name:           "basic",
			Description:    "Basic test",
			Input:          testInput{Name: "test"},
			ExpectedOutput: &testOutput{Result: "ok"},
		},
	)

	plugin := DefinePlugin(PluginDef{
		Name:    "test",
		Version: "1.0.0",
		Config:  &testConfig{},
	})

	// Simulate service registration with type info
	typeInfo, _ := getOpTypeInfo("TestOp")
	plugin.RegisterHandler(
		"test", "Test service",
		"test_op", "Test operation",
		func(ctx context.Context, req *Request) (*entities.Result, error) {
			return nil, nil
		},
		typeInfo.inputType,
		typeInfo.outputType,
		typeInfo.examples,
	)

	manifest := plugin.Manifest()

	// Verify service exists
	svc, ok := manifest.Services["test"]
	require.True(t, ok)
	require.Len(t, svc.Operations, 1)

	op := svc.Operations[0]
	assert.Equal(t, "test_op", op.Name)

	// Verify input fields
	assert.Contains(t, op.InputFields, "name")

	// Verify output schema
	require.NotNil(t, op.OutputSchema)
	var schema map[string]any
	json.Unmarshal(op.OutputSchema, &schema)
	props := schema["properties"].(map[string]any)
	assert.Contains(t, props, "result")

	// Verify examples
	require.Len(t, op.Examples, 1)
	assert.Equal(t, "basic", op.Examples[0].Name)
	assert.NotEmpty(t, op.Examples[0].Input)
	assert.NotEmpty(t, op.Examples[0].ExpectedOutput)
}
