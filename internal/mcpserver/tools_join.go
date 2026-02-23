package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/erraggy/oastools/internal/pathutil"
	"github.com/erraggy/oastools/joiner"
	"github.com/erraggy/oastools/parser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type joinInput struct {
	Specs          []specInput `json:"specs"                         jsonschema:"Array of OAS documents to join (minimum 2)"`
	PathStrategy   string      `json:"path_strategy,omitempty"       jsonschema:"Strategy for path collisions: accept-left or accept-right or fail or fail-on-paths"`
	SchemaStrategy string      `json:"schema_strategy,omitempty"     jsonschema:"Strategy for schema collisions: accept-left or accept-right or fail or rename-left or rename-right or deduplicate"`
	SemanticDedup  bool        `json:"semantic_dedup,omitempty"      jsonschema:"Enable semantic deduplication of equivalent schemas"`
	Output         string      `json:"output,omitempty"              jsonschema:"File path to write joined document. If omitted the result is returned inline."`
}

type joinWarning struct {
	Message string `json:"message"`
}

type joinOutput struct {
	SpecCount      int           `json:"spec_count"`
	Version        string        `json:"version"`
	PathCount      int           `json:"path_count"`
	SchemaCount    int           `json:"schema_count"`
	CollisionCount int           `json:"collision_count"`
	WarningCount   int           `json:"warning_count"`
	Warnings       []joinWarning `json:"warnings,omitempty"`
	WrittenTo      string        `json:"written_to,omitempty"`
	Document       string        `json:"document,omitempty"`
	Summary        string        `json:"summary"`
}

func handleJoin(_ context.Context, _ *mcp.CallToolRequest, input joinInput) (*mcp.CallToolResult, joinOutput, error) {
	// Apply config defaults.
	if input.PathStrategy == "" {
		input.PathStrategy = cfg.JoinPathStrategy
	}
	if input.SchemaStrategy == "" {
		input.SchemaStrategy = cfg.JoinSchemaStrategy
	}

	if len(input.Specs) < 2 {
		return errResult(fmt.Errorf("at least 2 specs are required for joining, got %d", len(input.Specs))), joinOutput{}, nil
	}
	if len(input.Specs) > cfg.MaxJoinSpecs {
		return errResult(fmt.Errorf("too many specs: got %d, maximum is %d; set OASTOOLS_MAX_JOIN_SPECS to increase",
			len(input.Specs), cfg.MaxJoinSpecs)), joinOutput{}, nil
	}
	if input.PathStrategy != "" && !validJoinStrategies[input.PathStrategy] {
		return errResult(fmt.Errorf("invalid path_strategy: %q; valid values: %s", input.PathStrategy, validJoinStrategyList)), joinOutput{}, nil
	}
	if input.SchemaStrategy != "" && !validJoinStrategies[input.SchemaStrategy] {
		return errResult(fmt.Errorf("invalid schema_strategy: %q; valid values: %s", input.SchemaStrategy, validJoinStrategyList)), joinOutput{}, nil
	}

	// Resolve all specs.
	parsed := make([]parser.ParseResult, 0, len(input.Specs))
	for i, spec := range input.Specs {
		result, err := spec.resolve()
		if err != nil {
			return errResult(fmt.Errorf("spec[%d]: %w", i, err)), joinOutput{}, nil
		}
		parsed = append(parsed, *result)
	}

	// Build joiner options.
	opts := []joiner.Option{
		joiner.WithParsed(parsed...),
	}
	if input.PathStrategy != "" {
		opts = append(opts, joiner.WithPathStrategy(joiner.CollisionStrategy(input.PathStrategy)))
	}
	if input.SchemaStrategy != "" {
		opts = append(opts, joiner.WithSchemaStrategy(joiner.CollisionStrategy(input.SchemaStrategy)))
	}
	if input.SemanticDedup {
		opts = append(opts, joiner.WithSemanticDeduplication(true))
	}

	result, err := joiner.JoinWithOptions(opts...)
	if err != nil {
		return errResult(err), joinOutput{}, nil
	}

	output := joinOutput{
		SpecCount:      len(input.Specs),
		Version:        result.Version,
		PathCount:      result.Stats.PathCount,
		SchemaCount:    result.Stats.SchemaCount,
		CollisionCount: result.CollisionCount,
		WarningCount:   len(result.Warnings),
	}

	output.Warnings = makeSlice[joinWarning](len(result.Warnings))
	for _, w := range result.Warnings {
		output.Warnings = append(output.Warnings, joinWarning{Message: w})
	}

	output.Summary = buildJoinSummary(output)

	// Marshal the joined document.
	pr := result.ToParseResult()
	var data []byte
	switch result.SourceFormat {
	case parser.SourceFormatJSON:
		data, err = pr.MarshalOrderedJSONIndent("", "  ")
	default:
		data, err = pr.MarshalOrderedYAML()
	}
	if err != nil {
		return errResult(err), joinOutput{}, nil
	}

	if input.Output != "" {
		cleanPath, pathErr := pathutil.SanitizeOutputPath(input.Output)
		if pathErr != nil {
			return errResult(fmt.Errorf("invalid output path: %w", pathErr)), joinOutput{}, nil
		}
		if err := os.WriteFile(cleanPath, data, 0o600); err != nil {
			return errResult(fmt.Errorf("failed to write output file: %w", err)), joinOutput{}, nil
		}
		output.WrittenTo = cleanPath
	} else {
		output.Document = string(data)
	}

	return nil, output, nil
}

func buildJoinSummary(output joinOutput) string {
	summary := "Joined " + strconv.Itoa(output.SpecCount) + " specs into " + output.Version + " document"
	summary += " with " + formatCount(output.PathCount, "path")
	summary += " and " + formatCount(output.SchemaCount, "schema") + "."

	if output.CollisionCount > 0 {
		summary += " " + formatCount(output.CollisionCount, "collision") + " resolved."
	}
	if output.WarningCount > 0 {
		summary += " " + formatCount(output.WarningCount, "warning") + "."
	}

	return summary
}
