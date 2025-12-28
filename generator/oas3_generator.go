package generator

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/erraggy/oastools/internal/httputil"
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
	// Build set of shared types
	sharedTypes := make(map[string]bool)
	for _, typeName := range cg.splitPlan.SharedTypes {
		sharedTypes[typeName] = true
	}

	// Build set of types per group
	groupTypes := make(map[string]map[string]bool)
	for _, group := range cg.splitPlan.Groups {
		if group.IsShared {
			continue
		}
		groupTypes[group.Name] = make(map[string]bool)
		for _, typeName := range group.Types {
			groupTypes[group.Name][typeName] = true
		}
	}

	// Get all schemas (with deduplication)
	schemaCount := 0
	if cg.doc.Components != nil && cg.doc.Components.Schemas != nil {
		schemaCount = len(cg.doc.Components.Schemas)
	}
	allSchemas := make([]schemaEntry, 0, schemaCount)
	if cg.doc.Components != nil && cg.doc.Components.Schemas != nil {
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

			allSchemas = append(allSchemas, schemaEntry{name: name, schema: schema})
			cg.schemaNames["#/components/schemas/"+name] = typeName
		}
	}
	sort.Slice(allSchemas, func(i, j int) bool {
		return allSchemas[i].name < allSchemas[j].name
	})

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
	buf.WriteString(fmt.Sprintf("package %s\n\n", cg.result.PackageName))

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
		var pathKeys []string
		for path := range cg.doc.Paths {
			pathKeys = append(pathKeys, path)
		}
		sort.Strings(pathKeys)

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

	// Format the code
	formatted, err := formatAndFixImports("generated.go", buf.Bytes())
	if err != nil {
		cg.addIssue("client.go", fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = buf.Bytes()
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    "client.go",
		Content: formatted,
	})

	return nil
}

// generateSplitClient generates client code split across multiple files
func (cg *oas3CodeGenerator) generateSplitClient() error {
	// Generate the base client.go (struct, constructor, options, helpers - no operations)
	if err := cg.generateBaseClient(); err != nil {
		return err
	}

	// Build a map of operation ID to path/method for quick lookup
	opToPathMethod := make(map[string]struct {
		path   string
		method string
		op     *parser.Operation
	})

	if cg.doc.Paths != nil {
		for path, pathItem := range cg.doc.Paths {
			if pathItem == nil {
				continue
			}
			operations := parser.GetOperations(pathItem, cg.doc.OASVersion)
			for method, op := range operations {
				if op == nil {
					continue
				}
				opID := operationToMethodName(op, path, method)
				opToPathMethod[opID] = struct {
					path   string
					method string
					op     *parser.Operation
				}{path, method, op}
			}
		}
	}

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
	var buf bytes.Buffer

	// Write header
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", cg.result.PackageName))

	// Write imports (base client needs these imports for clientHelpers)
	buf.WriteString("import (\n")
	buf.WriteString("\t\"bytes\"\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"encoding/json\"\n")
	buf.WriteString("\t\"fmt\"\n")
	buf.WriteString("\t\"io\"\n")
	buf.WriteString("\t\"net/http\"\n")
	buf.WriteString("\t\"net/url\"\n")
	buf.WriteString("\t\"strings\"\n")
	buf.WriteString(")\n\n")

	// Write client struct, types, constructor, and options using shared boilerplate
	writeClientBoilerplate(&buf, cg.doc.Info)

	// Write helper functions
	buf.WriteString(clientHelpers)

	// Format the code
	formatted, err := formatAndFixImports("generated.go", buf.Bytes())
	if err != nil {
		cg.addIssue("client.go", fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = buf.Bytes()
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    "client.go",
		Content: formatted,
	})

	return nil
}

// generateClientGroupFile generates a client_{group}.go file with operations for a specific group
//
//nolint:unparam // error return kept for API consistency with other generate methods
func (cg *oas3CodeGenerator) generateClientGroupFile(group FileGroup, opToPathMethod map[string]struct {
	path   string
	method string
	op     *parser.Operation
}) error {
	var buf bytes.Buffer

	// Write header with comment about the group
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n")
	buf.WriteString(fmt.Sprintf("// This file contains %s operations.\n\n", group.DisplayName))
	buf.WriteString(fmt.Sprintf("package %s\n\n", cg.result.PackageName))

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
		op := info.op

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

	// Generate each operation in this group
	for _, opID := range group.Operations {
		info, ok := opToPathMethod[opID]
		if !ok {
			continue
		}

		code, err := cg.generateClientMethod(info.path, info.method, info.op)
		if err != nil {
			cg.addIssue(fmt.Sprintf("paths.%s.%s", info.path, info.method), fmt.Sprintf("failed to generate client method: %v", err), SeverityWarning)
			continue
		}
		buf.WriteString(code)
		cg.result.GeneratedOperations++
	}

	// Format the code
	formatted, err := formatAndFixImports("generated.go", buf.Bytes())
	if err != nil {
		cg.addIssue(fmt.Sprintf("client_%s.go", group.Name), fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = buf.Bytes()
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    fmt.Sprintf("client_%s.go", group.Name),
		Content: formatted,
	})

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

	// Write method documentation - handle multiline descriptions properly
	if op.Summary != "" {
		buf.WriteString(formatMultilineComment(op.Summary, methodName, ""))
	} else if op.Description != "" {
		buf.WriteString(formatMultilineComment(op.Description, methodName, ""))
	} else {
		buf.WriteString(fmt.Sprintf("// %s calls %s %s\n", methodName, strings.ToUpper(method), path))
	}
	if op.Deprecated {
		buf.WriteString("// Deprecated: This operation is deprecated.\n")
	}

	// Write method signature
	responseType := cg.getResponseType(op)
	buf.WriteString(fmt.Sprintf("func (c *Client) %s(%s) (%s, error) {\n", methodName, strings.Join(params, ", "), responseType))

	// Build URL
	buf.WriteString("\tpath := ")
	if len(pathParams) > 0 {
		buf.WriteString("fmt.Sprintf(\"")
		pathTemplate := path
		for _, pp := range pathParams {
			pathTemplate = strings.ReplaceAll(pathTemplate, "{"+pp.name+"}", "%v")
		}
		buf.WriteString(pathTemplate)
		buf.WriteString("\"")
		for _, pp := range pathParams {
			buf.WriteString(", " + pp.varName)
		}
		buf.WriteString(")\n")
	} else {
		buf.WriteString(fmt.Sprintf("%q\n", path))
	}

	// Build query string
	if len(queryParams) > 0 {
		buf.WriteString("\tquery := make(url.Values)\n")
		buf.WriteString("\tif params != nil {\n")
		for _, param := range queryParams {
			paramName := toFieldName(param.Name)
			if param.Required {
				buf.WriteString(fmt.Sprintf("\t\tquery.Set(%q, fmt.Sprintf(\"%%v\", params.%s))\n", param.Name, paramName))
			} else {
				buf.WriteString(fmt.Sprintf("\t\tif params.%s != nil {\n", paramName))
				buf.WriteString(fmt.Sprintf("\t\t\tquery.Set(%q, fmt.Sprintf(\"%%v\", *params.%s))\n", param.Name, paramName))
				buf.WriteString("\t\t}\n")
			}
		}
		buf.WriteString("\t}\n")
		buf.WriteString("\tif len(query) > 0 {\n")
		buf.WriteString("\t\tpath += \"?\" + query.Encode()\n")
		buf.WriteString("\t}\n")
	}

	// Create request
	if hasBody {
		buf.WriteString("\tbodyData, err := json.Marshal(body)\n")
		buf.WriteString("\tif err != nil {\n")
		buf.WriteString(fmt.Sprintf("\t\treturn %s, fmt.Errorf(\"marshal request body: %%w\", err)\n", zeroValue(responseType)))
		buf.WriteString("\t}\n")
		buf.WriteString(fmt.Sprintf("\treq, err := http.NewRequestWithContext(ctx, %q, c.BaseURL+path, bytes.NewReader(bodyData))\n", strings.ToUpper(method)))
	} else {
		buf.WriteString(fmt.Sprintf("\treq, err := http.NewRequestWithContext(ctx, %q, c.BaseURL+path, nil)\n", strings.ToUpper(method)))
	}
	buf.WriteString("\tif err != nil {\n")
	buf.WriteString(fmt.Sprintf("\t\treturn %s, fmt.Errorf(\"create request: %%w\", err)\n", zeroValue(responseType)))
	buf.WriteString("\t}\n")

	// Set content type for requests with body
	if hasBody {
		buf.WriteString(fmt.Sprintf("\treq.Header.Set(\"Content-Type\", %q)\n", contentType))
	}
	buf.WriteString("\treq.Header.Set(\"Accept\", \"application/json\")\n")
	buf.WriteString("\tif c.UserAgent != \"\" {\n")
	buf.WriteString("\t\treq.Header.Set(\"User-Agent\", c.UserAgent)\n")
	buf.WriteString("\t}\n")

	// Apply request editors
	buf.WriteString("\tfor _, editor := range c.RequestEditors {\n")
	buf.WriteString("\t\tif err := editor(ctx, req); err != nil {\n")
	buf.WriteString(fmt.Sprintf("\t\t\treturn %s, fmt.Errorf(\"request editor: %%w\", err)\n", zeroValue(responseType)))
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t}\n")

	// Execute request
	buf.WriteString("\tresp, err := c.HTTPClient.Do(req)\n")
	buf.WriteString("\tif err != nil {\n")
	buf.WriteString(fmt.Sprintf("\t\treturn %s, fmt.Errorf(\"execute request: %%w\", err)\n", zeroValue(responseType)))
	buf.WriteString("\t}\n")
	buf.WriteString("\tdefer resp.Body.Close()\n")

	// Handle response
	buf.WriteString("\tif resp.StatusCode >= 400 {\n")
	buf.WriteString("\t\tbody, _ := io.ReadAll(resp.Body)\n")
	buf.WriteString(fmt.Sprintf("\t\treturn %s, &APIError{StatusCode: resp.StatusCode, Body: body}\n", zeroValue(responseType)))
	buf.WriteString("\t}\n")

	// Parse response body
	if responseType != "" && responseType != httpResponseType {
		if strings.HasPrefix(responseType, "*") {
			buf.WriteString(fmt.Sprintf("\tvar result %s\n", responseType[1:]))
			buf.WriteString("\tif err := json.NewDecoder(resp.Body).Decode(&result); err != nil {\n")
			buf.WriteString(fmt.Sprintf("\t\treturn %s, fmt.Errorf(\"decode response: %%w\", err)\n", zeroValue(responseType)))
			buf.WriteString("\t}\n")
			buf.WriteString("\treturn &result, nil\n")
		} else {
			buf.WriteString(fmt.Sprintf("\tvar result %s\n", responseType))
			buf.WriteString("\tif err := json.NewDecoder(resp.Body).Decode(&result); err != nil {\n")
			buf.WriteString(fmt.Sprintf("\t\treturn %s, fmt.Errorf(\"decode response: %%w\", err)\n", zeroValue(responseType)))
			buf.WriteString("\t}\n")
			buf.WriteString("\treturn result, nil\n")
		}
	} else {
		buf.WriteString("\treturn resp, nil\n")
	}

	buf.WriteString("}\n\n")

	// Generate params struct if needed
	if len(queryParams) > 0 {
		buf.WriteString(fmt.Sprintf("// %sParams contains query parameters for %s.\n", methodName, methodName))
		buf.WriteString(fmt.Sprintf("type %sParams struct {\n", methodName))
		for _, param := range queryParams {
			goType := cg.paramToGoType(param)
			fieldName := toFieldName(param.Name)
			if param.Description != "" {
				buf.WriteString(fmt.Sprintf("\t// %s\n", cleanDescription(param.Description)))
			}
			if !param.Required {
				buf.WriteString(fmt.Sprintf("\t%s *%s `json:%q`\n", fieldName, goType, param.Name+",omitempty"))
			} else {
				buf.WriteString(fmt.Sprintf("\t%s %s `json:%q`\n", fieldName, goType, param.Name))
			}
		}
		buf.WriteString("}\n\n")
	}

	return buf.String(), nil
}

type pathParam struct {
	name    string
	varName string
}

// paramToGoType converts a parameter to its Go type
func (cg *oas3CodeGenerator) paramToGoType(param *parser.Parameter) string {
	if param.Schema != nil {
		return cg.schemaToGoType(param.Schema, param.Required)
	}
	// Fallback for OAS 2.0 style parameters
	switch param.Type {
	case "string":
		return stringFormatToGoType(param.Format)
	case "integer":
		return integerFormatToGoType(param.Format)
	case "number":
		return numberFormatToGoType(param.Format)
	case "boolean":
		return "bool"
	case "array":
		return "[]string"
	default:
		return "string"
	}
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

// getResponseType determines the Go type for the success response
func (cg *oas3CodeGenerator) getResponseType(op *parser.Operation) string {
	if op.Responses == nil {
		return httpResponseType
	}

	// Check for 200, 201, 2XX responses
	for _, code := range []string{"200", "201", "2XX"} {
		if resp := op.Responses.Codes[code]; resp != nil {
			for contentType, mediaType := range resp.Content {
				if strings.Contains(contentType, "json") && mediaType != nil && mediaType.Schema != nil {
					goType := cg.schemaToGoType(mediaType.Schema, true)
					if !strings.HasPrefix(goType, "*") && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map") {
						return "*" + goType
					}
					return goType
				}
			}
		}
	}

	// Check default response
	if op.Responses.Default != nil {
		for contentType, mediaType := range op.Responses.Default.Content {
			if strings.Contains(contentType, "json") && mediaType != nil && mediaType.Schema != nil {
				goType := cg.schemaToGoType(mediaType.Schema, true)
				if !strings.HasPrefix(goType, "*") && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map") {
					return "*" + goType
				}
				return goType
			}
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
	var buf bytes.Buffer

	// Check if we need the time import
	needsTime := cg.operationsNeedTimeImport()

	// Write header
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", cg.result.PackageName))

	// Write imports
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"net/http\"\n")
	if needsTime {
		buf.WriteString("\t\"time\"\n")
	}
	buf.WriteString(")\n\n")

	// Track generated methods to avoid duplicates (can happen with duplicate operationIds).
	// NOTE: This map must be local per file generation to avoid stale data in split mode.
	generatedMethods := make(map[string]bool)

	// Generate server interface
	buf.WriteString("// ServerInterface represents the server API.\n")
	buf.WriteString("type ServerInterface interface {\n")

	if cg.doc.Paths != nil {
		// Sort paths for deterministic output
		var pathKeys []string
		for path := range cg.doc.Paths {
			pathKeys = append(pathKeys, path)
		}
		sort.Strings(pathKeys)

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

				methodName := operationToMethodName(op, path, method)
				if generatedMethods[methodName] {
					cg.addIssue(fmt.Sprintf("paths.%s.%s", path, method),
						fmt.Sprintf("duplicate method name %s - skipping", methodName), SeverityWarning)
					continue
				}
				generatedMethods[methodName] = true

				sig := cg.generateServerMethodSignature(path, method, op)
				buf.WriteString(sig)
			}
		}
	}

	buf.WriteString("}\n\n")

	// Generate request types (use same tracking map since request types are named after methods)
	for _, path := range func() []string {
		var keys []string
		if cg.doc.Paths != nil {
			for k := range cg.doc.Paths {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		return keys
	}() {
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

	// Write unimplemented server
	buf.WriteString("// UnimplementedServer provides default implementations that return errors.\n")
	buf.WriteString("type UnimplementedServer struct{}\n\n")

	// Track generated UnimplementedServer methods separately to avoid duplicates.
	// We can't reuse generatedMethods because it's used to check if a method was
	// added to the interface (i.e., wasn't filtered as duplicate).
	generatedUnimplemented := make(map[string]bool)

	// Generate unimplemented methods (methods already tracked from interface generation)
	if cg.doc.Paths != nil {
		var pathKeys []string
		for path := range cg.doc.Paths {
			pathKeys = append(pathKeys, path)
		}
		sort.Strings(pathKeys)

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

				methodName := operationToMethodName(op, path, method)
				// Skip if not in generated methods (was a duplicate in interface)
				if !generatedMethods[methodName] {
					continue
				}
				// Skip if already generated for UnimplementedServer
				if generatedUnimplemented[methodName] {
					continue
				}
				generatedUnimplemented[methodName] = true

				responseType := cg.getResponseType(op)

				buf.WriteString(fmt.Sprintf("func (s *UnimplementedServer) %s(ctx context.Context, req *%sRequest) (%s, error) {\n",
					methodName, methodName, responseType))
				buf.WriteString(fmt.Sprintf("\treturn %s, ErrNotImplemented\n", zeroValue(responseType)))
				buf.WriteString("}\n\n")
			}
		}
	}

	// Write error type
	buf.WriteString("// ErrNotImplemented is returned by UnimplementedServer methods.\n")
	buf.WriteString("var ErrNotImplemented = &NotImplementedError{}\n\n")
	buf.WriteString("// NotImplementedError indicates an operation is not implemented.\n")
	buf.WriteString("type NotImplementedError struct{}\n\n")
	buf.WriteString("func (e *NotImplementedError) Error() string { return \"not implemented\" }\n\n")

	// Format the code
	formatted, err := formatAndFixImports("generated.go", buf.Bytes())
	if err != nil {
		cg.addIssue("server.go", fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = buf.Bytes()
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    "server.go",
		Content: formatted,
	})

	return nil
}

// generateSplitServer generates server code split across multiple files
func (cg *oas3CodeGenerator) generateSplitServer() error {
	// Generate the base server.go (interface, unimplemented, error types - but no request types)
	// Returns the set of methods that were actually generated (excludes duplicates)
	generatedMethods, err := cg.generateBaseServer()
	if err != nil {
		return err
	}

	// Build a map of operation ID to path/method for quick lookup
	opToPathMethod := make(map[string]struct {
		path   string
		method string
		op     *parser.Operation
	})

	if cg.doc.Paths != nil {
		for path, pathItem := range cg.doc.Paths {
			if pathItem == nil {
				continue
			}
			operations := parser.GetOperations(pathItem, cg.doc.OASVersion)
			for method, op := range operations {
				if op == nil {
					continue
				}
				opID := operationToMethodName(op, path, method)
				opToPathMethod[opID] = struct {
					path   string
					method string
					op     *parser.Operation
				}{path, method, op}
			}
		}
	}

	// Generate a server file for each group (request types only)
	for _, group := range cg.splitPlan.Groups {
		if group.IsShared {
			continue // Skip shared types group
		}

		if err := cg.generateServerGroupFile(group, opToPathMethod, generatedMethods); err != nil {
			cg.addIssue(fmt.Sprintf("server_%s.go", group.Name), fmt.Sprintf("failed to generate: %v", err), SeverityWarning)
		}
	}

	return nil
}

// generateBaseServer generates the base server.go with interface and unimplemented struct.
// Returns a map of generated method names (to exclude duplicates from request type generation).
//
//nolint:unparam // error return kept for API consistency and future extensibility
func (cg *oas3CodeGenerator) generateBaseServer() (map[string]bool, error) {
	var buf bytes.Buffer

	// Check if we need the time import
	needsTime := cg.operationsNeedTimeImport()

	// Write header
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", cg.result.PackageName))

	// Write imports
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	if needsTime {
		buf.WriteString("\t\"time\"\n")
	}
	buf.WriteString(")\n\n")

	// Track generated methods to avoid duplicates (can happen with duplicate operationIds).
	// NOTE: This map must be local per file generation to avoid stale data in split mode.
	generatedMethods := make(map[string]bool)

	// Generate server interface (must be complete)
	buf.WriteString("// ServerInterface represents the server API.\n")
	buf.WriteString("type ServerInterface interface {\n")

	if cg.doc.Paths != nil {
		var pathKeys []string
		for path := range cg.doc.Paths {
			pathKeys = append(pathKeys, path)
		}
		sort.Strings(pathKeys)

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

				methodName := operationToMethodName(op, path, method)
				if generatedMethods[methodName] {
					cg.addIssue(fmt.Sprintf("paths.%s.%s", path, method),
						fmt.Sprintf("duplicate method name %s - skipping", methodName), SeverityWarning)
					continue
				}
				generatedMethods[methodName] = true

				sig := cg.generateServerMethodSignature(path, method, op)
				buf.WriteString(sig)
			}
		}
	}

	buf.WriteString("}\n\n")

	// Write unimplemented server (must be complete)
	buf.WriteString("// UnimplementedServer provides default implementations that return errors.\n")
	buf.WriteString("type UnimplementedServer struct{}\n\n")

	// Track generated UnimplementedServer methods separately to avoid duplicates.
	// We can't reuse generatedMethods because it's used to check if a method was
	// added to the interface (i.e., wasn't filtered as duplicate).
	generatedUnimplemented := make(map[string]bool)

	if cg.doc.Paths != nil {
		var pathKeys []string
		for path := range cg.doc.Paths {
			pathKeys = append(pathKeys, path)
		}
		sort.Strings(pathKeys)

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

				methodName := operationToMethodName(op, path, method)
				// Only generate if the method was added to the interface (not a duplicate)
				if !generatedMethods[methodName] {
					continue
				}
				// Skip if already generated for UnimplementedServer
				if generatedUnimplemented[methodName] {
					continue
				}
				generatedUnimplemented[methodName] = true

				responseType := cg.getResponseType(op)

				buf.WriteString(fmt.Sprintf("func (s *UnimplementedServer) %s(ctx context.Context, req *%sRequest) (%s, error) {\n",
					methodName, methodName, responseType))
				buf.WriteString(fmt.Sprintf("\treturn %s, ErrNotImplemented\n", zeroValue(responseType)))
				buf.WriteString("}\n\n")
			}
		}
	}

	// Write error type
	buf.WriteString("// ErrNotImplemented is returned by UnimplementedServer methods.\n")
	buf.WriteString("var ErrNotImplemented = &NotImplementedError{}\n\n")
	buf.WriteString("// NotImplementedError indicates an operation is not implemented.\n")
	buf.WriteString("type NotImplementedError struct{}\n\n")
	buf.WriteString("func (e *NotImplementedError) Error() string { return \"not implemented\" }\n\n")

	// Format the code
	formatted, err := formatAndFixImports("generated.go", buf.Bytes())
	if err != nil {
		cg.addIssue("server.go", fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = buf.Bytes()
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    "server.go",
		Content: formatted,
	})

	return generatedMethods, nil
}

// generateServerGroupFile generates a server_{group}.go file with request types for a specific group.
// The generatedMethods map indicates which methods were added to the interface (duplicates excluded).
//
//nolint:unparam // error return kept for API consistency with other generate methods
func (cg *oas3CodeGenerator) generateServerGroupFile(group FileGroup, opToPathMethod map[string]struct {
	path   string
	method string
	op     *parser.Operation
}, generatedMethods map[string]bool) error {
	var buf bytes.Buffer

	// Write header with comment about the group
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n")
	buf.WriteString(fmt.Sprintf("// This file contains %s server request types.\n\n", group.DisplayName))
	buf.WriteString(fmt.Sprintf("package %s\n\n", cg.result.PackageName))

	// Determine needed imports based on operations (only for non-duplicate methods)
	needsTime := false
	needsHTTP := false

	for _, opID := range group.Operations {
		// Skip if method was not added to interface (was filtered as duplicate)
		if !generatedMethods[opID] {
			continue
		}

		info, ok := opToPathMethod[opID]
		if !ok {
			continue
		}
		op := info.op

		// Check if any parameters need time
		for _, param := range op.Parameters {
			if param != nil && param.Schema != nil {
				if param.Schema.Format == "date-time" || param.Schema.Format == "date" {
					needsTime = true
				}
			}
		}

		// Request types might reference http
		needsHTTP = true
	}

	// Write imports
	buf.WriteString("import (\n")
	if needsHTTP {
		buf.WriteString("\t\"net/http\"\n")
	}
	if needsTime {
		buf.WriteString("\t\"time\"\n")
	}
	buf.WriteString(")\n\n")

	// Generate request types for each operation in this group
	for _, opID := range group.Operations {
		// Skip if method was not added to interface (was filtered as duplicate)
		if !generatedMethods[opID] {
			continue
		}

		info, ok := opToPathMethod[opID]
		if !ok {
			continue
		}

		reqType := cg.generateRequestType(info.path, info.method, info.op)
		if reqType != "" {
			buf.WriteString(reqType)
		}
	}

	// Format the code
	formatted, err := formatAndFixImports("generated.go", buf.Bytes())
	if err != nil {
		cg.addIssue(fmt.Sprintf("server_%s.go", group.Name), fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = buf.Bytes()
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    fmt.Sprintf("server_%s.go", group.Name),
		Content: formatted,
	})

	return nil
}

// generateServerMethodSignature generates the interface method signature
func (cg *oas3CodeGenerator) generateServerMethodSignature(path, method string, op *parser.Operation) string {
	var buf bytes.Buffer

	methodName := operationToMethodName(op, path, method)
	responseType := cg.getResponseType(op)

	// Write comment - handle multiline descriptions properly
	if op.Summary != "" {
		buf.WriteString(formatMultilineComment(op.Summary, methodName, "\t"))
	} else if op.Description != "" {
		buf.WriteString(formatMultilineComment(op.Description, methodName, "\t"))
	}
	if op.Deprecated {
		buf.WriteString("\t// Deprecated: This operation is deprecated.\n")
	}

	buf.WriteString(fmt.Sprintf("\t%s(ctx context.Context, req *%sRequest) (%s, error)\n", methodName, methodName, responseType))

	return buf.String()
}

// generateRequestType generates a request struct for an operation
func (cg *oas3CodeGenerator) generateRequestType(path, method string, op *parser.Operation) string {
	var buf bytes.Buffer

	methodName := operationToMethodName(op, path, method)

	buf.WriteString(fmt.Sprintf("// %sRequest contains the request data for %s.\n", methodName, methodName))
	buf.WriteString(fmt.Sprintf("type %sRequest struct {\n", methodName))

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
		buf.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, goType))
	}

	// Query parameters - paramToGoType already handles pointer for optional params
	for _, param := range queryParams {
		goType := cg.paramToGoType(param)
		fieldName := toFieldName(param.Name)
		buf.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, goType))
	}

	// Header parameters - paramToGoType already handles pointer for optional params
	for _, param := range headerParams {
		goType := cg.paramToGoType(param)
		fieldName := toFieldName(param.Name)
		buf.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, goType))
	}

	// Cookie parameters - paramToGoType already handles pointer for optional params
	for _, param := range cookieParams {
		goType := cg.paramToGoType(param)
		fieldName := toFieldName(param.Name)
		buf.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, goType))
	}

	// Request body
	if op.RequestBody != nil {
		bodyType := cg.getRequestBodyType(op.RequestBody)
		buf.WriteString(fmt.Sprintf("\tBody %s\n", bodyType))
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
func (cg *oas3CodeGenerator) generateSecurityHelpers() error {
	// Check if client generation is enabled and we have security schemes
	if !cg.g.GenerateClient {
		return nil
	}

	// Get security schemes
	var schemes map[string]*parser.SecurityScheme
	if cg.doc.Components != nil {
		schemes = cg.doc.Components.SecuritySchemes
	}

	// Generate security helpers if enabled
	if cg.g.GenerateSecurity && len(schemes) > 0 {
		if err := cg.generateSecurityHelpersFile(schemes); err != nil {
			return fmt.Errorf("failed to generate security helpers: %w", err)
		}
	}

	// Generate OAuth2 flows if enabled
	if cg.g.GenerateOAuth2Flows && len(schemes) > 0 {
		if err := cg.generateOAuth2Files(schemes); err != nil {
			return fmt.Errorf("failed to generate OAuth2 flows: %w", err)
		}
	}

	// Generate credential management if enabled
	if cg.g.GenerateCredentialMgmt {
		if err := cg.generateCredentialsFile(); err != nil {
			return fmt.Errorf("failed to generate credentials: %w", err)
		}
	}

	// Generate security enforcement if enabled
	if cg.g.GenerateSecurityEnforce {
		if err := cg.generateSecurityEnforceFile(); err != nil {
			return fmt.Errorf("failed to generate security enforcement: %w", err)
		}
	}

	// Generate OIDC discovery if enabled
	if cg.g.GenerateOIDCDiscovery && len(schemes) > 0 {
		if err := cg.generateOIDCDiscoveryFile(schemes); err != nil {
			return fmt.Errorf("failed to generate OIDC discovery: %w", err)
		}
	}

	// Generate README if enabled
	if cg.g.GenerateReadme {
		if err := cg.generateReadmeFile(schemes); err != nil {
			return fmt.Errorf("failed to generate README: %w", err)
		}
	}

	return nil
}

// generateSecurityHelpersFile generates the security_helpers.go file
//
//nolint:unparam // error return kept for API consistency and future extensibility
func (cg *oas3CodeGenerator) generateSecurityHelpersFile(schemes map[string]*parser.SecurityScheme) error {
	return generateSecurityHelpersFileShared(cg.securityContext(), schemes)
}

// generateOAuth2Files generates OAuth2 flow files for each OAuth2 security scheme
//
//nolint:unparam // error return kept for API consistency and future extensibility
func (cg *oas3CodeGenerator) generateOAuth2Files(schemes map[string]*parser.SecurityScheme) error {
	return generateOAuth2FilesShared(cg.securityContext(), schemes)
}

// generateCredentialsFile generates the credentials.go file
//
//nolint:unparam // error return kept for API consistency and future extensibility
func (cg *oas3CodeGenerator) generateCredentialsFile() error {
	return generateCredentialsFileShared(cg.securityContext())
}

// generateSecurityEnforceFile generates security enforcement code.
// If file splitting is enabled and needed, generates multiple files.
//
//nolint:unparam // error return kept for API consistency and future extensibility
func (cg *oas3CodeGenerator) generateSecurityEnforceFile() error {
	// Check if we should split
	if cg.splitPlan != nil && cg.splitPlan.NeedsSplit {
		return cg.generateSplitSecurityEnforce()
	}
	return cg.generateSingleSecurityEnforce()
}

// generateSingleSecurityEnforce generates all security enforcement in a single file.
func (cg *oas3CodeGenerator) generateSingleSecurityEnforce() error {
	opSecurity := ExtractOperationSecurityOAS3(cg.doc)
	return generateSingleSecurityEnforceShared(cg.securityContext(), opSecurity, cg.doc.Security)
}

// generateSplitSecurityEnforce generates security enforcement split across multiple files.
func (cg *oas3CodeGenerator) generateSplitSecurityEnforce() error {
	opSecurity := ExtractOperationSecurityOAS3(cg.doc)
	return generateSplitSecurityEnforceShared(cg.securityContext(), opSecurity, cg.doc.Security)
}

// generateOIDCDiscoveryFile generates the oidc_discovery.go file
//
//nolint:unparam // error return kept for API consistency and future extensibility
func (cg *oas3CodeGenerator) generateOIDCDiscoveryFile(schemes map[string]*parser.SecurityScheme) error {
	return generateOIDCDiscoveryFileShared(cg.securityContext(), schemes)
}

// generateReadmeFile generates the README.md file
//
//nolint:unparam // error return kept for API consistency and future extensibility
func (cg *oas3CodeGenerator) generateReadmeFile(schemes map[string]*parser.SecurityScheme) error {
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

	return nil
}

// securityContext returns a securityGenerationContext for shared security generation functions.
func (cg *oas3CodeGenerator) securityContext() *securityGenerationContext {
	return &securityGenerationContext{
		result:    cg.result,
		splitPlan: cg.splitPlan,
		addIssue:  cg.addIssue,
	}
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

// toFileName converts a name to a valid Go file name
func toFileName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
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

// generateServerResponses generates typed response helpers for each operation
func (cg *oas3CodeGenerator) generateServerResponses() error {
	if len(cg.doc.Paths) == 0 {
		return nil
	}

	// Build template data
	data := ServerResponsesFileData{
		Header: HeaderData{
			PackageName: cg.result.PackageName,
		},
		Operations: make([]ResponseOperationData, 0),
	}

	// Track generated methods to avoid duplicates
	generatedMethods := make(map[string]bool)

	// Sort paths for deterministic output
	pathKeys := make([]string, 0, len(cg.doc.Paths))
	for path := range cg.doc.Paths {
		pathKeys = append(pathKeys, path)
	}
	sort.Strings(pathKeys)

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

			methodName := operationToMethodName(op, path, method)
			if generatedMethods[methodName] {
				continue
			}
			generatedMethods[methodName] = true

			// Build response operation data
			opData := ResponseOperationData{
				MethodName:   methodName,
				ResponseType: methodName + "Response",
				StatusCodes:  cg.buildStatusCodes(op),
			}

			data.Operations = append(data.Operations, opData)
		}
	}

	// Execute template
	formatted, err := executeTemplate("responses.go.tmpl", data)
	if err != nil {
		cg.addIssue("server_responses.go", fmt.Sprintf("failed to execute template: %v", err), SeverityWarning)
		return err
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    "server_responses.go",
		Content: formatted,
	})

	return nil
}

// buildStatusCodes builds status code data for an operation's responses
func (cg *oas3CodeGenerator) buildStatusCodes(op *parser.Operation) []StatusCodeData {
	if op.Responses == nil {
		return nil
	}

	// Pre-allocate: 1 for default + status codes
	codes := make([]StatusCodeData, 0, 1+len(op.Responses.Codes))

	// Process default response first
	if op.Responses.Default != nil {
		statusData := cg.buildStatusCodeData("default", op.Responses.Default)
		codes = append(codes, statusData)
	}

	// Get sorted status codes from Codes map
	statusKeys := make([]string, 0, len(op.Responses.Codes))
	for code := range op.Responses.Codes {
		statusKeys = append(statusKeys, code)
	}
	sort.Strings(statusKeys)

	for _, code := range statusKeys {
		resp := op.Responses.Codes[code]
		if resp == nil {
			continue
		}
		statusData := cg.buildStatusCodeData(code, resp)
		codes = append(codes, statusData)
	}

	return codes
}

// generateServerBinder generates parameter binding code for each operation
func (cg *oas3CodeGenerator) generateServerBinder() error {
	if len(cg.doc.Paths) == 0 {
		return nil
	}

	// Build template data
	data := ServerBinderFileData{
		Header: HeaderData{
			PackageName: cg.result.PackageName,
		},
		Operations: make([]BinderOperationData, 0),
	}

	// Track generated methods to avoid duplicates
	generatedMethods := make(map[string]bool)

	// Sort paths for deterministic output
	pathKeys := make([]string, 0, len(cg.doc.Paths))
	for path := range cg.doc.Paths {
		pathKeys = append(pathKeys, path)
	}
	sort.Strings(pathKeys)

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

			methodName := operationToMethodName(op, path, method)
			if generatedMethods[methodName] {
				continue
			}
			generatedMethods[methodName] = true

			// Build binder operation data
			opData := cg.buildBinderOperationData(methodName, op)
			data.Operations = append(data.Operations, opData)
		}
	}

	// Execute template
	formatted, err := executeTemplate("binder.go.tmpl", data)
	if err != nil {
		cg.addIssue("server_binder.go", fmt.Sprintf("failed to execute template: %v", err), SeverityWarning)
		return err
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    "server_binder.go",
		Content: formatted,
	})

	return nil
}

// buildBinderOperationData builds binding data for a single operation
func (cg *oas3CodeGenerator) buildBinderOperationData(methodName string, op *parser.Operation) BinderOperationData {
	opData := BinderOperationData{
		MethodName:  methodName,
		RequestType: methodName + "Request",
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
	statusData := StatusCodeData{
		Code:        code,
		Description: resp.Description,
	}

	// Determine method name and code type
	switch {
	case code == "default":
		statusData.MethodName = "StatusDefault"
		statusData.IsDefault = true
		statusData.StatusCodeInt = 500 // Use 500 as default status
	case len(code) == 3 && strings.HasSuffix(code, "XX"):
		// Wildcard like 2XX, 4XX, 5XX
		statusData.IsWildcard = true
		statusData.MethodName = "Status" + code
		// Use first code in range
		switch code[0] {
		case '2':
			statusData.StatusCodeInt = 200
			statusData.IsSuccess = true
		case '3':
			statusData.StatusCodeInt = 300
		case '4':
			statusData.StatusCodeInt = 400
		case '5':
			statusData.StatusCodeInt = 500
		}
	default:
		// Specific status code
		statusData.MethodName = "Status" + code
		var statusInt int
		if _, err := fmt.Sscanf(code, "%d", &statusInt); err == nil {
			statusData.StatusCodeInt = statusInt
			statusData.IsSuccess = statusInt >= 200 && statusInt < 300
		}
	}

	// Determine body type from response content
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

// generateServerMiddleware generates validation middleware
func (cg *oas3CodeGenerator) generateServerMiddleware() error {
	return generateServerMiddlewareShared(cg.result, cg.addIssue)
}

// generateServerRouter generates HTTP router code
func (cg *oas3CodeGenerator) generateServerRouter() error {
	if len(cg.doc.Paths) == 0 {
		return nil
	}

	// Track generated methods to avoid duplicates
	generatedMethods := make(map[string]bool)

	// Sort paths for deterministic output
	pathKeys := make([]string, 0, len(cg.doc.Paths))
	for path := range cg.doc.Paths {
		pathKeys = append(pathKeys, path)
	}
	sort.Strings(pathKeys)

	// Build router data
	data := ServerRouterFileData{
		Header: HeaderData{
			PackageName: cg.result.PackageName,
		},
		Operations: make([]RouterOperationData, 0),
	}

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

			methodName := operationToMethodName(op, path, method)
			if generatedMethods[methodName] {
				continue
			}
			generatedMethods[methodName] = true

			opData := RouterOperationData{
				Path:        path,
				Method:      strings.ToUpper(method),
				MethodName:  methodName,
				RequestType: methodName + "Request",
			}

			// Collect path parameters with type info for proper conversion in templates
			for _, param := range op.Parameters {
				if param != nil && param.In == parser.ParamInPath {
					paramData := ParamBindData{
						Name:      param.Name,
						FieldName: toFieldName(param.Name),
						GoType:    cg.paramToGoType(param),
						Required:  param.Required,
					}
					if param.Schema != nil {
						paramData.SchemaType = cg.getSchemaType(param.Schema)
					}
					opData.PathParams = append(opData.PathParams, paramData)
				}
			}

			data.Operations = append(data.Operations, opData)
		}
	}

	// Select template based on router type
	templateName := "router.go.tmpl"
	if cg.g.ServerRouter == "chi" {
		templateName = "router_chi.go.tmpl"
	}

	formatted, err := executeTemplate(templateName, data)
	if err != nil {
		cg.addIssue("server_router.go", fmt.Sprintf("failed to execute template: %v", err), SeverityWarning)
		return err
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    "server_router.go",
		Content: formatted,
	})
	return nil
}

// generateServerStubs generates testable stub implementations
func (cg *oas3CodeGenerator) generateServerStubs() error {
	if len(cg.doc.Paths) == 0 {
		return nil
	}

	// Track generated methods to avoid duplicates
	generatedMethods := make(map[string]bool)

	// Sort paths for deterministic output
	pathKeys := make([]string, 0, len(cg.doc.Paths))
	for path := range cg.doc.Paths {
		pathKeys = append(pathKeys, path)
	}
	sort.Strings(pathKeys)

	// Build stubs data
	data := ServerStubsFileData{
		Header: HeaderData{
			PackageName: cg.result.PackageName,
		},
		Operations: make([]StubOperationData, 0),
	}

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

			methodName := operationToMethodName(op, path, method)
			if generatedMethods[methodName] {
				continue
			}
			generatedMethods[methodName] = true

			// Determine response type
			responseType := "*" + methodName + "Response"
			if !cg.g.ServerResponses {
				responseType = "any"
			}

			opData := StubOperationData{
				MethodName:   methodName,
				RequestType:  methodName + "Request",
				ResponseType: responseType,
			}

			data.Operations = append(data.Operations, opData)
		}
	}

	formatted, err := executeTemplate("stubs.go.tmpl", data)
	if err != nil {
		cg.addIssue("server_stubs.go", fmt.Sprintf("failed to execute template: %v", err), SeverityWarning)
		return err
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    "server_stubs.go",
		Content: formatted,
	})
	return nil
}
