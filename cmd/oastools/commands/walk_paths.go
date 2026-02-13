package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/erraggy/oastools/internal/httputil"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
)

// pathInfo holds collected path item data for filtering and rendering.
type pathInfo struct {
	pathTemplate string
	pathItem     *parser.PathItem
}

// handleWalkPaths implements the "walk paths" subcommand.
// It collects path items from the spec, applies filters, and renders output.
func handleWalkPaths(args []string) error {
	fs := flag.NewFlagSet("walk paths", flag.ContinueOnError)

	// Paths-specific flags
	path := fs.String("path", "", "Filter by path pattern (supports glob with *)")

	// Common walk flags
	var flags WalkFlags
	fs.StringVar(&flags.Format, "format", FormatText, "Output format: text, json, yaml")
	fs.BoolVar(&flags.Quiet, "quiet", false, "Suppress headers and decoration")
	fs.BoolVar(&flags.Quiet, "q", false, "Suppress headers and decoration (shorthand)")
	fs.BoolVar(&flags.Detail, "detail", false, "Show full path item instead of summary table")
	fs.StringVar(&flags.Extension, "extension", "", "Filter by extension (e.g., x-internal=true)")
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
		return fmt.Errorf("walk paths requires a spec file argument")
	}
	specPath := fs.Arg(0)

	// 1. Collect: parse spec and walk with path handler
	result, err := parseSpec(specPath, flags.ResolveRefs)
	if err != nil {
		return fmt.Errorf("walk paths: %w", err)
	}

	paths, err := collectPaths(result)
	if err != nil {
		return fmt.Errorf("walk paths: collecting paths: %w", err)
	}

	// 2. Filter
	paths, err = filterPaths(paths, *path, flags.Extension)
	if err != nil {
		return err
	}

	if len(paths) == 0 {
		renderNoResults("paths", flags.Quiet)
		return nil
	}

	// 3. Render
	if flags.Detail {
		return renderPathsDetail(paths, flags)
	}
	return renderPathsSummary(paths, flags)
}

// collectPaths walks the spec and collects all path items.
func collectPaths(result *parser.ParseResult) ([]pathInfo, error) {
	if result == nil {
		return nil, fmt.Errorf("walk paths: nil parse result")
	}
	var paths []pathInfo
	err := walker.Walk(result,
		walker.WithPathHandler(func(wc *walker.WalkContext, pi *parser.PathItem) walker.Action {
			paths = append(paths, pathInfo{pathTemplate: wc.PathTemplate, pathItem: pi})
			return walker.Continue
		}),
	)
	if err != nil {
		return nil, err
	}
	return paths, nil
}

// filterPaths applies path pattern and extension filters and returns the matching subset.
func filterPaths(paths []pathInfo, pathPattern, extension string) ([]pathInfo, error) {
	// Parse extension filter once if provided
	var extFilter *ExtensionFilter
	if extension != "" {
		ef, err := ParseExtensionFilter(extension)
		if err != nil {
			return nil, fmt.Errorf("walk paths: %w", err)
		}
		extFilter = &ef
	}

	var matched []pathInfo
	for _, p := range paths {
		if !matchPath(p.pathTemplate, pathPattern) {
			continue
		}
		if extFilter != nil && !extFilter.Match(p.pathItem.Extra) {
			continue
		}
		matched = append(matched, p)
	}
	return matched, nil
}

// pathMethods returns a comma-separated list of HTTP methods with non-nil operations.
func pathMethods(pi *parser.PathItem) string {
	var methods []string
	if pi.Get != nil {
		methods = append(methods, strings.ToUpper(httputil.MethodGet))
	}
	if pi.Put != nil {
		methods = append(methods, strings.ToUpper(httputil.MethodPut))
	}
	if pi.Post != nil {
		methods = append(methods, strings.ToUpper(httputil.MethodPost))
	}
	if pi.Delete != nil {
		methods = append(methods, strings.ToUpper(httputil.MethodDelete))
	}
	if pi.Options != nil {
		methods = append(methods, strings.ToUpper(httputil.MethodOptions))
	}
	if pi.Head != nil {
		methods = append(methods, strings.ToUpper(httputil.MethodHead))
	}
	if pi.Patch != nil {
		methods = append(methods, strings.ToUpper(httputil.MethodPatch))
	}
	if pi.Trace != nil {
		methods = append(methods, strings.ToUpper(httputil.MethodTrace))
	}
	if pi.Query != nil {
		methods = append(methods, strings.ToUpper(httputil.MethodQuery))
	}
	if len(pi.AdditionalOperations) > 0 {
		extra := make([]string, 0, len(pi.AdditionalOperations))
		for m := range pi.AdditionalOperations {
			extra = append(extra, strings.ToUpper(m))
		}
		slices.Sort(extra)
		methods = append(methods, extra...)
	}
	return strings.Join(methods, ", ")
}

// renderPathsSummary renders a summary table of path items.
func renderPathsSummary(paths []pathInfo, flags WalkFlags) error {
	headers := []string{"PATH", "METHODS", "SUMMARY", "EXTENSIONS"}
	rows := make([][]string, 0, len(paths))

	for _, p := range paths {
		rows = append(rows, []string{
			p.pathTemplate,
			pathMethods(p.pathItem),
			p.pathItem.Summary,
			FormatExtensions(p.pathItem.Extra),
		})
	}

	RenderSummaryTable(os.Stdout, headers, rows, flags.Quiet)
	return nil
}

// renderPathsDetail renders each matched path item in full detail.
func renderPathsDetail(paths []pathInfo, flags WalkFlags) error {
	for _, p := range paths {
		if err := RenderDetail(os.Stdout, p.pathItem, flags.Format, flags.Quiet); err != nil {
			return fmt.Errorf("walk paths: rendering detail: %w", err)
		}
	}
	return nil
}
