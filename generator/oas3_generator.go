package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"sort"
	"strings"

	"github.com/erraggy/oastools/internal/httputil"
	"github.com/erraggy/oastools/parser"
)

// oas3CodeGenerator handles code generation for OAS 3.x documents
type oas3CodeGenerator struct {
	g      *Generator
	doc    *parser.OAS3Document
	result *GenerateResult
	// schemaNames maps schema references to generated type names
	schemaNames map[string]string
}

func newOAS3CodeGenerator(g *Generator, doc *parser.OAS3Document, result *GenerateResult) *oas3CodeGenerator {
	return &oas3CodeGenerator{
		g:           g,
		doc:         doc,
		result:      result,
		schemaNames: make(map[string]string),
	}
}

// generateTypes generates type definitions from schemas
func (cg *oas3CodeGenerator) generateTypes() error {
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
	isNullable := schema.Nullable || isTypeNullable(schema.Type)
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
	var buf bytes.Buffer

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
	if responseType != "" && responseType != "*http.Response" {
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
		return "*http.Response"
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

	return "*http.Response"
}

// generateServer generates server interface code
func (cg *oas3CodeGenerator) generateServer() error {
	var buf bytes.Buffer

	// Write header
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", cg.result.PackageName))

	// Write imports
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"net/http\"\n")
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
