package mcpserver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/erraggy/oastools/overlay"
	"github.com/erraggy/oastools/parser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type overlayApplyInput struct {
	Spec    specInput `json:"spec"               jsonschema:"The OAS document to apply the overlay to"`
	Overlay specInput `json:"overlay"            jsonschema:"The Overlay document to apply"`
	DryRun  bool      `json:"dry_run,omitempty"  jsonschema:"Preview changes without applying"`
	Output  string    `json:"output,omitempty"   jsonschema:"File path to write result. If omitted the result is returned inline."`
}

type overlayApplyChange struct {
	ActionIndex int    `json:"action_index"`
	Target      string `json:"target"`
	Operation   string `json:"operation"`
	MatchCount  int    `json:"match_count"`
}

type overlayApplyOutput struct {
	ActionsApplied int                  `json:"actions_applied"`
	ActionsSkipped int                  `json:"actions_skipped"`
	Changes        []overlayApplyChange `json:"changes,omitempty"`
	Warnings       []string             `json:"warnings,omitempty"`
	WrittenTo      string               `json:"written_to,omitempty"`
	Document       string               `json:"document,omitempty"`
	Summary        string               `json:"summary"`
}

func handleOverlayApply(ctx context.Context, _ *mcp.CallToolRequest, input overlayApplyInput) (*mcp.CallToolResult, overlayApplyOutput, error) {
	specResult, err := input.Spec.resolve()
	if err != nil {
		return errResult(err), overlayApplyOutput{}, nil
	}

	o, err := resolveOverlayInput(ctx, input.Overlay)
	if err != nil {
		return errResult(err), overlayApplyOutput{}, nil
	}

	applier := overlay.NewApplier()

	if input.DryRun {
		return handleDryRun(applier, specResult, o)
	}

	result, err := applier.ApplyParsed(specResult, o)
	if err != nil {
		return errResult(err), overlayApplyOutput{}, nil
	}

	output := overlayApplyOutput{
		ActionsApplied: result.ActionsApplied,
		ActionsSkipped: result.ActionsSkipped,
		Warnings:       result.WarningStrings(),
	}

	output.Changes = makeSlice[overlayApplyChange](len(result.Changes))
	for _, c := range result.Changes {
		output.Changes = append(output.Changes, overlayApplyChange{
			ActionIndex: c.ActionIndex,
			Target:      c.Target,
			Operation:   c.Operation,
			MatchCount:  c.MatchCount,
		})
	}

	output.Summary = buildOverlayApplySummary(result.ActionsApplied, result.ActionsSkipped, len(result.Warnings))

	// Marshal the resulting document.
	pr := result.ToParseResult()
	var data []byte
	switch result.SourceFormat {
	case parser.SourceFormatJSON:
		data, err = pr.MarshalOrderedJSONIndent("", "  ")
	default:
		data, err = pr.MarshalOrderedYAML()
	}
	if err != nil {
		return errResult(err), overlayApplyOutput{}, nil
	}

	if input.Output != "" {
		if err := os.WriteFile(input.Output, data, 0o644); err != nil {
			return errResult(fmt.Errorf("failed to write output file: %w", err)), overlayApplyOutput{}, nil
		}
		output.WrittenTo = input.Output
	} else {
		output.Document = string(data)
	}

	return nil, output, nil
}

func handleDryRun(applier *overlay.Applier, specResult *parser.ParseResult, o *overlay.Overlay) (*mcp.CallToolResult, overlayApplyOutput, error) {
	dryResult, err := applier.DryRun(specResult, o)
	if err != nil {
		return errResult(err), overlayApplyOutput{}, nil
	}

	output := overlayApplyOutput{
		ActionsApplied: dryResult.WouldApply,
		ActionsSkipped: dryResult.WouldSkip,
		Warnings:       dryResult.Warnings,
	}

	output.Changes = makeSlice[overlayApplyChange](len(dryResult.Changes))
	for _, c := range dryResult.Changes {
		output.Changes = append(output.Changes, overlayApplyChange{
			ActionIndex: c.ActionIndex,
			Target:      c.Target,
			Operation:   c.Operation,
			MatchCount:  c.MatchCount,
		})
	}

	output.Summary = buildOverlayApplySummary(dryResult.WouldApply, dryResult.WouldSkip, len(dryResult.Warnings)) +
		" (dry run - no changes applied)"

	return nil, output, nil
}

func buildOverlayApplySummary(applied, skipped, warnings int) string {
	summary := formatCount(applied, "action") + " applied"
	if skipped > 0 {
		summary += ", " + formatCount(skipped, "action") + " skipped"
	}
	if warnings > 0 {
		summary += " with " + formatCount(warnings, "warning")
	}
	summary += "."
	return summary
}

// resolveOverlayInput parses an overlay document from the specInput.
// The overlay can be provided as a file path, URL, or inline content.
func resolveOverlayInput(ctx context.Context, s specInput) (*overlay.Overlay, error) {
	count := 0
	if s.File != "" {
		count++
	}
	if s.URL != "" {
		count++
	}
	if s.Content != "" {
		count++
	}
	if count != 1 {
		return nil, fmt.Errorf("exactly one of file, url, or content must be provided for overlay (got %d)", count)
	}

	switch {
	case s.File != "":
		return overlay.ParseOverlayFile(s.File)
	case s.URL != "":
		client := &http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil) //nolint:gosec // URL comes from user input
		if err != nil {
			return nil, fmt.Errorf("failed to create overlay request: %w", err)
		}
		resp, err := client.Do(req) //nolint:gosec // G704 - URL is user-provided input (MCP tool)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch overlay from URL: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("failed to fetch overlay from URL: server returned %s", resp.Status)
		}
		const maxOverlaySize = 10 << 20 // 10 MiB
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxOverlaySize))
		if err != nil {
			return nil, fmt.Errorf("failed to read overlay response: %w", err)
		}
		return overlay.ParseOverlay(data)
	default:
		return overlay.ParseOverlay([]byte(s.Content))
	}
}

// overlay_validate types and handler

type overlayValidateInput struct {
	Overlay specInput `json:"overlay" jsonschema:"The Overlay document to validate"`
}

type overlayValidateIssue struct {
	Field   string `json:"field,omitempty"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

type overlayValidateOutput struct {
	Valid      bool                   `json:"valid"`
	ErrorCount int                    `json:"error_count"`
	Errors     []overlayValidateIssue `json:"errors,omitempty"`
}

func handleOverlayValidate(ctx context.Context, _ *mcp.CallToolRequest, input overlayValidateInput) (*mcp.CallToolResult, overlayValidateOutput, error) {
	o, err := resolveOverlayInput(ctx, input.Overlay)
	if err != nil {
		return errResult(err), overlayValidateOutput{}, nil
	}

	errs := overlay.Validate(o)

	output := overlayValidateOutput{
		Valid:      len(errs) == 0,
		ErrorCount: len(errs),
	}

	output.Errors = makeSlice[overlayValidateIssue](len(errs))
	for _, e := range errs {
		output.Errors = append(output.Errors, overlayValidateIssue{
			Field:   e.Field,
			Path:    e.Path,
			Message: e.Message,
		})
	}

	return nil, output, nil
}
