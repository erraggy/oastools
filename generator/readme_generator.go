package generator

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Security scheme type constants for readme generation
const (
	schemeTypeAPIKey        = "apiKey"
	schemeTypeHTTP          = "http"
	schemeTypeBasic         = "basic"
	schemeTypeOAuth2        = "oauth2"
	schemeTypeOpenIDConnect = "openIdConnect"
)

// ReadmeGenerator generates README.md files for generated code.
type ReadmeGenerator struct{}

// NewReadmeGenerator creates a new ReadmeGenerator.
func NewReadmeGenerator() *ReadmeGenerator {
	return &ReadmeGenerator{}
}

// ReadmeContext contains all information needed to generate a README.
type ReadmeContext struct {
	// Timestamp is when the code was generated.
	Timestamp time.Time

	// OastoolsVersion is the version of oastools used.
	OastoolsVersion string

	// SourcePath is the path to the source OAS file(s).
	SourcePath string

	// OASVersion is the OpenAPI version (e.g., "3.0.3", "2.0").
	OASVersion string

	// APITitle is the title from the OAS info section.
	APITitle string

	// APIVersion is the version from the OAS info section.
	APIVersion string

	// APIDescription is the description from the OAS info section.
	APIDescription string

	// CLICommand is the command used to generate this code.
	CLICommand string

	// PackageName is the Go package name.
	PackageName string

	// GeneratedFiles lists all generated files with descriptions.
	GeneratedFiles []GeneratedFileSummary

	// SecuritySchemes describes the security schemes in the API.
	SecuritySchemes []SecuritySchemeSummary

	// SplitInfo contains information about file splitting (if applicable).
	SplitInfo *SplitSummary

	// Config contains the generator configuration used.
	Config *GeneratorConfigSummary
}

// GeneratedFileSummary describes a generated file.
type GeneratedFileSummary struct {
	// FileName is the name of the generated file.
	FileName string

	// Description describes what the file contains.
	Description string

	// LineCount is the number of lines in the file (optional).
	LineCount int
}

// SecuritySchemeSummary describes a security scheme.
type SecuritySchemeSummary struct {
	// Name is the security scheme name.
	Name string

	// Type is the security type (apiKey, http, oauth2, openIdConnect).
	Type string

	// Description is the security scheme description.
	Description string

	// Location is where the credential is sent (header, query, cookie) for apiKey.
	Location string

	// Scheme is the HTTP scheme (basic, bearer) for http type.
	Scheme string

	// Flows lists the OAuth2 flows available.
	Flows []string

	// OpenIDConnectURL is the OIDC discovery URL.
	OpenIDConnectURL string
}

// SplitSummary describes how files were split.
type SplitSummary struct {
	// WasSplit indicates if files were split.
	WasSplit bool

	// Strategy describes how files were split (by tag, by path prefix).
	Strategy string

	// Groups lists the split groups.
	Groups []string

	// SharedTypesFile is the name of the shared types file (if any).
	SharedTypesFile string
}

// GeneratorConfigSummary describes the generator configuration.
type GeneratorConfigSummary struct {
	// GenerateTypes indicates if types were generated.
	GenerateTypes bool

	// GenerateClient indicates if client code was generated.
	GenerateClient bool

	// GenerateSecurity indicates if security helpers were generated.
	GenerateSecurity bool

	// GenerateOAuth2Flows indicates if OAuth2 flow helpers were generated.
	GenerateOAuth2Flows bool

	// GenerateCredentialMgmt indicates if credential management was generated.
	GenerateCredentialMgmt bool

	// GenerateSecurityEnforce indicates if security enforcement was generated.
	GenerateSecurityEnforce bool

	// GenerateOIDCDiscovery indicates if OIDC discovery was generated.
	GenerateOIDCDiscovery bool
}

// GenerateReadme generates a README.md file.
func (g *ReadmeGenerator) GenerateReadme(ctx *ReadmeContext) string {
	var buf bytes.Buffer

	// Header
	buf.WriteString(g.generateHeader(ctx))

	// Overview
	buf.WriteString(g.generateOverview(ctx))

	// Generated files section
	buf.WriteString(g.generateFilesSection(ctx))

	// Security section (if applicable)
	if len(ctx.SecuritySchemes) > 0 {
		buf.WriteString(g.generateSecuritySection(ctx))
	}

	// Usage section
	buf.WriteString(g.generateUsageSection(ctx))

	// Regeneration section
	buf.WriteString(g.generateRegenerationSection(ctx))

	// Footer
	buf.WriteString(g.generateFooter(ctx))

	return buf.String()
}

// generateHeader generates the README header.
func (g *ReadmeGenerator) generateHeader(ctx *ReadmeContext) string {
	var buf bytes.Buffer

	title := ctx.APITitle
	if title == "" {
		title = "Generated API Client"
	}

	fmt.Fprintf(&buf, "# %s\n\n", title)

	if ctx.APIDescription != "" {
		fmt.Fprintf(&buf, "%s\n\n", ctx.APIDescription)
	}

	return buf.String()
}

// generateOverview generates the overview section.
func (g *ReadmeGenerator) generateOverview(ctx *ReadmeContext) string {
	var buf bytes.Buffer

	buf.WriteString("## Overview\n\n")

	buf.WriteString("This package was generated by [oastools](https://github.com/erraggy/oastools) from an OpenAPI specification.\n\n")

	buf.WriteString("| Property | Value |\n")
	buf.WriteString("|----------|-------|\n")

	if ctx.APIVersion != "" {
		fmt.Fprintf(&buf, "| API Version | %s |\n", ctx.APIVersion)
	}
	if ctx.OASVersion != "" {
		fmt.Fprintf(&buf, "| OpenAPI Version | %s |\n", ctx.OASVersion)
	}
	fmt.Fprintf(&buf, "| Package | `%s` |\n", ctx.PackageName)
	if ctx.OastoolsVersion != "" {
		fmt.Fprintf(&buf, "| Generator Version | %s |\n", ctx.OastoolsVersion)
	}
	fmt.Fprintf(&buf, "| Generated | %s |\n", ctx.Timestamp.Format(time.RFC3339))

	buf.WriteString("\n")

	return buf.String()
}

// generateFilesSection generates the files section.
func (g *ReadmeGenerator) generateFilesSection(ctx *ReadmeContext) string {
	var buf bytes.Buffer

	buf.WriteString("## Generated Files\n\n")

	if len(ctx.GeneratedFiles) == 0 {
		buf.WriteString("No files were generated.\n\n")
		return buf.String()
	}

	buf.WriteString("| File | Description |\n")
	buf.WriteString("|------|-------------|\n")

	for _, f := range ctx.GeneratedFiles {
		desc := f.Description
		if f.LineCount > 0 {
			desc = fmt.Sprintf("%s (%d lines)", desc, f.LineCount)
		}
		fmt.Fprintf(&buf, "| `%s` | %s |\n", f.FileName, desc)
	}

	buf.WriteString("\n")

	// Split information
	if ctx.SplitInfo != nil && ctx.SplitInfo.WasSplit {
		buf.WriteString("### File Organization\n\n")
		fmt.Fprintf(&buf, "Files were split %s.\n\n", ctx.SplitInfo.Strategy)

		if len(ctx.SplitInfo.Groups) > 0 {
			buf.WriteString("**Groups:**\n")
			for _, group := range ctx.SplitInfo.Groups {
				fmt.Fprintf(&buf, "- %s\n", group)
			}
			buf.WriteString("\n")
		}

		if ctx.SplitInfo.SharedTypesFile != "" {
			fmt.Fprintf(&buf, "**Shared types:** `%s`\n\n", ctx.SplitInfo.SharedTypesFile)
		}
	}

	return buf.String()
}

// generateSecuritySection generates the security section.
func (g *ReadmeGenerator) generateSecuritySection(ctx *ReadmeContext) string {
	var buf bytes.Buffer

	buf.WriteString("## Security\n\n")
	buf.WriteString("This API uses the following authentication methods:\n\n")

	for _, sec := range ctx.SecuritySchemes {
		fmt.Fprintf(&buf, "### %s\n\n", sec.Name)

		if sec.Description != "" {
			fmt.Fprintf(&buf, "%s\n\n", sec.Description)
		}

		fmt.Fprintf(&buf, "- **Type:** %s\n", sec.Type)

		switch sec.Type {
		case schemeTypeAPIKey:
			fmt.Fprintf(&buf, "- **Location:** %s\n", sec.Location)
		case schemeTypeHTTP:
			fmt.Fprintf(&buf, "- **Scheme:** %s\n", sec.Scheme)
		case schemeTypeOAuth2:
			if len(sec.Flows) > 0 {
				fmt.Fprintf(&buf, "- **Flows:** %s\n", strings.Join(sec.Flows, ", "))
			}
		case schemeTypeOpenIDConnect:
			if sec.OpenIDConnectURL != "" {
				fmt.Fprintf(&buf, "- **Discovery URL:** %s\n", sec.OpenIDConnectURL)
			}
		}

		buf.WriteString("\n")
	}

	return buf.String()
}

// generateUsageSection generates the usage section.
func (g *ReadmeGenerator) generateUsageSection(ctx *ReadmeContext) string {
	var buf bytes.Buffer

	buf.WriteString("## Usage\n\n")

	// Basic client creation
	buf.WriteString("### Creating a Client\n\n")
	buf.WriteString("```go\n")
	fmt.Fprintf(&buf, "import \"%s\"\n\n", ctx.PackageName)
	buf.WriteString("client, err := NewClient(\"https://api.example.com\")\n")
	buf.WriteString("if err != nil {\n")
	buf.WriteString("    log.Fatal(err)\n")
	buf.WriteString("}\n")
	buf.WriteString("```\n\n")

	// Security usage examples
	if len(ctx.SecuritySchemes) > 0 {
		buf.WriteString("### Authentication\n\n")

		for _, sec := range ctx.SecuritySchemes {
			funcName := sanitizeSecurityFunctionName(sec.Name)

			switch sec.Type {
			case schemeTypeAPIKey:
				switch sec.Location {
				case "header":
					fmt.Fprintf(&buf, "**%s (API Key in Header):**\n", sec.Name)
					buf.WriteString("```go\n")
					fmt.Fprintf(&buf, "client, err := NewClient(baseURL, With%sAPIKey(\"your-api-key\"))\n", funcName)
					buf.WriteString("```\n\n")
				case "query":
					fmt.Fprintf(&buf, "**%s (API Key in Query):**\n", sec.Name)
					buf.WriteString("```go\n")
					fmt.Fprintf(&buf, "client, err := NewClient(baseURL, With%sAPIKeyQuery(\"your-api-key\"))\n", funcName)
					buf.WriteString("```\n\n")
				case "cookie":
					fmt.Fprintf(&buf, "**%s (API Key in Cookie):**\n", sec.Name)
					buf.WriteString("```go\n")
					fmt.Fprintf(&buf, "client, err := NewClient(baseURL, With%sAPIKeyCookie(\"your-api-key\"))\n", funcName)
					buf.WriteString("```\n\n")
				}
			case schemeTypeHTTP:
				switch sec.Scheme {
				case "basic":
					fmt.Fprintf(&buf, "**%s (Basic Auth):**\n", sec.Name)
					buf.WriteString("```go\n")
					fmt.Fprintf(&buf, "client, err := NewClient(baseURL, With%sBasicAuth(\"username\", \"password\"))\n", funcName)
					buf.WriteString("```\n\n")
				case "bearer":
					fmt.Fprintf(&buf, "**%s (Bearer Token):**\n", sec.Name)
					buf.WriteString("```go\n")
					fmt.Fprintf(&buf, "client, err := NewClient(baseURL, With%sBearerToken(\"your-token\"))\n", funcName)
					buf.WriteString("```\n\n")
				}
			case schemeTypeOAuth2:
				fmt.Fprintf(&buf, "**%s (OAuth2):**\n", sec.Name)
				buf.WriteString("```go\n")
				fmt.Fprintf(&buf, "client, err := NewClient(baseURL, With%sOAuth2Token(\"your-access-token\"))\n", funcName)
				buf.WriteString("```\n\n")

				if ctx.Config != nil && ctx.Config.GenerateOAuth2Flows {
					buf.WriteString("For OAuth2 token management, see the generated OAuth2 client:\n")
					buf.WriteString("```go\n")
					fmt.Fprintf(&buf, "oauth2Client := New%sOAuth2Client(config)\n", funcName)
					buf.WriteString("// Use authorization code flow, client credentials, etc.\n")
					buf.WriteString("```\n\n")
				}
			case schemeTypeOpenIDConnect:
				fmt.Fprintf(&buf, "**%s (OpenID Connect):**\n", sec.Name)
				buf.WriteString("```go\n")
				fmt.Fprintf(&buf, "client, err := NewClient(baseURL, With%sToken(\"your-access-token\"))\n", funcName)
				buf.WriteString("```\n\n")

				if ctx.Config != nil && ctx.Config.GenerateOIDCDiscovery {
					buf.WriteString("For OIDC discovery, use the discovery client:\n")
					buf.WriteString("```go\n")
					buf.WriteString("discoveryClient := NewOIDCDiscoveryClient(discoveryURL)\n")
					buf.WriteString("config, err := discoveryClient.GetConfiguration(ctx)\n")
					buf.WriteString("```\n\n")
				}
			}
		}
	}

	// Credential provider usage
	if ctx.Config != nil && ctx.Config.GenerateCredentialMgmt {
		buf.WriteString("### Credential Providers\n\n")
		buf.WriteString("Use credential providers for dynamic credential management:\n\n")
		buf.WriteString("```go\n")
		buf.WriteString("// Environment variables\n")
		buf.WriteString("envProvider := NewEnvCredentialProvider(\"MYAPP_\")\n")
		buf.WriteString("client, err := NewClient(baseURL, WithCredentialProvider(envProvider, \"api_key\"))\n\n")
		buf.WriteString("// Memory provider (for testing)\n")
		buf.WriteString("memProvider := NewMemoryCredentialProvider()\n")
		buf.WriteString("memProvider.Set(\"api_key\", \"test-key\")\n")
		buf.WriteString("client, err := NewClient(baseURL, WithCredentialProvider(memProvider, \"api_key\"))\n\n")
		buf.WriteString("// Chain providers (try memory first, then env)\n")
		buf.WriteString("chain := NewCredentialChain(memProvider, envProvider)\n")
		buf.WriteString("client, err := NewClient(baseURL, WithCredentialProvider(chain, \"api_key\"))\n")
		buf.WriteString("```\n\n")
	}

	// Security validation usage
	if ctx.Config != nil && ctx.Config.GenerateSecurityEnforce {
		buf.WriteString("### Security Validation\n\n")
		buf.WriteString("Validate that required security is configured:\n\n")
		buf.WriteString("```go\n")
		buf.WriteString("validator := NewSecurityValidator()\n")
		buf.WriteString("validator.ConfigureScheme(\"oauth2\", \"read\", \"write\")\n\n")
		buf.WriteString("if err := validator.ValidateOperation(\"listUsers\"); err != nil {\n")
		buf.WriteString("    log.Printf(\"Security not configured: %v\", err)\n")
		buf.WriteString("}\n")
		buf.WriteString("```\n\n")
	}

	return buf.String()
}

// generateRegenerationSection generates the regeneration section.
func (g *ReadmeGenerator) generateRegenerationSection(ctx *ReadmeContext) string {
	var buf bytes.Buffer

	buf.WriteString("## Regeneration\n\n")
	buf.WriteString("To regenerate this code, run:\n\n")
	buf.WriteString("```bash\n")

	if ctx.CLICommand != "" {
		buf.WriteString(ctx.CLICommand)
	} else {
		// Build a reasonable default command
		cmd := "oastools generate"
		if ctx.SourcePath != "" {
			cmd += fmt.Sprintf(" %s", ctx.SourcePath)
		}
		if ctx.PackageName != "" {
			cmd += fmt.Sprintf(" --package %s", ctx.PackageName)
		}
		buf.WriteString(cmd)
	}

	buf.WriteString("\n```\n\n")

	buf.WriteString("> **Note:** Do not edit generated files directly. Make changes to the OpenAPI specification and regenerate.\n\n")

	return buf.String()
}

// generateFooter generates the footer.
func (g *ReadmeGenerator) generateFooter(ctx *ReadmeContext) string {
	var buf bytes.Buffer

	buf.WriteString("---\n\n")
	buf.WriteString("Generated by [oastools](https://github.com/erraggy/oastools)")

	if ctx.OastoolsVersion != "" {
		fmt.Fprintf(&buf, " v%s", ctx.OastoolsVersion)
	}

	buf.WriteString("\n")

	return buf.String()
}

// ExtractSecuritySchemeSummaries extracts security scheme summaries from a map.
func ExtractSecuritySchemeSummaries(schemes map[string]*SecurityScheme) []SecuritySchemeSummary {
	if len(schemes) == 0 {
		return nil
	}

	result := make([]SecuritySchemeSummary, 0, len(schemes))

	// Sort for deterministic output
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
		case schemeTypeAPIKey:
			summary.Location = scheme.In
		case schemeTypeHTTP:
			summary.Scheme = scheme.Scheme
		case schemeTypeOAuth2:
			summary.Flows = extractOAuth2FlowNames(scheme.Flows, scheme.Flow)
		case schemeTypeOpenIDConnect:
			summary.OpenIDConnectURL = scheme.OpenIDConnectURL
		}

		result = append(result, summary)
	}

	return result
}

// extractOAuth2FlowNames extracts the names of available OAuth2 flows.
func extractOAuth2FlowNames(flows *OAuthFlows, legacyFlow string) []string {
	var names []string

	// OAS 3.0+ style
	if flows != nil {
		if flows.AuthorizationCode != nil {
			names = append(names, "authorization_code")
		}
		if flows.ClientCredentials != nil {
			names = append(names, "client_credentials")
		}
		if flows.Password != nil {
			names = append(names, "password")
		}
		if flows.Implicit != nil {
			names = append(names, "implicit")
		}
	}

	// OAS 2.0 style
	if len(names) == 0 && legacyFlow != "" {
		switch legacyFlow {
		case "accessCode", "authorizationCode":
			names = append(names, "authorization_code")
		case "application", "clientCredentials":
			names = append(names, "client_credentials")
		case "password":
			names = append(names, "password")
		case "implicit":
			names = append(names, "implicit")
		}
	}

	return names
}

// SecurityScheme mirrors parser.SecurityScheme for use in readme generation.
type SecurityScheme struct {
	Type             string
	Description      string
	Name             string
	In               string
	Scheme           string
	BearerFormat     string
	Flows            *OAuthFlows
	OpenIDConnectURL string
	Flow             string // OAS 2.0
}

// OAuthFlows mirrors parser.OAuthFlows for use in readme generation.
type OAuthFlows struct {
	Implicit          *OAuthFlow
	Password          *OAuthFlow
	ClientCredentials *OAuthFlow
	AuthorizationCode *OAuthFlow
}

// OAuthFlow mirrors parser.OAuthFlow for use in readme generation.
type OAuthFlow struct {
	AuthorizationURL string
	TokenURL         string
	RefreshURL       string
	Scopes           map[string]string
}
