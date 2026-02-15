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
	Path        string    `json:"path,omitempty"             jsonschema:"Filter by path pattern (* = one segment\\, ** = zero or more segments\\, e.g. /users/* or /drives/**/workbook/**)"`
	Tag         string    `json:"tag,omitempty"              jsonschema:"Filter by tag name (exact match\\, case-sensitive)"`
	Deprecated  bool      `json:"deprecated,omitempty"       jsonschema:"Only show deprecated operations"`
	OperationID string    `json:"operation_id,omitempty"     jsonschema:"Select a single operation by operationId (exact match)"`
	Extension   string    `json:"extension,omitempty"        jsonschema:"Filter by extension key=value (e.g. x-internal=true)"`
	ResolveRefs bool      `json:"resolve_refs,omitempty"     jsonschema:"Resolve $ref pointers in output. Inlines referenced objects instead of showing $ref strings."`
	Detail      bool      `json:"detail,omitempty"           jsonschema:"Return full operation objects instead of summaries"`
	GroupBy     string    `json:"group_by,omitempty"         jsonschema:"Group results and return counts instead of individual items. Values: tag\\, method. Note: group_by=tag excludes untagged operations and counts multi-tagged operations once per tag."`
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
	Groups     []groupCount       `json:"groups,omitempty"`
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

	if err := validateGroupBy(input.GroupBy, input.Detail, []string{"tag", "method"}); err != nil {
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

	// group_by: aggregate matched operations and return counts.
	if input.GroupBy != "" {
		groups := groupAndSort(matched, func(op *walker.OperationInfo) []string {
			switch strings.ToLower(input.GroupBy) {
			case "tag":
				if len(op.Operation.Tags) == 0 {
					return nil
				}
				return op.Operation.Tags
			case "method":
				return []string{strings.ToUpper(op.Method)}
			default:
				return nil
			}
		})
		paged := paginate(groups, input.Offset, input.Limit)
		output := walkOperationsOutput{
			Total:    len(collector.All),
			Matched:  len(matched),
			Returned: len(paged),
			Groups:   paged,
		}
		return nil, output, nil
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
// Supports glob matching: * matches one path segment, ** matches zero or more segments.
func matchWalkPath(pathTemplate, pattern string) bool {
	if pattern == "" {
		return true
	}
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(pathTemplate, "/")
	return matchPathParts(pathParts, patternParts)
}

// matchPathParts recursively matches path segments against pattern segments.
// * matches exactly one segment, ** matches zero or more segments.
func matchPathParts(path, pattern []string) bool {
	for len(pattern) > 0 {
		seg := pattern[0]
		pattern = pattern[1:]

		if seg == "**" {
			// If ** is the last pattern segment, it matches everything remaining.
			if len(pattern) == 0 {
				return true
			}
			// Try matching the rest of the pattern at every possible position.
			for i := range len(path) + 1 {
				if matchPathParts(path[i:], pattern) {
					return true
				}
			}
			return false
		}

		if len(path) == 0 {
			return false
		}

		if seg != "*" && seg != path[0] {
			return false
		}

		path = path[1:]
	}
	return len(path) == 0
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
