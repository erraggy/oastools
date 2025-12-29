package generator

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/erraggy/oastools/parser"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Operation Mapping Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// OperationMapping holds the path, method, and operation for quick lookup by operation ID.
type OperationMapping struct {
	Path   string
	Method string
	Op     *parser.Operation
}

// buildOperationMap builds a map of operation IDs to their path/method/operation info.
// This uses parser.GetOperations which is version-aware and only returns methods valid
// for the given OAS version (e.g., TRACE for OAS 3.0+, QUERY for OAS 3.2+).
func buildOperationMap(paths parser.Paths, version parser.OASVersion) map[string]OperationMapping {
	result := make(map[string]OperationMapping)

	if paths == nil {
		return result
	}

	for path, pathItem := range paths {
		if pathItem == nil {
			continue
		}
		operations := parser.GetOperations(pathItem, version)
		for method, op := range operations {
			if op == nil {
				continue
			}
			opID := operationToMethodName(op, path, method)
			result[opID] = OperationMapping{
				Path:   path,
				Method: method,
				Op:     op,
			}
		}
	}

	return result
}

// sortedPathKeys returns the keys of a Paths map in sorted order.
// This is used to ensure deterministic iteration order over API paths.
func sortedPathKeys(paths parser.Paths) []string {
	keys := make([]string, 0, len(paths))
	for k := range paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ═══════════════════════════════════════════════════════════════════════════════
// Server Generation Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// buildServerMethodSignature builds an interface method signature for an operation.
// This is 100% identical between OAS 2.0 and OAS 3.x.
func buildServerMethodSignature(path, method string, op *parser.Operation, responseType string) string {
	var buf bytes.Buffer

	methodName := operationToMethodName(op, path, method)

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

// clientMethodGenerator is a callback for generating a client method.
type clientMethodGenerator func(path, method string, op *parser.Operation) (string, error)

// generateGroupClientMethods generates client methods for all operations in a group.
// This is 100% identical between OAS 2.0 and OAS 3.x.
func generateGroupClientMethods(
	buf *bytes.Buffer,
	group FileGroup,
	opToPathMethod map[string]OperationMapping,
	result *GenerateResult,
	addIssue issueAdder,
	generateMethod clientMethodGenerator,
) {
	for _, opID := range group.Operations {
		info, ok := opToPathMethod[opID]
		if !ok {
			continue
		}

		code, err := generateMethod(info.Path, info.Method, info.Op)
		if err != nil {
			addIssue(fmt.Sprintf("paths.%s.%s", info.Path, info.Method),
				fmt.Sprintf("failed to generate client method: %v", err), SeverityWarning)
			continue
		}
		buf.WriteString(code)
		result.GeneratedOperations++
	}
}

// writeNotImplementedError writes the ErrNotImplemented variable and NotImplementedError type.
// This is identical across OAS 2.0, OAS 3.x, and split-file generation.
func writeNotImplementedError(buf *bytes.Buffer) {
	buf.WriteString("// ErrNotImplemented is returned by UnimplementedServer methods.\n")
	buf.WriteString("var ErrNotImplemented = &NotImplementedError{}\n\n")
	buf.WriteString("// NotImplementedError indicates an operation is not implemented.\n")
	buf.WriteString("type NotImplementedError struct{}\n\n")
	buf.WriteString("func (e *NotImplementedError) Error() string { return \"not implemented\" }\n\n")
}

// generateServerMiddlewareShared generates the server middleware file.
// This is 100% identical between OAS 2.0 and OAS 3.x.
func generateServerMiddlewareShared(result *GenerateResult, addIssue issueAdder) error {
	data := ServerMiddlewareFileData{
		Header: HeaderData{
			PackageName: result.PackageName,
		},
	}

	formatted, err := executeTemplate("middleware.go.tmpl", data)
	if err != nil {
		addIssue("server_middleware.go", fmt.Sprintf("failed to execute template: %v", err), SeverityWarning)
		return err
	}

	result.Files = append(result.Files, GeneratedFile{
		Name:    "server_middleware.go",
		Content: formatted,
	})
	return nil
}

// serverRouterContext holds the context for generating server router code.
type serverRouterContext struct {
	paths        parser.Paths
	oasVersion   parser.OASVersion
	httpMethods  []string
	packageName  string
	serverRouter string
	result       *GenerateResult
	addIssue     issueAdder
	// paramToBindData converts a parameter to ParamBindData (version-specific)
	paramToBindData func(param *parser.Parameter) ParamBindData
}

// generateServerRouterShared generates HTTP router code.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateServerRouterShared(ctx *serverRouterContext) error {
	if len(ctx.paths) == 0 {
		return nil
	}

	// Track generated methods to avoid duplicates
	generatedMethods := make(map[string]bool)

	// Sort paths for deterministic output
	pathKeys := sortedPathKeys(ctx.paths)

	// Build router data
	data := ServerRouterFileData{
		Header: HeaderData{
			PackageName: ctx.packageName,
		},
		Operations: make([]RouterOperationData, 0),
	}

	for _, path := range pathKeys {
		pathItem := ctx.paths[path]
		if pathItem == nil {
			continue
		}

		operations := parser.GetOperations(pathItem, ctx.oasVersion)
		for _, method := range ctx.httpMethods {
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
					opData.PathParams = append(opData.PathParams, ctx.paramToBindData(param))
				}
			}

			data.Operations = append(data.Operations, opData)
		}
	}

	// Select template based on router type
	templateName := "router.go.tmpl"
	if ctx.serverRouter == "chi" {
		templateName = "router_chi.go.tmpl"
	}

	formatted, err := executeTemplate(templateName, data)
	if err != nil {
		ctx.addIssue("server_router.go", fmt.Sprintf("failed to execute template: %v", err), SeverityWarning)
		return err
	}

	ctx.result.Files = append(ctx.result.Files, GeneratedFile{
		Name:    "server_router.go",
		Content: formatted,
	})
	return nil
}

// serverStubsContext holds the context for generating server stubs.
type serverStubsContext struct {
	paths       parser.Paths
	oasVersion  parser.OASVersion
	httpMethods []string
	packageName string
	result      *GenerateResult
	addIssue    issueAdder
	// getResponseType returns the Go type for the operation's response given the method name
	getResponseType func(methodName string) string
}

// generateServerStubsShared generates testable stub implementations.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateServerStubsShared(ctx *serverStubsContext) error {
	if len(ctx.paths) == 0 {
		return nil
	}

	// Track generated methods to avoid duplicates
	generatedMethods := make(map[string]bool)

	// Sort paths for deterministic output
	pathKeys := sortedPathKeys(ctx.paths)

	// Build stubs data
	data := ServerStubsFileData{
		Header: HeaderData{
			PackageName: ctx.packageName,
		},
		Operations: make([]StubOperationData, 0),
	}

	for _, path := range pathKeys {
		pathItem := ctx.paths[path]
		if pathItem == nil {
			continue
		}

		operations := parser.GetOperations(pathItem, ctx.oasVersion)
		for _, method := range ctx.httpMethods {
			op := operations[method]
			if op == nil {
				continue
			}

			methodName := operationToMethodName(op, path, method)
			if generatedMethods[methodName] {
				continue
			}
			generatedMethods[methodName] = true

			opData := StubOperationData{
				MethodName:   methodName,
				RequestType:  methodName + "Request",
				ResponseType: ctx.getResponseType(methodName),
			}

			data.Operations = append(data.Operations, opData)
		}
	}

	formatted, err := executeTemplate("stubs.go.tmpl", data)
	if err != nil {
		ctx.addIssue("server_stubs.go", fmt.Sprintf("failed to execute template: %v", err), SeverityWarning)
		return err
	}

	ctx.result.Files = append(ctx.result.Files, GeneratedFile{
		Name:    "server_stubs.go",
		Content: formatted,
	})
	return nil
}

// baseServerContext holds the context for generating base server code.
type baseServerContext struct {
	paths       parser.Paths
	oasVersion  parser.OASVersion
	httpMethods []string
	packageName string
	needsTime   bool // Whether the time import is needed
	result      *GenerateResult
	addIssue    issueAdder
	// generateMethodSignature generates a server method signature for the interface
	generateMethodSignature func(path, method string, op *parser.Operation) string
	// getResponseType returns the Go type for the operation's response
	getResponseType func(op *parser.Operation) string
	// generateRequestTypes is an optional callback to generate request types inline.
	// When set, it's called after the interface and before UnimplementedServer.
	// This is used for single-file mode. For split mode, leave nil.
	generateRequestTypes func(buf *bytes.Buffer, generatedMethods map[string]bool)
}

// generateBaseServerShared generates the base server.go with interface and unimplemented server.
// Returns a map of generated method names (for use by group file generation).
func generateBaseServerShared(ctx *baseServerContext) (map[string]bool, error) {
	var buf bytes.Buffer

	// Write header
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", ctx.packageName))

	// Write imports - include net/http upfront to avoid expensive goimports scanning
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"net/http\"\n")
	if ctx.needsTime {
		buf.WriteString("\t\"time\"\n")
	}
	buf.WriteString(")\n\n")

	// Track generated methods to avoid duplicates (can happen with duplicate operationIds).
	// NOTE: This map must be local per file generation to avoid stale data in split mode.
	generatedMethods := make(map[string]bool)

	// Generate server interface (must be complete)
	buf.WriteString("// ServerInterface represents the server API.\n")
	buf.WriteString("type ServerInterface interface {\n")

	if ctx.paths != nil {
		for _, path := range sortedPathKeys(ctx.paths) {
			pathItem := ctx.paths[path]
			if pathItem == nil {
				continue
			}

			operations := parser.GetOperations(pathItem, ctx.oasVersion)
			for _, method := range ctx.httpMethods {
				op := operations[method]
				if op == nil {
					continue
				}

				methodName := operationToMethodName(op, path, method)
				if generatedMethods[methodName] {
					ctx.addIssue(fmt.Sprintf("paths.%s.%s", path, method),
						fmt.Sprintf("duplicate method name %s - skipping", methodName), SeverityWarning)
					continue
				}
				generatedMethods[methodName] = true

				sig := ctx.generateMethodSignature(path, method, op)
				buf.WriteString(sig)
			}
		}
	}

	buf.WriteString("}\n\n")

	// Generate request types if callback is provided (single-file mode)
	if ctx.generateRequestTypes != nil {
		ctx.generateRequestTypes(&buf, generatedMethods)
	}

	// Write unimplemented server (must be complete)
	buf.WriteString("// UnimplementedServer provides default implementations that return errors.\n")
	buf.WriteString("type UnimplementedServer struct{}\n\n")

	// Track generated UnimplementedServer methods separately to avoid duplicates.
	// We can't reuse generatedMethods because it's used to check if a method was
	// added to the interface (i.e., wasn't filtered as duplicate).
	generatedUnimplemented := make(map[string]bool)

	if ctx.paths != nil {
		for _, path := range sortedPathKeys(ctx.paths) {
			pathItem := ctx.paths[path]
			if pathItem == nil {
				continue
			}

			operations := parser.GetOperations(pathItem, ctx.oasVersion)
			for _, method := range ctx.httpMethods {
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

				responseType := ctx.getResponseType(op)

				buf.WriteString(fmt.Sprintf("func (s *UnimplementedServer) %s(ctx context.Context, req *%sRequest) (%s, error) {\n",
					methodName, methodName, responseType))
				buf.WriteString(fmt.Sprintf("\treturn %s, ErrNotImplemented\n", zeroValue(responseType)))
				buf.WriteString("}\n\n")
			}
		}
	}

	// Write error type
	writeNotImplementedError(&buf)

	// Format and append the file
	appendFormattedFile(ctx.result, "server.go", &buf, ctx.addIssue)

	return generatedMethods, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Split Server Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// splitServerContext holds context for generating split server files.
type splitServerContext struct {
	paths               parser.Paths
	oasVersion          parser.OASVersion
	splitPlan           *SplitPlan
	result              *GenerateResult
	addIssue            issueAdder
	generateBaseServer  func() (map[string]bool, error)
	generateRequestType func(path, method string, op *parser.Operation) string
}

// generateSplitServerShared generates server code split across multiple files.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateSplitServerShared(ctx *splitServerContext) error {
	// Generate the base server.go (interface, unimplemented, error types)
	// Returns the set of methods that were actually generated (excludes duplicates)
	generatedMethods, err := ctx.generateBaseServer()
	if err != nil {
		return err
	}

	// Build a map of operation ID to path/method for quick lookup
	opToPathMethod := buildOperationMap(ctx.paths, ctx.oasVersion)

	// Generate a server file for each group (request types only)
	for _, group := range ctx.splitPlan.Groups {
		if group.IsShared {
			continue // Skip shared types group
		}

		if err := generateServerGroupFileShared(&serverGroupContext{
			group:               group,
			opToPathMethod:      opToPathMethod,
			generatedMethods:    generatedMethods,
			packageName:         ctx.result.PackageName,
			result:              ctx.result,
			addIssue:            ctx.addIssue,
			generateRequestType: ctx.generateRequestType,
		}); err != nil {
			ctx.addIssue(fmt.Sprintf("server_%s.go", group.Name), fmt.Sprintf("failed to generate: %v", err), SeverityWarning)
		}
	}

	return nil
}

// serverGroupContext holds context for generating a single server group file.
type serverGroupContext struct {
	group               FileGroup
	opToPathMethod      map[string]OperationMapping
	generatedMethods    map[string]bool
	packageName         string
	result              *GenerateResult
	addIssue            issueAdder
	generateRequestType func(path, method string, op *parser.Operation) string
}

// generateServerGroupFileShared generates a server_{group}.go file with request types.
// The generatedMethods map indicates which methods were added to the interface.
// This is shared between OAS 2.0 and OAS 3.x generators.
//
//nolint:unparam // error return kept for API consistency and future extensibility
func generateServerGroupFileShared(ctx *serverGroupContext) error {
	var buf bytes.Buffer

	// Write header with comment about the group
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n")
	buf.WriteString(fmt.Sprintf("// This file contains %s server request types.\n\n", ctx.group.DisplayName))
	buf.WriteString(fmt.Sprintf("package %s\n\n", ctx.packageName))

	// Write minimal imports - formatAndFixImports will add/remove as needed
	buf.WriteString("import (\n")
	buf.WriteString("\t\"net/http\"\n")
	buf.WriteString(")\n\n")

	// Generate request types for each operation in this group
	for _, opID := range ctx.group.Operations {
		// Skip if method was not added to interface (was filtered as duplicate)
		if !ctx.generatedMethods[opID] {
			continue
		}

		info, ok := ctx.opToPathMethod[opID]
		if !ok {
			continue
		}

		reqType := ctx.generateRequestType(info.Path, info.Method, info.Op)
		if reqType != "" {
			buf.WriteString(reqType)
		}
	}

	// Format and append the file
	fileName := fmt.Sprintf("server_%s.go", ctx.group.Name)
	appendFormattedFile(ctx.result, fileName, &buf, ctx.addIssue)

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// OAuth2 Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// collectScopesFromFlows collects all unique scopes from OAS2 scopes map and OAS3 flows.
// This is shared between OAuth2Generator and SecurityHelperGenerator.
func collectScopesFromFlows(oas2Scopes map[string]string, flows *parser.OAuthFlows) []string {
	scopeSet := make(map[string]bool)

	// OAS 2.0 style
	for scope := range oas2Scopes {
		scopeSet[scope] = true
	}

	// OAS 3.0+ style
	if flows != nil {
		addScopesFromFlow(scopeSet, flows.Implicit)
		addScopesFromFlow(scopeSet, flows.Password)
		addScopesFromFlow(scopeSet, flows.ClientCredentials)
		addScopesFromFlow(scopeSet, flows.AuthorizationCode)
	}

	scopes := make([]string, 0, len(scopeSet))
	for scope := range scopeSet {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	return scopes
}

// addScopesFromFlow adds scopes from an OAuth flow to the scope set.
func addScopesFromFlow(scopeSet map[string]bool, flow *parser.OAuthFlow) {
	if flow == nil {
		return
	}
	for scope := range flow.Scopes {
		scopeSet[scope] = true
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Client Generation Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// generateBaseClientShared generates the base client.go with struct, constructor, and options.
// This is 100% identical between OAS 2.0 and OAS 3.x.
func generateBaseClientShared(packageName string, info *parser.Info, result *GenerateResult, addIssue issueAdder) error {
	var buf bytes.Buffer

	// Write header
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", packageName))

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
	writeClientBoilerplate(&buf, info)

	// Write helper functions
	buf.WriteString(clientHelpers)

	// Format and append the file
	appendFormattedFile(result, "client.go", &buf, addIssue)

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Security Generation Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// issueAdder is a callback for adding issues during generation.
type issueAdder func(location, message string, severity Severity)

// appendFormattedFile formats Go code and appends it to the result's file list.
// If formatting fails, the unformatted code is used with a warning issue.
func appendFormattedFile(result *GenerateResult, fileName string, buf *bytes.Buffer, addIssue issueAdder) {
	formatted, err := formatAndFixImports("generated.go", buf.Bytes())
	if err != nil {
		addIssue(fileName, fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = buf.Bytes()
	}
	result.Files = append(result.Files, GeneratedFile{Name: fileName, Content: formatted})
}

// securityGenerationContext holds the common context needed for security generation.
type securityGenerationContext struct {
	result    *GenerateResult
	splitPlan *SplitPlan
	addIssue  issueAdder
}

// generateSecurityHelpersFileShared generates the security_helpers.go file.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateSecurityHelpersFileShared(ctx *securityGenerationContext, schemes map[string]*parser.SecurityScheme) error {
	g := NewSecurityHelperGenerator(ctx.result.PackageName)
	code := g.GenerateSecurityHelpers(schemes)

	// Format the code
	formatted, err := formatAndFixImports("generated.go", []byte(code))
	if err != nil {
		ctx.addIssue("security_helpers.go", fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = []byte(code)
	}

	ctx.result.Files = append(ctx.result.Files, GeneratedFile{
		Name:    "security_helpers.go",
		Content: formatted,
	})

	return nil
}

// generateOAuth2FilesShared generates OAuth2 flow files for each OAuth2 security scheme.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateOAuth2FilesShared(ctx *securityGenerationContext, schemes map[string]*parser.SecurityScheme) error {
	for name, scheme := range schemes {
		if scheme == nil || scheme.Type != schemeTypeOAuth2 {
			continue
		}

		g := NewOAuth2Generator(name, scheme)
		if g == nil || !g.HasAnyFlow() {
			continue
		}

		code := g.GenerateOAuth2File(ctx.result.PackageName)

		// Format the code
		formatted, err := formatAndFixImports("generated.go", []byte(code))
		if err != nil {
			ctx.addIssue(fmt.Sprintf("oauth2_%s.go", name), fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
			formatted = []byte(code)
		}

		fileName := fmt.Sprintf("oauth2_%s.go", toFileName(name))
		ctx.result.Files = append(ctx.result.Files, GeneratedFile{
			Name:    fileName,
			Content: formatted,
		})
	}

	return nil
}

// generateCredentialsFileShared generates the credentials.go file.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateCredentialsFileShared(ctx *securityGenerationContext) error {
	g := NewCredentialGenerator(ctx.result.PackageName)
	code := g.GenerateCredentialsFile()

	// Format the code
	formatted, err := formatAndFixImports("generated.go", []byte(code))
	if err != nil {
		ctx.addIssue("credentials.go", fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = []byte(code)
	}

	ctx.result.Files = append(ctx.result.Files, GeneratedFile{
		Name:    "credentials.go",
		Content: formatted,
	})

	return nil
}

// generateOIDCDiscoveryFileShared generates the oidc_discovery.go file.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateOIDCDiscoveryFileShared(ctx *securityGenerationContext, schemes map[string]*parser.SecurityScheme) error {
	// Find the first OpenID Connect scheme to get the discovery URL
	var discoveryURL string
	for _, scheme := range schemes {
		if scheme != nil && scheme.Type == "openIdConnect" && scheme.OpenIDConnectURL != "" {
			discoveryURL = scheme.OpenIDConnectURL
			break
		}
	}

	g := NewOIDCDiscoveryGenerator(ctx.result.PackageName)
	code := g.GenerateOIDCDiscoveryFile(discoveryURL)

	// Format the code
	formatted, err := formatAndFixImports("generated.go", []byte(code))
	if err != nil {
		ctx.addIssue("oidc_discovery.go", fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = []byte(code)
	}

	ctx.result.Files = append(ctx.result.Files, GeneratedFile{
		Name:    "oidc_discovery.go",
		Content: formatted,
	})

	return nil
}

// generateSingleSecurityEnforceShared generates all security enforcement in a single file.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateSingleSecurityEnforceShared(ctx *securityGenerationContext, opSecurity OperationSecurityRequirements, globalSecurity []parser.SecurityRequirement) error {
	g := NewSecurityEnforceGenerator(ctx.result.PackageName)
	code := g.GenerateSecurityEnforceFile(opSecurity, globalSecurity)

	// Format the code
	formatted, err := formatAndFixImports("generated.go", []byte(code))
	if err != nil {
		ctx.addIssue("security_enforce.go", fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = []byte(code)
	}

	ctx.result.Files = append(ctx.result.Files, GeneratedFile{
		Name:    "security_enforce.go",
		Content: formatted,
	})

	return nil
}

// generateSplitSecurityEnforceShared generates security enforcement split across multiple files.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateSplitSecurityEnforceShared(ctx *securityGenerationContext, opSecurity OperationSecurityRequirements, globalSecurity []parser.SecurityRequirement) error {
	g := NewSecurityEnforceGenerator(ctx.result.PackageName)

	// Generate base file with shared types and empty map
	baseCode := g.GenerateBaseSecurityEnforceFile(globalSecurity)
	formatted, err := formatAndFixImports("generated.go", []byte(baseCode))
	if err != nil {
		ctx.addIssue("security_enforce.go", fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = []byte(baseCode)
	}
	ctx.result.Files = append(ctx.result.Files, GeneratedFile{
		Name:    "security_enforce.go",
		Content: formatted,
	})

	// Group operations by their file group
	for _, group := range ctx.splitPlan.Groups {
		if group.IsShared {
			continue
		}

		// Filter operation security for this group
		groupOpSecurity := make(OperationSecurityRequirements)
		for _, opID := range group.Operations {
			if sec, ok := opSecurity[opID]; ok {
				groupOpSecurity[opID] = sec
			}
		}

		if len(groupOpSecurity) == 0 {
			continue
		}

		// Generate group file
		groupCode := g.GenerateSecurityEnforceGroupFile(group.Name, group.DisplayName, groupOpSecurity)
		formatted, err := formatAndFixImports("generated.go", []byte(groupCode))
		if err != nil {
			ctx.addIssue(fmt.Sprintf("security_enforce_%s.go", group.Name),
				fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
			formatted = []byte(groupCode)
		}

		ctx.result.Files = append(ctx.result.Files, GeneratedFile{
			Name:    fmt.Sprintf("security_enforce_%s.go", group.Name),
			Content: formatted,
		})
	}

	return nil
}

// readmeContextBuilder holds common data needed to build a ReadmeContext.
type readmeContextBuilder struct {
	PackageName string
	OASVersion  string
	Config      *Generator
	SplitPlan   *SplitPlan
	Files       []GeneratedFile
	Info        *parser.Info
}

// buildReadmeContextShared builds the common parts of a ReadmeContext.
// The secSummaries are version-specific and must be provided by the caller.
func buildReadmeContextShared(b *readmeContextBuilder, secSummaries []SecuritySchemeSummary) *ReadmeContext {
	// Build generated file summaries
	fileSummaries := make([]GeneratedFileSummary, 0, len(b.Files))
	for _, f := range b.Files {
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
	if b.SplitPlan != nil && b.SplitPlan.NeedsSplit {
		strategy := "by tag"
		if !b.Config.SplitByTag {
			strategy = "by path prefix"
		}
		groups := make([]string, 0, len(b.SplitPlan.Groups))
		for _, grp := range b.SplitPlan.Groups {
			groups = append(groups, grp.DisplayName)
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
		PackageName: b.PackageName,
		OASVersion:  b.OASVersion,
		Config: &GeneratorConfigSummary{
			GenerateTypes:           b.Config.GenerateTypes,
			GenerateClient:          b.Config.GenerateClient,
			GenerateSecurity:        b.Config.GenerateSecurity,
			GenerateOAuth2Flows:     b.Config.GenerateOAuth2Flows,
			GenerateCredentialMgmt:  b.Config.GenerateCredentialMgmt,
			GenerateSecurityEnforce: b.Config.GenerateSecurityEnforce,
			GenerateOIDCDiscovery:   b.Config.GenerateOIDCDiscovery,
		},
		GeneratedFiles:  fileSummaries,
		SecuritySchemes: secSummaries,
		SplitInfo:       splitSummary,
	}

	// Extract API info
	if b.Info != nil {
		ctx.APITitle = b.Info.Title
		ctx.APIVersion = b.Info.Version
		ctx.APIDescription = b.Info.Description
	}

	return ctx
}

// buildSecuritySchemeSummariesOAS2 builds security scheme summaries for OAS 2.0.
func buildSecuritySchemeSummariesOAS2(schemes map[string]*parser.SecurityScheme) []SecuritySchemeSummary {
	if len(schemes) == 0 {
		return nil
	}

	// Sort scheme names for deterministic output
	names := make([]string, 0, len(schemes))
	for name := range schemes {
		names = append(names, name)
	}
	sort.Strings(names)

	summaries := make([]SecuritySchemeSummary, 0, len(names))
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
		case schemeTypeAPIKey:
			summary.Location = scheme.In
		case schemeTypeBasic:
			summary.Type = schemeTypeHTTP
			summary.Scheme = schemeTypeBasic
		case schemeTypeOAuth2:
			summary.Flows = extractOAuth2FlowNames(nil, scheme.Flow)
		}

		summaries = append(summaries, summary)
	}

	return summaries
}

// buildSecuritySchemeSummariesOAS3 builds security scheme summaries for OAS 3.x.
func buildSecuritySchemeSummariesOAS3(schemes map[string]*parser.SecurityScheme) []SecuritySchemeSummary {
	if len(schemes) == 0 {
		return nil
	}

	// Sort scheme names for deterministic output
	names := make([]string, 0, len(schemes))
	for name := range schemes {
		names = append(names, name)
	}
	sort.Strings(names)

	summaries := make([]SecuritySchemeSummary, 0, len(names))
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

		summaries = append(summaries, summary)
	}

	return summaries
}

// ═══════════════════════════════════════════════════════════════════════════════
// Status Code Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// parseStatusCodeMetadata parses status code metadata from a code string.
// This extracts common status code handling shared between OAS 2.0 and OAS 3.x.
// It handles:
//   - "default" → StatusDefault, IsDefault=true, StatusCodeInt=500
//   - "2XX", "4XX", etc. → Status2XX, IsWildcard=true, StatusCodeInt=200/400/etc.
//   - "200", "404", etc. → Status200, StatusCodeInt=200, IsSuccess=true if 2xx
//
// Returns a StatusCodeData with metadata fields populated. The caller should
// fill in Description, HasBody, ContentType, and BodyType as appropriate.
func parseStatusCodeMetadata(code string) StatusCodeData {
	data := StatusCodeData{Code: code}

	switch {
	case code == "default":
		data.MethodName = "StatusDefault"
		data.StatusCodeInt = 500
		data.IsDefault = true

	case len(code) == 3 && strings.HasSuffix(code, "XX"):
		// Wildcard like 2XX, 4XX, 5XX
		data.MethodName = "Status" + code
		data.IsWildcard = true
		switch code[0] {
		case '2':
			data.StatusCodeInt = 200
			data.IsSuccess = true
		case '3':
			data.StatusCodeInt = 300
		case '4':
			data.StatusCodeInt = 400
		case '5':
			data.StatusCodeInt = 500
		}

	default:
		// Specific status code
		data.MethodName = "Status" + code
		var statusInt int
		if _, err := fmt.Sscanf(code, "%d", &statusInt); err == nil {
			data.StatusCodeInt = statusInt
			data.IsSuccess = statusInt >= 200 && statusInt < 300
		}
	}

	return data
}

// ═══════════════════════════════════════════════════════════════════════════════
// Security Files Orchestration
// ═══════════════════════════════════════════════════════════════════════════════

// securityFileCallbacks holds callbacks for generating security-related files.
// Each callback handles version-specific file generation.
type securityFileCallbacks struct {
	generateCredentials     func() error
	generateSecurityEnforce func() error
	generateOIDCDiscovery   func(map[string]*parser.SecurityScheme) error
	generateReadme          func(map[string]*parser.SecurityScheme) error
}

// generateSecurityFilesOrchestrated orchestrates the generation of optional security files.
// This pattern is 100% identical between OAS 2.0 and OAS 3.x.
func generateSecurityFilesOrchestrated(g *Generator, schemes map[string]*parser.SecurityScheme, cb securityFileCallbacks) error {
	// Generate credential management if enabled
	if g.GenerateCredentialMgmt {
		if err := cb.generateCredentials(); err != nil {
			return fmt.Errorf("failed to generate credentials: %w", err)
		}
	}

	// Generate security enforcement if enabled
	if g.GenerateSecurityEnforce {
		if err := cb.generateSecurityEnforce(); err != nil {
			return fmt.Errorf("failed to generate security enforcement: %w", err)
		}
	}

	// Generate OIDC discovery if enabled
	if g.GenerateOIDCDiscovery && len(schemes) > 0 {
		if err := cb.generateOIDCDiscovery(schemes); err != nil {
			return fmt.Errorf("failed to generate OIDC discovery: %w", err)
		}
	}

	// Generate README if enabled
	if g.GenerateReadme {
		if err := cb.generateReadme(schemes); err != nil {
			return fmt.Errorf("failed to generate README: %w", err)
		}
	}

	return nil
}

// fullSecurityCallbacks extends securityFileCallbacks with additional generation callbacks.
// This type supports the unified generateAllSecurityHelpers orchestration function.
type fullSecurityCallbacks struct {
	generateSecurityHelpersFile func(map[string]*parser.SecurityScheme) error
	generateOAuth2Files         func(map[string]*parser.SecurityScheme) error
	generateCredentials         func() error
	generateSecurityEnforce     func() error
	generateOIDCDiscovery       func(map[string]*parser.SecurityScheme) error
	generateReadme              func(map[string]*parser.SecurityScheme) error
}

// generateAllSecurityHelpers orchestrates the complete security helper generation.
// This function is identical between OAS 2.0 and OAS 3.x - only the scheme source differs.
func generateAllSecurityHelpers(g *Generator, schemes map[string]*parser.SecurityScheme, cb fullSecurityCallbacks) error {
	if !g.GenerateClient {
		return nil
	}

	// Generate security helpers if enabled
	if g.GenerateSecurity && len(schemes) > 0 {
		if err := cb.generateSecurityHelpersFile(schemes); err != nil {
			return fmt.Errorf("failed to generate security helpers: %w", err)
		}
	}

	// Generate OAuth2 flows if enabled
	if g.GenerateOAuth2Flows && len(schemes) > 0 {
		if err := cb.generateOAuth2Files(schemes); err != nil {
			return fmt.Errorf("failed to generate OAuth2 flows: %w", err)
		}
	}

	// Generate optional security files using shared orchestration
	return generateSecurityFilesOrchestrated(g, schemes, securityFileCallbacks{
		generateCredentials:     cb.generateCredentials,
		generateSecurityEnforce: cb.generateSecurityEnforce,
		generateOIDCDiscovery:   cb.generateOIDCDiscovery,
		generateReadme:          cb.generateReadme,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Response Handling Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// statusCodeDataBuilder is a callback for building status code data.
// This allows version-specific handling of response schemas.
type statusCodeDataBuilder func(code string, resp *parser.Response) StatusCodeData

// buildStatusCodesShared builds status code data for an operation's responses.
// This is 100% identical between OAS 2.0 and OAS 3.x except for the buildData callback.
func buildStatusCodesShared(op *parser.Operation, buildData statusCodeDataBuilder) []StatusCodeData {
	if op.Responses == nil {
		return nil
	}

	// Pre-allocate: 1 for default + status codes
	codes := make([]StatusCodeData, 0, 1+len(op.Responses.Codes))

	// Process default response first
	if op.Responses.Default != nil {
		statusData := buildData("default", op.Responses.Default)
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
		statusData := buildData(code, resp)
		codes = append(codes, statusData)
	}

	return codes
}
