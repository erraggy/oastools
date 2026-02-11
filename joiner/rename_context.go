package joiner

import (
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/template"

	"github.com/erraggy/oastools/internal/maputil"
	"github.com/erraggy/oastools/internal/naming"
)

// UsageType indicates where a schema is referenced within an operation.
type UsageType string

const (
	// UsageTypeRequest indicates the schema is used in a request body.
	UsageTypeRequest UsageType = "request"
	// UsageTypeResponse indicates the schema is used in a response body.
	UsageTypeResponse UsageType = "response"
	// UsageTypeParameter indicates the schema is used in a parameter.
	UsageTypeParameter UsageType = "parameter"
	// UsageTypeHeader indicates the schema is used in a header.
	UsageTypeHeader UsageType = "header"
	// UsageTypeCallback indicates the schema is used in a callback.
	UsageTypeCallback UsageType = "callback"
)

// OperationRef represents a direct reference from an operation to a schema.
type OperationRef struct {
	Path        string    // API path: "/users/{id}"
	Method      string    // HTTP method: "get", "post", "delete"
	OperationID string    // Operation identifier if defined
	Tags        []string  // Tags associated with the operation
	UsageType   UsageType // Where the schema is referenced
	StatusCode  string    // Response status code (for response usage)
	ParamName   string    // Parameter name (for parameter usage)
	MediaType   string    // Media type: "application/json"
}

// RefGraph represents the complete reference structure of an OpenAPI document.
// It enables traversal from any schema to the operations that reference it,
// either directly or through intermediate schemas.
//
// The full implementation is in refgraph.go.
type RefGraph struct {
	// schemaRefs maps schema names to their direct references (schemas that reference them)
	schemaRefs map[string][]SchemaRef

	// operationRefs maps schema names to direct operation references
	operationRefs map[string][]OperationRef

	// resolved caches the fully resolved operation lineage for each schema
	resolved map[string][]OperationRef
}

// SchemaRef represents a reference from one schema to another.
type SchemaRef struct {
	FromSchema  string // The schema containing the $ref
	RefLocation string // Where in the schema: "properties.address", "items", "allOf[0]"
}

// ResolveLineage returns all operations that reference the given schema,
// either directly or through intermediate schemas.
func (g *RefGraph) ResolveLineage(schemaName string) []OperationRef {
	if g == nil {
		return nil
	}
	if cached, ok := g.resolved[schemaName]; ok {
		return cached
	}

	visited := make(map[string]bool)
	var lineage []OperationRef
	g.resolveLineageRecursive(schemaName, visited, &lineage)

	if g.resolved == nil {
		g.resolved = make(map[string][]OperationRef)
	}
	g.resolved[schemaName] = lineage
	return lineage
}

func (g *RefGraph) resolveLineageRecursive(schemaName string, visited map[string]bool, lineage *[]OperationRef) {
	if visited[schemaName] {
		return // Cycle detected, stop recursion
	}
	visited[schemaName] = true

	// Add direct operation references
	if ops, ok := g.operationRefs[schemaName]; ok {
		*lineage = append(*lineage, ops...)
	}

	// Traverse to parent schemas that reference this schema
	if refs, ok := g.schemaRefs[schemaName]; ok {
		for _, ref := range refs {
			g.resolveLineageRecursive(ref.FromSchema, visited, lineage)
		}
	}
}

// RenameContext provides comprehensive context for schema renaming decisions.
// It extends the original renameTemplateData with operation-derived fields.
type RenameContext struct {
	// Core fields (backward compatible with renameTemplateData)
	Name   string // Original schema name
	Source string // Source file name (sanitized)
	Index  int    // Document index (0-based)

	// Operation context (from primary operation reference)
	Path        string    // API path: "/users/{id}"
	Method      string    // HTTP method: "get", "post"
	OperationID string    // Operation ID if defined
	Tags        []string  // Tags from primary operation
	UsageType   UsageType // request, response, parameter, header, callback
	StatusCode  string    // Response status code
	ParamName   string    // Parameter name
	MediaType   string    // Media type

	// Aggregate context (when schema has multiple operation references)
	AllPaths        []string // All referencing paths
	AllMethods      []string // All methods (deduplicated)
	AllOperationIDs []string // All operation IDs (non-empty only)
	AllTags         []string // All tags (deduplicated)
	RefCount        int      // Total operation references
	PrimaryResource string   // Extracted resource name from path
	IsShared        bool     // True if referenced by multiple operations
}

// PrimaryOperationPolicy determines which operation provides primary context
// when a schema is referenced by multiple operations.
type PrimaryOperationPolicy int

const (
	// PolicyFirstEncountered uses the first operation found during traversal.
	PolicyFirstEncountered PrimaryOperationPolicy = iota

	// PolicyMostSpecific prefers operations with operationId, then tags.
	PolicyMostSpecific

	// PolicyAlphabetical sorts by path+method and uses alphabetically first.
	PolicyAlphabetical
)

// renameFuncs returns template functions for rename templates.
func renameFuncs() template.FuncMap {
	return template.FuncMap{
		// Path functions
		"pathSegment":  pathSegment,
		"pathResource": pathResource,
		"pathLast":     pathLast,
		"pathClean":    pathClean,

		// Tag functions
		"firstTag": firstTag,
		"joinTags": joinTags,
		"hasTag":   hasTag,

		// Case functions (from internal/naming)
		"pascalCase": naming.ToPascalCase,
		"camelCase":  naming.ToCamelCase,
		"snakeCase":  naming.ToSnakeCase,
		"kebabCase":  naming.ToKebabCase,

		// Conditional helpers
		"default":  defaultValue,
		"coalesce": coalesce,
	}
}

// pathSegment extracts the nth path segment (0-indexed), excluding path parameters.
// Negative indices count from the end.
// Example: pathSegment("/users/{id}/orders", 0) -> "users"
// Example: pathSegment("/users/{id}/orders", -1) -> "orders"
func pathSegment(path string, index int) string {
	segments := extractPathSegments(path)
	if len(segments) == 0 {
		return ""
	}
	if index < 0 {
		index = len(segments) + index
	}
	if index < 0 || index >= len(segments) {
		return ""
	}
	return segments[index]
}

// pathResource extracts the primary resource name from a path.
// Returns the first non-parameter segment.
// Example: pathResource("/users/{id}/orders") -> "users"
func pathResource(path string) string {
	segments := extractPathSegments(path)
	if len(segments) == 0 {
		return ""
	}
	return segments[0]
}

// pathLast extracts the last non-parameter segment from a path.
// Example: pathLast("/users/{id}/orders") -> "orders"
func pathLast(path string) string {
	segments := extractPathSegments(path)
	if len(segments) == 0 {
		return ""
	}
	return segments[len(segments)-1]
}

// pathClean sanitizes a path for use in naming.
// Removes slashes, replaces parameters with underscores.
// Example: pathClean("/users/{id}") -> "users_id"
func pathClean(path string) string {
	// Trim leading/trailing slashes
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}

	// Split by slashes
	parts := strings.Split(path, "/")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		// Remove braces from path parameters
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			part = strings.TrimPrefix(part, "{")
			part = strings.TrimSuffix(part, "}")
		}
		// Replace invalid characters
		part = strings.ReplaceAll(part, "-", "_")
		part = strings.ReplaceAll(part, ".", "_")
		result = append(result, part)
	}

	return strings.Join(result, "_")
}

// firstTag returns the first tag or empty string if none.
func firstTag(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	return tags[0]
}

// joinTags joins tags with a separator.
func joinTags(tags []string, sep string) string {
	return strings.Join(tags, sep)
}

// hasTag checks if a tag is present.
func hasTag(tags []string, tag string) bool {
	return slices.Contains(tags, tag)
}

// defaultValue returns the fallback if value is empty.
func defaultValue(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// coalesce returns the first non-empty string.
func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// buildRenameContext creates a RenameContext from schema info and reference graph.
// If graph is nil or has no lineage for the schema, only core fields are populated.
func buildRenameContext(
	schemaName string,
	sourcePath string,
	docIndex int,
	graph *RefGraph,
	policy PrimaryOperationPolicy,
) RenameContext {
	ctx := RenameContext{
		Name:   schemaName,
		Source: sanitizeSourcePath(sourcePath),
		Index:  docIndex,
	}

	// If no graph, return core context only
	if graph == nil {
		return ctx
	}

	// Get operation references for this schema
	refs := graph.ResolveLineage(schemaName)
	if len(refs) == 0 {
		return ctx
	}

	// Populate aggregate fields
	ctx.RefCount = len(refs)
	ctx.IsShared = len(refs) > 1

	pathSet := make(map[string]bool)
	methodSet := make(map[string]bool)
	tagSet := make(map[string]bool)

	for _, ref := range refs {
		pathSet[ref.Path] = true
		methodSet[ref.Method] = true
		if ref.OperationID != "" {
			ctx.AllOperationIDs = append(ctx.AllOperationIDs, ref.OperationID)
		}
		for _, tag := range ref.Tags {
			tagSet[tag] = true
		}
	}

	ctx.AllPaths = maputil.SortedKeys(pathSet)
	ctx.AllMethods = maputil.SortedKeys(methodSet)
	ctx.AllTags = maputil.SortedKeys(tagSet)

	// Select primary operation based on policy
	primary := selectPrimaryOperation(refs, policy)
	ctx.Path = primary.Path
	ctx.Method = primary.Method
	ctx.OperationID = primary.OperationID
	ctx.Tags = primary.Tags
	ctx.UsageType = primary.UsageType
	ctx.StatusCode = primary.StatusCode
	ctx.ParamName = primary.ParamName
	ctx.MediaType = primary.MediaType

	// Derive primary resource from path
	ctx.PrimaryResource = pathResource(primary.Path)

	return ctx
}

// selectPrimaryOperation selects the primary operation based on policy.
func selectPrimaryOperation(refs []OperationRef, policy PrimaryOperationPolicy) OperationRef {
	if len(refs) == 0 {
		return OperationRef{}
	}

	switch policy {
	case PolicyFirstEncountered:
		return refs[0]

	case PolicyAlphabetical:
		// Sort by path+method and return first
		sorted := make([]OperationRef, len(refs))
		copy(sorted, refs)
		sort.Slice(sorted, func(i, j int) bool {
			keyI := sorted[i].Path + sorted[i].Method
			keyJ := sorted[j].Path + sorted[j].Method
			return keyI < keyJ
		})
		return sorted[0]

	case PolicyMostSpecific:
		// Prefer refs with operationId
		for _, ref := range refs {
			if ref.OperationID != "" {
				return ref
			}
		}
		// Then prefer refs with tags
		for _, ref := range refs {
			if len(ref.Tags) > 0 {
				return ref
			}
		}
		// Fall back to first (already checked len > 0 above)
		fallthrough

	default:
		return refs[0]
	}
}

// sanitizeSourcePath extracts and cleans the filename from a path.
// Removes directory, extension, and invalid characters.
func sanitizeSourcePath(path string) string {
	if path == "" {
		return ""
	}

	// Extract base filename
	base := filepath.Base(path)

	// Remove extension
	ext := filepath.Ext(base)
	if ext != "" {
		base = strings.TrimSuffix(base, ext)
	}

	// Replace invalid characters for schema naming
	base = strings.ReplaceAll(base, "-", "_")
	base = strings.ReplaceAll(base, " ", "_")
	base = strings.ReplaceAll(base, ".", "_")

	return base
}

// extractPathSegments returns path segments excluding parameters.
func extractPathSegments(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var segments []string
	for _, p := range parts {
		if p != "" && !strings.HasPrefix(p, "{") {
			segments = append(segments, p)
		}
	}
	return segments
}

// buildRenameContextPtr is like buildRenameContext but returns a pointer.
// Returns nil if the context would have no useful information.
func buildRenameContextPtr(
	schemaName string,
	sourcePath string,
	docIndex int,
	graph *RefGraph,
	policy PrimaryOperationPolicy,
) *RenameContext {
	ctx := buildRenameContext(schemaName, sourcePath, docIndex, graph, policy)
	return &ctx
}
