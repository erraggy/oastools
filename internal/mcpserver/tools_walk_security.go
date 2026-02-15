package mcpserver

import (
	"context"
	"strings"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type walkSecurityInput struct {
	Spec        specInput `json:"spec"                     jsonschema:"The OAS document to walk"`
	Name        string    `json:"name,omitempty"           jsonschema:"Filter by security scheme name"`
	Type        string    `json:"type,omitempty"           jsonschema:"Filter by security scheme type (apiKey\\, http\\, oauth2\\, openIdConnect)"`
	Extension   string    `json:"extension,omitempty"      jsonschema:"Filter by extension key=value (e.g. x-internal=true)"`
	ResolveRefs bool      `json:"resolve_refs,omitempty"   jsonschema:"Resolve $ref pointers in output. Inlines referenced objects instead of showing $ref strings."`
	Detail      bool      `json:"detail,omitempty"         jsonschema:"Return full security scheme objects instead of summaries"`
	Limit       int       `json:"limit,omitempty"          jsonschema:"Maximum number of results to return (default 100)"`
	Offset      int       `json:"offset,omitempty"         jsonschema:"Skip the first N results (for pagination)"`
}

type securitySummary struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	In          string `json:"in,omitempty"`
	Description string `json:"description,omitempty"`
}

type securityDetail struct {
	Name           string                 `json:"name"`
	SecurityScheme *parser.SecurityScheme `json:"security_scheme"`
}

type walkSecurityOutput struct {
	Total     int               `json:"total"`
	Matched   int               `json:"matched"`
	Returned  int               `json:"returned"`
	Summaries []securitySummary `json:"summaries,omitempty"`
	Schemes   []securityDetail  `json:"schemes,omitempty"`
}

func handleWalkSecurity(_ context.Context, _ *mcp.CallToolRequest, input walkSecurityInput) (*mcp.CallToolResult, any, error) {
	var extraOpts []parser.Option
	if input.ResolveRefs {
		extraOpts = append(extraOpts, parser.WithResolveRefs(true))
	}

	result, err := input.Spec.resolve(extraOpts...)
	if err != nil {
		return errResult(err), nil, nil
	}

	collector, err := walker.CollectSecuritySchemes(result)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Filter security schemes.
	matched, err := filterWalkSecurity(collector.All, input)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Apply offset/limit pagination.
	returned := paginate(matched, input.Offset, input.Limit)

	output := walkSecurityOutput{
		Total:    len(collector.All),
		Matched:  len(matched),
		Returned: len(returned),
	}

	if input.Detail {
		output.Schemes = makeSlice[securityDetail](len(returned))
		for _, info := range returned {
			output.Schemes = append(output.Schemes, securityDetail{
				Name:           info.Name,
				SecurityScheme: info.SecurityScheme,
			})
		}
	} else {
		output.Summaries = makeSlice[securitySummary](len(returned))
		for _, info := range returned {
			output.Summaries = append(output.Summaries, securitySummary{
				Name:        info.Name,
				Type:        info.SecurityScheme.Type,
				In:          info.SecurityScheme.In,
				Description: info.SecurityScheme.Description,
			})
		}
	}

	return nil, output, nil
}

// filterWalkSecurity applies all security scheme filters and returns the matching subset.
func filterWalkSecurity(schemes []*walker.SecuritySchemeInfo, input walkSecurityInput) ([]*walker.SecuritySchemeInfo, error) {
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

	var matched []*walker.SecuritySchemeInfo
	for _, info := range schemes {
		if input.Name != "" && !strings.EqualFold(info.Name, input.Name) {
			continue
		}
		if input.Type != "" && !strings.EqualFold(info.SecurityScheme.Type, input.Type) {
			continue
		}
		if hasExtFilter && !matchExtension(info.SecurityScheme.Extra, extKey, extValue) {
			continue
		}
		matched = append(matched, info)
	}
	return matched, nil
}
