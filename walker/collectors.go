package walker

import (
	"github.com/erraggy/oastools/parser"
)

// SchemaInfo contains information about a collected schema.
type SchemaInfo struct {
	// Schema is the collected schema.
	Schema *parser.Schema

	// Name is the component name for component schemas.
	// Empty for inline schemas.
	Name string

	// JSONPath is the full JSON path to the schema.
	JSONPath string

	// IsComponent is true when the schema is defined in components/definitions.
	IsComponent bool
}

// SchemaCollector holds schemas collected during a walk.
type SchemaCollector struct {
	// All contains all schemas in traversal order.
	All []*SchemaInfo

	// Components contains only component schemas.
	Components []*SchemaInfo

	// Inline contains only inline schemas (not in components).
	Inline []*SchemaInfo

	// ByPath provides lookup by JSON path.
	ByPath map[string]*SchemaInfo

	// ByName provides lookup by name for schemas in the components section.
	// For top-level component schemas, this is the schema name (e.g., "Pet").
	// For nested property schemas within components, this is the property name.
	// Note: If multiple schemas have the same name, only the last one is stored.
	ByName map[string]*SchemaInfo
}

// CollectSchemas walks the document and collects all schemas.
// It returns a SchemaCollector containing all schemas organized by various criteria.
func CollectSchemas(result *parser.ParseResult) (*SchemaCollector, error) {
	collector := &SchemaCollector{
		All:        make([]*SchemaInfo, 0),
		Components: make([]*SchemaInfo, 0),
		Inline:     make([]*SchemaInfo, 0),
		ByPath:     make(map[string]*SchemaInfo),
		ByName:     make(map[string]*SchemaInfo),
	}

	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			info := &SchemaInfo{
				Schema:      schema,
				Name:        wc.Name,
				JSONPath:    wc.JSONPath,
				IsComponent: wc.IsComponent,
			}

			collector.All = append(collector.All, info)
			collector.ByPath[wc.JSONPath] = info

			if wc.IsComponent {
				collector.Components = append(collector.Components, info)
				if wc.Name != "" {
					collector.ByName[wc.Name] = info
				}
			} else {
				collector.Inline = append(collector.Inline, info)
			}

			return Continue
		}),
	)

	if err != nil {
		return nil, err
	}

	return collector, nil
}

// OperationInfo contains information about a collected operation.
type OperationInfo struct {
	// Operation is the collected operation.
	Operation *parser.Operation

	// PathTemplate is the URL path template (e.g., "/pets/{petId}").
	PathTemplate string

	// Method is the HTTP method (e.g., "get", "post").
	Method string

	// JSONPath is the full JSON path to the operation.
	JSONPath string
}

// OperationCollector holds operations collected during a walk.
type OperationCollector struct {
	// All contains all operations in traversal order.
	All []*OperationInfo

	// ByPath groups operations by path template.
	ByPath map[string][]*OperationInfo

	// ByMethod groups operations by HTTP method.
	ByMethod map[string][]*OperationInfo

	// ByTag groups operations by tag name.
	// Operations with multiple tags appear in multiple groups.
	// Operations without tags are not included in this map.
	ByTag map[string][]*OperationInfo
}

// CollectOperations walks the document and collects all operations.
// It returns an OperationCollector containing all operations organized by various criteria.
func CollectOperations(result *parser.ParseResult) (*OperationCollector, error) {
	collector := &OperationCollector{
		All:      make([]*OperationInfo, 0),
		ByPath:   make(map[string][]*OperationInfo),
		ByMethod: make(map[string][]*OperationInfo),
		ByTag:    make(map[string][]*OperationInfo),
	}

	err := Walk(result,
		WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
			info := &OperationInfo{
				Operation:    op,
				PathTemplate: wc.PathTemplate,
				Method:       wc.Method,
				JSONPath:     wc.JSONPath,
			}

			collector.All = append(collector.All, info)
			collector.ByPath[wc.PathTemplate] = append(collector.ByPath[wc.PathTemplate], info)
			collector.ByMethod[wc.Method] = append(collector.ByMethod[wc.Method], info)

			for _, tag := range op.Tags {
				collector.ByTag[tag] = append(collector.ByTag[tag], info)
			}

			return Continue
		}),
	)

	if err != nil {
		return nil, err
	}

	return collector, nil
}

// ParameterInfo contains information about a collected parameter.
type ParameterInfo struct {
	// Parameter is the collected parameter.
	Parameter *parser.Parameter

	// Name is the parameter name.
	Name string

	// In is the parameter location: query, header, path, cookie.
	In string

	// JSONPath is the full JSON path to the parameter.
	JSONPath string

	// PathTemplate is the owning path template.
	PathTemplate string

	// Method is the owning operation method (empty if path-level).
	Method string

	// IsComponent is true when the parameter is defined in components/definitions.
	IsComponent bool
}

// ParameterCollector holds parameters collected during a walk.
type ParameterCollector struct {
	// All contains all parameters in traversal order.
	All []*ParameterInfo

	// ByName groups parameters by name.
	ByName map[string][]*ParameterInfo

	// ByLocation groups parameters by location: "query", "header", "path", "cookie".
	ByLocation map[string][]*ParameterInfo

	// ByPath groups parameters by path template.
	ByPath map[string][]*ParameterInfo
}

// CollectParameters walks the document and collects all parameters.
// It returns a ParameterCollector containing all parameters organized by various criteria.
func CollectParameters(result *parser.ParseResult) (*ParameterCollector, error) {
	collector := &ParameterCollector{
		All:        make([]*ParameterInfo, 0),
		ByName:     make(map[string][]*ParameterInfo),
		ByLocation: make(map[string][]*ParameterInfo),
		ByPath:     make(map[string][]*ParameterInfo),
	}

	err := Walk(result,
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			info := &ParameterInfo{
				Parameter:    param,
				Name:         param.Name,
				In:           param.In,
				JSONPath:     wc.JSONPath,
				PathTemplate: wc.PathTemplate,
				Method:       wc.Method,
				IsComponent:  wc.IsComponent,
			}

			collector.All = append(collector.All, info)
			collector.ByName[param.Name] = append(collector.ByName[param.Name], info)
			collector.ByLocation[param.In] = append(collector.ByLocation[param.In], info)
			if wc.PathTemplate != "" {
				collector.ByPath[wc.PathTemplate] = append(collector.ByPath[wc.PathTemplate], info)
			}

			return Continue
		}),
	)

	if err != nil {
		return nil, err
	}

	return collector, nil
}

// ResponseInfo contains information about a collected response.
type ResponseInfo struct {
	// Response is the collected response.
	Response *parser.Response

	// StatusCode is the HTTP status code (e.g., "200", "404", "default").
	StatusCode string

	// JSONPath is the full JSON path to the response.
	JSONPath string

	// PathTemplate is the owning path template.
	PathTemplate string

	// Method is the owning operation method.
	Method string

	// IsComponent is true when the response is defined in components/responses.
	IsComponent bool
}

// ResponseCollector holds responses collected during a walk.
type ResponseCollector struct {
	// All contains all responses in traversal order.
	All []*ResponseInfo

	// ByStatusCode groups responses by HTTP status code.
	ByStatusCode map[string][]*ResponseInfo

	// ByPath groups responses by path template.
	ByPath map[string][]*ResponseInfo
}

// CollectResponses walks the document and collects all responses.
// It returns a ResponseCollector containing all responses organized by various criteria.
func CollectResponses(result *parser.ParseResult) (*ResponseCollector, error) {
	collector := &ResponseCollector{
		All:          make([]*ResponseInfo, 0),
		ByStatusCode: make(map[string][]*ResponseInfo),
		ByPath:       make(map[string][]*ResponseInfo),
	}

	err := Walk(result,
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
			info := &ResponseInfo{
				Response:     resp,
				StatusCode:   wc.StatusCode,
				JSONPath:     wc.JSONPath,
				PathTemplate: wc.PathTemplate,
				Method:       wc.Method,
				IsComponent:  wc.IsComponent,
			}

			collector.All = append(collector.All, info)
			if wc.StatusCode != "" {
				collector.ByStatusCode[wc.StatusCode] = append(collector.ByStatusCode[wc.StatusCode], info)
			}
			if wc.PathTemplate != "" {
				collector.ByPath[wc.PathTemplate] = append(collector.ByPath[wc.PathTemplate], info)
			}

			return Continue
		}),
	)

	if err != nil {
		return nil, err
	}

	return collector, nil
}

// SecuritySchemeInfo contains information about a collected security scheme.
type SecuritySchemeInfo struct {
	// SecurityScheme is the collected security scheme.
	SecurityScheme *parser.SecurityScheme

	// Name is the security scheme name from the components map key.
	Name string

	// JSONPath is the full JSON path to the security scheme.
	JSONPath string
}

// SecuritySchemeCollector holds security schemes collected during a walk.
type SecuritySchemeCollector struct {
	// All contains all security schemes in traversal order.
	All []*SecuritySchemeInfo

	// ByName provides lookup by name.
	ByName map[string]*SecuritySchemeInfo
}

// CollectSecuritySchemes walks the document and collects all security schemes.
// It returns a SecuritySchemeCollector containing all security schemes organized by various criteria.
func CollectSecuritySchemes(result *parser.ParseResult) (*SecuritySchemeCollector, error) {
	collector := &SecuritySchemeCollector{
		All:    make([]*SecuritySchemeInfo, 0),
		ByName: make(map[string]*SecuritySchemeInfo),
	}

	err := Walk(result,
		WithSecuritySchemeHandler(func(wc *WalkContext, scheme *parser.SecurityScheme) Action {
			info := &SecuritySchemeInfo{
				SecurityScheme: scheme,
				Name:           wc.Name,
				JSONPath:       wc.JSONPath,
			}

			collector.All = append(collector.All, info)
			if wc.Name != "" {
				collector.ByName[wc.Name] = info
			}

			return Continue
		}),
	)

	if err != nil {
		return nil, err
	}

	return collector, nil
}
