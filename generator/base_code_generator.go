// base_code_generator.go contains shared code generation logic for OAS 2.0 and 3.x

package generator

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/internal/maputil"
	"github.com/erraggy/oastools/parser"
)

// baseCodeGenerator contains shared fields and methods for both OAS versions
type baseCodeGenerator struct {
	g              *Generator
	result         *GenerateResult
	schemaNames    map[string]string // maps schema references to generated type names
	generatedTypes map[string]bool   // tracks which type names have been generated
	splitPlan      *SplitPlan        // file splitting plan for large APIs

	// Version-agnostic document access (set in constructor)
	paths       parser.Paths
	oasVersion  parser.OASVersion
	httpMethods []string

	// Version-specific callbacks (set in constructor)
	statusCodeDataBuilder      func(string, *parser.Response) StatusCodeData
	binderOperationDataBuilder func(string, *parser.Operation) BinderOperationData
}

// initBase initializes the base code generator fields
func (b *baseCodeGenerator) initBase(g *Generator, result *GenerateResult) {
	b.g = g
	b.result = result
	b.schemaNames = make(map[string]string)
	b.generatedTypes = make(map[string]bool)
}

// securityContext returns a securityGenerationContext for shared security generation functions.
func (b *baseCodeGenerator) securityContext() *securityGenerationContext {
	return &securityGenerationContext{
		result:    b.result,
		splitPlan: b.splitPlan,
		addIssue:  b.addIssue,
	}
}

// generateServerMiddleware generates validation middleware.
func (b *baseCodeGenerator) generateServerMiddleware() error {
	return generateServerMiddlewareShared(b.result, b.addIssue)
}

// generateCredentialsFile generates the credentials.go file.
func (b *baseCodeGenerator) generateCredentialsFile() {
	generateCredentialsFileShared(b.securityContext())
}

// generateSecurityHelpersFile generates the security_helpers.go file.
func (b *baseCodeGenerator) generateSecurityHelpersFile(schemes map[string]*parser.SecurityScheme) {
	generateSecurityHelpersFileShared(b.securityContext(), schemes)
}

// generateOAuth2Files generates OAuth2 flow files for each OAuth2 security scheme.
func (b *baseCodeGenerator) generateOAuth2Files(schemes map[string]*parser.SecurityScheme) {
	generateOAuth2FilesShared(b.securityContext(), schemes)
}

// generateOIDCDiscoveryFile generates the oidc_discovery.go file.
func (b *baseCodeGenerator) generateOIDCDiscoveryFile(schemes map[string]*parser.SecurityScheme) {
	generateOIDCDiscoveryFileShared(b.securityContext(), schemes)
}

// resolveRef resolves a $ref to a Go type name
func (b *baseCodeGenerator) resolveRef(ref string) string {
	if typeName, ok := b.schemaNames[ref]; ok {
		return typeName
	}
	// Extract name from ref path
	parts := strings.Split(ref, "/")
	if len(parts) > 0 {
		return toTypeName(parts[len(parts)-1])
	}
	return "any"
}

// addIssue adds a generation issue
//
//nolint:unparam // severity parameter kept for API consistency and future extensibility
func (b *baseCodeGenerator) addIssue(path, message string, severity Severity) {
	issue := GenerateIssue{
		Path:     path,
		Message:  message,
		Severity: severity,
	}
	b.populateIssueLocation(&issue, path)
	b.result.Issues = append(b.result.Issues, issue)
}

// populateIssueLocation fills in Line/Column/File from the SourceMap if available.
func (b *baseCodeGenerator) populateIssueLocation(issue *GenerateIssue, path string) {
	if b.g.SourceMap == nil {
		return
	}

	// Convert path format if needed (generator uses dotted paths like "definitions.Pet",
	// while SourceMap uses JSON path notation like "$.definitions.Pet")
	jsonPath := path
	if len(jsonPath) == 0 || jsonPath[0] != '$' {
		jsonPath = "$." + path
	}

	loc := b.g.SourceMap.Get(jsonPath)
	if loc.IsKnown() {
		issue.Line = loc.Line
		issue.Column = loc.Column
		issue.File = loc.File
	}
}

// getAdditionalPropertiesType extracts the Go type for additionalProperties
func (b *baseCodeGenerator) getAdditionalPropertiesType(schema *parser.Schema, schemaToGoType func(*parser.Schema, bool) string) string {
	if schema.AdditionalProperties == nil {
		return "any"
	}

	switch addProps := schema.AdditionalProperties.(type) {
	case *parser.Schema:
		return schemaToGoType(addProps, true)
	case map[string]any:
		return schemaTypeFromMap(addProps)
	case bool:
		if addProps {
			return "any"
		}
	}
	return "any"
}

// getArrayItemType extracts the Go type for array items, handling $ref properly
func (b *baseCodeGenerator) getArrayItemType(schema *parser.Schema, schemaToGoType func(*parser.Schema, bool) string) string {
	if schema.Items == nil {
		return "any"
	}

	switch items := schema.Items.(type) {
	case *parser.Schema:
		if items.Ref != "" {
			return b.resolveRef(items.Ref)
		}
		return schemaToGoType(items, true)
	case map[string]any:
		if ref, ok := items["$ref"].(string); ok {
			return b.resolveRef(ref)
		}
		return schemaTypeFromMap(items)
	}
	return "any"
}

// schemaToGoTypeBase is the shared logic for converting a schema to a Go type.
// isNullable is provided by the caller since OAS3 has additional nullable checks.
func (b *baseCodeGenerator) schemaToGoTypeBase(schema *parser.Schema, required bool, isNullable bool, schemaToGoType func(*parser.Schema, bool) string) string {
	if schema == nil {
		return "any"
	}

	// Handle $ref
	if schema.Ref != "" {
		refType := b.resolveRef(schema.Ref)
		if !required && b.g.UsePointers {
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
		goType = "[]" + b.getArrayItemType(schema, schemaToGoType)
	case "object":
		if schema.Properties == nil && schema.AdditionalProperties != nil {
			// Map type
			goType = "map[string]" + b.getAdditionalPropertiesType(schema, schemaToGoType)
		} else {
			goType = "map[string]any"
		}
	default:
		goType = "any"
	}

	// Handle optional fields with pointers
	if !required && b.g.UsePointers && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map") {
		return "*" + goType
	}

	// Handle nullable with pointers (OAS 3.x)
	if isNullable && b.g.UsePointers && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map") && !strings.HasPrefix(goType, "*") {
		return "*" + goType
	}

	return goType
}

// buildStatusCodes builds status code data for an operation's responses.
func (b *baseCodeGenerator) buildStatusCodes(op *parser.Operation) []StatusCodeData {
	return buildStatusCodesShared(op, b.statusCodeDataBuilder)
}

// generateServerResponses generates typed response helpers for each operation.
func (b *baseCodeGenerator) generateServerResponses() error {
	if len(b.paths) == 0 {
		return nil
	}

	// Build template data
	data := ServerResponsesFileData{
		Header: HeaderData{
			PackageName: b.result.PackageName,
		},
		Operations: make([]ResponseOperationData, 0),
	}

	// Track generated methods to avoid duplicates
	generatedMethods := make(map[string]bool)

	// Sort paths for deterministic output
	pathKeys := maputil.SortedKeys(b.paths)

	for _, path := range pathKeys {
		pathItem := b.paths[path]
		if pathItem == nil {
			continue
		}

		operations := parser.GetOperations(pathItem, b.oasVersion)
		for _, method := range b.httpMethods {
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
				StatusCodes:  b.buildStatusCodes(op),
			}

			data.Operations = append(data.Operations, opData)
		}
	}

	// Execute template
	formatted, err := executeTemplate("responses.go.tmpl", data)
	if err != nil {
		b.addIssue("server_responses.go", fmt.Sprintf("failed to execute template: %v", err), SeverityWarning)
		return err
	}

	b.result.Files = append(b.result.Files, GeneratedFile{
		Name:    "server_responses.go",
		Content: formatted,
	})

	return nil
}

// generateServerBinder generates parameter binding helpers.
func (b *baseCodeGenerator) generateServerBinder() error {
	if len(b.paths) == 0 {
		return nil
	}

	// Build template data
	data := ServerBinderFileData{
		Header: HeaderData{
			PackageName: b.result.PackageName,
		},
		Operations: make([]BinderOperationData, 0),
	}

	// Track generated methods to avoid duplicates
	generatedMethods := make(map[string]bool)

	// Sort paths for deterministic output
	pathKeys := maputil.SortedKeys(b.paths)

	for _, path := range pathKeys {
		pathItem := b.paths[path]
		if pathItem == nil {
			continue
		}

		operations := parser.GetOperations(pathItem, b.oasVersion)
		for _, method := range b.httpMethods {
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
			opData := b.binderOperationDataBuilder(methodName, op)
			data.Operations = append(data.Operations, opData)
		}
	}

	// Execute template
	formatted, err := executeTemplate("binder.go.tmpl", data)
	if err != nil {
		b.addIssue("server_binder.go", fmt.Sprintf("failed to execute template: %v", err), SeverityWarning)
		return err
	}

	b.result.Files = append(b.result.Files, GeneratedFile{
		Name:    "server_binder.go",
		Content: formatted,
	})

	return nil
}

// generateServerStubs generates testable stub implementations.
func (b *baseCodeGenerator) generateServerStubs() error {
	return generateServerStubsShared(&serverStubsContext{
		paths:       b.paths,
		oasVersion:  b.oasVersion,
		httpMethods: b.httpMethods,
		packageName: b.result.PackageName,
		schemaTypes: b.generatedTypes,
		result:      b.result,
		addIssue:    b.addIssue,
		getResponseType: func(methodName string) string {
			if !b.g.ServerResponses {
				return "any"
			}
			return "*" + methodName + "Response"
		},
	})
}
