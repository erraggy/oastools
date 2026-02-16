package mcpserver

import (
	"context"
	"strings"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type walkHeadersInput struct {
	Spec        specInput `json:"spec"                     jsonschema:"The OAS document to walk"`
	Name        string    `json:"name,omitempty"           jsonschema:"Filter by header name (exact match\\, or glob with * and ? for pattern matching\\, e.g. X-Rate-* or X-Request-*)"`
	Path        string    `json:"path,omitempty"           jsonschema:"Filter by path pattern (* = one segment\\, ** = zero or more segments\\, e.g. /users/* or /drives/**/workbook/**)"`
	Method      string    `json:"method,omitempty"         jsonschema:"Filter by HTTP method (get\\, post\\, put\\, delete\\, patch\\, etc.)"`
	Status      string    `json:"status,omitempty"         jsonschema:"Filter by status code: exact (200\\, 404)\\, wildcard (2xx\\, 4xx\\, 5xx)\\, or default (case-insensitive)"`
	Extension   string    `json:"extension,omitempty"      jsonschema:"Filter by extension key=value (e.g. x-internal=true)"`
	Component   bool      `json:"component,omitempty"      jsonschema:"Only show component headers (defined in components/headers)"`
	ResolveRefs bool      `json:"resolve_refs,omitempty"   jsonschema:"Resolve $ref pointers in output. Inlines referenced objects instead of showing $ref strings."`
	Detail      bool      `json:"detail,omitempty"         jsonschema:"Return full header objects instead of summaries"`
	GroupBy     string    `json:"group_by,omitempty"       jsonschema:"Group results and return counts instead of individual items. Values: name\\, status_code"`
	Limit       int       `json:"limit,omitempty"          jsonschema:"Maximum number of results to return (default 100; 25 in detail mode)"`
	Offset      int       `json:"offset,omitempty"         jsonschema:"Skip the first N results (for pagination)"`
}

// headerInfo holds walker context for a single header occurrence.
type headerInfo struct {
	Name        string
	Path        string
	Method      string
	StatusCode  string
	IsComponent bool
	Header      *parser.Header
}

type headerSummary struct {
	Name        string `json:"name"`
	Path        string `json:"path,omitempty"`
	Method      string `json:"method,omitempty"`
	Status      string `json:"status,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Deprecated  bool   `json:"deprecated,omitempty"`
}

type headerDetail struct {
	Name   string         `json:"name"`
	Path   string         `json:"path,omitempty"`
	Method string         `json:"method,omitempty"`
	Status string         `json:"status,omitempty"`
	Header *parser.Header `json:"header"`
}

type walkHeadersOutput struct {
	Total     int             `json:"total"`
	Matched   int             `json:"matched"`
	Returned  int             `json:"returned"`
	Summaries []headerSummary `json:"summaries,omitempty"`
	Headers   []headerDetail  `json:"headers,omitempty"`
	Groups    []groupCount    `json:"groups,omitempty"`
}

func handleWalkHeaders(_ context.Context, _ *mcp.CallToolRequest, input walkHeadersInput) (*mcp.CallToolResult, any, error) {
	var extraOpts []parser.Option
	if input.ResolveRefs {
		extraOpts = append(extraOpts, parser.WithResolveRefs(true))
	}

	result, err := input.Spec.resolve(extraOpts...)
	if err != nil {
		return errResult(err), nil, nil
	}

	if err := validateGroupBy(input.GroupBy, input.Detail, []string{"name", "status_code"}); err != nil {
		return errResult(err), nil, nil
	}

	// Collect all headers using the walker.
	var all []*headerInfo
	err = walker.Walk(result,
		walker.WithHeaderHandler(func(wc *walker.WalkContext, header *parser.Header) walker.Action {
			all = append(all, &headerInfo{
				Name:        wc.Name,
				Path:        wc.PathTemplate,
				Method:      wc.Method,
				StatusCode:  wc.StatusCode,
				IsComponent: wc.IsComponent,
				Header:      header,
			})
			return walker.Continue
		}),
	)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Filter headers.
	matched, err := filterWalkHeaders(all, input)
	if err != nil {
		return errResult(err), nil, nil
	}

	// group_by: aggregate matched headers and return counts.
	if input.GroupBy != "" {
		groups := groupAndSort(matched, func(info *headerInfo) []string {
			switch strings.ToLower(input.GroupBy) {
			case "name":
				return []string{info.Name}
			case "status_code":
				if info.StatusCode == "" {
					return nil
				}
				return []string{info.StatusCode}
			default:
				return nil
			}
		})
		paged := paginate(groups, input.Offset, input.Limit)
		output := walkHeadersOutput{
			Total:    len(all),
			Matched:  len(matched),
			Returned: len(paged),
			Groups:   paged,
		}
		return nil, output, nil
	}

	// Apply offset/limit pagination.
	limit := input.Limit
	if input.Detail {
		limit = detailLimit(limit)
	}
	returned := paginate(matched, input.Offset, limit)

	output := walkHeadersOutput{
		Total:    len(all),
		Matched:  len(matched),
		Returned: len(returned),
	}

	if input.Detail {
		output.Headers = makeSlice[headerDetail](len(returned))
		for _, info := range returned {
			output.Headers = append(output.Headers, headerDetail{
				Name:   info.Name,
				Path:   info.Path,
				Method: strings.ToUpper(info.Method),
				Status: info.StatusCode,
				Header: info.Header,
			})
		}
	} else {
		output.Summaries = makeSlice[headerSummary](len(returned))
		for _, info := range returned {
			s := headerSummary{
				Name:   info.Name,
				Path:   info.Path,
				Method: strings.ToUpper(info.Method),
				Status: info.StatusCode,
			}
			if info.Header != nil {
				s.Description = info.Header.Description
				s.Required = info.Header.Required
				s.Deprecated = info.Header.Deprecated
			}
			output.Summaries = append(output.Summaries, s)
		}
	}

	return nil, output, nil
}

// filterWalkHeaders applies all header filters and returns the matching subset.
func filterWalkHeaders(headers []*headerInfo, input walkHeadersInput) ([]*headerInfo, error) {
	if err := validateGlobPattern(input.Name); err != nil {
		return nil, err
	}

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

	var matched []*headerInfo
	for _, info := range headers {
		if input.Name != "" && !matchGlobName(info.Name, input.Name) {
			continue
		}
		if input.Path != "" && !matchWalkPath(info.Path, input.Path) {
			continue
		}
		if input.Method != "" && !strings.EqualFold(info.Method, input.Method) {
			continue
		}
		if input.Status != "" && !statusCodeMatches(info.StatusCode, input.Status) {
			continue
		}
		if hasExtFilter && (info.Header == nil || !matchExtension(info.Header.Extra, extKey, extValue)) {
			continue
		}
		if input.Component && !info.IsComponent {
			continue
		}
		matched = append(matched, info)
	}
	return matched, nil
}
