// Package mcpserver implements an MCP (Model Context Protocol) server
// that exposes oastools capabilities as MCP tools over stdio.
package mcpserver

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/erraggy/oastools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverInstructions = `oastools MCP server — validates, fixes, converts, diffs, joins, walks, and generates OpenAPI specs.

Configuration: All defaults are configurable via OASTOOLS_* environment variables set in your MCP client config. The Go MCP SDK does not support initializationOptions; use env vars instead.

Key settings:
- OASTOOLS_CACHE_FILE_TTL (default: 15m) — cache TTL for local file specs
- OASTOOLS_CACHE_URL_TTL (default: 5m) — cache TTL for URL-fetched specs
- OASTOOLS_CACHE_ENABLED (default: true) — disable spec caching entirely
- OASTOOLS_WALK_LIMIT (default: 100) — default result limit for walk tools
- OASTOOLS_WALK_DETAIL_LIMIT (default: 25) — default limit in detail mode
- OASTOOLS_VALIDATE_STRICT (default: false) — enable strict validation by default
- OASTOOLS_VALIDATE_NO_WARNINGS (default: false) — suppress warnings by default
- OASTOOLS_JOIN_PATH_STRATEGY — default path collision strategy for join
- OASTOOLS_JOIN_SCHEMA_STRATEGY — default schema collision strategy for join

Caching: Parsed specs are cached per session. File entries use path+mtime as key (auto-invalidated on change). URL entries are cached with a shorter TTL. A background sweeper removes expired entries every 60s.`

// Run starts the MCP server over stdio and blocks until the client disconnects
// or the context is cancelled.
func Run(ctx context.Context) error {
	if cfg.CacheEnabled {
		specCache.startSweeper(ctx, cfg.CacheSweepInterval)
	}

	server := mcp.NewServer(
		&mcp.Implementation{Name: "oastools", Version: oastools.Version()},
		&mcp.ServerOptions{
			Instructions: serverInstructions,
		},
	)
	registerAllTools(server)
	return server.Run(ctx, &mcp.StdioTransport{})
}

func registerAllTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "validate",
		Description: "Validate an OpenAPI Specification document against its version schema. Returns errors and warnings with JSON path locations. For large specs, use no_warnings to focus on errors first. Use offset/limit to paginate through results. Strict mode and warning suppression defaults are configurable via OASTOOLS_VALIDATE_STRICT and OASTOOLS_VALIDATE_NO_WARNINGS env vars.",
	}, handleValidate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "parse",
		Description: "Parse an OpenAPI Specification document. Returns a structural summary: title, version, OAS version, path/operation/schema counts, servers, and tags. Use full=true only for small specs; for large specs use walk_* tools to explore specific sections.",
	}, handleParse)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fix",
		Description: "Automatically fix common issues in an OpenAPI Specification document. Fix types: generic schema names, duplicate operationIds, missing path parameters, unused schemas/empty paths (prune), missing $ref targets (stub). Use dry_run=true to preview fixes before applying. Use output to write to a file instead of returning inline.",
	}, handleFix)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "convert",
		Description: "Convert an OpenAPI Specification document between versions (2.0, 3.0, 3.1). Returns conversion issues and the converted document.",
	}, handleConvert)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "diff",
		Description: "Compare two versions of the same OpenAPI Specification document and report differences. Detects breaking changes, additions, removals, and modifications with severity levels. Use breaking_only=true to focus on breaking changes first. Both base and revision must be provided.",
	}, handleDiff)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "join",
		Description: "Join multiple OpenAPI Specification documents into a single merged document. Requires at least 2 specs via the specs array. Collision strategies: accept-left, accept-right, fail (paths/schemas), rename (schemas only). Use semantic_dedup to merge equivalent schemas. Default collision strategies are configurable via OASTOOLS_JOIN_PATH_STRATEGY and OASTOOLS_JOIN_SCHEMA_STRATEGY env vars.",
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
		Description: "Generate Go code from an OpenAPI Specification document. Set exactly one of: types (type definitions only), client (HTTP client), or server (server interfaces and handlers). Requires output_dir. Returns a manifest of generated files.",
	}, handleGenerate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "walk_operations",
		Description: "Walk and query operations in an OpenAPI Specification document. Filter by method, path, tag, operationId, deprecated status, or extension. Returns summaries (method, path, operationId, tags) by default or full operation objects with detail=true. For large APIs, filter by tag first (most selective), then narrow with path or method. Path patterns support * (one segment) and ** (zero or more segments). Use group_by (tag or method) to get distribution counts instead of individual items. Default limit is configurable via OASTOOLS_WALK_LIMIT (default 100, 25 in detail mode).",
	}, handleWalkOperations)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "walk_schemas",
		Description: "Walk and query schemas in an OpenAPI Specification document. Filter by name, type, component/inline location, or extension. Returns summaries (name, type, JSON path, component status) by default or full schema objects with detail=true. Use component=true to see only named component schemas (skips inline schemas, reducing results 3-5x). Avoid detail=true without filters on large specs. Use group_by (type or location) to get distribution counts instead of individual items.",
	}, handleWalkSchemas)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "walk_parameters",
		Description: "Walk and query parameters in an OpenAPI Specification document. Filter by location (in), name, path pattern, method, or extension. Returns summaries (name, location, path, method) by default or full parameter objects with detail=true. Use group_by (location or name) to get distribution counts instead of individual items.",
	}, handleWalkParameters)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "walk_responses",
		Description: "Walk and query responses in an OpenAPI Specification document. Filter by status code, path pattern, method, or extension. Returns summaries (status code, path, method, description) by default or full response objects with detail=true. Use group_by (status_code or method) to get distribution counts instead of individual items.",
	}, handleWalkResponses)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "walk_security",
		Description: "Walk and query security schemes defined in components. Filter by name or type (apiKey, http, oauth2, openIdConnect). Returns summaries (name, type, location) by default or full security scheme objects with detail=true.",
	}, handleWalkSecurity)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "walk_paths",
		Description: "Walk and query path items in an OpenAPI Specification document. Filter by path pattern or extension. Returns summaries (path, method count) by default or full path item objects with detail=true. Path patterns support * (one segment) and ** (zero or more segments), e.g. /users/** matches all paths under /users. Use group_by=segment to group paths by their first URL segment.",
	}, handleWalkPaths)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "walk_refs",
		Description: "Walk and count $ref references in an OpenAPI Specification document. By default, returns unique ref targets ranked by reference count (most-referenced first). Use target to filter to a specific ref (supports * glob, e.g. *schemas/microsoft.graph.*). Use detail=true to see individual source locations instead of counts. Filter by node_type to narrow to schema, parameter, response, requestBody, header, or pathItem refs. Use group_by=node_type to get distribution counts by ref type.",
	}, handleWalkRefs)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "walk_headers",
		Description: "Walk and query response headers and component headers in an OpenAPI Specification document. Filter by name, path, method, status code, or component location. Returns summaries (name, path, method, status, description) by default or full header objects with detail=true. Use group_by=name to find the most commonly used headers across the API.",
	}, handleWalkHeaders)
}

// paginate applies offset/limit pagination to a slice, returning the
// requested page. A non-positive limit defaults to cfg.WalkLimit.
func paginate[T any](items []T, offset, limit int) []T {
	if limit <= 0 {
		limit = cfg.WalkLimit
	}
	if limit > cfg.MaxLimit {
		limit = cfg.MaxLimit
	}
	if offset < 0 || offset >= len(items) {
		return nil
	}
	end := offset + limit
	if end < offset || end > len(items) { // overflow or beyond slice
		end = len(items)
	}
	return items[offset:end]
}

// detailLimit returns a lower default limit for detail mode output.
// When the user hasn't specified an explicit limit (limit <= 0),
// detail mode defaults to cfg.WalkDetailLimit to keep output manageable.
func detailLimit(limit int) int {
	if limit <= 0 {
		return cfg.WalkDetailLimit
	}
	return limit
}

// makeSlice returns nil when n is 0 (preserving omitempty JSON semantics),
// otherwise returns make([]T, 0, n) for pre-allocated appending.
func makeSlice[T any](n int) []T {
	if n == 0 {
		return nil
	}
	return make([]T, 0, n)
}

// sanitizeError strips absolute filesystem paths from error messages
// to prevent leaking internal directory structure to MCP clients.
var pathPattern = regexp.MustCompile(`(?:/(?:home|tmp|var|Users|etc|opt|usr|private|root|mnt|srv|run|snap|nix)[a-zA-Z0-9._/-]*)`)

func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	return pathPattern.ReplaceAllString(err.Error(), "<path>")
}

// errResult creates an MCP error result from an error.
func errResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: sanitizeError(err)}},
	}
}

// groupCount represents a single group in group_by results.
type groupCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

// groupAndSort groups items by key, sorts by count descending (ties
// broken alphabetically by key), and returns the sorted groups.
func groupAndSort[T any](items []T, keyFn func(T) []string) []groupCount {
	counts := make(map[string]int)
	for _, item := range items {
		for _, key := range keyFn(item) {
			counts[key]++
		}
	}
	groups := make([]groupCount, 0, len(counts))
	for key, count := range counts {
		groups = append(groups, groupCount{Key: key, Count: count})
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Count != groups[j].Count {
			return groups[i].Count > groups[j].Count
		}
		return groups[i].Key < groups[j].Key
	})
	return groups
}

// validateGroupBy checks that group_by is a valid value and is not combined with detail.
func validateGroupBy(groupBy string, detail bool, allowed []string) error {
	if groupBy == "" {
		return nil
	}
	if detail {
		return fmt.Errorf("cannot use both group_by and detail")
	}
	for _, a := range allowed {
		if strings.EqualFold(groupBy, a) {
			return nil
		}
	}
	return fmt.Errorf("invalid group_by value %q; valid values: %s", groupBy, strings.Join(allowed, ", "))
}

// validateGlobPattern checks whether a glob pattern is syntactically valid.
// Call this once before a filter loop so matchGlobName/matchRefGlob never
// encounter an invalid pattern at match time.
func validateGlobPattern(pattern string) error {
	if pattern == "" || !strings.ContainsAny(pattern, "*?[") {
		return nil
	}
	if _, err := filepath.Match(pattern, ""); err != nil {
		return fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}
	return nil
}
