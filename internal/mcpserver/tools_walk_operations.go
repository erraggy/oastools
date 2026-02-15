package mcpserver

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type walkOperationsInput struct {
	Spec        specInput `json:"spec"                       jsonschema:"The OAS document to walk"`
	Method      string    `json:"method,omitempty"           jsonschema:"Filter by HTTP method (get\\, post\\, put\\, delete\\, patch\\, etc.)"`
	Path        string    `json:"path,omitempty"             jsonschema:"Filter by path pattern (supports * glob)"`
	Tag         string    `json:"tag,omitempty"              jsonschema:"Filter by tag name"`
	Deprecated  bool      `json:"deprecated,omitempty"       jsonschema:"Only show deprecated operations"`
	OperationID string    `json:"operation_id,omitempty"     jsonschema:"Select by operationId"`
	Extension   string    `json:"extension,omitempty"        jsonschema:"Filter by extension key=value (e.g. x-internal=true)"`
	ResolveRefs bool      `json:"resolve_refs,omitempty"     jsonschema:"Resolve $ref pointers before output"`
	Detail      bool      `json:"detail,omitempty"           jsonschema:"Return full operation objects instead of summaries"`
	Limit       int       `json:"limit,omitempty"            jsonschema:"Maximum number of results to return (default 100)"`
	Offset      int       `json:"offset,omitempty"           jsonschema:"Skip the first N results (for pagination)"`
}

type operationSummary struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	OperationID string   `json:"operation_id,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Deprecated  bool     `json:"deprecated,omitempty"`
}

type operationDetail struct {
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Operation *parser.Operation `json:"operation"`
}

type walkOperationsOutput struct {
	Total      int                `json:"total"`
	Matched    int                `json:"matched"`
	Returned   int                `json:"returned"`
	Summaries  []operationSummary `json:"summaries,omitempty"`
	Operations []operationDetail  `json:"operations,omitempty"`
}

const defaultWalkLimit = 100

func handleWalkOperations(_ context.Context, _ *mcp.CallToolRequest, input walkOperationsInput) (*mcp.CallToolResult, any, error) {
	var extraOpts []parser.Option
	if input.ResolveRefs {
		extraOpts = append(extraOpts, parser.WithResolveRefs(true))
	}

	result, err := input.Spec.resolve(extraOpts...)
	if err != nil {
		return errResult(err), nil, nil
	}

	collector, err := walker.CollectOperations(result)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Filter operations.
	matched, err := filterWalkOperations(collector.All, input)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Apply offset/limit pagination.
	returned := paginate(matched, input.Offset, input.Limit)

	output := walkOperationsOutput{
		Total:    len(collector.All),
		Matched:  len(matched),
		Returned: len(returned),
	}

	if input.Detail {
		output.Operations = makeSlice[operationDetail](len(returned))
		for _, op := range returned {
			output.Operations = append(output.Operations, operationDetail{
				Method:    strings.ToUpper(op.Method),
				Path:      op.PathTemplate,
				Operation: op.Operation,
			})
		}
	} else {
		output.Summaries = makeSlice[operationSummary](len(returned))
		for _, op := range returned {
			output.Summaries = append(output.Summaries, operationSummary{
				Method:      strings.ToUpper(op.Method),
				Path:        op.PathTemplate,
				OperationID: op.Operation.OperationID,
				Summary:     op.Operation.Summary,
				Tags:        op.Operation.Tags,
				Deprecated:  op.Operation.Deprecated,
			})
		}
	}

	return nil, output, nil
}

// filterWalkOperations applies all operation filters and returns the matching subset.
func filterWalkOperations(ops []*walker.OperationInfo, input walkOperationsInput) ([]*walker.OperationInfo, error) {
	// Parse extension filter once if provided.
	var extKey, extValue string
	var hasExtFilter bool
	if input.Extension != "" {
		key, val, err := parseExtensionKeyValue(input.Extension)
		if err != nil {
			return nil, err
		}
		extKey = key
		extValue = val
		hasExtFilter = true
	}

	var matched []*walker.OperationInfo
	for _, op := range ops {
		if input.Method != "" && !strings.EqualFold(op.Method, input.Method) {
			continue
		}
		if input.Path != "" && !matchWalkPath(op.PathTemplate, input.Path) {
			continue
		}
		if input.Tag != "" && !slices.Contains(op.Operation.Tags, input.Tag) {
			continue
		}
		if input.Deprecated && !op.Operation.Deprecated {
			continue
		}
		if input.OperationID != "" && op.Operation.OperationID != input.OperationID {
			continue
		}
		if hasExtFilter && !matchExtension(op.Operation.Extra, extKey, extValue) {
			continue
		}
		matched = append(matched, op)
	}
	return matched, nil
}

// matchWalkPath checks if a path template matches a pattern.
// Supports simple glob matching where * matches exactly one path segment.
func matchWalkPath(pathTemplate, pattern string) bool {
	if pattern == "" {
		return true
	}
	if strings.Contains(pattern, "*") {
		patternParts := strings.Split(pattern, "/")
		pathParts := strings.Split(pathTemplate, "/")
		if len(patternParts) != len(pathParts) {
			return false
		}
		for i, pp := range patternParts {
			if pp == "*" {
				continue
			}
			if pp != pathParts[i] {
				return false
			}
		}
		return true
	}
	return pathTemplate == pattern
}

// parseExtensionKeyValue parses a simple "key=value" extension filter.
// Returns the key and value. If no "=" is present, value is empty (existence check).
func parseExtensionKeyValue(filter string) (string, string, error) {
	if filter == "" {
		return "", "", fmt.Errorf("empty extension filter")
	}
	parts := strings.SplitN(filter, "=", 2)
	key := parts[0]
	if !strings.HasPrefix(key, "x-") {
		return "", "", fmt.Errorf("invalid extension key %q: must start with \"x-\"", key)
	}
	if len(parts) == 2 {
		return key, parts[1], nil
	}
	return key, "", nil
}

// matchExtension checks if a node's extensions match a key=value filter.
// If value is empty, it checks for existence only.
func matchExtension(extensions map[string]any, key, value string) bool {
	val, exists := extensions[key]
	if !exists {
		return false
	}
	if value == "" {
		return true // existence check
	}
	return fmt.Sprintf("%v", val) == value
}
