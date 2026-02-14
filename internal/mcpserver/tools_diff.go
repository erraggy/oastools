package mcpserver

import (
	"context"
	"strconv"

	"github.com/erraggy/oastools/differ"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type diffInput struct {
	Base         specInput `json:"base"                    jsonschema:"The base/original OAS document"`
	Revision     specInput `json:"revision"                jsonschema:"The revised OAS document to compare against the base"`
	BreakingOnly bool      `json:"breaking_only,omitempty" jsonschema:"Only show breaking changes"`
	NoInfo       bool      `json:"no_info,omitempty"       jsonschema:"Suppress informational changes"`
}

type diffChange struct {
	Severity string `json:"severity"`
	Type     string `json:"type"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

type diffOutput struct {
	TotalChanges  int          `json:"total_changes"`
	BreakingCount int          `json:"breaking_count"`
	WarningCount  int          `json:"warning_count"`
	InfoCount     int          `json:"info_count"`
	Changes       []diffChange `json:"changes,omitempty"`
	Summary       string       `json:"summary"`
}

func handleDiff(_ context.Context, _ *mcp.CallToolRequest, input diffInput) (*mcp.CallToolResult, diffOutput, error) {
	baseResult, err := input.Base.resolve()
	if err != nil {
		return errResult(err), diffOutput{}, nil
	}

	revisionResult, err := input.Revision.resolve()
	if err != nil {
		return errResult(err), diffOutput{}, nil
	}

	opts := []differ.Option{
		differ.WithSourceParsed(*baseResult),
		differ.WithTargetParsed(*revisionResult),
	}
	if input.BreakingOnly {
		opts = append(opts, differ.WithMode(differ.ModeBreaking))
	}
	if input.NoInfo {
		opts = append(opts, differ.WithIncludeInfo(false))
	}

	result, err := differ.DiffWithOptions(opts...)
	if err != nil {
		return errResult(err), diffOutput{}, nil
	}

	output := diffOutput{
		Changes: makeSlice[diffChange](len(result.Changes)),
	}

	for _, c := range result.Changes {
		// When breaking_only is set, skip non-breaking changes.
		if input.BreakingOnly && c.Severity != differ.SeverityCritical && c.Severity != differ.SeverityError {
			continue
		}

		output.Changes = append(output.Changes, diffChange{
			Severity: c.Severity.String(),
			Type:     string(c.Type),
			Path:     c.Path,
			Message:  c.Message,
		})

		// Count by severity from the displayed changes.
		switch c.Severity {
		case differ.SeverityCritical, differ.SeverityError:
			output.BreakingCount++
		case differ.SeverityWarning:
			output.WarningCount++
		default:
			output.InfoCount++
		}
	}

	output.TotalChanges = len(output.Changes)
	output.Summary = buildDiffSummary(output)

	return nil, output, nil
}

func buildDiffSummary(output diffOutput) string {
	if output.TotalChanges == 0 {
		return "No changes detected."
	}

	summary := ""
	if output.BreakingCount > 0 {
		summary = "Breaking changes detected. "
	}

	summary += formatCount(output.TotalChanges, "change") + " found"
	if output.BreakingCount > 0 {
		summary += " (" + formatCount(output.BreakingCount, "breaking change") + ")."
	} else {
		summary += "."
	}

	return summary
}

func formatCount(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}
