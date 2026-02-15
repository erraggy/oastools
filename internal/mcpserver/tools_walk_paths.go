package mcpserver

import (
	"context"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type walkPathsInput struct {
	Spec        specInput `json:"spec"                     jsonschema:"The OAS document to walk"`
	Path        string    `json:"path,omitempty"           jsonschema:"Filter by path pattern (* = one segment\\, ** = zero or more segments\\, e.g. /users/* or /drives/**/workbook/**)"`
	Extension   string    `json:"extension,omitempty"      jsonschema:"Filter by extension key=value (e.g. x-internal=true)"`
	ResolveRefs bool      `json:"resolve_refs,omitempty"   jsonschema:"Resolve $ref pointers in output. Inlines referenced objects instead of showing $ref strings."`
	Detail      bool      `json:"detail,omitempty"         jsonschema:"Return full path item objects instead of summaries"`
	Limit       int       `json:"limit,omitempty"          jsonschema:"Maximum number of results to return (default 100)"`
	Offset      int       `json:"offset,omitempty"         jsonschema:"Skip the first N results (for pagination)"`
}

type pathSummary struct {
	Path        string `json:"path"`
	MethodCount int    `json:"method_count"`
	Summary     string `json:"summary,omitempty"`
}

type pathDetail struct {
	Path     string           `json:"path"`
	PathItem *parser.PathItem `json:"path_item"`
}

// pathInfo holds collected path item information.
type pathInfo struct {
	PathTemplate string
	PathItem     *parser.PathItem
}

type walkPathsOutput struct {
	Total     int           `json:"total"`
	Matched   int           `json:"matched"`
	Returned  int           `json:"returned"`
	Summaries []pathSummary `json:"summaries,omitempty"`
	Paths     []pathDetail  `json:"paths,omitempty"`
}

func handleWalkPaths(_ context.Context, _ *mcp.CallToolRequest, input walkPathsInput) (*mcp.CallToolResult, any, error) {
	var extraOpts []parser.Option
	if input.ResolveRefs {
		extraOpts = append(extraOpts, parser.WithResolveRefs(true))
	}

	result, err := input.Spec.resolve(extraOpts...)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Collect path items using the walker.
	var all []*pathInfo
	err = walker.Walk(result,
		walker.WithPathItemHandler(func(wc *walker.WalkContext, pathItem *parser.PathItem) walker.Action {
			// Only collect top-level path items (not component pathItems or callbacks).
			if wc.PathTemplate != "" {
				all = append(all, &pathInfo{
					PathTemplate: wc.PathTemplate,
					PathItem:     pathItem,
				})
			}
			return walker.Continue
		}),
	)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Filter paths.
	matched, err := filterWalkPaths(all, input)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Apply offset/limit pagination.
	returned := paginate(matched, input.Offset, input.Limit)

	output := walkPathsOutput{
		Total:    len(all),
		Matched:  len(matched),
		Returned: len(returned),
	}

	if input.Detail {
		output.Paths = makeSlice[pathDetail](len(returned))
		for _, info := range returned {
			output.Paths = append(output.Paths, pathDetail{
				Path:     info.PathTemplate,
				PathItem: info.PathItem,
			})
		}
	} else {
		output.Summaries = makeSlice[pathSummary](len(returned))
		for _, info := range returned {
			output.Summaries = append(output.Summaries, pathSummary{
				Path:        info.PathTemplate,
				MethodCount: countMethods(info.PathItem),
				Summary:     info.PathItem.Summary,
			})
		}
	}

	return nil, output, nil
}

// filterWalkPaths applies all path filters and returns the matching subset.
func filterWalkPaths(paths []*pathInfo, input walkPathsInput) ([]*pathInfo, error) {
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

	var matched []*pathInfo
	for _, info := range paths {
		if input.Path != "" && !matchWalkPath(info.PathTemplate, input.Path) {
			continue
		}
		if hasExtFilter && !matchExtension(info.PathItem.Extra, extKey, extValue) {
			continue
		}
		matched = append(matched, info)
	}
	return matched, nil
}

// countMethods returns the number of HTTP methods defined on a PathItem.
func countMethods(p *parser.PathItem) int {
	count := 0
	if p.Get != nil {
		count++
	}
	if p.Put != nil {
		count++
	}
	if p.Post != nil {
		count++
	}
	if p.Delete != nil {
		count++
	}
	if p.Options != nil {
		count++
	}
	if p.Head != nil {
		count++
	}
	if p.Patch != nil {
		count++
	}
	if p.Trace != nil {
		count++
	}
	if p.Query != nil {
		count++
	}
	count += len(p.AdditionalOperations)
	return count
}
