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

// responseDetailView wraps a response with its walker context for serialization.
type responseDetailView struct {
	StatusCode string           `json:"statusCode" yaml:"statusCode"`
	Path       string           `json:"path" yaml:"path"`
	Method     string           `json:"method" yaml:"method"`
	Response   *parser.Response `json:"response" yaml:"response"`
}

// WalkResponsesFlags contains flags for the walk responses subcommand.
type WalkResponsesFlags struct {
	Status string
	Path   string
	Method string
	WalkFlags
}

// SetupWalkResponsesFlags creates and configures a FlagSet for the walk responses subcommand.
func SetupWalkResponsesFlags() (*flag.FlagSet, *WalkResponsesFlags) {
	fs := flag.NewFlagSet("walk responses", flag.ContinueOnError)
	flags := &WalkResponsesFlags{}

	fs.StringVar(&flags.Status, "status", "", "Filter by status code (200, 4xx, etc.)")
	fs.StringVar(&flags.Path, "path", "", "Filter by owning path pattern (supports glob)")
	fs.StringVar(&flags.Method, "method", "", "Filter by owning operation method")

	fs.StringVar(&flags.Format, "format", FormatText, "Output format: text, json, yaml")
	fs.BoolVar(&flags.Quiet, "quiet", false, "Suppress headers and decoration")
	fs.BoolVar(&flags.Quiet, "q", false, "Suppress headers and decoration (shorthand)")
	fs.BoolVar(&flags.Detail, "detail", false, "Show full node instead of summary table")
	fs.StringVar(&flags.Extension, "extension", "", "Filter by extension (e.g., x-internal=true)")
	fs.BoolVar(&flags.ResolveRefs, "resolve-refs", false, "Resolve $ref pointers before output")

	return fs, flags
}

// handleWalkResponses implements the "walk responses" subcommand.
func handleWalkResponses(args []string) error {
	fs, flags := SetupWalkResponsesFlags()

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
		return fmt.Errorf("walk responses: missing spec file argument")
	}

	specPath := fs.Arg(0)

	// 1. Collect
	result, err := parseSpec(specPath, flags.ResolveRefs)
	if err != nil {
		return fmt.Errorf("walk responses: %w", err)
	}

	collector, err := walker.CollectResponses(result)
	if err != nil {
		return fmt.Errorf("walk responses: %w", err)
	}

	// 2. Filter
	filtered := collector.All

	if flags.Status != "" {
		filtered = filterResponsesByStatus(filtered, flags.Status)
	}
	if flags.Path != "" {
		filtered = filterResponsesByPath(filtered, flags.Path)
	}
	if flags.Method != "" {
		filtered = filterResponsesByMethod(filtered, flags.Method)
	}
	if flags.Extension != "" {
		extFilter, err := ParseExtensionFilter(flags.Extension)
		if err != nil {
			return fmt.Errorf("walk responses: %w", err)
		}
		filtered = filterResponsesByExtension(filtered, extFilter)
	}

	if len(filtered) == 0 {
		renderNoResults("responses", flags.Quiet)
		return nil
	}

	// 3. Render
	if flags.Detail {
		for _, info := range filtered {
			view := responseDetailView{
				StatusCode: info.StatusCode,
				Path:       info.PathTemplate,
				Method:     strings.ToUpper(info.Method),
				Response:   info.Response,
			}
			if err := RenderDetail(os.Stdout, view, flags.Format); err != nil {
				return fmt.Errorf("walk responses: %w", err)
			}
		}
		return nil
	}

	headers := []string{"STATUS", "DESCRIPTION", "PATH", "METHOD", "EXTENSIONS"}
	rows := make([][]string, 0, len(filtered))
	for _, info := range filtered {
		rows = append(rows, []string{
			info.StatusCode,
			info.Response.Description,
			info.PathTemplate,
			strings.ToUpper(info.Method),
			FormatExtensions(info.Response.Extra),
		})
	}

	if flags.Format != FormatText {
		return RenderSummaryStructured(os.Stdout, headers, rows, flags.Format)
	}
	RenderSummaryTable(os.Stdout, headers, rows, flags.Quiet)
	return nil
}

// filterResponsesByStatus filters responses by status code pattern.
func filterResponsesByStatus(responses []*walker.ResponseInfo, pattern string) []*walker.ResponseInfo {
	var result []*walker.ResponseInfo
	for _, r := range responses {
		if matchStatusCode(r.StatusCode, pattern) {
			result = append(result, r)
		}
	}
	return result
}

// filterResponsesByPath filters responses by owning path template.
func filterResponsesByPath(responses []*walker.ResponseInfo, pattern string) []*walker.ResponseInfo {
	var result []*walker.ResponseInfo
	for _, r := range responses {
		if matchPath(r.PathTemplate, pattern) {
			result = append(result, r)
		}
	}
	return result
}

// filterResponsesByMethod filters responses by owning operation method.
func filterResponsesByMethod(responses []*walker.ResponseInfo, method string) []*walker.ResponseInfo {
	method = strings.ToLower(method)
	var result []*walker.ResponseInfo
	for _, r := range responses {
		if strings.ToLower(r.Method) == method {
			result = append(result, r)
		}
	}
	return result
}

// filterResponsesByExtension filters responses by extension filter.
func filterResponsesByExtension(responses []*walker.ResponseInfo, filter ExtensionFilter) []*walker.ResponseInfo {
	var result []*walker.ResponseInfo
	for _, r := range responses {
		if filter.Match(r.Response.Extra) {
			result = append(result, r)
		}
	}
	return result
}
