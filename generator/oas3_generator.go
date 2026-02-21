package generator

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/erraggy/oastools/internal/httputil"
	"github.com/erraggy/oastools/internal/maputil"
	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
)

// oas3CodeGenerator handles code generation for OAS 3.x documents
type oas3CodeGenerator struct {
	baseCodeGenerator
	doc *parser.OAS3Document
}

func newOAS3CodeGenerator(g *Generator, doc *parser.OAS3Document, result *GenerateResult) *oas3CodeGenerator {
	cg := &oas3CodeGenerator{
		doc: doc,
	}
	cg.initBase(g, result)

	// Analyze document for file splitting
	splitter := &FileSplitter{
		MaxLinesPerFile:      g.MaxLinesPerFile,
		MaxTypesPerFile:      g.MaxTypesPerFile,
		MaxOperationsPerFile: g.MaxOperationsPerFile,
		SplitByTag:           g.SplitByTag,
		SplitByPathPrefix:    g.SplitByPathPrefix,
	}
	cg.splitPlan = splitter.AnalyzeOAS3(doc)

	// Populate base fields for shared methods
	cg.paths = doc.Paths
	cg.oasVersion = doc.OASVersion
	cg.httpMethods = httpMethods
	cg.statusCodeDataBuilder = cg.buildStatusCodeData
	cg.binderOperationDataBuilder = cg.buildBinderOperationData

	return cg
}

// generateTypes generates type definitions from schemas
func (cg *oas3CodeGenerator) generateTypes() error {
	// Check if we should split into multiple files
	if cg.splitPlan != nil && cg.splitPlan.NeedsSplit {
		return cg.generateSplitTypes()
	}

	return cg.generateSingleTypes()
}

// generateSingleTypes generates all types in a single file (original behavior)
func (cg *oas3CodeGenerator) generateSingleTypes() error {
	// Build template data
	data := cg.buildTypesFileData()

	// Execute template
	formatted, err := executeTemplate("types.go.tmpl", data)
	if err != nil {
		cg.addIssue("types.go", fmt.Sprintf("failed to execute template: %v", err), SeverityWarning)
		return err
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    "types.go",
		Content: formatted,
	})

	return nil
}

// generateSplitTypes generates types split across multiple files
func (cg *oas3CodeGenerator) generateSplitTypes() error {
	// Build type maps from split plan
	sharedTypes, groupTypes := buildTypeGroupMaps(cg.splitPlan)

	// Get all schemas (with deduplication)
	allSchemas := cg.collectSchemas()

	// Generate shared types file (types.go)
	if err := cg.generateTypesFile("types.go", "Shared types used across multiple operations", allSchemas, sharedTypes); err != nil {
		return err
	}

	// Generate per-group type files
	for _, group := range cg.splitPlan.Groups {
		if group.IsShared {
			continue
		}

		types := groupTypes[group.Name]
		if len(types) == 0 {
			continue
		}

		fileName := fmt.Sprintf("types_%s.go", group.Name)
		comment := fmt.Sprintf("%s types", group.DisplayName)
		if err := cg.generateTypesFile(fileName, comment, allSchemas, types); err != nil {
			cg.addIssue(fileName, fmt.Sprintf("failed to generate: %v", err), SeverityWarning)
		}
	}

	return nil
}

// generateTypesFile generates a types file with only the specified types
func (cg *oas3CodeGenerator) generateTypesFile(fileName, comment string, allSchemas []schemaEntry, includeTypes map[string]bool) error {
	// Build filtered types list (pre-allocate with reasonable capacity)
	filteredSchemas := make([]schemaEntry, 0, len(includeTypes))
	for _, entry := range allSchemas {
		typeName := toTypeName(entry.name)
		if includeTypes[typeName] || includeTypes[entry.name] {
			filteredSchemas = append(filteredSchemas, entry)
		}
	}

	if len(filteredSchemas) == 0 {
		return nil // Skip empty files
	}

	// Build template data for filtered types
	data := cg.buildTypesFileDataForSchemas(filteredSchemas, comment)

	// Execute template
	formatted, err := executeTemplate("types.go.tmpl", data)
	if err != nil {
		cg.addIssue(fileName, fmt.Sprintf("failed to execute template: %v", err), SeverityWarning)
		return err
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    fileName,
		Content: formatted,
	})

	return nil
}

// buildTypesFileDataForSchemas builds template data for a specific set of schemas
func (cg *oas3CodeGenerator) buildTypesFileDataForSchemas(schemas []schemaEntry, comment string) *TypesFileData {
	data := &TypesFileData{}

	// Build header
	imports := make(map[string]bool)
	for _, entry := range schemas {
		if needsTimeImport(entry.schema) {
			imports["time"] = true
		}
		if hasDiscriminator(entry.schema) {
			imports["encoding/json"] = true
		}
	}

	importList := make([]string, 0, len(imports))
	for imp := range imports {
		importList = append(importList, imp)
	}
	sort.Strings(importList)

	data.Header = HeaderData{
		PackageName: cg.result.PackageName,
		Imports:     importList,
		Comment:     comment,
	}

	// Build type definitions
	for _, entry := range schemas {
		typeDef := cg.buildTypeDefinition(entry.name, entry.schema)
		data.Types = append(data.Types, typeDef)
		cg.result.GeneratedTypes++
	}

	return data
}

// schemaEntry holds a schema name and its definition
type schemaEntry struct {
	name   string
	schema *parser.Schema
}

// collectSchemas collects and deduplicates schemas from components.
// It tracks generated types to avoid duplicates and populates schemaNames map.
func (cg *oas3CodeGenerator) collectSchemas() []schemaEntry {
	if cg.doc.Components == nil || cg.doc.Components.Schemas == nil {
		return nil
	}

	schemas := make([]schemaEntry, 0, len(cg.doc.Components.Schemas))
	for name, schema := range cg.doc.Components.Schemas {
		if schema == nil {
			continue
		}
		// Check for duplicate type names (e.g., "user_profile" and "UserProfile" both become "UserProfile")
		typeName := toTypeName(name)
		if cg.generatedTypes[typeName] {
			cg.addIssue(fmt.Sprintf("components.schemas.%s", name),
				fmt.Sprintf("duplicate type name %s - skipping", typeName), SeverityWarning)
			continue
		}
		cg.generatedTypes[typeName] = true

		schemas = append(schemas, schemaEntry{name: name, schema: schema})
		cg.schemaNames["#/components/schemas/"+name] = typeName
	}

	// Sort for deterministic output
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].name < schemas[j].name
	})

	return schemas
}

// schemaToGoType converts a schema to a Go type string
func (cg *oas3CodeGenerator) schemaToGoType(schema *parser.Schema, required bool) string {
	// OAS 3.x handles nullable via schema.Nullable or type array with null
	isNullable := schema != nil && (schema.Nullable || schemautil.IsNullable(schema))
	return cg.schemaToGoTypeBase(schema, required, isNullable, cg.schemaToGoType)
}

// getArrayItemType extracts the Go type for array items, handling $ref properly
func (cg *oas3CodeGenerator) getArrayItemType(schema *parser.Schema) string {
	return cg.baseCodeGenerator.getArrayItemType(schema, cg.schemaToGoType)
}

// buildValidateTag builds a validate tag from schema constraints
func (cg *oas3CodeGenerator) buildValidateTag(schema *parser.Schema, required bool) string {
	if schema == nil {
		return ""
	}

	var parts []string

	if required {
		parts = append(parts, "required")
	}

	schemaType := getSchemaType(schema)

	// String constraints
	if schemaType == "string" {
		if schema.MinLength != nil && *schema.MinLength > 0 {
			parts = append(parts, fmt.Sprintf("min=%d", *schema.MinLength))
		}
		if schema.MaxLength != nil {
			parts = append(parts, fmt.Sprintf("max=%d", *schema.MaxLength))
		}
		if schema.Pattern != "" {
			// Note: complex patterns may need escaping
			parts = append(parts, "regexp")
		}
		if schema.Format == "email" {
			parts = append(parts, "email")
		}
		if schema.Format == "uri" || schema.Format == "url" {
			parts = append(parts, "url")
		}
	}

	// Numeric constraints
	if schemaType == "integer" || schemaType == "number" {
		if schema.Minimum != nil {
			isExclusive := false
			if schema.ExclusiveMinimum != nil {
				if b, ok := schema.ExclusiveMinimum.(bool); ok && b {
					isExclusive = true
				}
			}
			if isExclusive {
				parts = append(parts, fmt.Sprintf("gt=%v", *schema.Minimum))
			} else {
				parts = append(parts, fmt.Sprintf("gte=%v", *schema.Minimum))
			}
		}
		if schema.Maximum != nil {
			isExclusive := false
			if schema.ExclusiveMaximum != nil {
				if b, ok := schema.ExclusiveMaximum.(bool); ok && b {
					isExclusive = true
				}
			}
			if isExclusive {
				parts = append(parts, fmt.Sprintf("lt=%v", *schema.Maximum))
			} else {
				parts = append(parts, fmt.Sprintf("lte=%v", *schema.Maximum))
			}
		}
	}

	// Array constraints
	if schemaType == "array" {
		if schema.MinItems != nil && *schema.MinItems > 0 {
			parts = append(parts, fmt.Sprintf("min=%d", *schema.MinItems))
		}
		if schema.MaxItems != nil {
			parts = append(parts, fmt.Sprintf("max=%d", *schema.MaxItems))
		}
	}

	// Enum constraint
	if len(schema.Enum) > 0 {
		var enumVals []string
		for _, e := range schema.Enum {
			enumVals = append(enumVals, fmt.Sprintf("%v", e))
		}
		parts = append(parts, "oneof="+strings.Join(enumVals, " "))
	}

	return strings.Join(parts, ",")
}

// generateClient generates HTTP client code
func (cg *oas3CodeGenerator) generateClient() error {
	// Check if we should split into multiple files
	if cg.splitPlan != nil && cg.splitPlan.NeedsSplit {
		return cg.generateSplitClient()
	}

	return cg.generateSingleClient()
}

// generateSingleClient generates all client code in a single file (original behavior)
func (cg *oas3CodeGenerator) generateSingleClient() error {
	var buf bytes.Buffer

	// Check if we need the time import
	needsTime := cg.operationsNeedTimeImport()

	// Write header
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n\n")
	fmt.Fprintf(&buf, "package %s\n\n", cg.result.PackageName)

	// Write imports
	buf.WriteString("import (\n")
	buf.WriteString("\t\"bytes\"\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"encoding/json\"\n")
	buf.WriteString("\t\"fmt\"\n")
	buf.WriteString("\t\"io\"\n")
	buf.WriteString("\t\"net/http\"\n")
	buf.WriteString("\t\"net/url\"\n")
	buf.WriteString("\t\"strings\"\n")
	if needsTime {
		buf.WriteString("\t\"time\"\n")
	}
	buf.WriteString(")\n\n")

	// Write client struct, types, constructor, and options using shared boilerplate
	writeClientBoilerplate(&buf, cg.doc.Info)

	// Generate methods for each operation
	if cg.doc.Paths != nil {
		// Sort paths for deterministic output
		pathKeys := maputil.SortedKeys(cg.doc.Paths)

		for _, path := range pathKeys {
			pathItem := cg.doc.Paths[path]
			if pathItem == nil {
				continue
			}

			operations := parser.GetOperations(pathItem, cg.doc.OASVersion)
			for _, method := range httpMethods {
				op := operations[method]
				if op == nil {
					continue
				}

				code, err := cg.generateClientMethod(path, method, op)
				if err != nil {
					cg.addIssue(fmt.Sprintf("paths.%s.%s", path, method), fmt.Sprintf("failed to generate client method: %v", err), SeverityWarning)
					continue
				}
				buf.WriteString(code)
				cg.result.GeneratedOperations++
			}
		}
	}

	// Write helper functions
	buf.WriteString(clientHelpers)

	// Format and append the file
	appendFormattedFile(cg.result, "client.go", &buf, cg.addIssue)

	return nil
}

// generateSplitClient generates client code split across multiple files
func (cg *oas3CodeGenerator) generateSplitClient() error {
	// Generate the base client.go (struct, constructor, options, helpers - no operations)
	if err := cg.generateBaseClient(); err != nil {
		return err
	}

	// Build a map of operation ID to path/method for quick lookup
	opToPathMethod := buildOperationMap(cg.doc.Paths, cg.doc.OASVersion)

	// Generate a client file for each group
	for _, group := range cg.splitPlan.Groups {
		if group.IsShared {
			continue // Skip shared types group
		}

		if err := cg.generateClientGroupFile(group, opToPathMethod); err != nil {
			cg.addIssue(fmt.Sprintf("client_%s.go", group.Name), fmt.Sprintf("failed to generate: %v", err), SeverityWarning)
		}
	}

	return nil
}

// generateBaseClient generates the base client.go with struct, constructor, and options (no operations)
func (cg *oas3CodeGenerator) generateBaseClient() error {
	return generateBaseClientShared(cg.result.PackageName, cg.doc.Info, cg.result, cg.addIssue)
}

// generateClientGroupFile generates a client_{group}.go file with operations for a specific group
//
//nolint:unparam // error return kept for API consistency with other generate methods
func (cg *oas3CodeGenerator) generateClientGroupFile(group FileGroup, opToPathMethod map[string]OperationMapping) error {
	var buf bytes.Buffer

	// Write header with comment about the group
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n")
	fmt.Fprintf(&buf, "// This file contains %s operations.\n\n", group.DisplayName)
	fmt.Fprintf(&buf, "package %s\n\n", cg.result.PackageName)

	// Determine needed imports based on operations
	needsTime := false
	needsBytes := false
	needsJSON := false
	needsURL := false

	for _, opID := range group.Operations {
		info, ok := opToPathMethod[opID]
		if !ok {
			continue
		}
		op := info.Op

		// Check if any parameters need time
		for _, param := range op.Parameters {
			if param != nil && param.Schema != nil {
				if param.Schema.Format == "date-time" || param.Schema.Format == "date" {
					needsTime = true
				}
			}
		}

		// Request body means we need bytes and json
		if op.RequestBody != nil {
			needsBytes = true
			needsJSON = true
		}

		// Response parsing needs json
		if op.Responses != nil {
			needsJSON = true
		}

		// Query params need url
		for _, param := range op.Parameters {
			if param != nil && param.In == "query" {
				needsURL = true
			}
		}
	}

	// Write imports
	buf.WriteString("import (\n")
	if needsBytes {
		buf.WriteString("\t\"bytes\"\n")
	}
	buf.WriteString("\t\"context\"\n")
	if needsJSON {
		buf.WriteString("\t\"encoding/json\"\n")
	}
	buf.WriteString("\t\"fmt\"\n")
	buf.WriteString("\t\"io\"\n")
	buf.WriteString("\t\"net/http\"\n")
	if needsURL {
		buf.WriteString("\t\"net/url\"\n")
	}
	buf.WriteString("\t\"strings\"\n")
	if needsTime {
		buf.WriteString("\t\"time\"\n")
	}
	buf.WriteString(")\n\n")

	// Generate each operation in this group using shared helper
	generateGroupClientMethods(&buf, group, opToPathMethod, cg.result, cg.addIssue, cg.generateClientMethod)

	// Format and append the file
	fileName := fmt.Sprintf("client_%s.go", group.Name)
	appendFormattedFile(cg.result, fileName, &buf, cg.addIssue)

	return nil
}

// generateClientMethod generates a client method for an operation
//
//nolint:unparam // error return kept for API consistency with interface requirements
func (cg *oas3CodeGenerator) generateClientMethod(path, method string, op *parser.Operation) (string, error) {
	var buf bytes.Buffer

	methodName := operationToMethodName(op, path, method)

	// Build parameter list
	var params []string
	params = append(params, "ctx context.Context")

	// Process all parameters in a single pass
	var pathParams []pathParam
	var queryParams []*parser.Parameter
	for _, param := range op.Parameters {
		if param == nil {
			continue
		}
		switch param.In {
		case parser.ParamInPath:
			goType := cg.paramToGoType(param)
			paramName := toParamName(param.Name)
			params = append(params, fmt.Sprintf("%s %s", paramName, goType))
			pathParams = append(pathParams, pathParam{name: param.Name, varName: paramName})
		case parser.ParamInQuery:
			queryParams = append(queryParams, param)
		}
	}
	if len(queryParams) > 0 {
		params = append(params, "params *"+methodName+"Params")
	}

	// Request body
	hasBody := op.RequestBody != nil
	contentType := "application/json" // default
	if hasBody {
		bodyType := cg.getRequestBodyType(op.RequestBody)
		params = append(params, "body "+bodyType)
		contentType = cg.getRequestBodyContentType(op.RequestBody)
	}

	// Generate method using shared helpers
	responseType := cg.getResponseType(op)

	writeClientMethod(&buf, op, methodName, method, path, params, pathParams, queryParams,
		hasBody, contentType, responseType, cg.paramToGoType)

	return buf.String(), nil
}

// paramToGoType converts a parameter to its Go type
func (cg *oas3CodeGenerator) paramToGoType(param *parser.Parameter) string {
	if param.Schema != nil {
		return cg.schemaToGoType(param.Schema, param.Required)
	}
	// Fallback for OAS 2.0 style parameters
	return paramTypeToGoType(param.Type, param.Format)
}

// getRequestBodyType determines the Go type for a request body
func (cg *oas3CodeGenerator) getRequestBodyType(rb *parser.RequestBody) string {
	if rb == nil {
		return "any"
	}
	// Look for JSON content type
	for contentType, mediaType := range rb.Content {
		if strings.Contains(contentType, "json") && mediaType != nil && mediaType.Schema != nil {
			return cg.schemaToGoType(mediaType.Schema, true)
		}
	}
	return "any"
}

// getRequestBodyContentType returns the primary content type for a request body
func (cg *oas3CodeGenerator) getRequestBodyContentType(rb *parser.RequestBody) string {
	if rb == nil || rb.Content == nil {
		return "application/json"
	}
	// Prefer JSON content types
	for contentType := range rb.Content {
		if strings.Contains(contentType, "json") {
			return contentType
		}
	}
	// Fall back to first available content type
	for contentType := range rb.Content {
		return contentType
	}
	return "application/json"
}

// findJSONSchemaType searches a content map for a JSON schema and returns its Go type.
// Returns the type string and true if found, or empty string and false if not found.
func (cg *oas3CodeGenerator) findJSONSchemaType(content map[string]*parser.MediaType) (string, bool) {
	for contentType, mediaType := range content {
		if strings.Contains(contentType, "json") && mediaType != nil && mediaType.Schema != nil {
			goType := cg.schemaToGoType(mediaType.Schema, true)
			if !strings.HasPrefix(goType, "*") && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map") {
				return "*" + goType, true
			}
			return goType, true
		}
	}
	return "", false
}

// getResponseType determines the Go type for the success response
func (cg *oas3CodeGenerator) getResponseType(op *parser.Operation) string {
	if op.Responses == nil {
		return httpResponseType
	}

	// Check for 200, 201, 2XX responses
	for _, code := range []string{"200", "201", "2XX"} {
		if resp := op.Responses.Codes[code]; resp != nil {
			if goType, found := cg.findJSONSchemaType(resp.Content); found {
				return goType
			}
		}
	}

	// Check default response
	if op.Responses.Default != nil {
		if goType, found := cg.findJSONSchemaType(op.Responses.Default.Content); found {
			return goType
		}
	}

	return httpResponseType
}

// generateServer generates server interface code
func (cg *oas3CodeGenerator) generateServer() error {
	// Check if we should split into multiple files
	if cg.splitPlan != nil && cg.splitPlan.NeedsSplit {
		return cg.generateSplitServer()
	}

	return cg.generateSingleServer()
}

// generateSingleServer generates all server code in a single file (original behavior)
func (cg *oas3CodeGenerator) generateSingleServer() error {
	// Use shared function with request type callback for single-file mode
	_, err := generateBaseServerShared(&baseServerContext{
		paths:                   cg.doc.Paths,
		oasVersion:              cg.doc.OASVersion,
		httpMethods:             httpMethods,
		packageName:             cg.result.PackageName,
		needsTime:               cg.operationsNeedTimeImport(),
		result:                  cg.result,
		addIssue:                cg.addIssue,
		generateMethodSignature: cg.generateServerMethodSignature,
		getResponseType:         cg.getResponseType,
		generateRequestTypes:    cg.writeRequestTypes,
		schemaTypes:             cg.generatedTypes,
	})
	return err
}

// writeRequestTypes generates request types for all operations.
// This is used as a callback for single-file server generation.
func (cg *oas3CodeGenerator) writeRequestTypes(buf *bytes.Buffer, generatedMethods map[string]bool) {
	for _, path := range maputil.SortedKeys(cg.doc.Paths) {
		pathItem := cg.doc.Paths[path]
		if pathItem == nil {
			continue
		}

		operations := parser.GetOperations(pathItem, cg.doc.OASVersion)
		for _, method := range httpMethods {
			op := operations[method]
			if op == nil {
				continue
			}

			methodName := operationToMethodName(op, path, method)
			// Skip if method was not added to interface (was filtered as duplicate)
			if !generatedMethods[methodName] {
				continue
			}

			reqType := cg.generateRequestType(path, method, op)
			if reqType != "" {
				buf.WriteString(reqType)
			}
		}
	}
}

// generateSplitServer generates server code split across multiple files
func (cg *oas3CodeGenerator) generateSplitServer() error {
	return generateSplitServerShared(&splitServerContext{
		paths:               cg.doc.Paths,
		oasVersion:          cg.doc.OASVersion,
		splitPlan:           cg.splitPlan,
		result:              cg.result,
		addIssue:            cg.addIssue,
		generateBaseServer:  cg.generateBaseServer,
		generateRequestType: cg.generateRequestType,
	})
}

// generateBaseServer generates the base server.go with interface and unimplemented struct.
// Returns a map of generated method names (to exclude duplicates from request type generation).
//
//nolint:unparam // error return kept for API consistency and future extensibility
func (cg *oas3CodeGenerator) generateBaseServer() (map[string]bool, error) {
	return generateBaseServerShared(&baseServerContext{
		paths:                   cg.doc.Paths,
		oasVersion:              cg.doc.OASVersion,
		httpMethods:             httpMethods,
		packageName:             cg.result.PackageName,
		needsTime:               cg.operationsNeedTimeImport(),
		schemaTypes:             cg.generatedTypes,
		result:                  cg.result,
		addIssue:                cg.addIssue,
		generateMethodSignature: cg.generateServerMethodSignature,
		getResponseType:         cg.getResponseType,
	})
}

// generateServerMethodSignature generates the interface method signature
func (cg *oas3CodeGenerator) generateServerMethodSignature(path, method string, op *parser.Operation) string {
	return buildServerMethodSignature(path, method, op, cg.getResponseType(op), cg.generatedTypes)
}

// generateRequestType generates a request struct for an operation
func (cg *oas3CodeGenerator) generateRequestType(path, method string, op *parser.Operation) string {
	var buf bytes.Buffer

	methodName := operationToMethodName(op, path, method)
	wrapperName := resolveWrapperName(methodName, cg.generatedTypes)

	fmt.Fprintf(&buf, "// %s contains the request data for %s.\n", wrapperName, methodName)
	fmt.Fprintf(&buf, "type %s struct {\n", wrapperName)

	// Categorize parameters in a single pass
	var pathParams, queryParams, headerParams, cookieParams []*parser.Parameter
	for _, param := range op.Parameters {
		if param == nil {
			continue
		}
		switch param.In {
		case parser.ParamInPath:
			pathParams = append(pathParams, param)
		case parser.ParamInQuery:
			queryParams = append(queryParams, param)
		case parser.ParamInHeader:
			headerParams = append(headerParams, param)
		case parser.ParamInCookie:
			cookieParams = append(cookieParams, param)
		}
	}

	// Path parameters
	for _, param := range pathParams {
		goType := cg.paramToGoType(param)
		fieldName := toFieldName(param.Name)
		fmt.Fprintf(&buf, "\t%s %s\n", fieldName, goType)
	}

	// Query parameters - paramToGoType already handles pointer for optional params
	for _, param := range queryParams {
		goType := cg.paramToGoType(param)
		fieldName := toFieldName(param.Name)
		fmt.Fprintf(&buf, "\t%s %s\n", fieldName, goType)
	}

	// Header parameters - paramToGoType already handles pointer for optional params
	for _, param := range headerParams {
		goType := cg.paramToGoType(param)
		fieldName := toFieldName(param.Name)
		fmt.Fprintf(&buf, "\t%s %s\n", fieldName, goType)
	}

	// Cookie parameters - paramToGoType already handles pointer for optional params
	for _, param := range cookieParams {
		goType := cg.paramToGoType(param)
		fieldName := toFieldName(param.Name)
		fmt.Fprintf(&buf, "\t%s %s\n", fieldName, goType)
	}

	// Request body
	if op.RequestBody != nil {
		bodyType := cg.getRequestBodyType(op.RequestBody)
		fmt.Fprintf(&buf, "\tBody %s\n", bodyType)
	}

	// HTTP request
	buf.WriteString("\tHTTPRequest *http.Request\n")

	buf.WriteString("}\n\n")

	return buf.String()
}

// httpMethods returns all HTTP methods in a consistent order
var httpMethods = []string{
	httputil.MethodGet,
	httputil.MethodPut,
	httputil.MethodPost,
	httputil.MethodDelete,
	httputil.MethodOptions,
	httputil.MethodHead,
	httputil.MethodPatch,
	httputil.MethodTrace,
	httputil.MethodQuery,
}

// operationsNeedTimeImport checks if any operation parameters or responses use time.Time
func (cg *oas3CodeGenerator) operationsNeedTimeImport() bool {
	if cg.doc.Paths == nil {
		return false
	}

	for _, pathItem := range cg.doc.Paths {
		if pathItem == nil {
			continue
		}

		operations := parser.GetOperations(pathItem, cg.doc.OASVersion)
		for _, op := range operations {
			if op == nil {
				continue
			}

			// Check parameters
			for _, param := range op.Parameters {
				if param == nil || param.Schema == nil {
					continue
				}
				if needsTimeImport(param.Schema) {
					return true
				}
			}

			// Check request body
			if op.RequestBody != nil {
				for _, mediaType := range op.RequestBody.Content {
					if mediaType != nil && mediaType.Schema != nil && needsTimeImport(mediaType.Schema) {
						return true
					}
				}
			}

			// Check responses
			if op.Responses != nil {
				for _, resp := range op.Responses.Codes {
					if resp == nil {
						continue
					}
					for _, mediaType := range resp.Content {
						if mediaType != nil && mediaType.Schema != nil && needsTimeImport(mediaType.Schema) {
							return true
						}
					}
				}
				if op.Responses.Default != nil {
					for _, mediaType := range op.Responses.Default.Content {
						if mediaType != nil && mediaType.Schema != nil && needsTimeImport(mediaType.Schema) {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

// Helper constants
const clientHelpers = `
// APIError represents an API error response.
type APIError struct {
	StatusCode int
	Body       []byte
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error: status %d: %s", e.StatusCode, string(e.Body))
}
`

// generateSecurityHelpers generates security helper code based on configuration
func (cg *oas3CodeGenerator) generateSecurityHelpers() {
	var schemes map[string]*parser.SecurityScheme
	if cg.doc.Components != nil {
		schemes = cg.doc.Components.SecuritySchemes
	}
	generateAllSecurityHelpers(cg.g, schemes, fullSecurityCallbacks{
		generateSecurityHelpersFile: cg.generateSecurityHelpersFile,
		generateOAuth2Files:         cg.generateOAuth2Files,
		generateCredentials:         cg.generateCredentialsFile,
		generateSecurityEnforce:     cg.generateSecurityEnforceFile,
		generateOIDCDiscovery:       cg.generateOIDCDiscoveryFile,
		generateReadme:              cg.generateReadmeFile,
	})
}

// generateSecurityEnforceFile generates security enforcement code.
// If file splitting is enabled and needed, generates multiple files.
func (cg *oas3CodeGenerator) generateSecurityEnforceFile() {
	// Check if we should split
	if cg.splitPlan != nil && cg.splitPlan.NeedsSplit {
		cg.generateSplitSecurityEnforce()
		return
	}
	cg.generateSingleSecurityEnforce()
}

// generateSingleSecurityEnforce generates all security enforcement in a single file.
func (cg *oas3CodeGenerator) generateSingleSecurityEnforce() {
	opSecurity := ExtractOperationSecurityOAS3(cg.doc)
	generateSingleSecurityEnforceShared(cg.securityContext(), opSecurity, cg.doc.Security)
}

// generateSplitSecurityEnforce generates security enforcement split across multiple files.
func (cg *oas3CodeGenerator) generateSplitSecurityEnforce() {
	opSecurity := ExtractOperationSecurityOAS3(cg.doc)
	generateSplitSecurityEnforceShared(cg.securityContext(), opSecurity, cg.doc.Security)
}

// generateReadmeFile generates the README.md file
func (cg *oas3CodeGenerator) generateReadmeFile(schemes map[string]*parser.SecurityScheme) {
	g := NewReadmeGenerator()

	// Build version-specific security scheme summaries
	secSummaries := buildSecuritySchemeSummariesOAS3(schemes)

	// Build context using shared helper
	builder := &readmeContextBuilder{
		PackageName: cg.result.PackageName,
		OASVersion:  cg.doc.OpenAPI, // Use the OpenAPI version string field
		Config:      cg.g,
		SplitPlan:   cg.splitPlan,
		Files:       cg.result.Files,
		Info:        cg.doc.Info,
	}
	ctx := buildReadmeContextShared(builder, secSummaries)

	content := g.GenerateReadme(ctx)

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    "README.md",
		Content: []byte(content),
	})
}

// getFileDescription returns a description for a generated file
func getFileDescription(name string) string {
	switch name {
	case "types.go":
		return "Data types and models"
	case "client.go":
		return "HTTP client implementation"
	case "server.go":
		return "Server interface"
	case "security_helpers.go":
		return "Security authentication helpers"
	case "credentials.go":
		return "Credential management"
	case "security_enforce.go":
		return "Security enforcement and validation"
	case "oidc_discovery.go":
		return "OpenID Connect discovery client"
	default:
		if strings.HasPrefix(name, "oauth2_") {
			return "OAuth2 token flow management"
		}
		return "Generated code"
	}
}

// toFileName converts a name to a valid Go file name.
// It lowercases, replaces hyphens/spaces with underscores, and strips all
// characters outside the [a-z0-9_] allowlist to prevent path traversal.
func toFileName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	// Strip all characters except [a-z0-9_]
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// convertFlows converts parser.OAuthFlows to readme generator OAuthFlows
func convertFlows(flows *parser.OAuthFlows) *OAuthFlows {
	if flows == nil {
		return nil
	}

	result := &OAuthFlows{}

	if flows.Implicit != nil {
		result.Implicit = &OAuthFlow{
			AuthorizationURL: flows.Implicit.AuthorizationURL,
			Scopes:           flows.Implicit.Scopes,
		}
	}
	if flows.Password != nil {
		result.Password = &OAuthFlow{
			TokenURL: flows.Password.TokenURL,
			Scopes:   flows.Password.Scopes,
		}
	}
	if flows.ClientCredentials != nil {
		result.ClientCredentials = &OAuthFlow{
			TokenURL: flows.ClientCredentials.TokenURL,
			Scopes:   flows.ClientCredentials.Scopes,
		}
	}
	if flows.AuthorizationCode != nil {
		result.AuthorizationCode = &OAuthFlow{
			AuthorizationURL: flows.AuthorizationCode.AuthorizationURL,
			TokenURL:         flows.AuthorizationCode.TokenURL,
			RefreshURL:       flows.AuthorizationCode.RefreshURL,
			Scopes:           flows.AuthorizationCode.Scopes,
		}
	}

	return result
}

// buildBinderOperationData builds binding data for a single operation
func (cg *oas3CodeGenerator) buildBinderOperationData(methodName string, op *parser.Operation) BinderOperationData {
	opData := BinderOperationData{
		MethodName:  methodName,
		RequestType: resolveWrapperName(methodName, cg.generatedTypes),
	}

	// Process parameters
	for _, param := range op.Parameters {
		if param == nil {
			continue
		}

		paramData := ParamBindData{
			Name:      param.Name,
			FieldName: toFieldName(param.Name),
			GoType:    cg.paramToGoType(param),
			Required:  param.Required,
			IsPointer: !param.Required && param.In != parser.ParamInPath,
		}

		// Determine schema type
		if param.Schema != nil {
			paramData.SchemaType = cg.getSchemaType(param.Schema)
		}

		switch param.In {
		case parser.ParamInPath:
			opData.PathParams = append(opData.PathParams, paramData)
		case parser.ParamInQuery:
			opData.QueryParams = append(opData.QueryParams, paramData)
		case parser.ParamInHeader:
			opData.HeaderParams = append(opData.HeaderParams, paramData)
		case parser.ParamInCookie:
			opData.CookieParams = append(opData.CookieParams, paramData)
		}
	}

	// Check for request body
	if op.RequestBody != nil {
		opData.HasBody = true
		if op.RequestBody.Content != nil {
			for _, mediaType := range op.RequestBody.Content {
				if mediaType != nil && mediaType.Schema != nil {
					opData.BodyType = cg.schemaToGoType(mediaType.Schema, true)
					break
				}
			}
		}
	}

	return opData
}

// getSchemaType returns the basic schema type string
func (cg *oas3CodeGenerator) getSchemaType(schema *parser.Schema) string {
	if schema == nil {
		return "string"
	}

	// Handle type as string or array
	switch t := schema.Type.(type) {
	case string:
		return t
	case []any:
		if len(t) > 0 {
			if s, ok := t[0].(string); ok {
				return s
			}
		}
	case []string:
		if len(t) > 0 {
			return t[0]
		}
	}

	return "string"
}

// buildStatusCodeData builds data for a single status code response
func (cg *oas3CodeGenerator) buildStatusCodeData(code string, resp *parser.Response) StatusCodeData {
	// Parse status code metadata using shared helper
	statusData := parseStatusCodeMetadata(code)
	statusData.Description = resp.Description

	// OAS 3.x: Determine body type from response content map
	if resp.Content != nil {
		for contentType, mediaType := range resp.Content {
			if mediaType != nil && mediaType.Schema != nil {
				statusData.HasBody = true
				statusData.ContentType = contentType
				statusData.BodyType = cg.schemaToGoType(mediaType.Schema, true)
				break // Use first content type
			}
		}
	}

	return statusData
}

// generateServerRouter generates HTTP router code
func (cg *oas3CodeGenerator) generateServerRouter() error {
	return generateServerRouterShared(&serverRouterContext{
		paths:        cg.doc.Paths,
		oasVersion:   cg.doc.OASVersion,
		httpMethods:  httpMethods,
		packageName:  cg.result.PackageName,
		serverRouter: cg.g.ServerRouter,
		schemaTypes:  cg.generatedTypes,
		result:       cg.result,
		addIssue:     cg.addIssue,
		paramToBindData: func(param *parser.Parameter) ParamBindData {
			data := ParamBindData{
				Name:      param.Name,
				FieldName: toFieldName(param.Name),
				GoType:    cg.paramToGoType(param),
				Required:  param.Required,
			}
			if param.Schema != nil {
				data.SchemaType = cg.getSchemaType(param.Schema)
			}
			return data
		},
	})
}
