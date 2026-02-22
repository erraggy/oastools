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

// schemaDetailView wraps a schema with its walker context for serialization.
type schemaDetailView struct {
	Name        string         `json:"name" yaml:"name"`
	JSONPath    string         `json:"jsonPath" yaml:"jsonPath"`
	IsComponent bool           `json:"isComponent" yaml:"isComponent"`
	Schema      *parser.Schema `json:"schema" yaml:"schema"`
}

// WalkSchemasFlags contains flags for the walk schemas subcommand.
type WalkSchemasFlags struct {
	Name      string
	Component bool
	Inline    bool
	Type      string
	WalkFlags
}

// SetupWalkSchemasFlags creates and configures a FlagSet for the walk schemas subcommand.
func SetupWalkSchemasFlags() (*flag.FlagSet, *WalkSchemasFlags) {
	fs := flag.NewFlagSet("walk schemas", flag.ContinueOnError)
	flags := &WalkSchemasFlags{}

	fs.StringVar(&flags.Name, "name", "", "Select by schema name")
	fs.BoolVar(&flags.Component, "component", false, "Only show component schemas")
	fs.BoolVar(&flags.Inline, "inline", false, "Only show inline schemas")
	fs.StringVar(&flags.Type, "type", "", "Filter by schema type (object, array, string, etc.)")

	fs.StringVar(&flags.Format, "format", FormatText, "Output format: text, json, yaml")
	fs.BoolVar(&flags.Quiet, "quiet", false, "Suppress headers and decoration")
	fs.BoolVar(&flags.Quiet, "q", false, "Suppress headers and decoration (shorthand)")
	fs.BoolVar(&flags.Detail, "detail", false, "Show full node instead of summary table")
	fs.StringVar(&flags.Extension, "extension", "", "Filter by extension (e.g., x-internal=true)")
	fs.BoolVar(&flags.ResolveRefs, "resolve-refs", false, "Resolve $ref pointers")

	return fs, flags
}

// handleWalkSchemas implements the "walk schemas" subcommand.
func handleWalkSchemas(args []string) error {
	fs, flags := SetupWalkSchemasFlags()

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if err := ValidateOutputFormat(flags.Format); err != nil {
		return err
	}

	if flags.Component && flags.Inline {
		return fmt.Errorf("walk schemas: cannot use both --component and --inline")
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("walk schemas requires a spec file argument")
	}
	specPath := fs.Arg(0)

	// 1. Collect: parse spec and collect schemas
	result, err := parseSpec(specPath, flags.ResolveRefs)
	if err != nil {
		return fmt.Errorf("walk schemas: %w", err)
	}

	collector, err := walker.CollectSchemas(result)
	if err != nil {
		return fmt.Errorf("walk schemas: collecting schemas: %w", err)
	}

	// Choose base set based on component/inline filter
	schemas := collector.All
	if flags.Component {
		schemas = collector.Components
	} else if flags.Inline {
		schemas = collector.Inline
	}

	// 2. Filter: apply name, type, and extension filters
	var extFilter *ExtensionFilter
	if flags.Extension != "" {
		ef, err := ParseExtensionFilter(flags.Extension)
		if err != nil {
			return fmt.Errorf("walk schemas: parsing extension filter: %w", err)
		}
		extFilter = &ef
	}

	var filtered []*walker.SchemaInfo
	for _, info := range schemas {
		if flags.Name != "" && !strings.EqualFold(info.Name, flags.Name) {
			continue
		}
		if flags.Type != "" && !schemaTypeMatches(info.Schema.Type, flags.Type) {
			continue
		}
		if extFilter != nil && !extFilter.Match(info.Schema.Extra) {
			continue
		}
		filtered = append(filtered, info)
	}

	if len(filtered) == 0 {
		renderNoResults("schemas", flags.Quiet)
		return nil
	}

	// 3. Render: summary table or detail output
	if flags.Detail {
		for _, info := range filtered {
			view := schemaDetailView{
				Name:        info.Name,
				JSONPath:    info.JSONPath,
				IsComponent: info.IsComponent,
				Schema:      info.Schema,
			}
			if err := RenderDetail(os.Stdout, view, flags.Format); err != nil {
				return fmt.Errorf("walk schemas: rendering detail: %w", err)
			}
		}
		return nil
	}

	headers := []string{"NAME", "TYPE", "PROPERTIES", "LOCATION", "EXTENSIONS"}
	rows := make([][]string, 0, len(filtered))
	for _, info := range filtered {
		displayName := info.Name
		if displayName == "" {
			displayName = info.JSONPath
		}

		schemaType := ""
		if info.Schema.Type != nil {
			schemaType = fmt.Sprintf("%v", info.Schema.Type)
		}

		props := fmt.Sprintf("%d props", len(info.Schema.Properties))

		location := "inline"
		if info.IsComponent {
			location = "component"
		}

		extensions := FormatExtensions(info.Schema.Extra)

		rows = append(rows, []string{displayName, schemaType, props, location, extensions})
	}

	if flags.Format != FormatText {
		return RenderSummaryStructured(os.Stdout, headers, rows, flags.Format)
	}
	RenderSummaryTable(os.Stdout, headers, rows, flags.Quiet)
	return nil
}

// schemaTypeMatches checks if a schema's Type field matches the given filter string.
// The Type field is `any` because it can be a string (OAS 3.0) or []string (OAS 3.1+).
func schemaTypeMatches(schemaType any, filter string) bool {
	switch t := schemaType.(type) {
	case string:
		return strings.EqualFold(t, filter)
	case []string:
		for _, s := range t {
			if strings.EqualFold(s, filter) {
				return true
			}
		}
	case []any:
		for _, s := range t {
			if str, ok := s.(string); ok && strings.EqualFold(str, filter) {
				return true
			}
		}
	}
	return false
}
