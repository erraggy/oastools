//go:build integration

package harness

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// InjectProblems modifies a parsed document by injecting the specified problems.
func InjectProblems(doc *parser.ParseResult, problems *Problems) error {
	if problems == nil {
		return nil
	}

	// Inject missing path parameters
	for _, mpp := range problems.MissingPathParams {
		if err := injectMissingPathParam(doc, mpp); err != nil {
			return fmt.Errorf("inject missing-path-params: %w", err)
		}
	}

	// Inject generic schemas
	for _, schemaName := range problems.GenericSchemas {
		if err := injectGenericSchema(doc, schemaName); err != nil {
			return fmt.Errorf("inject generic-schemas: %w", err)
		}
	}

	// Inject duplicate operation IDs
	for _, dup := range problems.DuplicateOperationIDs {
		if err := injectDuplicateOperationID(doc, dup); err != nil {
			return fmt.Errorf("inject duplicate-operationids: %w", err)
		}
	}

	// Inject CSV enums
	for _, csvEnum := range problems.CSVEnums {
		if err := injectCSVEnum(doc, csvEnum); err != nil {
			return fmt.Errorf("inject csv-enums: %w", err)
		}
	}

	// Inject unused schemas
	for _, schemaName := range problems.UnusedSchemas {
		if err := injectUnusedSchema(doc, schemaName); err != nil {
			return fmt.Errorf("inject unused-schemas: %w", err)
		}
	}

	// Inject empty paths
	for _, path := range problems.EmptyPaths {
		if err := injectEmptyPath(doc, path); err != nil {
			return fmt.Errorf("inject empty-paths: %w", err)
		}
	}

	// Inject duplicate schemas (identical structure - for join testing)
	for _, dup := range problems.DuplicateSchemaIdentical {
		if err := injectDuplicateSchemaIdentical(doc, dup); err != nil {
			return fmt.Errorf("inject duplicate-schema-identical: %w", err)
		}
	}

	// Inject duplicate schemas (different structure - for collision testing)
	for _, dup := range problems.DuplicateSchemaDifferent {
		if err := injectDuplicateSchemaDifferent(doc, dup); err != nil {
			return fmt.Errorf("inject duplicate-schema-different: %w", err)
		}
	}

	// Inject duplicate paths
	for _, dup := range problems.DuplicatePath {
		if err := injectDuplicatePath(doc, dup); err != nil {
			return fmt.Errorf("inject duplicate-path: %w", err)
		}
	}

	// Inject semantic duplicates (same structure, different name)
	for _, dup := range problems.SemanticDuplicate {
		if err := injectSemanticDuplicate(doc, dup); err != nil {
			return fmt.Errorf("inject semantic-duplicate: %w", err)
		}
	}

	// Inject differ problems (Phase 6)

	// Remove endpoints (breaking change)
	for _, path := range problems.RemoveEndpoint {
		if err := injectRemoveEndpoint(doc, path); err != nil {
			return fmt.Errorf("inject remove-endpoint: %w", err)
		}
	}

	// Remove operations (breaking change)
	for _, op := range problems.RemoveOperation {
		if err := injectRemoveOperation(doc, op); err != nil {
			return fmt.Errorf("inject remove-operation: %w", err)
		}
	}

	// Add required parameters (breaking change)
	for _, param := range problems.AddRequiredParam {
		if err := injectAddRequiredParam(doc, param); err != nil {
			return fmt.Errorf("inject add-required-param: %w", err)
		}
	}

	// Remove response codes (breaking change)
	for _, resp := range problems.RemoveResponseCode {
		if err := injectRemoveResponseCode(doc, resp); err != nil {
			return fmt.Errorf("inject remove-response-code: %w", err)
		}
	}

	// Add endpoints (non-breaking)
	for _, ep := range problems.AddEndpoint {
		if err := injectAddEndpoint(doc, ep); err != nil {
			return fmt.Errorf("inject add-endpoint: %w", err)
		}
	}

	// Add optional parameters (non-breaking)
	for _, param := range problems.AddOptionalParam {
		if err := injectAddOptionalParam(doc, param); err != nil {
			return fmt.Errorf("inject add-optional-param: %w", err)
		}
	}

	return nil
}

// injectMissingPathParam adds a path with template variables but no parameter declarations.
func injectMissingPathParam(doc *parser.ParseResult, mpp MissingPathParam) error {
	if doc.IsOAS3() {
		return injectMissingPathParamOAS3(doc, mpp)
	}
	return injectMissingPathParamOAS2(doc, mpp)
}

func injectMissingPathParamOAS3(doc *parser.ParseResult, mpp MissingPathParam) error {
	oas3Doc, ok := doc.OAS3Document()
	if !ok {
		return fmt.Errorf("expected OAS3 document")
	}

	if oas3Doc.Paths == nil {
		oas3Doc.Paths = make(parser.Paths)
	}

	// Create a path item with an operation but no parameters for the template variables
	pathItem := &parser.PathItem{}
	operation := &parser.Operation{
		Summary:     "Test operation with missing path params",
		OperationID: generateOperationID(mpp.Path, mpp.Method),
		Responses: &parser.Responses{
			Codes: map[string]*parser.Response{
				"200": {Description: "Success"},
			},
		},
		// Deliberately NOT adding parameters for the template variables in the path
	}

	// Set the operation on the appropriate method
	setOperationOnPathItem(pathItem, mpp.Method, operation)

	oas3Doc.Paths[mpp.Path] = pathItem
	return nil
}

func injectMissingPathParamOAS2(doc *parser.ParseResult, mpp MissingPathParam) error {
	oas2Doc, ok := doc.OAS2Document()
	if !ok {
		return fmt.Errorf("expected OAS2 document")
	}

	if oas2Doc.Paths == nil {
		oas2Doc.Paths = make(map[string]*parser.PathItem)
	}

	pathItem := &parser.PathItem{}
	operation := &parser.Operation{
		Summary:     "Test operation with missing path params",
		OperationID: generateOperationID(mpp.Path, mpp.Method),
		Responses: &parser.Responses{
			Codes: map[string]*parser.Response{
				"200": {Description: "Success"},
			},
		},
	}

	setOperationOnPathItem(pathItem, mpp.Method, operation)
	oas2Doc.Paths[mpp.Path] = pathItem
	return nil
}

// injectGenericSchema adds a schema with bracket syntax (e.g., Response[Pet]).
func injectGenericSchema(doc *parser.ParseResult, schemaName string) error {
	schema := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"data": {Type: "string"},
		},
	}

	if doc.IsOAS3() {
		oas3Doc, ok := doc.OAS3Document()
		if !ok {
			return fmt.Errorf("expected OAS3 document")
		}
		if oas3Doc.Components == nil {
			oas3Doc.Components = &parser.Components{}
		}
		if oas3Doc.Components.Schemas == nil {
			oas3Doc.Components.Schemas = make(map[string]*parser.Schema)
		}
		oas3Doc.Components.Schemas[schemaName] = schema
	} else {
		oas2Doc, ok := doc.OAS2Document()
		if !ok {
			return fmt.Errorf("expected OAS2 document")
		}
		if oas2Doc.Definitions == nil {
			oas2Doc.Definitions = make(map[string]*parser.Schema)
		}
		oas2Doc.Definitions[schemaName] = schema
	}

	return nil
}

// injectDuplicateOperationID creates multiple operations with the same operationId.
func injectDuplicateOperationID(doc *parser.ParseResult, dup DuplicateOperationID) error {
	if doc.IsOAS3() {
		return injectDuplicateOperationIDOAS3(doc, dup)
	}
	return injectDuplicateOperationIDOAS2(doc, dup)
}

func injectDuplicateOperationIDOAS3(doc *parser.ParseResult, dup DuplicateOperationID) error {
	oas3Doc, ok := doc.OAS3Document()
	if !ok {
		return fmt.Errorf("expected OAS3 document")
	}

	if oas3Doc.Paths == nil {
		oas3Doc.Paths = make(parser.Paths)
	}

	methods := []string{"get", "post", "put", "delete", "patch"}

	for i := range dup.Count {
		path := fmt.Sprintf("/duplicate-%s-%d", dup.ID, i)
		method := methods[i%len(methods)]

		pathItem := &parser.PathItem{}
		operation := &parser.Operation{
			Summary:     fmt.Sprintf("Duplicate operation %d", i),
			OperationID: dup.ID, // Same operationId for all
			Responses: &parser.Responses{
				Codes: map[string]*parser.Response{
					"200": {Description: "Success"},
				},
			},
		}
		setOperationOnPathItem(pathItem, method, operation)
		oas3Doc.Paths[path] = pathItem
	}

	return nil
}

func injectDuplicateOperationIDOAS2(doc *parser.ParseResult, dup DuplicateOperationID) error {
	oas2Doc, ok := doc.OAS2Document()
	if !ok {
		return fmt.Errorf("expected OAS2 document")
	}

	if oas2Doc.Paths == nil {
		oas2Doc.Paths = make(map[string]*parser.PathItem)
	}

	methods := []string{"get", "post", "put", "delete", "patch"}

	for i := range dup.Count {
		path := fmt.Sprintf("/duplicate-%s-%d", dup.ID, i)
		method := methods[i%len(methods)]

		pathItem := &parser.PathItem{}
		operation := &parser.Operation{
			Summary:     fmt.Sprintf("Duplicate operation %d", i),
			OperationID: dup.ID,
			Responses: &parser.Responses{
				Codes: map[string]*parser.Response{
					"200": {Description: "Success"},
				},
			},
		}
		setOperationOnPathItem(pathItem, method, operation)
		oas2Doc.Paths[path] = pathItem
	}

	return nil
}

// injectCSVEnum adds a schema with enum values stored as a CSV string.
// Note: The CSV enum fixer only works for integer/number types.
func injectCSVEnum(doc *parser.ParseResult, csvEnum CSVEnum) error {
	// Create schema with CSV-style enum (single string value that contains commas)
	// The fixer expects integer/number type with CSV values like "1,2,3"
	schema := &parser.Schema{
		Type: "integer",
		Enum: []any{csvEnum.Values}, // Store as single CSV string - this is the problem we're testing
	}

	if doc.IsOAS3() {
		oas3Doc, ok := doc.OAS3Document()
		if !ok {
			return fmt.Errorf("expected OAS3 document")
		}
		if oas3Doc.Components == nil {
			oas3Doc.Components = &parser.Components{}
		}
		if oas3Doc.Components.Schemas == nil {
			oas3Doc.Components.Schemas = make(map[string]*parser.Schema)
		}
		oas3Doc.Components.Schemas[csvEnum.Schema] = schema
	} else {
		oas2Doc, ok := doc.OAS2Document()
		if !ok {
			return fmt.Errorf("expected OAS2 document")
		}
		if oas2Doc.Definitions == nil {
			oas2Doc.Definitions = make(map[string]*parser.Schema)
		}
		oas2Doc.Definitions[csvEnum.Schema] = schema
	}

	return nil
}

// injectUnusedSchema adds a schema that isn't referenced anywhere.
func injectUnusedSchema(doc *parser.ParseResult, schemaName string) error {
	schema := &parser.Schema{
		Type:        "object",
		Description: "An orphaned schema that is not referenced anywhere",
		Properties: map[string]*parser.Schema{
			"orphanedField": {Type: "string"},
		},
	}

	if doc.IsOAS3() {
		oas3Doc, ok := doc.OAS3Document()
		if !ok {
			return fmt.Errorf("expected OAS3 document")
		}
		if oas3Doc.Components == nil {
			oas3Doc.Components = &parser.Components{}
		}
		if oas3Doc.Components.Schemas == nil {
			oas3Doc.Components.Schemas = make(map[string]*parser.Schema)
		}
		oas3Doc.Components.Schemas[schemaName] = schema
	} else {
		oas2Doc, ok := doc.OAS2Document()
		if !ok {
			return fmt.Errorf("expected OAS2 document")
		}
		if oas2Doc.Definitions == nil {
			oas2Doc.Definitions = make(map[string]*parser.Schema)
		}
		oas2Doc.Definitions[schemaName] = schema
	}

	return nil
}

// injectEmptyPath adds a path with no HTTP operations.
func injectEmptyPath(doc *parser.ParseResult, path string) error {
	// Create an empty path item (no operations)
	pathItem := &parser.PathItem{
		Summary:     "Empty path with no operations",
		Description: "This path has no HTTP methods defined",
	}

	if doc.IsOAS3() {
		oas3Doc, ok := doc.OAS3Document()
		if !ok {
			return fmt.Errorf("expected OAS3 document")
		}
		if oas3Doc.Paths == nil {
			oas3Doc.Paths = make(parser.Paths)
		}
		oas3Doc.Paths[path] = pathItem
	} else {
		oas2Doc, ok := doc.OAS2Document()
		if !ok {
			return fmt.Errorf("expected OAS2 document")
		}
		if oas2Doc.Paths == nil {
			oas2Doc.Paths = make(map[string]*parser.PathItem)
		}
		oas2Doc.Paths[path] = pathItem
	}

	return nil
}

// setOperationOnPathItem sets an operation on the path item for the given method.
func setOperationOnPathItem(pathItem *parser.PathItem, method string, operation *parser.Operation) {
	switch strings.ToLower(method) {
	case "get":
		pathItem.Get = operation
	case "post":
		pathItem.Post = operation
	case "put":
		pathItem.Put = operation
	case "delete":
		pathItem.Delete = operation
	case "patch":
		pathItem.Patch = operation
	case "options":
		pathItem.Options = operation
	case "head":
		pathItem.Head = operation
	case "trace":
		pathItem.Trace = operation
	case "query":
		pathItem.Query = operation
	}
}

// generateOperationID creates an operationId from a path and method.
func generateOperationID(path, method string) string {
	// Remove leading slash and replace special chars
	cleanPath := strings.TrimPrefix(path, "/")
	cleanPath = strings.ReplaceAll(cleanPath, "/", "_")
	cleanPath = strings.ReplaceAll(cleanPath, "{", "")
	cleanPath = strings.ReplaceAll(cleanPath, "}", "")
	cleanPath = strings.ReplaceAll(cleanPath, "-", "_")

	return strings.ToLower(method) + "_" + cleanPath
}

// --- Joiner Problem Injectors ---

// injectDuplicateSchemaIdentical adds a schema with the same name and identical structure.
// If CopyFrom is specified, copies structure from that schema; otherwise creates a simple object.
func injectDuplicateSchemaIdentical(doc *parser.ParseResult, dup DuplicateSchema) error {
	var schema *parser.Schema

	if dup.CopyFrom != "" {
		// Copy structure from an existing schema
		existingSchema, err := getSchema(doc, dup.CopyFrom)
		if err != nil {
			return fmt.Errorf("copy-from schema %q not found: %w", dup.CopyFrom, err)
		}
		schema = copySchema(existingSchema)
	} else {
		// Create a simple object schema (this will match if the target is also a simple object)
		schema = &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"id":   {Type: "integer", Format: "int64"},
				"name": {Type: "string"},
			},
		}
	}

	return setSchema(doc, dup.Name, schema)
}

// injectDuplicateSchemaDifferent adds a schema with the same name but different structure.
// This creates a collision scenario where schemas have the same name but incompatible definitions.
func injectDuplicateSchemaDifferent(doc *parser.ParseResult, dup DuplicateSchema) error {
	// Create a schema with different structure than what typically exists
	// We use a completely different set of properties to ensure it's different
	schema := &parser.Schema{
		Type:        "object",
		Description: "Conflicting schema with different structure",
		Properties: map[string]*parser.Schema{
			"conflictField1": {Type: "string", Description: "Unique field for collision"},
			"conflictField2": {Type: "boolean"},
			"conflictField3": {Type: "number", Format: "double"},
		},
		Required: []string{"conflictField1"},
	}

	return setSchema(doc, dup.Name, schema)
}

// injectDuplicatePath adds a path that already exists (for path collision testing).
func injectDuplicatePath(doc *parser.ParseResult, dup DuplicatePathConfig) error {
	method := dup.Method
	if method == "" {
		method = "get"
	}

	pathItem := &parser.PathItem{}
	operation := &parser.Operation{
		Summary:     "Duplicate path operation for collision testing",
		OperationID: generateOperationID(dup.Path+"_dup", method),
		Responses: &parser.Responses{
			Codes: map[string]*parser.Response{
				"200": {Description: "Success"},
			},
		},
	}
	setOperationOnPathItem(pathItem, method, operation)

	if doc.IsOAS3() {
		oas3Doc, ok := doc.OAS3Document()
		if !ok {
			return fmt.Errorf("expected OAS3 document")
		}
		if oas3Doc.Paths == nil {
			oas3Doc.Paths = make(parser.Paths)
		}
		oas3Doc.Paths[dup.Path] = pathItem
	} else {
		oas2Doc, ok := doc.OAS2Document()
		if !ok {
			return fmt.Errorf("expected OAS2 document")
		}
		if oas2Doc.Paths == nil {
			oas2Doc.Paths = make(map[string]*parser.PathItem)
		}
		oas2Doc.Paths[dup.Path] = pathItem
	}

	return nil
}

// injectSemanticDuplicate adds a schema with a different name but identical structure to another.
// This is for testing semantic deduplication where schemas are structurally equivalent.
func injectSemanticDuplicate(doc *parser.ParseResult, dup SemanticDuplicateConfig) error {
	// Get the original schema
	original, err := getSchema(doc, dup.Original)
	if err != nil {
		return fmt.Errorf("original schema %q not found: %w", dup.Original, err)
	}

	// Create an identical copy with a different name
	duplicate := copySchema(original)

	return setSchema(doc, dup.DuplicateName, duplicate)
}

// getSchema retrieves a schema by name from the document.
func getSchema(doc *parser.ParseResult, name string) (*parser.Schema, error) {
	if doc.IsOAS3() {
		oas3Doc, ok := doc.OAS3Document()
		if !ok {
			return nil, fmt.Errorf("expected OAS3 document")
		}
		if oas3Doc.Components == nil || oas3Doc.Components.Schemas == nil {
			return nil, fmt.Errorf("schema %q not found: no schemas defined", name)
		}
		schema, ok := oas3Doc.Components.Schemas[name]
		if !ok {
			return nil, fmt.Errorf("schema %q not found", name)
		}
		return schema, nil
	}

	oas2Doc, ok := doc.OAS2Document()
	if !ok {
		return nil, fmt.Errorf("expected OAS2 document")
	}
	if oas2Doc.Definitions == nil {
		return nil, fmt.Errorf("schema %q not found: no definitions defined", name)
	}
	schema, ok := oas2Doc.Definitions[name]
	if !ok {
		return nil, fmt.Errorf("schema %q not found", name)
	}
	return schema, nil
}

// setSchema adds or replaces a schema in the document.
func setSchema(doc *parser.ParseResult, name string, schema *parser.Schema) error {
	if doc.IsOAS3() {
		oas3Doc, ok := doc.OAS3Document()
		if !ok {
			return fmt.Errorf("expected OAS3 document")
		}
		if oas3Doc.Components == nil {
			oas3Doc.Components = &parser.Components{}
		}
		if oas3Doc.Components.Schemas == nil {
			oas3Doc.Components.Schemas = make(map[string]*parser.Schema)
		}
		oas3Doc.Components.Schemas[name] = schema
	} else {
		oas2Doc, ok := doc.OAS2Document()
		if !ok {
			return fmt.Errorf("expected OAS2 document")
		}
		if oas2Doc.Definitions == nil {
			oas2Doc.Definitions = make(map[string]*parser.Schema)
		}
		oas2Doc.Definitions[name] = schema
	}
	return nil
}

// --- Differ Problem Injectors ---

// injectRemoveEndpoint removes an existing path from the document.
// Returns an error if the path does not exist (likely a configuration error).
func injectRemoveEndpoint(doc *parser.ParseResult, path string) error {
	if doc.IsOAS3() {
		oas3Doc, ok := doc.OAS3Document()
		if !ok {
			return fmt.Errorf("expected OAS3 document")
		}
		if oas3Doc.Paths == nil {
			return fmt.Errorf("cannot remove endpoint %q: no paths defined in document", path)
		}
		if _, exists := oas3Doc.Paths[path]; !exists {
			return fmt.Errorf("cannot remove endpoint %q: path does not exist", path)
		}
		delete(oas3Doc.Paths, path)
	} else {
		oas2Doc, ok := doc.OAS2Document()
		if !ok {
			return fmt.Errorf("expected OAS2 document")
		}
		if oas2Doc.Paths == nil {
			return fmt.Errorf("cannot remove endpoint %q: no paths defined in document", path)
		}
		if _, exists := oas2Doc.Paths[path]; !exists {
			return fmt.Errorf("cannot remove endpoint %q: path does not exist", path)
		}
		delete(oas2Doc.Paths, path)
	}
	return nil
}

// injectRemoveOperation removes an operation from a path.
// Returns an error if the path or operation does not exist (likely a configuration error).
func injectRemoveOperation(doc *parser.ParseResult, op RemoveOperationConfig) error {
	var pathItem *parser.PathItem

	if doc.IsOAS3() {
		oas3Doc, ok := doc.OAS3Document()
		if !ok {
			return fmt.Errorf("expected OAS3 document")
		}
		if oas3Doc.Paths == nil {
			return fmt.Errorf("cannot remove operation %s %s: no paths defined in document", op.Method, op.Path)
		}
		pathItem = oas3Doc.Paths[op.Path]
	} else {
		oas2Doc, ok := doc.OAS2Document()
		if !ok {
			return fmt.Errorf("expected OAS2 document")
		}
		if oas2Doc.Paths == nil {
			return fmt.Errorf("cannot remove operation %s %s: no paths defined in document", op.Method, op.Path)
		}
		pathItem = oas2Doc.Paths[op.Path]
	}

	if pathItem == nil {
		return fmt.Errorf("cannot remove operation %s %s: path does not exist", op.Method, op.Path)
	}

	// Remove the operation (check it exists first)
	method := strings.ToLower(op.Method)
	var exists bool
	switch method {
	case "get":
		exists = pathItem.Get != nil
		pathItem.Get = nil
	case "post":
		exists = pathItem.Post != nil
		pathItem.Post = nil
	case "put":
		exists = pathItem.Put != nil
		pathItem.Put = nil
	case "delete":
		exists = pathItem.Delete != nil
		pathItem.Delete = nil
	case "patch":
		exists = pathItem.Patch != nil
		pathItem.Patch = nil
	case "options":
		exists = pathItem.Options != nil
		pathItem.Options = nil
	case "head":
		exists = pathItem.Head != nil
		pathItem.Head = nil
	case "trace":
		exists = pathItem.Trace != nil
		pathItem.Trace = nil
	default:
		return fmt.Errorf("cannot remove operation %s %s: unknown HTTP method", op.Method, op.Path)
	}

	if !exists {
		return fmt.Errorf("cannot remove operation %s %s: operation does not exist", op.Method, op.Path)
	}

	return nil
}

// injectAddRequiredParam adds a new required parameter to an operation.
func injectAddRequiredParam(doc *parser.ParseResult, cfg AddRequiredParamConfig) error {
	operation, err := getOperation(doc, cfg.Path, cfg.Method)
	if err != nil {
		return err
	}
	if operation == nil {
		return fmt.Errorf("operation %s %s not found", cfg.Method, cfg.Path)
	}

	// Create the new required parameter
	param := &parser.Parameter{
		Name:        cfg.ParamName,
		In:          cfg.In,
		Required:    true,
		Description: "New required parameter added for diff testing",
		Schema:      &parser.Schema{Type: "string"},
	}

	operation.Parameters = append(operation.Parameters, param)
	return nil
}

// injectRemoveResponseCode removes a response code from an operation.
func injectRemoveResponseCode(doc *parser.ParseResult, cfg RemoveResponseCodeConfig) error {
	operation, err := getOperation(doc, cfg.Path, cfg.Method)
	if err != nil {
		return err
	}
	if operation == nil {
		return fmt.Errorf("operation %s %s not found", cfg.Method, cfg.Path)
	}

	if operation.Responses != nil && operation.Responses.Codes != nil {
		delete(operation.Responses.Codes, cfg.Code)
	}
	return nil
}

// injectAddEndpoint adds a new endpoint to the document.
func injectAddEndpoint(doc *parser.ParseResult, cfg AddEndpointConfig) error {
	pathItem := &parser.PathItem{}
	operation := &parser.Operation{
		Summary:     "New endpoint added for diff testing",
		OperationID: generateOperationID(cfg.Path, cfg.Method),
		Responses: &parser.Responses{
			Codes: map[string]*parser.Response{
				"200": {Description: "Success"},
			},
		},
	}
	setOperationOnPathItem(pathItem, cfg.Method, operation)

	if doc.IsOAS3() {
		oas3Doc, ok := doc.OAS3Document()
		if !ok {
			return fmt.Errorf("expected OAS3 document")
		}
		if oas3Doc.Paths == nil {
			oas3Doc.Paths = make(parser.Paths)
		}
		oas3Doc.Paths[cfg.Path] = pathItem
	} else {
		oas2Doc, ok := doc.OAS2Document()
		if !ok {
			return fmt.Errorf("expected OAS2 document")
		}
		if oas2Doc.Paths == nil {
			oas2Doc.Paths = make(map[string]*parser.PathItem)
		}
		oas2Doc.Paths[cfg.Path] = pathItem
	}

	return nil
}

// injectAddOptionalParam adds a new optional parameter to an operation.
func injectAddOptionalParam(doc *parser.ParseResult, cfg AddOptionalParamConfig) error {
	operation, err := getOperation(doc, cfg.Path, cfg.Method)
	if err != nil {
		return err
	}
	if operation == nil {
		return fmt.Errorf("operation %s %s not found", cfg.Method, cfg.Path)
	}

	// Create the new optional parameter
	param := &parser.Parameter{
		Name:        cfg.ParamName,
		In:          cfg.In,
		Required:    false,
		Description: "New optional parameter added for diff testing",
		Schema:      &parser.Schema{Type: "string"},
	}

	operation.Parameters = append(operation.Parameters, param)
	return nil
}

// getOperation retrieves an operation from a path.
func getOperation(doc *parser.ParseResult, path, method string) (*parser.Operation, error) {
	var pathItem *parser.PathItem

	if doc.IsOAS3() {
		oas3Doc, ok := doc.OAS3Document()
		if !ok {
			return nil, fmt.Errorf("expected OAS3 document")
		}
		if oas3Doc.Paths == nil {
			return nil, nil
		}
		pathItem = oas3Doc.Paths[path]
	} else {
		oas2Doc, ok := doc.OAS2Document()
		if !ok {
			return nil, fmt.Errorf("expected OAS2 document")
		}
		if oas2Doc.Paths == nil {
			return nil, nil
		}
		pathItem = oas2Doc.Paths[path]
	}

	if pathItem == nil {
		return nil, nil
	}

	switch strings.ToLower(method) {
	case "get":
		return pathItem.Get, nil
	case "post":
		return pathItem.Post, nil
	case "put":
		return pathItem.Put, nil
	case "delete":
		return pathItem.Delete, nil
	case "patch":
		return pathItem.Patch, nil
	case "options":
		return pathItem.Options, nil
	case "head":
		return pathItem.Head, nil
	case "trace":
		return pathItem.Trace, nil
	}

	return nil, nil
}

// copySchema creates a deep copy of a schema.
// For simplicity, we recreate the essential fields rather than using JSON marshaling.
func copySchema(original *parser.Schema) *parser.Schema {
	if original == nil {
		return nil
	}

	copy := &parser.Schema{
		Type:                 original.Type,
		Format:               original.Format,
		Description:          original.Description,
		Default:              original.Default,
		Minimum:              original.Minimum,
		Maximum:              original.Maximum,
		ExclusiveMinimum:     original.ExclusiveMinimum,
		ExclusiveMaximum:     original.ExclusiveMaximum,
		MinLength:            original.MinLength,
		MaxLength:            original.MaxLength,
		Pattern:              original.Pattern,
		MinItems:             original.MinItems,
		MaxItems:             original.MaxItems,
		UniqueItems:          original.UniqueItems,
		MinProperties:        original.MinProperties,
		MaxProperties:        original.MaxProperties,
		Nullable:             original.Nullable,
		ReadOnly:             original.ReadOnly,
		WriteOnly:            original.WriteOnly,
		Deprecated:           original.Deprecated,
		Discriminator:        original.Discriminator,
		Example:              original.Example,
		ExternalDocs:         original.ExternalDocs,
		AdditionalProperties: original.AdditionalProperties,
		XML:                  original.XML,
		ContentMediaType:     original.ContentMediaType,
		ContentEncoding:      original.ContentEncoding,
		Title:                original.Title,
		Const:                original.Const,
		Ref:                  original.Ref,
	}

	// Copy slices
	if len(original.Required) > 0 {
		copy.Required = make([]string, len(original.Required))
		copy.Required = append(copy.Required[:0], original.Required...)
	}
	if len(original.Enum) > 0 {
		copy.Enum = make([]any, len(original.Enum))
		copy.Enum = append(copy.Enum[:0], original.Enum...)
	}

	// Copy properties (shallow copy of the map, deep copy of values)
	if original.Properties != nil {
		copy.Properties = make(map[string]*parser.Schema)
		for k, v := range original.Properties {
			copy.Properties[k] = copySchema(v)
		}
	}

	// Copy items (Items can be *Schema or bool in OAS 3.1+)
	if original.Items != nil {
		switch items := original.Items.(type) {
		case *parser.Schema:
			copy.Items = copySchema(items)
		case bool:
			copy.Items = items
		default:
			// Keep original as-is for unknown types
			copy.Items = original.Items
		}
	}

	// Copy allOf, oneOf, anyOf
	if len(original.AllOf) > 0 {
		copy.AllOf = make([]*parser.Schema, len(original.AllOf))
		for i, s := range original.AllOf {
			copy.AllOf[i] = copySchema(s)
		}
	}
	if len(original.OneOf) > 0 {
		copy.OneOf = make([]*parser.Schema, len(original.OneOf))
		for i, s := range original.OneOf {
			copy.OneOf[i] = copySchema(s)
		}
	}
	if len(original.AnyOf) > 0 {
		copy.AnyOf = make([]*parser.Schema, len(original.AnyOf))
		for i, s := range original.AnyOf {
			copy.AnyOf[i] = copySchema(s)
		}
	}
	if original.Not != nil {
		copy.Not = copySchema(original.Not)
	}

	return copy
}
