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
