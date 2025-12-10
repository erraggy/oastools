package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"sort"
	"strings"
	"time"

	"github.com/erraggy/oastools/internal/httputil"
	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
)

// oas3CodeGenerator handles code generation for OAS 3.x documents
type oas3CodeGenerator struct {
	g      *Generator
	doc    *parser.OAS3Document
	result *GenerateResult
	// schemaNames maps schema references to generated type names
	schemaNames map[string]string
	// splitPlan contains the file splitting plan for large APIs
	splitPlan *SplitPlan
}

func newOAS3CodeGenerator(g *Generator, doc *parser.OAS3Document, result *GenerateResult) *oas3CodeGenerator {
	cg := &oas3CodeGenerator{
		g:           g,
		doc:         doc,
		result:      result,
		schemaNames: make(map[string]string),
	}

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

	// Get all schemas
	var allSchemas []schemaEntry
	if cg.doc.Components != nil && cg.doc.Components.Schemas != nil {
		for name, schema := range cg.doc.Components.Schemas {
			if schema == nil {
				continue
			}
			allSchemas = append(allSchemas, schemaEntry{name: name, schema: schema})
			cg.schemaNames["#/components/schemas/"+name] = toTypeName(name)
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
	// Build filtered types list
	var filteredSchemas []schemaEntry
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
	if schema == nil {
		return "any"
	}

	// Handle $ref
	if schema.Ref != "" {
		refType := cg.resolveRef(schema.Ref)
		if !required && cg.g.UsePointers {
			return "*" + refType
		}
		return refType
	}

	schemaType := getSchemaType(schema)
	var goType string

	switch schemaType {
	case "string":
		goType = stringFormatToGoType(schema.Format)
	case "integer":
		goType = integerFormatToGoType(schema.Format)
	case "number":
		goType = numberFormatToGoType(schema.Format)
	case "boolean":
		goType = "bool"
	case "array":
		goType = "[]" + cg.getArrayItemType(schema)
	case "object":
		if schema.Properties == nil && schema.AdditionalProperties != nil {
			// Map type
			goType = "map[string]" + cg.getAdditionalPropertiesType(schema)
		} else {
			goType = "map[string]any"
		}
	default:
		goType = "any"
	}

	// Handle nullable (OAS 3.0) or type array with null (OAS 3.1+)
	isNullable := schema.Nullable || schemautil.IsNullable(schema)
	if !required && cg.g.UsePointers && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map") {
		return "*" + goType
	}
	if isNullable && cg.g.UsePointers && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map") && !strings.HasPrefix(goType, "*") {
		return "*" + goType
	}

	return goType
}

// getArrayItemType extracts the Go type for array items, handling $ref properly
func (cg *oas3CodeGenerator) getArrayItemType(schema *parser.Schema) string {
	if schema.Items == nil {
		return "any"
	}

	switch items := schema.Items.(type) {
	case *parser.Schema:
		// Check if items has a $ref
		if items.Ref != "" {
			return cg.resolveRef(items.Ref)
		}
		return cg.schemaToGoType(items, true)
	case map[string]interface{}:
		// Handle inline schema as map
		if ref, ok := items["$ref"].(string); ok {
			return cg.resolveRef(ref)
		}
		return schemaTypeFromMap(items)
	}
	return "any"
}

// getAdditionalPropertiesType extracts the Go type for additionalProperties
func (cg *oas3CodeGenerator) getAdditionalPropertiesType(schema *parser.Schema) string {
	if schema.AdditionalProperties == nil {
		return "any"
	}

	switch addProps := schema.AdditionalProperties.(type) {
	case *parser.Schema:
		return cg.schemaToGoType(addProps, true)
	case map[string]interface{}:
		return schemaTypeFromMap(addProps)
	case bool:
		if addProps {
			return "any"
		}
	}
	return "any"
}

// resolveRef resolves a $ref to a Go type name
func (cg *oas3CodeGenerator) resolveRef(ref string) string {
	if typeName, ok := cg.schemaNames[ref]; ok {
		return typeName
	}
	// Extract name from ref path
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return toTypeName(parts[len(parts)-1])
	}
	return "any"
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

// addIssue adds a generation issue
func (cg *oas3CodeGenerator) addIssue(path, message string, severity Severity) {
	cg.result.Issues = append(cg.result.Issues, GenerateIssue{
		Path:     path,
		Message:  message,
		Severity: severity,
	})
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

	// Write client struct
	buf.WriteString("// Client is the API client.\n")
	buf.WriteString("type Client struct {\n")
	buf.WriteString("\t// BaseURL is the base URL for API requests.\n")
	buf.WriteString("\tBaseURL string\n")
	buf.WriteString("\t// HTTPClient is the HTTP client to use for requests.\n")
	buf.WriteString("\tHTTPClient *http.Client\n")
	buf.WriteString("\t// UserAgent is the User-Agent header value for requests.\n")
	buf.WriteString("\tUserAgent string\n")
	buf.WriteString("\t// RequestEditors are functions that can modify requests before sending.\n")
	buf.WriteString("\tRequestEditors []RequestEditorFn\n")
	buf.WriteString("}\n\n")

	// Write types
	buf.WriteString("// RequestEditorFn is a function that can modify an HTTP request.\n")
	buf.WriteString("type RequestEditorFn func(ctx context.Context, req *http.Request) error\n\n")

	buf.WriteString("// ClientOption is a function that configures a Client.\n")
	buf.WriteString("type ClientOption func(*Client) error\n\n")

	// Write constructor
	defaultUserAgent := buildDefaultUserAgent(cg.doc.Info)
	buf.WriteString("// NewClient creates a new API client.\n")
	buf.WriteString("func NewClient(baseURL string, opts ...ClientOption) (*Client, error) {\n")
	buf.WriteString("\tc := &Client{\n")
	buf.WriteString("\t\tBaseURL:    strings.TrimSuffix(baseURL, \"/\"),\n")
	buf.WriteString("\t\tHTTPClient: http.DefaultClient,\n")
	buf.WriteString(fmt.Sprintf("\t\tUserAgent:  %q,\n", defaultUserAgent))
	buf.WriteString("\t}\n")
	buf.WriteString("\tfor _, opt := range opts {\n")
	buf.WriteString("\t\tif err := opt(c); err != nil {\n")
	buf.WriteString("\t\t\treturn nil, err\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\treturn c, nil\n")
	buf.WriteString("}\n\n")

	// Write client options
	buf.WriteString("// WithHTTPClient sets the HTTP client.\n")
	buf.WriteString("func WithHTTPClient(client *http.Client) ClientOption {\n")
	buf.WriteString("\treturn func(c *Client) error {\n")
	buf.WriteString("\t\tc.HTTPClient = client\n")
	buf.WriteString("\t\treturn nil\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	buf.WriteString("// WithRequestEditor adds a request editor function.\n")
	buf.WriteString("func WithRequestEditor(fn RequestEditorFn) ClientOption {\n")
	buf.WriteString("\treturn func(c *Client) error {\n")
	buf.WriteString("\t\tc.RequestEditors = append(c.RequestEditors, fn)\n")
	buf.WriteString("\t\treturn nil\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	buf.WriteString("// WithUserAgent sets the User-Agent header value.\n")
	buf.WriteString("func WithUserAgent(ua string) ClientOption {\n")
	buf.WriteString("\treturn func(c *Client) error {\n")
	buf.WriteString("\t\tc.UserAgent = ua\n")
	buf.WriteString("\t\treturn nil\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

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
	formatted, err := format.Source(buf.Bytes())
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

	// Write imports (base client needs fewer imports)
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"fmt\"\n")
	buf.WriteString("\t\"io\"\n")
	buf.WriteString("\t\"net/http\"\n")
	buf.WriteString("\t\"strings\"\n")
	buf.WriteString(")\n\n")

	// Write client struct
	buf.WriteString("// Client is the API client.\n")
	buf.WriteString("type Client struct {\n")
	buf.WriteString("\t// BaseURL is the base URL for API requests.\n")
	buf.WriteString("\tBaseURL string\n")
	buf.WriteString("\t// HTTPClient is the HTTP client to use for requests.\n")
	buf.WriteString("\tHTTPClient *http.Client\n")
	buf.WriteString("\t// UserAgent is the User-Agent header value for requests.\n")
	buf.WriteString("\tUserAgent string\n")
	buf.WriteString("\t// RequestEditors are functions that can modify requests before sending.\n")
	buf.WriteString("\tRequestEditors []RequestEditorFn\n")
	buf.WriteString("}\n\n")

	// Write types
	buf.WriteString("// RequestEditorFn is a function that can modify an HTTP request.\n")
	buf.WriteString("type RequestEditorFn func(ctx context.Context, req *http.Request) error\n\n")

	buf.WriteString("// ClientOption is a function that configures a Client.\n")
	buf.WriteString("type ClientOption func(*Client) error\n\n")

	// Write constructor
	defaultUserAgent := buildDefaultUserAgent(cg.doc.Info)
	buf.WriteString("// NewClient creates a new API client.\n")
	buf.WriteString("func NewClient(baseURL string, opts ...ClientOption) (*Client, error) {\n")
	buf.WriteString("\tc := &Client{\n")
	buf.WriteString("\t\tBaseURL:    strings.TrimSuffix(baseURL, \"/\"),\n")
	buf.WriteString("\t\tHTTPClient: http.DefaultClient,\n")
	buf.WriteString(fmt.Sprintf("\t\tUserAgent:  %q,\n", defaultUserAgent))
	buf.WriteString("\t}\n")
	buf.WriteString("\tfor _, opt := range opts {\n")
	buf.WriteString("\t\tif err := opt(c); err != nil {\n")
	buf.WriteString("\t\t\treturn nil, err\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t}\n")
	buf.WriteString("\treturn c, nil\n")
	buf.WriteString("}\n\n")

	// Write client options
	buf.WriteString("// WithHTTPClient sets the HTTP client.\n")
	buf.WriteString("func WithHTTPClient(client *http.Client) ClientOption {\n")
	buf.WriteString("\treturn func(c *Client) error {\n")
	buf.WriteString("\t\tc.HTTPClient = client\n")
	buf.WriteString("\t\treturn nil\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	buf.WriteString("// WithRequestEditor adds a request editor function.\n")
	buf.WriteString("func WithRequestEditor(fn RequestEditorFn) ClientOption {\n")
	buf.WriteString("\treturn func(c *Client) error {\n")
	buf.WriteString("\t\tc.RequestEditors = append(c.RequestEditors, fn)\n")
	buf.WriteString("\t\treturn nil\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	buf.WriteString("// WithUserAgent sets the User-Agent header value.\n")
	buf.WriteString("func WithUserAgent(ua string) ClientOption {\n")
	buf.WriteString("\treturn func(c *Client) error {\n")
	buf.WriteString("\t\tc.UserAgent = ua\n")
	buf.WriteString("\t\treturn nil\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	// Write helper functions
	buf.WriteString(clientHelpers)

	// Format the code
	formatted, err := format.Source(buf.Bytes())
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
	formatted, err := format.Source(buf.Bytes())
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

	// Write method documentation
	if op.Summary != "" {
		buf.WriteString(fmt.Sprintf("// %s %s\n", methodName, op.Summary))
	} else if op.Description != "" {
		buf.WriteString(fmt.Sprintf("// %s %s\n", methodName, cleanDescription(op.Description)))
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

				sig := cg.generateServerMethodSignature(path, method, op)
				buf.WriteString(sig)
			}
		}
	}

	buf.WriteString("}\n\n")

	// Generate request types
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

			reqType := cg.generateRequestType(path, method, op)
			if reqType != "" {
				buf.WriteString(reqType)
			}
		}
	}

	// Write unimplemented server
	buf.WriteString("// UnimplementedServer provides default implementations that return errors.\n")
	buf.WriteString("type UnimplementedServer struct{}\n\n")

	// Generate unimplemented methods
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
	formatted, err := format.Source(buf.Bytes())
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
	if err := cg.generateBaseServer(); err != nil {
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

		if err := cg.generateServerGroupFile(group, opToPathMethod); err != nil {
			cg.addIssue(fmt.Sprintf("server_%s.go", group.Name), fmt.Sprintf("failed to generate: %v", err), SeverityWarning)
		}
	}

	return nil
}

// generateBaseServer generates the base server.go with interface and unimplemented struct
func (cg *oas3CodeGenerator) generateBaseServer() error {
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

				sig := cg.generateServerMethodSignature(path, method, op)
				buf.WriteString(sig)
			}
		}
	}

	buf.WriteString("}\n\n")

	// Write unimplemented server (must be complete)
	buf.WriteString("// UnimplementedServer provides default implementations that return errors.\n")
	buf.WriteString("type UnimplementedServer struct{}\n\n")

	// Generate unimplemented methods
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
	formatted, err := format.Source(buf.Bytes())
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

// generateServerGroupFile generates a server_{group}.go file with request types for a specific group
//
//nolint:unparam // error return kept for API consistency with other generate methods
func (cg *oas3CodeGenerator) generateServerGroupFile(group FileGroup, opToPathMethod map[string]struct {
	path   string
	method string
	op     *parser.Operation
}) error {
	var buf bytes.Buffer

	// Write header with comment about the group
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n")
	buf.WriteString(fmt.Sprintf("// This file contains %s server request types.\n\n", group.DisplayName))
	buf.WriteString(fmt.Sprintf("package %s\n\n", cg.result.PackageName))

	// Determine needed imports based on operations
	needsTime := false
	needsHTTP := false

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
	formatted, err := format.Source(buf.Bytes())
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

	// Write comment
	if op.Summary != "" {
		buf.WriteString(fmt.Sprintf("\t// %s %s\n", methodName, op.Summary))
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

	// Query parameters
	for _, param := range queryParams {
		goType := cg.paramToGoType(param)
		fieldName := toFieldName(param.Name)
		if !param.Required {
			buf.WriteString(fmt.Sprintf("\t%s *%s\n", fieldName, goType))
		} else {
			buf.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, goType))
		}
	}

	// Header parameters
	for _, param := range headerParams {
		goType := cg.paramToGoType(param)
		fieldName := toFieldName(param.Name)
		if !param.Required {
			buf.WriteString(fmt.Sprintf("\t%s *%s\n", fieldName, goType))
		} else {
			buf.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, goType))
		}
	}

	// Cookie parameters
	for _, param := range cookieParams {
		goType := cg.paramToGoType(param)
		fieldName := toFieldName(param.Name)
		if !param.Required {
			buf.WriteString(fmt.Sprintf("\t%s *%s\n", fieldName, goType))
		} else {
			buf.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, goType))
		}
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

// Ensure unused imports don't cause errors
var (
	_ = bytes.NewReader
	_ = context.Background
	_ = json.Marshal
	_ = fmt.Sprintf
	_ = io.ReadAll
	_ = http.NewRequest
	_ = url.Values{}
	_ = strings.TrimSpace
)
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
	g := NewSecurityHelperGenerator(cg.result.PackageName)
	code := g.GenerateSecurityHelpers(schemes)

	// Format the code
	formatted, err := format.Source([]byte(code))
	if err != nil {
		cg.addIssue("security_helpers.go", fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = []byte(code)
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    "security_helpers.go",
		Content: formatted,
	})

	return nil
}

// generateOAuth2Files generates OAuth2 flow files for each OAuth2 security scheme
//
//nolint:unparam // error return kept for API consistency and future extensibility
func (cg *oas3CodeGenerator) generateOAuth2Files(schemes map[string]*parser.SecurityScheme) error {
	for name, scheme := range schemes {
		if scheme == nil || scheme.Type != "oauth2" {
			continue
		}

		g := NewOAuth2Generator(name, scheme)
		if g == nil || !g.HasAnyFlow() {
			continue
		}

		code := g.GenerateOAuth2File(cg.result.PackageName)

		// Format the code
		formatted, err := format.Source([]byte(code))
		if err != nil {
			cg.addIssue(fmt.Sprintf("oauth2_%s.go", name), fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
			formatted = []byte(code)
		}

		fileName := fmt.Sprintf("oauth2_%s.go", toFileName(name))
		cg.result.Files = append(cg.result.Files, GeneratedFile{
			Name:    fileName,
			Content: formatted,
		})
	}

	return nil
}

// generateCredentialsFile generates the credentials.go file
//
//nolint:unparam // error return kept for API consistency and future extensibility
func (cg *oas3CodeGenerator) generateCredentialsFile() error {
	g := NewCredentialGenerator(cg.result.PackageName)
	code := g.GenerateCredentialsFile()

	// Format the code
	formatted, err := format.Source([]byte(code))
	if err != nil {
		cg.addIssue("credentials.go", fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = []byte(code)
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    "credentials.go",
		Content: formatted,
	})

	return nil
}

// generateSecurityEnforceFile generates the security_enforce.go file
//
//nolint:unparam // error return kept for API consistency and future extensibility
func (cg *oas3CodeGenerator) generateSecurityEnforceFile() error {
	g := NewSecurityEnforceGenerator(cg.result.PackageName)

	// Extract operation security requirements
	opSecurity := ExtractOperationSecurityOAS3(cg.doc)

	code := g.GenerateSecurityEnforceFile(opSecurity, cg.doc.Security)

	// Format the code
	formatted, err := format.Source([]byte(code))
	if err != nil {
		cg.addIssue("security_enforce.go", fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = []byte(code)
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    "security_enforce.go",
		Content: formatted,
	})

	return nil
}

// generateOIDCDiscoveryFile generates the oidc_discovery.go file
//
//nolint:unparam // error return kept for API consistency and future extensibility
func (cg *oas3CodeGenerator) generateOIDCDiscoveryFile(schemes map[string]*parser.SecurityScheme) error {
	// Find the first OpenID Connect scheme to get the discovery URL
	var discoveryURL string
	for _, scheme := range schemes {
		if scheme != nil && scheme.Type == "openIdConnect" && scheme.OpenIDConnectURL != "" {
			discoveryURL = scheme.OpenIDConnectURL
			break
		}
	}

	g := NewOIDCDiscoveryGenerator(cg.result.PackageName)
	code := g.GenerateOIDCDiscoveryFile(discoveryURL)

	// Format the code
	formatted, err := format.Source([]byte(code))
	if err != nil {
		cg.addIssue("oidc_discovery.go", fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = []byte(code)
	}

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    "oidc_discovery.go",
		Content: formatted,
	})

	return nil
}

// generateReadmeFile generates the README.md file
//
//nolint:unparam // error return kept for API consistency and future extensibility
func (cg *oas3CodeGenerator) generateReadmeFile(schemes map[string]*parser.SecurityScheme) error {
	g := NewReadmeGenerator()

	// Build security scheme summaries
	var secSummaries []SecuritySchemeSummary
	if len(schemes) > 0 {
		// Sort scheme names for deterministic output
		names := make([]string, 0, len(schemes))
		for name := range schemes {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			scheme := schemes[name]
			if scheme == nil {
				continue
			}

			summary := SecuritySchemeSummary{
				Name:        name,
				Type:        scheme.Type,
				Description: scheme.Description,
			}

			switch scheme.Type {
			case "apiKey":
				summary.Location = scheme.In
			case "http":
				summary.Scheme = scheme.Scheme
			case "oauth2":
				summary.Flows = extractOAuth2FlowNames(convertFlows(scheme.Flows), scheme.Flow)
			case "openIdConnect":
				summary.OpenIDConnectURL = scheme.OpenIDConnectURL
			}

			secSummaries = append(secSummaries, summary)
		}
	}

	// Build generated file summaries
	fileSummaries := make([]GeneratedFileSummary, 0, len(cg.result.Files))
	for _, f := range cg.result.Files {
		lineCount := strings.Count(string(f.Content), "\n")
		desc := getFileDescription(f.Name)
		fileSummaries = append(fileSummaries, GeneratedFileSummary{
			FileName:    f.Name,
			Description: desc,
			LineCount:   lineCount,
		})
	}

	// Build split summary if applicable
	var splitSummary *SplitSummary
	if cg.splitPlan != nil && cg.splitPlan.NeedsSplit {
		strategy := "by tag"
		if !cg.g.SplitByTag {
			strategy = "by path prefix"
		}
		groups := make([]string, 0, len(cg.splitPlan.Groups))
		for _, g := range cg.splitPlan.Groups {
			groups = append(groups, g.DisplayName)
		}
		splitSummary = &SplitSummary{
			WasSplit:        true,
			Strategy:        strategy,
			Groups:          groups,
			SharedTypesFile: "types.go",
		}
	}

	// Build context
	ctx := &ReadmeContext{
		Timestamp:   time.Now(),
		PackageName: cg.result.PackageName,
		OASVersion:  cg.doc.OpenAPI, // Use the OpenAPI version string field
		Config: &GeneratorConfigSummary{
			GenerateTypes:           cg.g.GenerateTypes,
			GenerateClient:          cg.g.GenerateClient,
			GenerateSecurity:        cg.g.GenerateSecurity,
			GenerateOAuth2Flows:     cg.g.GenerateOAuth2Flows,
			GenerateCredentialMgmt:  cg.g.GenerateCredentialMgmt,
			GenerateSecurityEnforce: cg.g.GenerateSecurityEnforce,
			GenerateOIDCDiscovery:   cg.g.GenerateOIDCDiscovery,
		},
		GeneratedFiles:  fileSummaries,
		SecuritySchemes: secSummaries,
		SplitInfo:       splitSummary,
	}

	// Extract API info
	if cg.doc.Info != nil {
		ctx.APITitle = cg.doc.Info.Title
		ctx.APIVersion = cg.doc.Info.Version
		ctx.APIDescription = cg.doc.Info.Description
	}

	content := g.GenerateReadme(ctx)

	cg.result.Files = append(cg.result.Files, GeneratedFile{
		Name:    "README.md",
		Content: []byte(content),
	})

	return nil
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
