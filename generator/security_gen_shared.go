package generator

import (
	"bytes"
	"fmt"
	"net/url"
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
// Security Generation Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// isSecureURL checks whether a URL uses HTTPS, or HTTP only for localhost/127.0.0.1.
// This is used to warn about insecure OAuth2/OIDC endpoint URLs during code generation.
func isSecureURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	if u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1") {
		return true
	}
	return false
}

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
func generateSecurityHelpersFileShared(ctx *securityGenerationContext, schemes map[string]*parser.SecurityScheme) {
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
}

// generateOAuth2FilesShared generates OAuth2 flow files for each OAuth2 security scheme.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateOAuth2FilesShared(ctx *securityGenerationContext, schemes map[string]*parser.SecurityScheme) {
	for name, scheme := range schemes {
		if scheme == nil || scheme.Type != schemeTypeOAuth2 {
			continue
		}

		g := NewOAuth2Generator(name, scheme)
		if g == nil || !g.HasAnyFlow() {
			continue
		}

		// Validate OAuth2 endpoint URLs use HTTPS (or HTTP for localhost)
		authURL, tokenURL := g.getURLs()
		if authURL != "" && !isSecureURL(authURL) {
			ctx.addIssue(fmt.Sprintf("securitySchemes.%s", name),
				fmt.Sprintf("OAuth2 authorization URL uses insecure scheme: %s", authURL), SeverityWarning)
		}
		if tokenURL != "" && !isSecureURL(tokenURL) {
			ctx.addIssue(fmt.Sprintf("securitySchemes.%s", name),
				fmt.Sprintf("OAuth2 token URL uses insecure scheme: %s", tokenURL), SeverityWarning)
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
}

// generateCredentialsFileShared generates the credentials.go file.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateCredentialsFileShared(ctx *securityGenerationContext) {
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
}

// generateOIDCDiscoveryFileShared generates the oidc_discovery.go file.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateOIDCDiscoveryFileShared(ctx *securityGenerationContext, schemes map[string]*parser.SecurityScheme) {
	// Find the first OpenID Connect scheme to get the discovery URL
	var discoveryURL string
	for _, scheme := range schemes {
		if scheme != nil && scheme.Type == "openIdConnect" && scheme.OpenIDConnectURL != "" {
			discoveryURL = scheme.OpenIDConnectURL
			break
		}
	}

	// Validate OIDC discovery URL uses HTTPS (or HTTP for localhost)
	if discoveryURL != "" && !isSecureURL(discoveryURL) {
		ctx.addIssue("securitySchemes.openIdConnect",
			fmt.Sprintf("OpenID Connect discovery URL uses insecure scheme: %s", discoveryURL), SeverityWarning)
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
}

// generateSingleSecurityEnforceShared generates all security enforcement in a single file.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateSingleSecurityEnforceShared(ctx *securityGenerationContext, opSecurity OperationSecurityRequirements, globalSecurity []parser.SecurityRequirement) {
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
}

// generateSplitSecurityEnforceShared generates security enforcement split across multiple files.
// This is shared between OAS 2.0 and OAS 3.x generators.
func generateSplitSecurityEnforceShared(ctx *securityGenerationContext, opSecurity OperationSecurityRequirements, globalSecurity []parser.SecurityRequirement) {
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
	generateCredentials     func()
	generateSecurityEnforce func()
	generateOIDCDiscovery   func(map[string]*parser.SecurityScheme)
	generateReadme          func(map[string]*parser.SecurityScheme)
}

// generateSecurityFilesOrchestrated orchestrates the generation of optional security files.
// This pattern is 100% identical between OAS 2.0 and OAS 3.x.
func generateSecurityFilesOrchestrated(g *Generator, schemes map[string]*parser.SecurityScheme, cb securityFileCallbacks) {
	// Generate credential management if enabled
	if g.GenerateCredentialMgmt {
		cb.generateCredentials()
	}

	// Generate security enforcement if enabled
	if g.GenerateSecurityEnforce {
		cb.generateSecurityEnforce()
	}

	// Generate OIDC discovery if enabled
	if g.GenerateOIDCDiscovery && len(schemes) > 0 {
		cb.generateOIDCDiscovery(schemes)
	}

	// Generate README if enabled
	if g.GenerateReadme {
		cb.generateReadme(schemes)
	}
}

// fullSecurityCallbacks extends securityFileCallbacks with additional generation callbacks.
// This type supports the unified generateAllSecurityHelpers orchestration function.
type fullSecurityCallbacks struct {
	generateSecurityHelpersFile func(map[string]*parser.SecurityScheme)
	generateOAuth2Files         func(map[string]*parser.SecurityScheme)
	generateCredentials         func()
	generateSecurityEnforce     func()
	generateOIDCDiscovery       func(map[string]*parser.SecurityScheme)
	generateReadme              func(map[string]*parser.SecurityScheme)
}

// generateAllSecurityHelpers orchestrates the complete security helper generation.
// This function is identical between OAS 2.0 and OAS 3.x - only the scheme source differs.
func generateAllSecurityHelpers(g *Generator, schemes map[string]*parser.SecurityScheme, cb fullSecurityCallbacks) {
	if !g.GenerateClient {
		return
	}

	// Generate security helpers if enabled
	if g.GenerateSecurity && len(schemes) > 0 {
		cb.generateSecurityHelpersFile(schemes)
	}

	// Generate OAuth2 flows if enabled
	if g.GenerateOAuth2Flows && len(schemes) > 0 {
		cb.generateOAuth2Files(schemes)
	}

	// Generate optional security files using shared orchestration
	generateSecurityFilesOrchestrated(g, schemes, securityFileCallbacks{
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
