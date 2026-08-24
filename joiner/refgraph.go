package joiner

import (
	"fmt"

	"github.com/erraggy/oastools/internal/httputil"
	"github.com/erraggy/oastools/internal/schemarefs"
	"github.com/erraggy/oastools/parser"
)

// newRefGraph creates a new empty RefGraph.
func newRefGraph() *RefGraph {
	return &RefGraph{
		schemaRefs:    make(map[string][]SchemaRef),
		operationRefs: make(map[string][]OperationRef),
		resolved:      make(map[string][]OperationRef),
	}
}

// buildRefGraphOAS3 builds a reference graph from an OAS 3.x document.
func buildRefGraphOAS3(doc *parser.OAS3Document, version parser.OASVersion) *RefGraph {
	g := newRefGraph()
	if doc == nil {
		return g
	}

	// Traverse paths for operation references
	for path, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}
		g.recordPathItemOAS3(path, pathItem, version)
	}

	// Traverse webhooks (OAS 3.1+)
	for name, pathItem := range doc.Webhooks {
		if pathItem == nil {
			continue
		}
		// Use webhook name as the path for traceability
		g.recordPathItemOAS3(fmt.Sprintf("webhook:%s", name), pathItem, version)
	}

	// Traverse component schemas for schema-to-schema references
	if doc.Components != nil {
		for schemaName, schema := range doc.Components.Schemas {
			if schema != nil {
				g.recordSchemaRefs(schemaName, schema)
			}
		}
	}

	return g
}

// recordPathItemOAS3 records all schema references from a PathItem's operations.
func (g *RefGraph) recordPathItemOAS3(path string, pathItem *parser.PathItem, version parser.OASVersion) {
	ops := parser.GetOperations(pathItem, version)

	for method, op := range ops {
		if op == nil {
			continue
		}

		baseRef := OperationRef{
			Path:        path,
			Method:      method,
			OperationID: op.OperationID,
			Tags:        op.Tags,
		}

		// Record request body schema references
		if op.RequestBody != nil && op.RequestBody.Content != nil {
			for mediaType, content := range op.RequestBody.Content {
				if content != nil && content.Schema != nil {
					g.recordOperationSchemaRef(content.Schema, baseRef, UsageTypeRequest, "", "", mediaType)
				}
			}
		}

		// Record response schema references
		if op.Responses != nil {
			// Default response
			if op.Responses.Default != nil {
				g.recordResponseSchemaRefs(op.Responses.Default, baseRef, defaultKey)
			}
			// Status code responses
			for statusCode, response := range op.Responses.Codes {
				if response != nil {
					g.recordResponseSchemaRefs(response, baseRef, statusCode)
				}
			}
		}

		// Record parameter schema references
		for _, param := range op.Parameters {
			if param != nil && param.Schema != nil {
				g.recordOperationSchemaRef(param.Schema, baseRef, UsageTypeParameter, "", param.Name, "")
			}
		}

		// Record callback schema references (OAS 3.0+)
		for callbackName, callback := range op.Callbacks {
			if callback == nil {
				continue
			}
			for cbPath, cbPathItem := range *callback {
				if cbPathItem != nil {
					// Recursively record the callback path item
					cbRef := OperationRef{
						Path:        fmt.Sprintf("%s->%s:%s", path, callbackName, cbPath),
						Method:      method,
						OperationID: op.OperationID,
						Tags:        op.Tags,
						UsageType:   UsageTypeCallback,
					}
					g.recordPathItemCallbackOAS3(cbPathItem, cbRef, version)
				}
			}
		}
	}

	// Also check path-level parameters
	for _, param := range pathItem.Parameters {
		if param != nil && param.Schema != nil {
			// Path-level parameters don't have a specific operation
			baseRef := OperationRef{
				Path: path,
			}
			g.recordOperationSchemaRef(param.Schema, baseRef, UsageTypeParameter, "", param.Name, "")
		}
	}
}

// recordPathItemCallbackOAS3 records schema references from a callback PathItem.
func (g *RefGraph) recordPathItemCallbackOAS3(pathItem *parser.PathItem, baseRef OperationRef, version parser.OASVersion) {
	ops := parser.GetOperations(pathItem, version)

	for method, op := range ops {
		if op == nil {
			continue
		}

		callbackRef := baseRef
		callbackRef.Method = method
		if op.OperationID != "" {
			callbackRef.OperationID = op.OperationID
		}
		if len(op.Tags) > 0 {
			callbackRef.Tags = op.Tags
		}

		// Record request body
		if op.RequestBody != nil && op.RequestBody.Content != nil {
			for mediaType, content := range op.RequestBody.Content {
				if content != nil && content.Schema != nil {
					g.recordOperationSchemaRef(content.Schema, callbackRef, UsageTypeCallback, "", "", mediaType)
				}
			}
		}

		// Record responses
		if op.Responses != nil {
			if op.Responses.Default != nil {
				g.recordResponseSchemaRefs(op.Responses.Default, callbackRef, defaultKey)
			}
			for statusCode, response := range op.Responses.Codes {
				if response != nil {
					g.recordResponseSchemaRefs(response, callbackRef, statusCode)
				}
			}
		}
	}
}

// recordResponseSchemaRefs records schema references from a response.
func (g *RefGraph) recordResponseSchemaRefs(response *parser.Response, baseRef OperationRef, statusCode string) {
	// Record content schema references
	for mediaType, content := range response.Content {
		if content != nil && content.Schema != nil {
			g.recordOperationSchemaRef(content.Schema, baseRef, UsageTypeResponse, statusCode, "", mediaType)
		}
	}

	// Record header schema references
	for headerName, header := range response.Headers {
		if header != nil && header.Schema != nil {
			g.recordOperationSchemaRef(header.Schema, baseRef, UsageTypeHeader, statusCode, headerName, "")
		}
	}
}

// recordOperationSchemaRef records a schema reference from an operation.
func (g *RefGraph) recordOperationSchemaRef(schema *parser.Schema, baseRef OperationRef, usage UsageType, statusCode, paramName, mediaType string) {
	schemaName := extractSchemaNameFromRef(schema.Ref)
	if schemaName == "" {
		// Not a $ref, but might contain nested $refs - we track the immediate ref only
		return
	}

	opRef := OperationRef{
		Path:        baseRef.Path,
		Method:      baseRef.Method,
		OperationID: baseRef.OperationID,
		Tags:        baseRef.Tags,
		UsageType:   usage,
		StatusCode:  statusCode,
		ParamName:   paramName,
		MediaType:   mediaType,
	}

	g.operationRefs[schemaName] = append(g.operationRefs[schemaName], opRef)
}

// buildRefGraphOAS2 builds a reference graph from an OAS 2.0 document.
func buildRefGraphOAS2(doc *parser.OAS2Document) *RefGraph {
	g := newRefGraph()
	if doc == nil {
		return g
	}

	// Traverse paths for operation references
	for path, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}
		g.recordPathItemOAS2(path, pathItem)
	}

	// Traverse definitions for schema-to-schema references
	for schemaName, schema := range doc.Definitions {
		if schema != nil {
			g.recordSchemaRefs(schemaName, schema)
		}
	}

	return g
}

// recordPathItemOAS2 records all schema references from a PathItem's operations (OAS 2.0).
func (g *RefGraph) recordPathItemOAS2(path string, pathItem *parser.PathItem) {
	// OAS 2.0 operations
	operations := map[string]*parser.Operation{
		"get":               pathItem.Get,
		"put":               pathItem.Put,
		httputil.MethodPost: pathItem.Post,
		"delete":            pathItem.Delete,
		"options":           pathItem.Options,
		"head":              pathItem.Head,
		"patch":             pathItem.Patch,
	}

	for method, op := range operations {
		if op == nil {
			continue
		}

		baseRef := OperationRef{
			Path:        path,
			Method:      method,
			OperationID: op.OperationID,
			Tags:        op.Tags,
		}

		// Record parameter schema references (including body parameters)
		for _, param := range op.Parameters {
			if param == nil {
				continue
			}
			if param.In == "body" && param.Schema != nil {
				// Body parameter - this is the request body in OAS 2.0
				g.recordOperationSchemaRef(param.Schema, baseRef, UsageTypeRequest, "", param.Name, "")
			} else if param.Schema != nil {
				// Other parameters with schema
				g.recordOperationSchemaRef(param.Schema, baseRef, UsageTypeParameter, "", param.Name, "")
			}
		}

		// Record response schema references
		if op.Responses != nil {
			// Default response
			if op.Responses.Default != nil && op.Responses.Default.Schema != nil {
				g.recordOperationSchemaRef(op.Responses.Default.Schema, baseRef, UsageTypeResponse, defaultKey, "", "")
			}
			// Status code responses
			for statusCode, response := range op.Responses.Codes {
				if response != nil && response.Schema != nil {
					g.recordOperationSchemaRef(response.Schema, baseRef, UsageTypeResponse, statusCode, "", "")
				}
			}
		}
	}

	// Also check path-level parameters
	for _, param := range pathItem.Parameters {
		if param == nil {
			continue
		}
		baseRef := OperationRef{
			Path: path,
		}
		if param.In == "body" && param.Schema != nil {
			g.recordOperationSchemaRef(param.Schema, baseRef, UsageTypeRequest, "", param.Name, "")
		} else if param.Schema != nil {
			g.recordOperationSchemaRef(param.Schema, baseRef, UsageTypeParameter, "", param.Name, "")
		}
	}
}

// extractSchemaNameFromRef extracts the schema name from a $ref string.
func extractSchemaNameFromRef(ref string) string {
	return schemarefs.SchemaName(ref)
}

// recordSchemaRefs records every reference schema makes to a component schema,
// attributing each to schemaName.
func (g *RefGraph) recordSchemaRefs(schemaName string, schema *parser.Schema) {
	schemarefs.EachRef(schema, "", func(name, at string) {
		g.schemaRefs[name] = append(g.schemaRefs[name], SchemaRef{
			FromSchema:  schemaName,
			RefLocation: at,
		})
	})
}

// deduplicateOperationRefs removes duplicate operation references.
// Duplicates are identified by path + method + usageType + statusCode.
func deduplicateOperationRefs(refs []OperationRef) []OperationRef {
	if len(refs) == 0 {
		return refs
	}

	seen := make(map[string]bool)
	result := make([]OperationRef, 0, len(refs))

	for _, ref := range refs {
		key := fmt.Sprintf("%s|%s|%s|%s", ref.Path, ref.Method, ref.UsageType, ref.StatusCode)
		if !seen[key] {
			seen[key] = true
			result = append(result, ref)
		}
	}

	return result
}
