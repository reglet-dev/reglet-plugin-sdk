package plugin

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types
type ResolveInput struct {
	Hostname string `json:"hostname"`
}

type ResolveOutput struct {
	Records []string `json:"records"`
}

// Test service with typed Op
type TypedDNSService struct {
	Service `name:"dns" desc:"DNS resolution"`

	Resolve Op[ResolveInput, ResolveOutput] `desc:"Resolve hostname" method:"ResolveHandler"`
}

func (s *TypedDNSService) ResolveHandler(ctx context.Context, in *ResolveInput) (*ResolveOutput, error) {
	// Get mock client from context
	client := GetClient[*mockResolver](ctx)
	records := client.Lookup(in.Hostname)
	return &ResolveOutput{Records: records}, nil
}

type mockResolver struct {
	records map[string][]string
}

func (m *mockResolver) Lookup(host string) []string {
	return m.records[host]
}

func TestTypedServiceRegistration(t *testing.T) {
	clearOpRegistry()

	// 1. Register op types
	RegisterOp[ResolveInput, ResolveOutput]("Resolve",
		Example[ResolveInput, ResolveOutput]{
			Name:  "basic",
			Input: ResolveInput{Hostname: "example.com"},
		},
	)

	// 2. Create plugin
	plugin := DefinePlugin(PluginDef{
		Name:    "dns",
		Version: "1.0.0",
	})

	// 3. Register service
	err := RegisterService(plugin, &TypedDNSService{})
	require.NoError(t, err)

	// 4. Get handler and execute
	handler, ok := plugin.GetHandler("dns", "resolve")
	require.True(t, ok)

	mockClient := &mockResolver{
		records: map[string][]string{
			"example.com": {"1.2.3.4"},
		},
	}

	req := &Request{
		Client: mockClient,
		Raw:    []byte(`{"hostname": "example.com"}`),
	}

	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, entities.ResultStatusSuccess, result.Status)
	assert.Equal(t, []any{"1.2.3.4"}, result.Data["records"])
}

func TestTypedHandler_Error(t *testing.T) {
	clearOpRegistry()

	// Service that returns error
	type ErrorService struct {
		Service `name:"err" desc:"Error test"`
		Fail    Op[testInput, testOutput] `desc:"Always fails"`
	}

	handlerFunc := func(ctx context.Context, in *testInput) (*testOutput, error) {
		return nil, errors.New("intentional error")
	}

	method := reflect.ValueOf(handlerFunc)
	wrapped, err := wrapTypedMethod(method, reflect.TypeOf(testInput{}), reflect.TypeOf(testOutput{}))
	require.NoError(t, err)

	req := &Request{
		Client: nil,
		Raw:    []byte(`{"name": "test"}`),
	}

	result, err := wrapped(context.Background(), req)
	require.NoError(t, err) // Handler errors are wrapped in Result
	assert.Equal(t, entities.ResultStatusError, result.Status)
	assert.Contains(t, result.Error.Message, "intentional error")
}

func TestTypedHandler_InvalidJSON(t *testing.T) {
	clearOpRegistry()

	handlerFunc := func(ctx context.Context, in *testInput) (*testOutput, error) {
		return &testOutput{Result: "ok"}, nil
	}

	method := reflect.ValueOf(handlerFunc)
	wrapped, err := wrapTypedMethod(method, reflect.TypeOf(testInput{}), reflect.TypeOf(testOutput{}))
	require.NoError(t, err)

	req := &Request{
		Client: nil,
		Raw:    []byte(`{invalid json}`),
	}

	result, err := wrapped(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, entities.ResultStatusError, result.Status)
	assert.Contains(t, result.Error.Message, "failed to parse config")
}

func TestOpNotRegistered(t *testing.T) {
	clearOpRegistry()

	type UnregisteredService struct {
		Service `name:"unreg" desc:"Unregistered"`
		Op1     Op[testInput, testOutput] `desc:"Not registered"`
	}

	plugin := DefinePlugin(PluginDef{Name: "unreg", Version: "1.0.0"})

	err := RegisterService(plugin, &UnregisteredService{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}
