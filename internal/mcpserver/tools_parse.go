package mcpserver

import (
	"context"

	"github.com/erraggy/oastools/parser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type parseInput struct {
	Spec        specInput `json:"spec"                     jsonschema:"The OAS document to parse"`
	ResolveRefs bool      `json:"resolve_refs,omitempty"   jsonschema:"Resolve $ref pointers before returning"`
	Full        bool      `json:"full,omitempty"           jsonschema:"Return full parsed document instead of summary"`
}

type parseSummaryServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type parseOutput struct {
	Version        string               `json:"version"`
	Title          string               `json:"title"`
	Description    string               `json:"description,omitempty"`
	PathCount      int                  `json:"path_count"`
	OperationCount int                  `json:"operation_count"`
	SchemaCount    int                  `json:"schema_count"`
	Servers        []parseSummaryServer `json:"servers,omitempty"`
	Tags           []string             `json:"tags,omitempty"`
	Format         string               `json:"format"`
	FullDocument   string               `json:"full_document,omitempty"`
}

func handleParse(_ context.Context, _ *mcp.CallToolRequest, input parseInput) (*mcp.CallToolResult, parseOutput, error) {
	var extraOpts []parser.Option
	if input.ResolveRefs {
		extraOpts = append(extraOpts, parser.WithResolveRefs(true))
	}

	result, err := input.Spec.resolve(extraOpts...)
	if err != nil {
		return errResult(err), parseOutput{}, nil
	}

	output := parseOutput{
		Version:        result.Version,
		Format:         string(result.SourceFormat),
		PathCount:      result.Stats.PathCount,
		OperationCount: result.Stats.OperationCount,
		SchemaCount:    result.Stats.SchemaCount,
	}

	accessor := result.AsAccessor()
	if accessor != nil {
		if info := accessor.GetInfo(); info != nil {
			output.Title = info.Title
			output.Description = info.Description
		}
		for _, tag := range accessor.GetTags() {
			if tag != nil {
				output.Tags = append(output.Tags, tag.Name)
			}
		}
	}

	// Servers are OAS 3.x only
	if doc, ok := result.OAS3Document(); ok {
		for _, s := range doc.Servers {
			if s != nil {
				output.Servers = append(output.Servers, parseSummaryServer{
					URL:         s.URL,
					Description: s.Description,
				})
			}
		}
	}

	if input.Full {
		var data []byte
		switch result.SourceFormat {
		case parser.SourceFormatJSON:
			data, err = result.MarshalOrderedJSONIndent("", "  ")
		default:
			data, err = result.MarshalOrderedYAML()
		}
		if err != nil {
			return errResult(err), parseOutput{}, nil
		}
		output.FullDocument = string(data)
	}

	return nil, output, nil
}
