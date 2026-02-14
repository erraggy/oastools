// Package mcpserver implements an MCP (Model Context Protocol) server
// that exposes oastools capabilities as MCP tools over stdio.
package mcpserver

import (
	"context"

	"github.com/erraggy/oastools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Run starts the MCP server over stdio and blocks until the client disconnects
// or the context is cancelled.
func Run(ctx context.Context) error {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "oastools", Version: oastools.Version()},
		nil,
	)
	registerAllTools(server)
	return server.Run(ctx, &mcp.StdioTransport{})
}

func registerAllTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "validate",
		Description: "Validate an OpenAPI Specification document against its version schema. Returns validation errors and warnings.",
	}, handleValidate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "parse",
		Description: "Parse an OpenAPI Specification document and return a summary of its structure (paths, schemas, servers, tags) or the full parsed document.",
	}, handleParse)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fix",
		Description: "Automatically fix common issues in an OpenAPI Specification document. Supports fixing missing path parameters, duplicate operationIds, generic schema names, pruning unused schemas/empty paths, and stubbing missing $ref targets.",
	}, handleFix)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "convert",
		Description: "Convert an OpenAPI Specification document between versions (2.0, 3.0, 3.1). Returns conversion issues and the converted document.",
	}, handleConvert)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "diff",
		Description: "Compare two OpenAPI Specification documents and report differences. Detects breaking changes, additions, removals, and modifications with severity levels.",
	}, handleDiff)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "join",
		Description: "Join multiple OpenAPI Specification documents into a single merged document. Requires at least 2 specs. Supports collision strategies for paths and schemas.",
	}, handleJoin)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "overlay_apply",
		Description: "Apply an Overlay document to an OpenAPI Specification. Overlays use JSONPath expressions to update or remove parts of the spec. Supports dry-run preview and file output.",
	}, handleOverlayApply)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "overlay_validate",
		Description: "Validate an Overlay document structure. Checks required fields, supported version, valid JSONPath syntax in action targets, and that actions have update or remove operations.",
	}, handleOverlayValidate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "generate",
		Description: "Generate Go code (types, client, server) from an OpenAPI Specification document. Writes generated files to the specified output directory and returns a manifest of what was generated.",
	}, handleGenerate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "walk_operations",
		Description: "Walk and query operations in an OpenAPI Specification document. Filter by method, path, tag, operationId, deprecated status, or extension. Returns summaries by default or full operation objects with detail mode.",
	}, handleWalkOperations)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "walk_schemas",
		Description: "Walk and query schemas in an OpenAPI Specification document. Filter by name, type, component/inline location, or extension. Returns summaries by default or full schema objects with detail mode.",
	}, handleWalkSchemas)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "walk_parameters",
		Description: "Walk and query parameters in an OpenAPI Specification document. Filter by location (query/header/path/cookie), name, path, method, or extension. Returns summaries by default or full parameter objects with detail mode.",
	}, handleWalkParameters)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "walk_responses",
		Description: "Walk and query responses in an OpenAPI Specification document. Filter by status code (200, 4xx, default), path, method, or extension. Returns summaries by default or full response objects with detail mode.",
	}, handleWalkResponses)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "walk_security",
		Description: "Walk and query security schemes in an OpenAPI Specification document. Filter by name, type (apiKey/http/oauth2/openIdConnect), or extension. Returns summaries by default or full security scheme objects with detail mode.",
	}, handleWalkSecurity)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "walk_paths",
		Description: "Walk and query path items in an OpenAPI Specification document. Filter by path pattern (supports * glob) or extension. Returns summaries with method counts by default or full path item objects with detail mode.",
	}, handleWalkPaths)
}

// makeSlice returns nil when n is 0 (preserving omitempty JSON semantics),
// otherwise returns make([]T, 0, n) for pre-allocated appending.
func makeSlice[T any](n int) []T {
	if n == 0 {
		return nil
	}
	return make([]T, 0, n)
}

// errResult creates an MCP error result from an error.
func errResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}
}
