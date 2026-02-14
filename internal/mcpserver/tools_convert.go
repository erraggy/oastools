package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/erraggy/oastools/converter"
	"github.com/erraggy/oastools/parser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type convertInput struct {
	Spec   specInput `json:"spec"               jsonschema:"The OAS document to convert"`
	Target string    `json:"target"             jsonschema:"Target OAS version (2.0\\, 3.0\\, or 3.1)"`
	Output string    `json:"output,omitempty"   jsonschema:"File path to write converted document. If omitted the document is returned inline."`
}

type convertIssue struct {
	Severity string `json:"severity"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

type convertOutput struct {
	SourceVersion string         `json:"source_version"`
	TargetVersion string         `json:"target_version"`
	Success       bool           `json:"success"`
	IssueCount    int            `json:"issue_count"`
	Issues        []convertIssue `json:"issues,omitempty"`
	WrittenTo     string         `json:"written_to,omitempty"`
	Document      string         `json:"document,omitempty"`
}

func handleConvert(_ context.Context, _ *mcp.CallToolRequest, input convertInput) (*mcp.CallToolResult, convertOutput, error) {
	if input.Target == "" {
		return errResult(fmt.Errorf("target version is required")), convertOutput{}, nil
	}

	opts, err := buildConverterOptions(input)
	if err != nil {
		return errResult(err), convertOutput{}, nil
	}

	result, err := converter.ConvertWithOptions(opts...)
	if err != nil {
		return errResult(err), convertOutput{}, nil
	}

	output := convertOutput{
		SourceVersion: result.SourceVersion,
		TargetVersion: result.TargetVersion,
		Success:       result.Success,
		IssueCount:    len(result.Issues),
	}

	output.Issues = makeSlice[convertIssue](len(result.Issues))
	for _, issue := range result.Issues {
		output.Issues = append(output.Issues, convertIssue{
			Severity: issue.Severity.String(),
			Path:     issue.Path,
			Message:  issue.Message,
		})
	}

	// Marshal the converted document.
	pr := result.ToParseResult()
	var data []byte
	switch result.SourceFormat {
	case parser.SourceFormatJSON:
		data, err = pr.MarshalOrderedJSONIndent("", "  ")
	default:
		data, err = pr.MarshalOrderedYAML()
	}
	if err != nil {
		return errResult(err), convertOutput{}, nil
	}

	if input.Output != "" {
		if err := os.WriteFile(input.Output, data, 0o644); err != nil {
			return errResult(fmt.Errorf("failed to write output file: %w", err)), convertOutput{}, nil
		}
		output.WrittenTo = input.Output
	} else {
		output.Document = string(data)
	}

	return nil, output, nil
}

// buildConverterOptions translates the MCP input into converter options,
// handling the three input modes (file, url, content) and the target version.
func buildConverterOptions(input convertInput) ([]converter.Option, error) {
	var opts []converter.Option

	switch {
	case input.Spec.File != "":
		opts = append(opts, converter.WithFilePath(input.Spec.File))
	case input.Spec.URL != "":
		opts = append(opts, converter.WithFilePath(input.Spec.URL))
	case input.Spec.Content != "":
		parseResult, err := parser.ParseWithOptions(
			parser.WithReader(strings.NewReader(input.Spec.Content)),
		)
		if err != nil {
			return nil, err
		}
		opts = append(opts, converter.WithParsed(*parseResult))
	default:
		return nil, fmt.Errorf("exactly one of file, url, or content must be provided")
	}

	opts = append(opts, converter.WithTargetVersion(input.Target))

	return opts, nil
}
