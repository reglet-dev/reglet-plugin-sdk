package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/reglet-dev/reglet-sdk/domain/entities"
)

// Service is embedded in service structs to provide metadata.
// Tag format: `name:"service_name" desc:"Service description"`
type Service struct{}

// Request contains the context for a handler invocation.
type Request struct {
	Client interface{} // Plugin-specific client (e.g., *AWSClient)
	Config interface{} // Parsed config struct
	Raw    []byte      // Raw config JSON
}

// HandlerFunc is the signature for operation handlers.
type HandlerFunc func(ctx context.Context, req *Request) (*entities.Result, error)

// MustRegisterService registers a service or panics.
// Use this in init() functions.
func MustRegisterService(plugin *PluginDefinition, svc interface{}) {
	if err := RegisterService(plugin, svc); err != nil {
		panic(fmt.Sprintf("failed to register service: %v", err))
	}
}

// RegisterService registers all operations from a service struct.
func RegisterService(plugin *PluginDefinition, svc interface{}) error {
	svcType := reflect.TypeOf(svc)
	svcValue := reflect.ValueOf(svc)

	// Must be a pointer to struct
	if svcType.Kind() != reflect.Ptr || svcType.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("service must be a pointer to struct, got %T", svc)
	}

	structType := svcType.Elem()

	// Find embedded Service field and extract metadata
	serviceName, serviceDesc, err := extractServiceMetadata(structType)
	if err != nil {
		return err
	}

	// Find all Op fields and their descriptions
	ops, err := extractOperations(structType)
	if err != nil {
		return err
	}

	// Match operations to methods and register
	for _, op := range ops {
		method := svcValue.MethodByName(op.methodName)
		if !method.IsValid() {
			return fmt.Errorf("service %s: no method %s for operation %s (field %s)",
				serviceName, op.methodName, op.name, op.fieldName)
		}

		var handler HandlerFunc
		var wrapErr error

		if op.isTyped {
			// Typed handler: func(ctx, *Input) (*Output, error)
			handler, wrapErr = wrapTypedMethod(method, op.inputType, op.outputType)
		} else {
			// Legacy handler: func(ctx, *Request) (*Result, error)
			handler, wrapErr = wrapLegacyMethod(method)
		}

		if wrapErr != nil {
			return fmt.Errorf("service %s, operation %s: %w",
				serviceName, op.name, wrapErr)
		}

		plugin.RegisterHandler(
			serviceName, serviceDesc,
			op.name, op.description,
			handler,
			op.inputType,
			op.outputType,
			op.examples,
		)
	}

	return nil
}

// extractServiceMetadata finds the embedded Service field and parses its tags.
func extractServiceMetadata(t reflect.Type) (name, desc string, err error) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Type == reflect.TypeOf(Service{}) {
			tag := field.Tag
			name = tag.Get("name")
			desc = tag.Get("desc")
			if name == "" {
				return "", "", fmt.Errorf("Service field missing 'name' tag")
			}
			return name, desc, nil
		}
	}
	return "", "", fmt.Errorf("struct must embed plugin.Service")
}

// opInfo holds operation metadata extracted from struct fields.
type opInfo struct {
	fieldName   string // PascalCase field name
	methodName  string // Method name to invoke
	name        string // snake_case operation name
	description string
	isTyped     bool         // true if Op[I,O], false if legacy Op
	inputType   reflect.Type // nil for legacy
	outputType  reflect.Type // nil for legacy
	examples    []any        // nil for legacy
}

// extractOperations finds all Op fields and extracts their metadata.
func extractOperations(t reflect.Type) ([]opInfo, error) {
	var ops []opInfo

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Check if field implements operation marker interface
		if !isOpType(field.Type) {
			continue
		}

		methodName := field.Tag.Get("method")
		if methodName == "" {
			// Default to field name if no method tag
			methodName = field.Name
		}

		op := opInfo{
			fieldName:   field.Name,
			methodName:  methodName,
			name:        toSnakeCase(field.Name),
			description: field.Tag.Get("desc"),
		}

		// Check if typed (Op[I,O]) or legacy (Op)
		if isTypedOp(field.Type) {
			typeInfo, ok := getOpTypeInfo(field.Name)
			if !ok {
				// Require registration for typed ops
				return nil, fmt.Errorf(
					"Op field %s not registered - call plugin.RegisterOp[I,O](%q, ...) in init() before MustRegisterService",
					field.Name, field.Name)
			}
			op.isTyped = true
			op.inputType = typeInfo.inputType
			op.outputType = typeInfo.outputType
			op.examples = typeInfo.examples
		}

		ops = append(ops, op)
	}

	if len(ops) == 0 {
		return nil, fmt.Errorf("service has no operations (no Op fields)")
	}

	return ops, nil
}

// isOpType checks if a type is the generic Op[I,O].
func isOpType(t reflect.Type) bool {
	return t.Implements(reflect.TypeOf((*operation)(nil)).Elem())
}

// isTypedOp checks if a type is the generic Op[I,O].
// Since we only have Op[I,O], this is effectively always true for fields passing isOpType.
func isTypedOp(t reflect.Type) bool {
	return strings.HasPrefix(t.Name(), "Op[")
}

// wrapTypedMethod wraps a typed handler as HandlerFunc.
// Expected signature: func(ctx context.Context, in *I) (*O, error)
func wrapTypedMethod(method reflect.Value, inputType, outputType reflect.Type) (HandlerFunc, error) {
	methodType := method.Type()

	// Validate signature
	if methodType.NumIn() != 2 || methodType.NumOut() != 2 {
		return nil, fmt.Errorf("typed handler must have signature (context.Context, *Input) (*Output, error)")
	}

	// Validate context parameter
	ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if !methodType.In(0).Implements(ctxType) {
		return nil, fmt.Errorf("first parameter must be context.Context")
	}

	// Validate input parameter (must be pointer to inputType)
	expectedInputPtr := reflect.PointerTo(inputType)
	if methodType.In(1) != expectedInputPtr {
		return nil, fmt.Errorf("second parameter must be *%s, got %s", inputType.Name(), methodType.In(1))
	}

	// Validate output parameter (must be pointer to outputType)
	expectedOutputPtr := reflect.PointerTo(outputType)
	if methodType.Out(0) != expectedOutputPtr {
		return nil, fmt.Errorf("first return must be *%s, got %s", outputType.Name(), methodType.Out(0))
	}

	// Validate error return
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if !methodType.Out(1).Implements(errorType) {
		return nil, fmt.Errorf("second return must be error")
	}

	return func(ctx context.Context, req *Request) (*entities.Result, error) {
		// 1. Inject client into context
		ctx = WithClient(ctx, req.Client)

		// 2. Parse config JSON into input type
		inputPtr := reflect.New(inputType)
		if len(req.Raw) > 0 {
			if err := json.Unmarshal(req.Raw, inputPtr.Interface()); err != nil {
				return entities.ResultErrorPtr("config", fmt.Sprintf("failed to parse config: %v", err)), nil
			}
		}

		// 3. Call the typed handler
		args := []reflect.Value{
			reflect.ValueOf(ctx),
			inputPtr,
		}
		results := method.Call(args)

		// 4. Handle error return
		if !results[1].IsNil() {
			err := results[1].Interface().(error)
			return entities.ResultErrorPtr("execution", err.Error()), nil
		}

		// 5. Handle nil output
		if results[0].IsNil() {
			return entities.ResultSuccessPtr("ok", nil), nil
		}

		// 6. Convert output struct to map[string]any for Result.Data
		output := results[0].Interface()
		data, err := structToMap(output)
		if err != nil {
			return entities.ResultErrorPtr("output", fmt.Sprintf("failed to serialize output: %v", err)), nil
		}

		return entities.ResultSuccessPtr("ok", data), nil
	}, nil
}

// wrapLegacyMethod wraps a legacy handler (existing signature).
func wrapLegacyMethod(method reflect.Value) (HandlerFunc, error) {
	methodType := method.Type()

	if methodType.NumIn() != 2 || methodType.NumOut() != 2 {
		return nil, fmt.Errorf("method must have signature (context.Context, *Request) (*entities.Result, error)")
	}

	ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
	reqType := reflect.TypeOf((*Request)(nil))
	if !methodType.In(0).Implements(ctxType) {
		return nil, fmt.Errorf("first parameter must be context.Context")
	}
	if methodType.In(1) != reqType {
		return nil, fmt.Errorf("second parameter must be *plugin.Request")
	}

	resultType := reflect.TypeOf((*entities.Result)(nil))
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	if methodType.Out(0) != resultType {
		return nil, fmt.Errorf("first return value must be *entities.Result")
	}
	if !methodType.Out(1).Implements(errorType) {
		return nil, fmt.Errorf("second return value must be error")
	}

	return func(ctx context.Context, req *Request) (*entities.Result, error) {
		args := []reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(req),
		}
		results := method.Call(args)

		var result *entities.Result
		if !results[0].IsNil() {
			result = results[0].Interface().(*entities.Result)
		}

		var err error
		if !results[1].IsNil() {
			err = results[1].Interface().(error)
		}

		return result, err
	}, nil
}

// structToMap converts a struct to map[string]any via JSON round-trip.
func structToMap(v any) (map[string]any, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(bytes, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// toSnakeCase converts PascalCase to snake_case.
var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
