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

// parameterDetailView wraps a parameter with its walker context for serialization.
type parameterDetailView struct {
	Path      string            `json:"path" yaml:"path"`
	Method    string            `json:"method" yaml:"method"`
	Parameter *parser.Parameter `json:"parameter" yaml:"parameter"`
}

// WalkParametersFlags contains flags for the walk parameters subcommand.
type WalkParametersFlags struct {
	In     string
	Name   string
	Path   string
	Method string
	WalkFlags
}

// SetupWalkParametersFlags creates and configures a FlagSet for the walk parameters subcommand.
func SetupWalkParametersFlags() (*flag.FlagSet, *WalkParametersFlags) {
	fs := flag.NewFlagSet("walk parameters", flag.ContinueOnError)
	flags := &WalkParametersFlags{}

	fs.StringVar(&flags.In, "in", "", "Filter by location (path, query, header, cookie)")
	fs.StringVar(&flags.Name, "name", "", "Filter by parameter name")
	fs.StringVar(&flags.Path, "path", "", "Filter by owning path pattern (supports glob with *)")
	fs.StringVar(&flags.Method, "method", "", "Filter by owning operation method")

	fs.StringVar(&flags.Format, "format", FormatText, "Output format: text, json, yaml")
	fs.BoolVar(&flags.Quiet, "quiet", false, "Suppress headers and decoration")
	fs.BoolVar(&flags.Quiet, "q", false, "Suppress headers and decoration (shorthand)")
	fs.BoolVar(&flags.Detail, "detail", false, "Show full parameter instead of summary table")
	fs.StringVar(&flags.Extension, "extension", "", "Filter by extension (e.g., x-internal=true)")
	fs.BoolVar(&flags.ResolveRefs, "resolve-refs", false, "Resolve $ref pointers before output")

	return fs, flags
}

// handleWalkParameters implements the "walk parameters" subcommand.
func handleWalkParameters(args []string) error {
	fs, flags := SetupWalkParametersFlags()

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
		return fmt.Errorf("walk parameters: missing spec file argument")
	}
	specPath := fs.Arg(0)

	// 1. Collect
	result, err := parseSpec(specPath, flags.ResolveRefs)
	if err != nil {
		return fmt.Errorf("walk parameters: %w", err)
	}

	collector, err := walker.CollectParameters(result)
	if err != nil {
		return fmt.Errorf("walk parameters: %w", err)
	}

	// 2. Filter
	var extFilter ExtensionFilter
	var hasExtFilter bool
	if flags.Extension != "" {
		extFilter, err = ParseExtensionFilter(flags.Extension)
		if err != nil {
			return fmt.Errorf("walk parameters: %w", err)
		}
		hasExtFilter = true
	}

	var filtered []*walker.ParameterInfo
	for _, info := range collector.All {
		if flags.In != "" && !strings.EqualFold(info.In, flags.In) {
			continue
		}
		if flags.Name != "" && !strings.EqualFold(info.Name, flags.Name) {
			continue
		}
		if !matchPath(info.PathTemplate, flags.Path) {
			continue
		}
		if flags.Method != "" && !strings.EqualFold(info.Method, flags.Method) {
			continue
		}
		if hasExtFilter && !extFilter.Match(info.Parameter.Extra) {
			continue
		}
		filtered = append(filtered, info)
	}

	// 3. Render
	if len(filtered) == 0 {
		renderNoResults("parameters", flags.Quiet)
		return nil
	}

	if flags.Detail {
		for _, info := range filtered {
			view := parameterDetailView{
				Path:      info.PathTemplate,
				Method:    strings.ToUpper(info.Method),
				Parameter: info.Parameter,
			}
			if err := RenderDetail(os.Stdout, view, flags.Format); err != nil {
				return fmt.Errorf("walk parameters: %w", err)
			}
		}
		return nil
	}

	// Summary table
	headers := []string{"NAME", "IN", "REQUIRED", "PATH", "METHOD", "EXTENSIONS"}
	var rows [][]string
	for _, info := range filtered {
		rows = append(rows, []string{
			info.Name,
			info.In,
			fmt.Sprintf("%v", info.Parameter.Required),
			info.PathTemplate,
			strings.ToUpper(info.Method),
			FormatExtensions(info.Parameter.Extra),
		})
	}

	if flags.Format != FormatText {
		return RenderSummaryStructured(os.Stdout, headers, rows, flags.Format)
	}
	RenderSummaryTable(os.Stdout, headers, rows, flags.Quiet)
	return nil
}
