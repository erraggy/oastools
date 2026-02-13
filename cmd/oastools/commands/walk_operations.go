package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/erraggy/oastools/walker"
)

// handleWalkOperations implements the "walk operations" subcommand.
// It collects operations from the spec, applies filters, and renders output.
func handleWalkOperations(args []string) error {
	fs := flag.NewFlagSet("walk operations", flag.ContinueOnError)

	// Operation-specific flags
	method := fs.String("method", "", "Filter by HTTP method (e.g., get, post)")
	path := fs.String("path", "", "Filter by path pattern (supports glob with *)")
	tag := fs.String("tag", "", "Filter by tag")
	deprecated := fs.Bool("deprecated", false, "Only show deprecated operations")
	operationID := fs.String("operationId", "", "Select by operationId")

	// Common walk flags
	var flags WalkFlags
	fs.StringVar(&flags.Format, "format", FormatText, "Output format: text, json, yaml")
	fs.BoolVar(&flags.Quiet, "quiet", false, "Suppress headers and decoration")
	fs.BoolVar(&flags.Quiet, "q", false, "Suppress headers and decoration (shorthand)")
	fs.BoolVar(&flags.Detail, "detail", false, "Show full operation instead of summary table")
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
		return fmt.Errorf("walk operations requires a spec file argument")
	}
	specPath := fs.Arg(0)

	// 1. Collect: parse spec and collect operations
	result, err := parseSpec(specPath, flags.ResolveRefs)
	if err != nil {
		return fmt.Errorf("walk operations: %w", err)
	}

	collector, err := walker.CollectOperations(result)
	if err != nil {
		return fmt.Errorf("walk operations: collecting operations: %w", err)
	}

	// 2. Filter
	matched := collector.All

	matched, err = filterOperations(matched, *method, *path, *tag, *deprecated, *operationID, flags.Extension)
	if err != nil {
		return err
	}

	if len(matched) == 0 {
		renderNoResults("operations", flags.Quiet)
		return nil
	}

	// 3. Render
	if flags.Detail {
		return renderOperationsDetail(matched, flags)
	}
	return renderOperationsSummary(matched, flags)
}

// filterOperations applies all operation filters and returns the matching subset.
func filterOperations(
	ops []*walker.OperationInfo,
	method, path, tag string,
	deprecated bool,
	operationID, extension string,
) ([]*walker.OperationInfo, error) {
	// Parse extension filter once if provided
	var extFilter *ExtensionFilter
	if extension != "" {
		ef, err := ParseExtensionFilter(extension)
		if err != nil {
			return nil, fmt.Errorf("walk operations: %w", err)
		}
		extFilter = &ef
	}

	var matched []*walker.OperationInfo
	for _, op := range ops {
		if !matchOperationMethod(op.Method, method) {
			continue
		}
		if !matchPath(op.PathTemplate, path) {
			continue
		}
		if !matchOperationTag(op.Operation.Tags, tag) {
			continue
		}
		if deprecated && !op.Operation.Deprecated {
			continue
		}
		if operationID != "" && op.Operation.OperationID != operationID {
			continue
		}
		if extFilter != nil && !extFilter.Match(op.Operation.Extra) {
			continue
		}
		matched = append(matched, op)
	}
	return matched, nil
}

// matchOperationMethod checks if an operation's method matches the filter.
func matchOperationMethod(opMethod, filter string) bool {
	if filter == "" {
		return true
	}
	return strings.EqualFold(opMethod, filter)
}

// matchOperationTag checks if an operation has the specified tag.
func matchOperationTag(tags []string, filter string) bool {
	if filter == "" {
		return true
	}
	return slices.Contains(tags, filter)
}

// renderOperationsSummary renders a summary table of operations.
func renderOperationsSummary(ops []*walker.OperationInfo, flags WalkFlags) error {
	headers := []string{"METHOD", "PATH", "SUMMARY", "TAGS", "EXTENSIONS"}
	rows := make([][]string, 0, len(ops))

	for _, op := range ops {
		rows = append(rows, []string{
			strings.ToUpper(op.Method),
			op.PathTemplate,
			op.Operation.Summary,
			strings.Join(op.Operation.Tags, ", "),
			FormatExtensions(op.Operation.Extra),
		})
	}

	RenderSummaryTable(os.Stdout, headers, rows, flags.Quiet)
	return nil
}

// renderOperationsDetail renders each matched operation in full detail.
func renderOperationsDetail(ops []*walker.OperationInfo, flags WalkFlags) error {
	for _, op := range ops {
		if err := RenderDetail(os.Stdout, op.Operation, flags.Format, flags.Quiet); err != nil {
			return fmt.Errorf("walk operations: rendering detail: %w", err)
		}
	}
	return nil
}
