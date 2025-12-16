// prune.go implements shared pruning logic for removing orphaned content
// from OpenAPI documents. These helpers are version-agnostic and used by
// both OAS 2.0 and OAS 3.x implementations.
package fixer

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// buildReferencedSchemaSet builds the transitive closure of referenced schemas.
// Starting from refs collected by RefCollector, it follows schema-to-schema references
// to ensure schemas that are indirectly referenced are not pruned.
//
// Example: If operation refs A, and A refs B, and B refs C, all three are "referenced".
func buildReferencedSchemaSet(collector *RefCollector, schemas map[string]*parser.Schema, version parser.OASVersion) map[string]bool {
	referenced := make(map[string]bool)
	queue := make([]string, 0)

	// Determine the appropriate prefix based on version
	prefix := schemaRefPrefix(version)

	// 1. Get directly referenced schemas from collector
	for ref := range collector.RefsByType[RefTypeSchema] {
		name := extractSchemaName(ref, prefix)
		if name == "" {
			continue
		}
		if _, exists := schemas[name]; exists && !referenced[name] {
			referenced[name] = true
			queue = append(queue, name)
		}
	}

	// 2. Process transitive references (BFS)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		schema := schemas[name]
		if schema == nil {
			continue
		}

		// Find refs in this schema
		schemaRefs := collectSchemaRefs(schema, prefix)
		for _, refName := range schemaRefs {
			if _, exists := schemas[refName]; exists && !referenced[refName] {
				referenced[refName] = true
				queue = append(queue, refName)
			}
		}
	}

	return referenced
}

// schemaRefPrefix returns the reference prefix for schemas based on OAS version.
func schemaRefPrefix(version parser.OASVersion) string {
	if version == parser.OASVersion20 {
		return "#/definitions/"
	}
	return "#/components/schemas/"
}

// extractSchemaName extracts the schema name from a reference path.
// Handles both URL-encoded and non-encoded refs.
func extractSchemaName(ref, prefix string) string {
	// Try direct prefix match first
	if strings.HasPrefix(ref, prefix) {
		return strings.TrimPrefix(ref, prefix)
	}

	// Try URL-decoded version
	decoded, err := url.PathUnescape(ref)
	if err == nil && strings.HasPrefix(decoded, prefix) {
		return strings.TrimPrefix(decoded, prefix)
	}

	return ""
}

// collectSchemaRefs extracts all schema reference names from a schema.
// This is used to find transitive references (schemas referencing other schemas).
// prefix should be "#/definitions/" for OAS 2.0 or "#/components/schemas/" for OAS 3.x
func collectSchemaRefs(schema *parser.Schema, prefix string) []string {
	visited := make(map[*parser.Schema]bool)
	return collectSchemaRefsRecursive(schema, prefix, visited)
}

// collectSchemaRefsRecursive is the internal implementation with circular reference protection.
func collectSchemaRefsRecursive(schema *parser.Schema, prefix string, visited map[*parser.Schema]bool) []string {
	if schema == nil || visited[schema] {
		return nil
	}
	visited[schema] = true

	var refs []string

	// Direct schema ref
	if name := extractSchemaName(schema.Ref, prefix); name != "" {
		refs = append(refs, name)
	}

	// Properties
	for _, propSchema := range schema.Properties {
		refs = append(refs, collectSchemaRefsRecursive(propSchema, prefix, visited)...)
	}

	// AdditionalProperties (can be *Schema or bool)
	if schema.AdditionalProperties != nil {
		if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
			refs = append(refs, collectSchemaRefsRecursive(addProps, prefix, visited)...)
		}
	}

	// Items (can be *Schema or bool in OAS 3.1+)
	if schema.Items != nil {
		if items, ok := schema.Items.(*parser.Schema); ok {
			refs = append(refs, collectSchemaRefsRecursive(items, prefix, visited)...)
		}
	}

	// AdditionalItems (can be *Schema or bool)
	if schema.AdditionalItems != nil {
		if addItems, ok := schema.AdditionalItems.(*parser.Schema); ok {
			refs = append(refs, collectSchemaRefsRecursive(addItems, prefix, visited)...)
		}
	}

	// Schema composition
	for _, s := range schema.AllOf {
		refs = append(refs, collectSchemaRefsRecursive(s, prefix, visited)...)
	}
	for _, s := range schema.AnyOf {
		refs = append(refs, collectSchemaRefsRecursive(s, prefix, visited)...)
	}
	for _, s := range schema.OneOf {
		refs = append(refs, collectSchemaRefsRecursive(s, prefix, visited)...)
	}
	if schema.Not != nil {
		refs = append(refs, collectSchemaRefsRecursive(schema.Not, prefix, visited)...)
	}

	// OAS 3.1+ / JSON Schema Draft 2020-12 fields
	for _, s := range schema.PrefixItems {
		refs = append(refs, collectSchemaRefsRecursive(s, prefix, visited)...)
	}
	if schema.Contains != nil {
		refs = append(refs, collectSchemaRefsRecursive(schema.Contains, prefix, visited)...)
	}
	if schema.PropertyNames != nil {
		refs = append(refs, collectSchemaRefsRecursive(schema.PropertyNames, prefix, visited)...)
	}
	for _, depSchema := range schema.DependentSchemas {
		refs = append(refs, collectSchemaRefsRecursive(depSchema, prefix, visited)...)
	}

	// Conditional schemas (OAS 3.1+)
	if schema.If != nil {
		refs = append(refs, collectSchemaRefsRecursive(schema.If, prefix, visited)...)
	}
	if schema.Then != nil {
		refs = append(refs, collectSchemaRefsRecursive(schema.Then, prefix, visited)...)
	}
	if schema.Else != nil {
		refs = append(refs, collectSchemaRefsRecursive(schema.Else, prefix, visited)...)
	}

	// $defs (OAS 3.1+)
	for _, defSchema := range schema.Defs {
		refs = append(refs, collectSchemaRefsRecursive(defSchema, prefix, visited)...)
	}

	// Pattern properties
	for _, propSchema := range schema.PatternProperties {
		refs = append(refs, collectSchemaRefsRecursive(propSchema, prefix, visited)...)
	}

	// Discriminator mapping values are references
	if schema.Discriminator != nil {
		for _, mappingRef := range schema.Discriminator.Mapping {
			if name := extractSchemaName(mappingRef, prefix); name != "" {
				refs = append(refs, name)
			}
		}
	}

	return refs
}

// isPathItemEmpty returns true if the path item has no operations defined.
// A path with only parameters but no HTTP methods is considered empty.
// A path with a $ref is NOT considered empty.
func isPathItemEmpty(pathItem *parser.PathItem, version parser.OASVersion) bool {
	if pathItem == nil {
		return true
	}

	// A path with a $ref is not empty - it references another path item
	if pathItem.Ref != "" {
		return false
	}

	// Check all HTTP methods available in all versions
	if pathItem.Get != nil ||
		pathItem.Put != nil ||
		pathItem.Post != nil ||
		pathItem.Delete != nil ||
		pathItem.Options != nil ||
		pathItem.Head != nil ||
		pathItem.Patch != nil {
		return false
	}

	// TRACE method is OAS 3.0+
	if version >= parser.OASVersion300 && pathItem.Trace != nil {
		return false
	}

	// QUERY method is OAS 3.2+
	if version >= parser.OASVersion320 && pathItem.Query != nil {
		return false
	}

	return true
}

// pruneEmptyPaths removes path items that have no operations defined.
// This is a shared implementation used by both OAS 2.0 and OAS 3.x.
func (f *Fixer) pruneEmptyPaths(paths parser.Paths, result *FixResult, version parser.OASVersion) {
	if paths == nil {
		return
	}

	for pathKey, pathItem := range paths {
		if isPathItemEmpty(pathItem, version) {
			delete(paths, pathKey)
			fix := Fix{
				Type:        FixTypePrunedEmptyPath,
				Path:        fmt.Sprintf("paths.%s", pathKey),
				Description: fmt.Sprintf("removed empty path item '%s' with no operations", pathKey),
				Before:      pathItem,
				After:       nil,
			}
			f.populateFixLocation(&fix)
			result.Fixes = append(result.Fixes, fix)
		}
	}
}

// resolveNameCollision ensures a new name doesn't conflict with existing schemas.
// If newName exists in schemas (and is not being renamed away), appends a numeric suffix.
// Returns the resolved unique name.
func resolveNameCollision(newName string, schemas map[string]*parser.Schema, pendingRenames map[string]string) string {
	// Check if the name is available
	if isNameAvailable(newName, schemas, pendingRenames) {
		return newName
	}

	// Find a unique name by appending a numeric suffix
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s%d", newName, i)
		if isNameAvailable(candidate, schemas, pendingRenames) {
			return candidate
		}
	}
}

// isNameAvailable checks if a name is available for use.
// A name is available if:
// 1. It doesn't exist in schemas, OR
// 2. It exists but is being renamed away (in pendingRenames as a key)
func isNameAvailable(name string, schemas map[string]*parser.Schema, pendingRenames map[string]string) bool {
	// If the name doesn't exist in schemas, it's available
	if _, exists := schemas[name]; !exists {
		return true
	}

	// If the name exists but is being renamed to something else, it's available
	if pendingRenames != nil {
		if _, beingRenamed := pendingRenames[name]; beingRenamed {
			return true
		}
	}

	return false
}
