package generator

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/erraggy/oastools/parser"
)

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
