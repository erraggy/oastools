package mcpserver

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	"github.com/erraggy/oastools/walker"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type walkRefsInput struct {
	Spec     specInput `json:"spec"                   jsonschema:"The OAS document to walk"`
	Target   string    `json:"target,omitempty"        jsonschema:"Filter by ref target (supports * and ? glob, e.g. *schemas/Pet or *responses/*)"`
	NodeType string    `json:"node_type,omitempty"     jsonschema:"Filter by ref node type: schema, parameter, response, requestBody, header, pathItem, link, example, securityScheme"`
	Detail   bool      `json:"detail,omitempty"        jsonschema:"Return individual source locations instead of aggregated counts"`
	GroupBy  string    `json:"group_by,omitempty"      jsonschema:"Group results and return counts instead of individual items. Values: node_type"`
	Limit    int       `json:"limit,omitempty"         jsonschema:"Maximum number of results to return (default 100; 25 in detail mode)"`
	Offset   int       `json:"offset,omitempty"        jsonschema:"Skip the first N results (for pagination)"`
}

type refSummary struct {
	Ref   string `json:"ref"`
	Count int    `json:"count"`
}

type refDetail struct {
	Ref        string `json:"ref"`
	SourcePath string `json:"source_path"`
	NodeType   string `json:"node_type"`
}

// walkRefsOutput holds results from walk_refs. In summary mode, Total and
// Matched count unique ref targets. In detail and group_by modes, they count
// individual ref occurrences (a single target referenced 3 times counts as 3).
type walkRefsOutput struct {
	Total     int          `json:"total"`
	Matched   int          `json:"matched"`
	Returned  int          `json:"returned"`
	Summaries []refSummary `json:"refs,omitempty"`
	Details   []refDetail  `json:"details,omitempty"`
	Groups    []groupCount `json:"groups,omitempty"`
}

func handleWalkRefs(_ context.Context, _ *mcp.CallToolRequest, input walkRefsInput) (*mcp.CallToolResult, any, error) {
	// Validate glob pattern before expensive walk.
	if err := validateGlobPattern(input.Target); err != nil {
		return errResult(err), nil, nil
	}

	if err := validateGroupBy(input.GroupBy, input.Detail, []string{"node_type"}); err != nil {
		return errResult(err), nil, nil
	}

	result, err := input.Spec.resolve()
	if err != nil {
		return errResult(err), nil, nil
	}

	// Collect all refs via the walker.
	var allRefs []*walker.RefInfo
	err = walker.Walk(result,
		walker.WithMapRefTracking(),
		walker.WithRefHandler(func(_ *walker.WalkContext, ref *walker.RefInfo) walker.Action {
			allRefs = append(allRefs, ref)
			return walker.Continue
		}),
	)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Filter refs.
	filtered := filterRefs(allRefs, input)

	// group_by: aggregate by node type and return counts.
	if input.GroupBy != "" {
		groups := groupAndSort(filtered, func(ref *walker.RefInfo) []string {
			return []string{string(ref.NodeType)}
		})
		paged := paginate(groups, input.Offset, input.Limit)
		output := walkRefsOutput{
			Total:    len(allRefs),
			Matched:  len(filtered),
			Returned: len(paged),
			Groups:   paged,
		}
		return nil, output, nil
	}

	if input.Detail {
		// Detail mode: return individual ref locations.
		limit := detailLimit(input.Limit)
		paged := paginate(filtered, input.Offset, limit)
		output := walkRefsOutput{
			Total:    len(allRefs),
			Matched:  len(filtered),
			Returned: len(paged),
			Details:  makeSlice[refDetail](len(paged)),
		}
		for _, ref := range paged {
			output.Details = append(output.Details, refDetail{
				Ref:        ref.Ref,
				SourcePath: ref.SourcePath,
				NodeType:   string(ref.NodeType),
			})
		}
		return nil, output, nil
	}

	// Summary mode: aggregate by ref target, sort by count desc.
	counts := make(map[string]int)
	for _, ref := range filtered {
		counts[ref.Ref]++
	}

	summaries := make([]refSummary, 0, len(counts))
	for ref, count := range counts {
		summaries = append(summaries, refSummary{Ref: ref, Count: count})
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Count != summaries[j].Count {
			return summaries[i].Count > summaries[j].Count
		}
		return summaries[i].Ref < summaries[j].Ref
	})

	paged := paginate(summaries, input.Offset, input.Limit)
	output := walkRefsOutput{
		Total:     countUniqueRefs(allRefs),
		Matched:   countUniqueRefs(filtered),
		Returned:  len(paged),
		Summaries: paged,
	}
	return nil, output, nil
}

// filterRefs applies target and node_type filters to refs.
func filterRefs(refs []*walker.RefInfo, input walkRefsInput) []*walker.RefInfo {
	if input.Target == "" && input.NodeType == "" {
		return refs
	}
	var filtered []*walker.RefInfo
	for _, ref := range refs {
		if input.Target != "" && !matchRefGlob(ref.Ref, input.Target) {
			continue
		}
		if input.NodeType != "" && !strings.EqualFold(string(ref.NodeType), input.NodeType) {
			continue
		}
		filtered = append(filtered, ref)
	}
	return filtered
}

// countUniqueRefs returns the number of distinct ref targets.
func countUniqueRefs(refs []*walker.RefInfo) int {
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		seen[ref.Ref] = struct{}{}
	}
	return len(seen)
}

// matchRefGlob matches a $ref value against a glob pattern. Unlike matchGlobName,
// this function allows * and ? to match across / separators in ref paths like
// "#/components/schemas/Pet". It does this by replacing / with a non-separator
// character before calling filepath.Match.
func matchRefGlob(ref, pattern string) bool {
	if !strings.ContainsAny(pattern, "*?") {
		return strings.EqualFold(ref, pattern)
	}
	// Replace / with : so filepath.Match's * can cross path boundaries.
	normalizedRef := strings.ReplaceAll(strings.ToLower(ref), "/", ":")
	normalizedPattern := strings.ReplaceAll(strings.ToLower(pattern), "/", ":")
	matched, err := filepath.Match(normalizedPattern, normalizedRef)
	return err == nil && matched
}
