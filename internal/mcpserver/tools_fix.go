package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/erraggy/oastools/fixer"
	"github.com/erraggy/oastools/parser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fixInput struct {
	Spec                     specInput `json:"spec"                                  jsonschema:"The OAS document to fix"`
	FixSchemaNames           bool      `json:"fix_schema_names,omitempty"            jsonschema:"Rename generic schema names (Object1\\, Model2) to meaningful names"`
	FixDuplicateOperationIds bool      `json:"fix_duplicate_operationids,omitempty"  jsonschema:"Fix duplicate operationId values"`
	Prune                    bool      `json:"prune,omitempty"                       jsonschema:"Remove empty paths and unused schemas"`
	StubMissingRefs          bool      `json:"stub_missing_refs,omitempty"           jsonschema:"Create stub schemas for missing $ref targets"`
	DryRun                   bool      `json:"dry_run,omitempty"                     jsonschema:"Preview fixes without applying them"`
	IncludeDocument          bool      `json:"include_document,omitempty"            jsonschema:"Include the full corrected document in output"`
	Output                   string    `json:"output,omitempty"                     jsonschema:"File path to write the fixed document. If omitted the document is returned inline when include_document is true."`
	Offset                   int       `json:"offset,omitempty"                     jsonschema:"Skip the first N fixes (for pagination)"`
	Limit                    int       `json:"limit,omitempty"                      jsonschema:"Maximum number of fixes to return (default 100)"`
}

type fixApplied struct {
	Type        string `json:"type"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

type fixOutput struct {
	FixCount  int          `json:"fix_count"`
	Returned  int          `json:"returned"`
	Fixes     []fixApplied `json:"fixes,omitempty"`
	Version   string       `json:"version"`
	WrittenTo string       `json:"written_to,omitempty"`
	Document  string       `json:"document,omitempty"`
}

func handleFix(_ context.Context, _ *mcp.CallToolRequest, input fixInput) (*mcp.CallToolResult, fixOutput, error) {
	opts, err := buildFixerOptions(input)
	if err != nil {
		return errResult(err), fixOutput{}, nil
	}

	result, err := fixer.FixWithOptions(opts...)
	if err != nil {
		return errResult(err), fixOutput{}, nil
	}

	output := fixOutput{
		FixCount: result.FixCount,
		Version:  result.SourceVersion,
	}

	output.Fixes = makeSlice[fixApplied](len(result.Fixes))
	for _, f := range result.Fixes {
		output.Fixes = append(output.Fixes, fixApplied{
			Type:        string(f.Type),
			Path:        f.Path,
			Description: f.Description,
		})
	}

	output.Fixes = paginate(output.Fixes, input.Offset, input.Limit)
	output.Returned = len(output.Fixes)

	needsDocument := !input.DryRun && (input.Output != "" || input.IncludeDocument)
	if needsDocument {
		pr := result.ToParseResult()
		var data []byte
		switch result.SourceFormat {
		case parser.SourceFormatJSON:
			data, err = pr.MarshalOrderedJSONIndent("", "  ")
		default:
			data, err = pr.MarshalOrderedYAML()
		}
		if err != nil {
			return errResult(err), fixOutput{}, nil
		}

		if input.Output != "" {
			if err := os.WriteFile(input.Output, data, 0o644); err != nil {
				return errResult(fmt.Errorf("failed to write output file: %w", err)), fixOutput{}, nil
			}
			output.WrittenTo = input.Output
		}
		if input.IncludeDocument {
			output.Document = string(data)
		}
	}

	return nil, output, nil
}

// buildFixerOptions translates the MCP input into fixer options, handling
// the three input modes (file, url, content) and all optional fix flags.
func buildFixerOptions(input fixInput) ([]fixer.Option, error) {
	var opts []fixer.Option

	// Determine input source. For file/url, use fixer.WithFilePath directly
	// (the fixer handles its own parsing). For inline content, parse first
	// and pass the result via fixer.WithParsed.
	switch {
	case input.Spec.File != "":
		opts = append(opts, fixer.WithFilePath(input.Spec.File))
	case input.Spec.URL != "":
		opts = append(opts, fixer.WithFilePath(input.Spec.URL))
	case input.Spec.Content != "":
		parseResult, err := parser.ParseWithOptions(
			parser.WithReader(strings.NewReader(input.Spec.Content)),
		)
		if err != nil {
			return nil, err
		}
		opts = append(opts, fixer.WithParsed(*parseResult), fixer.WithMutableInput(true))
	default:
		return nil, fmt.Errorf("exactly one of file, url, or content must be provided")
	}

	// Build the list of enabled fix types based on input flags.
	var fixes []fixer.FixType
	// Missing path parameters are always enabled (default fixer behavior).
	fixes = append(fixes, fixer.FixTypeMissingPathParameter)
	if input.FixSchemaNames {
		fixes = append(fixes, fixer.FixTypeRenamedGenericSchema)
	}
	if input.FixDuplicateOperationIds {
		fixes = append(fixes, fixer.FixTypeDuplicateOperationId)
	}
	if input.Prune {
		fixes = append(fixes, fixer.FixTypePrunedEmptyPath, fixer.FixTypePrunedUnusedSchema)
	}
	if input.StubMissingRefs {
		fixes = append(fixes, fixer.FixTypeStubMissingRef)
	}
	opts = append(opts, fixer.WithEnabledFixes(fixes...))

	if input.DryRun {
		opts = append(opts, fixer.WithDryRun(true))
	}

	return opts, nil
}
