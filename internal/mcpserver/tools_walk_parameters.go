package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type walkParametersInput struct {
	Spec        specInput `json:"spec"                     jsonschema:"The OAS document to walk"`
	In          string    `json:"in,omitempty"             jsonschema:"Filter by parameter location (query\\, header\\, path\\, cookie)"`
	Name        string    `json:"name,omitempty"           jsonschema:"Filter by parameter name"`
	Path        string    `json:"path,omitempty"           jsonschema:"Filter by path pattern (supports * glob)"`
	Method      string    `json:"method,omitempty"         jsonschema:"Filter by HTTP method (get\\, post\\, put\\, delete\\, patch\\, etc.)"`
	Extension   string    `json:"extension,omitempty"      jsonschema:"Filter by extension key=value (e.g. x-internal=true)"`
	ResolveRefs bool      `json:"resolve_refs,omitempty"   jsonschema:"Resolve $ref pointers before output"`
	Detail      bool      `json:"detail,omitempty"         jsonschema:"Return full parameter objects instead of summaries"`
	Limit       int       `json:"limit,omitempty"          jsonschema:"Maximum number of results to return (default 100)"`
	Offset      int       `json:"offset,omitempty"         jsonschema:"Skip the first N results (for pagination)"`
}

type parameterSummary struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Path     string `json:"path,omitempty"`
	Method   string `json:"method,omitempty"`
	Required bool   `json:"required,omitempty"`
	Type     string `json:"type,omitempty"`
}

type parameterDetail struct {
	Name      string            `json:"name"`
	Path      string            `json:"path,omitempty"`
	Method    string            `json:"method,omitempty"`
	Parameter *parser.Parameter `json:"parameter"`
}

type walkParametersOutput struct {
	Total      int                `json:"total"`
	Matched    int                `json:"matched"`
	Returned   int                `json:"returned"`
	Summaries  []parameterSummary `json:"summaries,omitempty"`
	Parameters []parameterDetail  `json:"parameters,omitempty"`
}

func handleWalkParameters(_ context.Context, _ *mcp.CallToolRequest, input walkParametersInput) (*mcp.CallToolResult, any, error) {
	var extraOpts []parser.Option
	if input.ResolveRefs {
		extraOpts = append(extraOpts, parser.WithResolveRefs(true))
	}

	result, err := input.Spec.resolve(extraOpts...)
	if err != nil {
		return errResult(err), nil, nil
	}

	collector, err := walker.CollectParameters(result)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Filter parameters.
	matched, err := filterWalkParameters(collector.All, input)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Apply offset/limit pagination.
	returned := paginate(matched, input.Offset, input.Limit)

	output := walkParametersOutput{
		Total:    len(collector.All),
		Matched:  len(matched),
		Returned: len(returned),
	}

	if input.Detail {
		output.Parameters = makeSlice[parameterDetail](len(returned))
		for _, info := range returned {
			output.Parameters = append(output.Parameters, parameterDetail{
				Name:      info.Name,
				Path:      info.PathTemplate,
				Method:    strings.ToUpper(info.Method),
				Parameter: info.Parameter,
			})
		}
	} else {
		output.Summaries = makeSlice[parameterSummary](len(returned))
		for _, info := range returned {
			output.Summaries = append(output.Summaries, parameterSummary{
				Name:     info.Name,
				In:       info.In,
				Path:     info.PathTemplate,
				Method:   strings.ToUpper(info.Method),
				Required: info.Parameter.Required,
				Type:     parameterTypeString(info.Parameter),
			})
		}
	}

	return nil, output, nil
}

// filterWalkParameters applies all parameter filters and returns the matching subset.
func filterWalkParameters(params []*walker.ParameterInfo, input walkParametersInput) ([]*walker.ParameterInfo, error) {
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

	var matched []*walker.ParameterInfo
	for _, info := range params {
		if input.In != "" && !strings.EqualFold(info.In, input.In) {
			continue
		}
		if input.Name != "" && !strings.EqualFold(info.Name, input.Name) {
			continue
		}
		if input.Path != "" && !matchWalkPath(info.PathTemplate, input.Path) {
			continue
		}
		if input.Method != "" && !strings.EqualFold(info.Method, input.Method) {
			continue
		}
		if hasExtFilter && !matchExtension(info.Parameter.Extra, extKey, extValue) {
			continue
		}
		matched = append(matched, info)
	}
	return matched, nil
}

// parameterTypeString returns the parameter's type as a string.
// For OAS 3.0+ this comes from the schema; for OAS 2.0 from the type field.
// The Schema.Type field is `any` because it can be a string (OAS 3.0) or
// []string (OAS 3.1+), so we handle both via type assertion.
func parameterTypeString(param *parser.Parameter) string {
	if param.Schema != nil {
		switch t := param.Schema.Type.(type) {
		case string:
			return t
		case []string:
			return strings.Join(t, ", ")
		case []any:
			parts := make([]string, 0, len(t))
			for _, v := range t {
				parts = append(parts, fmt.Sprintf("%v", v))
			}
			return strings.Join(parts, ", ")
		default:
			if t != nil {
				return fmt.Sprintf("%v", t)
			}
			return ""
		}
	}
	return param.Type
}
