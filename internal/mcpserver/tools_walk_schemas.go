package mcpserver

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type walkSchemasInput struct {
	Spec        specInput `json:"spec"                     jsonschema:"The OAS document to walk"`
	Name        string    `json:"name,omitempty"           jsonschema:"Filter by schema name (exact match\\, or glob with * and ? for pattern matching\\, e.g. *workbook* or microsoft.graph.*)"`
	Type        string    `json:"type,omitempty"           jsonschema:"Filter by schema type (object\\, array\\, string\\, integer\\, etc.)"`
	Component   bool      `json:"component,omitempty"      jsonschema:"Only show component schemas (defined in components/schemas or definitions). Mutually exclusive with inline."`
	Inline      bool      `json:"inline,omitempty"         jsonschema:"Only show inline schemas (embedded in operations\\, not in components). Mutually exclusive with component."`
	Extension   string    `json:"extension,omitempty"      jsonschema:"Filter by extension key=value (e.g. x-internal=true)"`
	ResolveRefs bool      `json:"resolve_refs,omitempty"   jsonschema:"Resolve $ref pointers in output. Inlines referenced objects instead of showing $ref strings."`
	Detail      bool      `json:"detail,omitempty"         jsonschema:"Return full schema objects. WARNING: produces large output without name/type filters on big specs."`
	GroupBy     string    `json:"group_by,omitempty"       jsonschema:"Group results and return counts instead of individual items. Values: type\\, location"`
	Limit       int       `json:"limit,omitempty"          jsonschema:"Maximum results (default 100)"`
	Offset      int       `json:"offset,omitempty"         jsonschema:"Skip the first N results (for pagination)"`
}

type schemaSummary struct {
	Name          string   `json:"name"`
	Type          string   `json:"type,omitempty"`
	Location      string   `json:"location"`
	PropertyCount int      `json:"property_count"`
	Required      []string `json:"required,omitempty"`
}

type schemaDetail struct {
	Name        string         `json:"name"`
	JSONPath    string         `json:"json_path"`
	IsComponent bool           `json:"is_component"`
	Schema      *parser.Schema `json:"schema"`
}

type walkSchemasOutput struct {
	Total     int             `json:"total"`
	Matched   int             `json:"matched"`
	Returned  int             `json:"returned"`
	Summaries []schemaSummary `json:"summaries,omitempty"`
	Schemas   []schemaDetail  `json:"schemas,omitempty"`
	Groups    []groupCount    `json:"groups,omitempty"`
}

func handleWalkSchemas(_ context.Context, _ *mcp.CallToolRequest, input walkSchemasInput) (*mcp.CallToolResult, any, error) {
	if input.Component && input.Inline {
		return errResult(fmt.Errorf("cannot use both component and inline filters")), nil, nil
	}

	if err := validateGroupBy(input.GroupBy, input.Detail, []string{"type", "location"}); err != nil {
		return errResult(err), nil, nil
	}

	var extraOpts []parser.Option
	if input.ResolveRefs {
		extraOpts = append(extraOpts, parser.WithResolveRefs(true))
	}

	result, err := input.Spec.resolve(extraOpts...)
	if err != nil {
		return errResult(err), nil, nil
	}

	collector, err := walker.CollectSchemas(result)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Choose base set based on component/inline filter.
	schemas := collector.All
	if input.Component {
		schemas = collector.Components
	} else if input.Inline {
		schemas = collector.Inline
	}

	// Apply additional filters.
	filtered, err := filterWalkSchemas(schemas, input)
	if err != nil {
		return errResult(err), nil, nil
	}

	// group_by: aggregate matched schemas and return counts.
	if input.GroupBy != "" {
		groups := groupAndSort(filtered, func(info *walker.SchemaInfo) []string {
			switch strings.ToLower(input.GroupBy) {
			case "type":
				t := schemaTypeString(info.Schema.Type)
				if t == "" {
					return []string{""}
				}
				return []string{t}
			case "location":
				return []string{schemaLocation(info.IsComponent)}
			default:
				return nil
			}
		})
		paged := paginate(groups, input.Offset, input.Limit)
		output := walkSchemasOutput{
			Total:    len(collector.All),
			Matched:  len(filtered),
			Returned: len(paged),
			Groups:   paged,
		}
		return nil, output, nil
	}

	// Apply offset/limit pagination.
	returned := paginate(filtered, input.Offset, input.Limit)

	output := walkSchemasOutput{
		Total:    len(collector.All),
		Matched:  len(filtered),
		Returned: len(returned),
	}

	if input.Detail {
		output.Schemas = makeSlice[schemaDetail](len(returned))
		for _, info := range returned {
			output.Schemas = append(output.Schemas, schemaDetail{
				Name:        schemaDisplayName(info),
				JSONPath:    info.JSONPath,
				IsComponent: info.IsComponent,
				Schema:      info.Schema,
			})
		}
	} else {
		output.Summaries = makeSlice[schemaSummary](len(returned))
		for _, info := range returned {
			output.Summaries = append(output.Summaries, schemaSummary{
				Name:          schemaDisplayName(info),
				Type:          schemaTypeString(info.Schema.Type),
				Location:      schemaLocation(info.IsComponent),
				PropertyCount: len(info.Schema.Properties),
				Required:      info.Schema.Required,
			})
		}
	}

	return nil, output, nil
}

// filterWalkSchemas applies name, type, and extension filters.
func filterWalkSchemas(schemas []*walker.SchemaInfo, input walkSchemasInput) ([]*walker.SchemaInfo, error) {
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

	var filtered []*walker.SchemaInfo
	for _, info := range schemas {
		if input.Name != "" && !matchGlobName(info.Name, input.Name) {
			continue
		}
		if input.Type != "" && !schemaTypeMatches(info.Schema.Type, input.Type) {
			continue
		}
		if hasExtFilter && !matchExtension(info.Schema.Extra, extKey, extValue) {
			continue
		}
		filtered = append(filtered, info)
	}
	return filtered, nil
}

// schemaTypeMatches checks if a schema's Type field matches the given filter string.
// The Type field is `any` because it can be a string (OAS 3.0) or []string (OAS 3.1+).
func schemaTypeMatches(schemaType any, filter string) bool {
	switch t := schemaType.(type) {
	case string:
		return strings.EqualFold(t, filter)
	case []string:
		for _, s := range t {
			if strings.EqualFold(s, filter) {
				return true
			}
		}
	case []any:
		for _, s := range t {
			if str, ok := s.(string); ok && strings.EqualFold(str, filter) {
				return true
			}
		}
	}
	return false
}

// schemaTypeString returns a display string for a schema's type field.
// Handles OAS 3.1+ where type can be string, []string, or []any.
func schemaTypeString(t any) string {
	switch v := t.(type) {
	case nil:
		return ""
	case string:
		return v
	case []string:
		return strings.Join(v, ", ")
	case []any:
		parts := make([]string, 0, len(v))
		for _, s := range v {
			parts = append(parts, fmt.Sprintf("%v", s))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", t)
	}
}

// schemaDisplayName returns a display name for a schema.
// For component schemas, this is the component name.
// For inline schemas, this falls back to the JSON path.
func schemaDisplayName(info *walker.SchemaInfo) string {
	if info.Name != "" {
		return info.Name
	}
	return info.JSONPath
}

// schemaLocation returns "component" or "inline" based on the IsComponent flag.
func schemaLocation(isComponent bool) string {
	if isComponent {
		return "component"
	}
	return "inline"
}

// matchGlobName matches a name against a pattern. If the pattern contains
// glob characters (* or ?), it uses case-insensitive filepath.Match.
// Otherwise, it falls back to case-insensitive exact match.
func matchGlobName(name, pattern string) bool {
	if strings.ContainsAny(pattern, "*?") {
		matched, err := filepath.Match(strings.ToLower(pattern), strings.ToLower(name))
		return err == nil && matched
	}
	return strings.EqualFold(name, pattern)
}
