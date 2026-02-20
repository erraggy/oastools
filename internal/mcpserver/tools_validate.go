package mcpserver

import (
	"context"

	"github.com/erraggy/oastools/validator"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type validateInput struct {
	Spec       specInput `json:"spec"                    jsonschema:"The OAS document to validate"`
	Strict     *bool     `json:"strict,omitempty"        jsonschema:"Enable strict validation mode"`
	NoWarnings *bool     `json:"no_warnings,omitempty"   jsonschema:"Suppress warnings from output"`
	Offset     int       `json:"offset,omitempty"        jsonschema:"Skip the first N errors/warnings (for pagination)"`
	Limit      int       `json:"limit,omitempty"         jsonschema:"Maximum number of errors/warnings to return (default 100). Applied independently to errors and warnings arrays."`
}

type validateIssue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

type validateOutput struct {
	Valid        bool            `json:"valid"`
	Version      string          `json:"version"`
	ErrorCount   int             `json:"error_count"`
	WarningCount int             `json:"warning_count"`
	Returned     int             `json:"returned"`
	Errors       []validateIssue `json:"errors,omitempty"`
	Warnings     []validateIssue `json:"warnings,omitempty"`
}

func handleValidate(_ context.Context, _ *mcp.CallToolRequest, input validateInput) (*mcp.CallToolResult, validateOutput, error) {
	// Apply config defaults when input fields are omitted (nil).
	strict := cfg.ValidateStrict
	if input.Strict != nil {
		strict = *input.Strict
	}
	noWarnings := cfg.ValidateNoWarnings
	if input.NoWarnings != nil {
		noWarnings = *input.NoWarnings
	}

	parseResult, err := input.Spec.resolve()
	if err != nil {
		return errResult(err), validateOutput{}, nil
	}

	opts := []validator.Option{
		validator.WithParsed(*parseResult),
	}
	if strict {
		opts = append(opts, validator.WithStrictMode(true))
	}

	result, err := validator.ValidateWithOptions(opts...)
	if err != nil {
		return errResult(err), validateOutput{}, nil
	}

	output := validateOutput{
		Valid:      result.Valid,
		Version:    result.Version,
		ErrorCount: result.ErrorCount,
	}

	output.Errors = makeSlice[validateIssue](len(result.Errors))
	for _, e := range result.Errors {
		output.Errors = append(output.Errors, validateIssue{
			Path:    e.Path,
			Message: e.Message,
			Field:   e.Field,
		})
	}
	if !noWarnings {
		output.WarningCount = result.WarningCount
		output.Warnings = makeSlice[validateIssue](len(result.Warnings))
		for _, w := range result.Warnings {
			output.Warnings = append(output.Warnings, validateIssue{
				Path:    w.Path,
				Message: w.Message,
				Field:   w.Field,
			})
		}
	}

	// Paginate errors and warnings.
	output.Errors = paginate(output.Errors, input.Offset, input.Limit)
	if !noWarnings {
		output.Warnings = paginate(output.Warnings, input.Offset, input.Limit)
	}
	output.Returned = len(output.Errors) + len(output.Warnings)

	return nil, output, nil
}
