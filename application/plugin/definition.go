package plugin

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"

	"github.com/reglet-dev/reglet-plugin-sdk/application/schema"
	"github.com/reglet-dev/reglet-plugin-sdk/domain/entities"
)

// PluginDef defines plugin identity and configuration.
type PluginDef struct {
	Name         string
	Version      string
	Description  string
	Config       interface{} // Struct for schema generation
	Capabilities entities.GrantSet
}

// PluginDefinition holds the parsed plugin definition and registered services.
type PluginDefinition struct {
	services     map[string]*serviceEntry
	def          PluginDef
	configSchema json.RawMessage
	mu           sync.RWMutex
}

// serviceEntry holds a registered service.
type serviceEntry struct {
	operations  map[string]*operationEntry
	name        string
	description string
}

// operationEntry holds a registered operation.
type operationEntry struct {
	handler     HandlerFunc
	name        string
	description string
	// NEW: Type info for schema generation
	inputType  reflect.Type
	outputType reflect.Type
	examples   []any
}

// DefinePlugin creates a new plugin definition.
// Call this once at package level in your plugin.
func DefinePlugin(def PluginDef) *PluginDefinition {
	var configSchema []byte
	var err error
	if def.Config != nil {
		configSchema, err = schema.GenerateSchema(def.Config)
		if err != nil {
			panic("failed to generate config schema: " + err.Error())
		}
	} else {
		// Empty schema or default
		configSchema = []byte("{}")
	}

	return &PluginDefinition{
		def:          def,
		configSchema: configSchema,
		services:     make(map[string]*serviceEntry),
	}
}

// Manifest returns the complete plugin manifest.
func (p *PluginDefinition) Manifest() *entities.Manifest {
	p.mu.RLock()
	defer p.mu.RUnlock()

	services := make(map[string]entities.ServiceManifest)
	for name, svc := range p.services {
		ops := make([]entities.OperationManifest, 0, len(svc.operations))
		for _, op := range svc.operations {
			opManifest := entities.OperationManifest{
				Name:        op.name,
				Description: op.description,
			}

			// Generate input fields from input type (if available)
			if op.inputType != nil {
				opManifest.InputFields = extractFieldNames(op.inputType)
			}

			// Generate output schema from output type (if available)
			if op.outputType != nil {
				// We create a new instance of the type to ensure it works with GenerateSchema
				// reflect.New returns a pointer to the type
				val := reflect.New(op.outputType).Elem().Interface()
				outputSchema, err := schema.GenerateSchema(val)
				if err == nil {
					opManifest.OutputSchema = outputSchema
				}
			}

			// Convert examples to manifest format
			if len(op.examples) > 0 {
				opManifest.Examples = convertExamplesToManifest(op.examples)
			}

			ops = append(ops, opManifest)
		}
		services[name] = entities.ServiceManifest{
			Name:        svc.name,
			Description: svc.description,
			Operations:  ops,
		}
	}

	return &entities.Manifest{
		Name:         p.def.Name,
		Version:      p.def.Version,
		Description:  p.def.Description,
		SDKVersion:   Version, // From sdk version.go
		Capabilities: p.def.Capabilities,
		ConfigSchema: p.configSchema,
		Services:     services,
	}
}

// RegisterHandler registers a handler for a service/operation.
// Called internally by RegisterService.
func (p *PluginDefinition) RegisterHandler(
	serviceName, serviceDesc string,
	opName, opDesc string,
	handler HandlerFunc,
	inputType, outputType reflect.Type,
	examples []any,
) {
	p.mu.Lock()
	defer p.mu.Unlock()

	svc, ok := p.services[serviceName]
	if !ok {
		svc = &serviceEntry{
			name:        serviceName,
			description: serviceDesc,
			operations:  make(map[string]*operationEntry),
		}
		p.services[serviceName] = svc
	}

	svc.operations[opName] = &operationEntry{
		name:        opName,
		description: opDesc,
		handler:     handler,
		inputType:   inputType,
		outputType:  outputType,
		examples:    examples,
	}
}

// GetHandler returns a handler for the given service/operation.
func (p *PluginDefinition) GetHandler(serviceName, opName string) (HandlerFunc, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	svc, ok := p.services[serviceName]
	if !ok {
		return nil, false
	}

	op, ok := svc.operations[opName]
	if !ok {
		return nil, false
	}

	return op.handler, true
}

// extractFieldNames gets JSON field names from a struct type.
func extractFieldNames(t reflect.Type) []string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	var names []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		name := strings.Split(jsonTag, ",")[0]
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// convertExamplesToManifest converts typed examples to manifest format.
func convertExamplesToManifest(examples []any) []entities.OperationExample {
	result := make([]entities.OperationExample, 0, len(examples))

	for _, ex := range examples {
		v := reflect.ValueOf(ex)

		name := v.FieldByName("Name").String()
		desc := v.FieldByName("Description").String()
		input := v.FieldByName("Input").Interface()
		expectedOutput := v.FieldByName("ExpectedOutput")
		expectedError := v.FieldByName("ExpectedError").String()

		inputJSON, _ := json.Marshal(input)

		var outputJSON json.RawMessage
		if !expectedOutput.IsNil() {
			outputJSON, _ = json.Marshal(expectedOutput.Interface())
		}

		result = append(result, entities.OperationExample{
			Name:           name,
			Description:    desc,
			Input:          inputJSON,
			ExpectedOutput: outputJSON,
			ExpectedError:  expectedError,
		})
	}

	return result
}
