package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
)

// securityDetailView wraps a security scheme with its name for serialization.
type securityDetailView struct {
	Name           string                 `json:"name" yaml:"name"`
	SecurityScheme *parser.SecurityScheme `json:"securityScheme" yaml:"securityScheme"`
}

// handleWalkSecurity implements the "walk security" subcommand.
// It collects security schemes from the spec, applies filters, and renders output.
func handleWalkSecurity(args []string) error {
	fs := flag.NewFlagSet("walk security", flag.ContinueOnError)

	// Security-specific flags
	name := fs.String("name", "", "Filter by security scheme name")
	schemeType := fs.String("type", "", "Filter by type (apiKey, http, oauth2, openIdConnect)")

	// Common walk flags
	var flags WalkFlags
	fs.StringVar(&flags.Format, "format", FormatText, "Output format: text, json, yaml")
	fs.BoolVar(&flags.Quiet, "quiet", false, "Suppress headers and decoration")
	fs.BoolVar(&flags.Quiet, "q", false, "Suppress headers and decoration (shorthand)")
	fs.BoolVar(&flags.Detail, "detail", false, "Show full security scheme instead of summary table")
	fs.StringVar(&flags.Extension, "extension", "", "Filter by extension (e.g., x-scope=internal)")
	fs.BoolVar(&flags.ResolveRefs, "resolve-refs", false, "Resolve $ref pointers before output")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if err := ValidateOutputFormat(flags.Format); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("walk security requires a spec file argument")
	}
	specPath := fs.Arg(0)

	// 1. Collect: parse spec and collect security schemes
	result, err := parseSpec(specPath, flags.ResolveRefs)
	if err != nil {
		return fmt.Errorf("walk security: %w", err)
	}

	collector, err := walker.CollectSecuritySchemes(result)
	if err != nil {
		return fmt.Errorf("walk security: collecting security schemes: %w", err)
	}

	// 2. Filter
	matched, err := filterSecuritySchemes(collector.All, *name, *schemeType, flags.Extension)
	if err != nil {
		return err
	}

	if len(matched) == 0 {
		renderNoResults("security schemes", flags.Quiet)
		return nil
	}

	// 3. Render
	if flags.Detail {
		return renderSecurityDetail(matched, flags)
	}
	return renderSecuritySummary(matched, flags)
}

// filterSecuritySchemes applies all security scheme filters and returns the matching subset.
func filterSecuritySchemes(
	schemes []*walker.SecuritySchemeInfo,
	name, schemeType, extension string,
) ([]*walker.SecuritySchemeInfo, error) {
	// Parse extension filter once if provided
	var extFilter *ExtensionFilter
	if extension != "" {
		ef, err := ParseExtensionFilter(extension)
		if err != nil {
			return nil, fmt.Errorf("walk security: %w", err)
		}
		extFilter = &ef
	}

	var matched []*walker.SecuritySchemeInfo
	for _, info := range schemes {
		if info == nil || info.SecurityScheme == nil {
			continue
		}
		if name != "" && info.Name != name {
			continue
		}
		if schemeType != "" && !strings.EqualFold(info.SecurityScheme.Type, schemeType) {
			continue
		}
		if extFilter != nil && !extFilter.Match(info.SecurityScheme.Extra) {
			continue
		}
		matched = append(matched, info)
	}
	return matched, nil
}

// renderSecuritySummary renders a summary table of security schemes.
func renderSecuritySummary(schemes []*walker.SecuritySchemeInfo, flags WalkFlags) error {
	headers := []string{"NAME", "TYPE", "SCHEME", "IN", "EXTENSIONS"}
	rows := make([][]string, 0, len(schemes))

	for _, info := range schemes {
		rows = append(rows, []string{
			info.Name,
			info.SecurityScheme.Type,
			info.SecurityScheme.Scheme,
			info.SecurityScheme.In,
			FormatExtensions(info.SecurityScheme.Extra),
		})
	}

	if flags.Format != FormatText {
		return RenderSummaryStructured(os.Stdout, headers, rows, flags.Format)
	}
	RenderSummaryTable(os.Stdout, headers, rows, flags.Quiet)
	return nil
}

// renderSecurityDetail renders each matched security scheme in full detail.
func renderSecurityDetail(schemes []*walker.SecuritySchemeInfo, flags WalkFlags) error {
	for _, info := range schemes {
		view := securityDetailView{
			Name:           info.Name,
			SecurityScheme: info.SecurityScheme,
		}
		if err := RenderDetail(os.Stdout, view, flags.Format, flags.Quiet); err != nil {
			return fmt.Errorf("walk security: rendering detail: %w", err)
		}
	}
	return nil
}
