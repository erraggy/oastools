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
}

// generateBaseServerShared generates the base server.go with interface and unimplemented server.
// Returns a map of generated method names (for use by group file generation).
func generateBaseServerShared(ctx *baseServerContext) (map[string]bool, error) {
	var buf bytes.Buffer

	// Write header
	buf.WriteString("// Code generated by oastools. DO NOT EDIT.\n\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", ctx.packageName))

	// Write imports
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
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
	buf.WriteString("// ErrNotImplemented is returned by UnimplementedServer methods.\n")
	buf.WriteString("var ErrNotImplemented = &NotImplementedError{}\n\n")
	buf.WriteString("// NotImplementedError indicates an operation is not implemented.\n")
	buf.WriteString("type NotImplementedError struct{}\n\n")
	buf.WriteString("func (e *NotImplementedError) Error() string { return \"not implemented\" }\n\n")

	// Format the code
	formatted, err := formatAndFixImports("generated.go", buf.Bytes())
	if err != nil {
		ctx.addIssue("server.go", fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = buf.Bytes()
	}

	ctx.result.Files = append(ctx.result.Files, GeneratedFile{
		Name:    "server.go",
		Content: formatted,
	})

	return generatedMethods, nil
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

	// Format the code
	formatted, err := formatAndFixImports("generated.go", buf.Bytes())
	if err != nil {
		addIssue("client.go", fmt.Sprintf("failed to format generated code: %v", err), SeverityWarning)
		formatted = buf.Bytes()
	}

	result.Files = append(result.Files, GeneratedFile{
		Name:    "client.go",
		Content: formatted,
	})

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Security Generation Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// issueAdder is a callback for adding issues during generation.
type issueAdder func(location, message string, severity Severity)

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
