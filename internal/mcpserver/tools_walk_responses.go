package mcpserver

import (
	"context"
	"strings"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type walkResponsesInput struct {
	Spec        specInput `json:"spec"                     jsonschema:"The OAS document to walk"`
	Status      string    `json:"status,omitempty"         jsonschema:"Filter by status code (200\\, 4xx\\, default)"`
	Path        string    `json:"path,omitempty"           jsonschema:"Filter by path pattern (supports * glob)"`
	Method      string    `json:"method,omitempty"         jsonschema:"Filter by HTTP method (get\\, post\\, put\\, delete\\, patch\\, etc.)"`
	Extension   string    `json:"extension,omitempty"      jsonschema:"Filter by extension key=value (e.g. x-internal=true)"`
	ResolveRefs bool      `json:"resolve_refs,omitempty"   jsonschema:"Resolve $ref pointers before output"`
	Detail      bool      `json:"detail,omitempty"         jsonschema:"Return full response objects instead of summaries"`
	Limit       int       `json:"limit,omitempty"          jsonschema:"Maximum number of results to return (default 100)"`
}

type responseSummary struct {
	Status      string `json:"status"`
	Path        string `json:"path,omitempty"`
	Method      string `json:"method,omitempty"`
	Description string `json:"description,omitempty"`
}

type responseDetail struct {
	Status   string           `json:"status"`
	Path     string           `json:"path,omitempty"`
	Method   string           `json:"method,omitempty"`
	Response *parser.Response `json:"response"`
}

type walkResponsesOutput struct {
	Total     int               `json:"total"`
	Matched   int               `json:"matched"`
	Returned  int               `json:"returned"`
	Summaries []responseSummary `json:"summaries,omitempty"`
	Responses []responseDetail  `json:"responses,omitempty"`
}

func handleWalkResponses(_ context.Context, _ *mcp.CallToolRequest, input walkResponsesInput) (*mcp.CallToolResult, any, error) {
	var extraOpts []parser.Option
	if input.ResolveRefs {
		extraOpts = append(extraOpts, parser.WithResolveRefs(true))
	}

	result, err := input.Spec.resolve(extraOpts...)
	if err != nil {
		return errResult(err), nil, nil
	}

	collector, err := walker.CollectResponses(result)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Filter responses.
	matched, err := filterWalkResponses(collector.All, input)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Apply limit.
	limit := input.Limit
	if limit <= 0 {
		limit = defaultWalkLimit
	}
	returned := matched
	if len(returned) > limit {
		returned = returned[:limit]
	}

	output := walkResponsesOutput{
		Total:    len(collector.All),
		Matched:  len(matched),
		Returned: len(returned),
	}

	if input.Detail {
		output.Responses = makeSlice[responseDetail](len(returned))
		for _, info := range returned {
			output.Responses = append(output.Responses, responseDetail{
				Status:   info.StatusCode,
				Path:     info.PathTemplate,
				Method:   strings.ToUpper(info.Method),
				Response: info.Response,
			})
		}
	} else {
		output.Summaries = makeSlice[responseSummary](len(returned))
		for _, info := range returned {
			s := responseSummary{
				Status: info.StatusCode,
				Path:   info.PathTemplate,
				Method: strings.ToUpper(info.Method),
			}
			if info.Response != nil {
				s.Description = info.Response.Description
			}
			output.Summaries = append(output.Summaries, s)
		}
	}

	return nil, output, nil
}

// filterWalkResponses applies all response filters and returns the matching subset.
func filterWalkResponses(responses []*walker.ResponseInfo, input walkResponsesInput) ([]*walker.ResponseInfo, error) {
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

	var matched []*walker.ResponseInfo
	for _, info := range responses {
		if input.Status != "" && !statusCodeMatches(info.StatusCode, input.Status) {
			continue
		}
		if input.Path != "" && !matchWalkPath(info.PathTemplate, input.Path) {
			continue
		}
		if input.Method != "" && !strings.EqualFold(info.Method, input.Method) {
			continue
		}
		if hasExtFilter && (info.Response == nil || !matchExtension(info.Response.Extra, extKey, extValue)) {
			continue
		}
		matched = append(matched, info)
	}
	return matched, nil
}

// statusCodeMatches checks if a status code matches a filter pattern.
// Supports exact match ("200"), wildcard patterns ("4xx"), and "default".
func statusCodeMatches(statusCode, filter string) bool {
	if strings.EqualFold(statusCode, filter) {
		return true
	}
	// Support wildcard patterns like "2xx", "4xx", "5xx".
	if len(filter) == 3 && strings.HasSuffix(strings.ToLower(filter), "xx") && len(statusCode) == 3 {
		return statusCode[0] == filter[0]
	}
	return false
}
