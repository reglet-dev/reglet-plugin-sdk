package plugin_test

import (
	"context"
	"testing"

	"github.com/reglet-dev/reglet-sdk/application/plugin"
	"github.com/reglet-dev/reglet-sdk/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestService is a struct with methods to be registered as operations.
type TestService struct {
	plugin.Service `name:"test_service" desc:"Test Service"`
	EchoOp         plugin.Op[EchoRequest, EchoResponse] `desc:"Echoes the message back" method:"Echo"`
	AddOp          plugin.Op[AddRequest, AddResponse]   `desc:"Adds two numbers" method:"Add"`
}

type EchoRequest struct {
	Message string `json:"message"`
}

type EchoResponse struct {
	Reply string `json:"reply"`
}

type AddRequest struct {
	A int `json:"a"`
	B int `json:"b"`
}

type AddResponse struct {
	Sum int `json:"sum"`
}

// Echo is a typed operation.
func (s *TestService) Echo(ctx context.Context, req *EchoRequest) (*EchoResponse, error) {
	return &EchoResponse{Reply: req.Message}, nil
}

// Add is a typed operation.
func (s *TestService) Add(ctx context.Context, req *AddRequest) (*AddResponse, error) {
	return &AddResponse{Sum: req.A + req.B}, nil
}

func TestServiceRegistration(t *testing.T) {
	// 0. Register Ops (Required for typed ops)
	plugin.RegisterOp[EchoRequest, EchoResponse]("EchoOp")
	plugin.RegisterOp[AddRequest, AddResponse]("AddOp")

	// 1. Define plugin
	def := plugin.DefinePlugin(plugin.PluginDef{
		Name:    "test-plugin",
		Version: "1.0.0",
	})

	// 2. Register the service
	svc := &TestService{}
	err := plugin.RegisterService(def, svc)
	require.NoError(t, err)

	// 3. Generate Manifest
	manifest := def.Manifest()
	assert.NotNil(t, manifest)

	// Verify Service Manifest
	svcManifest, ok := manifest.Services["test_service"]
	assert.True(t, ok, "service 'test_service' not found")

	// Check Operations
	foundEcho := false
	for _, op := range svcManifest.Operations {
		if op.Name == "echo_op" { // Converted from field name 'EchoOp'
			foundEcho = true
			assert.Equal(t, "Echoes the message back", op.Description)
			assert.Contains(t, op.InputFields, "message")
		}
	}
	assert.True(t, foundEcho, "echo_op operation not found")

	// 5. Verify Handler Registration
	// Field EchoOp -> name echo_op
	handlerEcho, ok := def.GetHandler("test_service", "echo_op")
	assert.True(t, ok)
	assert.NotNil(t, handlerEcho)

	// 6. Execute Handler (Echo)
	reqBody := []byte(`{"message": "hello"}`)
	req := &plugin.Request{
		Raw: reqBody,
	}

	res, err := handlerEcho(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, entities.ResultStatusSuccess, res.Status)
	assert.Equal(t, "ok", res.Message)

	// Verify output data
	// Result.Data is map[string]any
	assert.Equal(t, "hello", res.Data["reply"])
}
