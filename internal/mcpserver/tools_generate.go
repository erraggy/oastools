package mcpserver

import (
	"context"
	"fmt"

	"github.com/erraggy/oastools/generator"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type generateInput struct {
	Spec        specInput `json:"spec"                    jsonschema:"The OAS document to generate code from"`
	Client      bool      `json:"client,omitempty"        jsonschema:"Generate client code"`
	Server      bool      `json:"server,omitempty"        jsonschema:"Generate server code"`
	Types       bool      `json:"types,omitempty"         jsonschema:"Generate type definitions only"`
	PackageName string    `json:"package_name,omitempty"  jsonschema:"Go package name for generated code (default: api)"`
	OutputDir   string    `json:"output_dir"              jsonschema:"Directory to write generated files to"`
}

type generatedFileInfo struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

type generateOutput struct {
	Success             bool                `json:"success"`
	OutputDir           string              `json:"output_dir"`
	PackageName         string              `json:"package_name"`
	FileCount           int                 `json:"file_count"`
	Files               []generatedFileInfo `json:"files"`
	GeneratedTypes      int                 `json:"generated_types"`
	GeneratedOperations int                 `json:"generated_operations"`
	WarningCount        int                 `json:"warning_count"`
	CriticalCount       int                 `json:"critical_count"`
}

func handleGenerate(_ context.Context, _ *mcp.CallToolRequest, input generateInput) (*mcp.CallToolResult, generateOutput, error) {
	if input.OutputDir == "" {
		return errResult(fmt.Errorf("output_dir is required")), generateOutput{}, nil
	}

	parseResult, err := input.Spec.resolve()
	if err != nil {
		return errResult(err), generateOutput{}, nil
	}

	opts := []generator.Option{
		generator.WithParsed(*parseResult),
		generator.WithReadme(false),
	}

	if input.PackageName != "" {
		opts = append(opts, generator.WithPackageName(input.PackageName))
	}
	if input.Client {
		opts = append(opts, generator.WithClient(true))
	}
	if input.Server {
		opts = append(opts, generator.WithServer(true))
	}
	if input.Types {
		opts = append(opts, generator.WithTypes(true))
	}

	result, err := generator.GenerateWithOptions(opts...)
	if err != nil {
		return errResult(err), generateOutput{}, nil
	}

	if err := result.WriteFiles(input.OutputDir); err != nil {
		return errResult(fmt.Errorf("failed to write generated files: %w", err)), generateOutput{}, nil
	}

	output := generateOutput{
		Success:             result.Success,
		OutputDir:           input.OutputDir,
		PackageName:         result.PackageName,
		FileCount:           len(result.Files),
		GeneratedTypes:      result.GeneratedTypes,
		GeneratedOperations: result.GeneratedOperations,
		WarningCount:        result.WarningCount,
		CriticalCount:       result.CriticalCount,
	}

	output.Files = makeSlice[generatedFileInfo](len(result.Files))
	for _, f := range result.Files {
		output.Files = append(output.Files, generatedFileInfo{
			Name: f.Name,
			Size: len(f.Content),
		})
	}

	return nil, output, nil
}
