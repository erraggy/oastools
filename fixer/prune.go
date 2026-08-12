// This file implements shared pruning logic for removing orphaned content
// from OpenAPI documents. These helpers are version-agnostic and used by
// both OAS 2.0 and OAS 3.x implementations.

package fixer

import (
	"fmt"
	"slices"
	"strings"

	"github.com/erraggy/oastools/internal/pathutil"
	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
)

// collectPolymorphicSchemaRefs collects refs from a schema-or-bool field, plus the
// raw map form that [schemautil.SchemaOrBoolSchemas] does not cover.
func collectPolymorphicSchemaRefs(refs *[]string, field any, prefix string, visited map[*parser.Schema]bool) {
	if field == nil {
		return
	}
	for _, s := range schemautil.SchemaOrBoolSchemas(field) {
		collectSchemaRefsRecursive(refs, s, prefix, visited)
	}
	if mapField, ok := field.(map[string]any); ok {
		collectRefsFromMap(refs, mapField, prefix)
	}
}

// refHasEntryPointOrigin returns true when any of the following are true:
//   - ref is not in poolRefs or its origins are empty
//   - at least one origin is outside the schema pool
func refHasEntryPointOrigin(poolPrefix string, poolRefs map[string][]string, ref string) bool {
	origins, ok := poolRefs[ref]
	if !ok || len(origins) == 0 {
		return true
	}
	for _, origin := range origins {
		if !strings.HasPrefix(origin, poolPrefix) {
			return true
		}
	}
	return false
}

// buildReferencedSchemaSet builds the transitive closure of referenced schemas.
// Starting from refs collected by RefCollector, it follows schema-to-schema references
// to ensure schemas that are indirectly referenced are not pruned.
//
// Example: If operation refs A, and A refs B, and B refs C, all three are "referenced".
func buildReferencedSchemaSet(collector *RefCollector, schemas map[string]*parser.Schema, accessor parser.DocumentAccessor) map[string]bool {
	referenced := make(map[string]bool)
	queue := make([]string, 0)

	prefix := accessor.SchemaRefPrefix()
	poolPrefix := schemaPathPrefix(accessor.GetVersion()) + "."

	// 1. Get directly referenced schemas from collector. names is reused across
	// refs so the candidate spellings cost no allocation after the first ref.
	var names []string
	for ref := range collector.RefsByType[RefTypeSchema] {
		// exclude refs whose origins are all local to this schema pool
		// see: Issue #474
		if !refHasEntryPointOrigin(poolPrefix, collector.Refs, ref) {
			continue
		}
		names = appendSchemaNames(names[:0], ref, prefix)
		for _, name := range names {
			if _, exists := schemas[name]; exists && !referenced[name] {
				referenced[name] = true
				queue = append(queue, name)
			}
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

// appendSchemaNames appends to dst every schema name a reference path could
// denote, most-faithful spelling first. It appends nothing for a ref that does
// not name a local schema.
//
// More than one candidate is needed because generators disagree about how to
// escape a name, and a wrong guess makes pruning delete a schema that is in
// fact referenced. Escaping is reversed so the name matches the key it was
// built from: without it, a schema named "pet/summary" is referenced as
// "#/definitions/pet~1summary" and the recovered name never matches the schemas
// map. But one reversal is not enough either — generators mix the conventions,
// so "#/definitions/Paged%5Bpkg/store.Pet%5D" percent-encodes the brackets and
// leaves the slashes raw, and RFC 6901 unescaping alone recovers a name no key
// matches.
//
// Returning candidates rather than picking one keeps both readings alive: a
// schema genuinely named "Foo%20Bar" is matched by the undecoded spelling, and
// a mixed-encoding ref by the decoded one. Every caller looks each name up in
// the schemas map before acting, so a candidate that names nothing costs a
// lookup and no more.
//
// Appending to dst rather than returning a fresh slice keeps the common case
// allocation-free: this runs for every $ref in the document, and almost all of
// them are unencoded and yield a single name.
func appendSchemaNames(dst []string, ref, prefix string) []string {
	// Shares [pathutil.CutRefPrefix] with splitSchemaRef so a ref whose pointer
	// separators are percent-encoded is recognized identically by pruning and by
	// rewriting. Recognizing it in only one of the two is what made a rename leave
	// such a ref dangling while its target survived the prune.
	name, found := pathutil.CutRefPrefix(ref, prefix)
	if !found {
		return dst
	}

	start := len(dst)
	dst = append(dst, name)

	// Without an escape sequence every spelling of the name is the same string.
	if !strings.ContainsAny(name, "%~") {
		return dst
	}

	for _, candidate := range [2]string{pathutil.UnescapeRefToken(name), pathutil.DecodeRefToken(name)} {
		if !slices.Contains(dst[start:], candidate) {
			dst = append(dst, candidate)
		}
	}
	return dst
}

// collectSchemaRefs extracts all schema reference names from a schema.
// This is used to find transitive references (schemas referencing other schemas).
// prefix should be "#/definitions/" for OAS 2.0 or "#/components/schemas/" for OAS 3.x
func collectSchemaRefs(schema *parser.Schema, prefix string) []string {
	visited := make(map[*parser.Schema]bool)
	refs := make([]string, 0, 32)
	collectSchemaRefsRecursive(&refs, schema, prefix, visited)
	return refs
}

// appendOptionalSchemaRefs appends refs from an optional schema field if non-nil.
// This helper reduces duplication in collectSchemaRefsRecursive.
func appendOptionalSchemaRefs(refs *[]string, s *parser.Schema, prefix string, visited map[*parser.Schema]bool) {
	if s != nil {
		collectSchemaRefsRecursive(refs, s, prefix, visited)
	}
}

// collectSchemaRefsRecursive is the internal implementation with circular reference protection.
// It appends found refs to the pre-allocated slice pointed to by refs.
func collectSchemaRefsRecursive(refs *[]string, schema *parser.Schema, prefix string, visited map[*parser.Schema]bool) {
	if schema == nil || visited[schema] {
		return
	}
	visited[schema] = true

	// Direct schema ref
	*refs = appendSchemaNames(*refs, schema.Ref, prefix)

	// Properties
	for _, propSchema := range schema.Properties {
		collectSchemaRefsRecursive(refs, propSchema, prefix, visited)
	}

	// Schema-or-bool fields
	collectPolymorphicSchemaRefs(refs, schema.AdditionalProperties, prefix, visited)
	collectPolymorphicSchemaRefs(refs, schema.Items, prefix, visited)
	collectPolymorphicSchemaRefs(refs, schema.AdditionalItems, prefix, visited)

	// Schema composition
	for _, s := range schema.AllOf {
		collectSchemaRefsRecursive(refs, s, prefix, visited)
	}
	for _, s := range schema.AnyOf {
		collectSchemaRefsRecursive(refs, s, prefix, visited)
	}
	for _, s := range schema.OneOf {
		collectSchemaRefsRecursive(refs, s, prefix, visited)
	}
	appendOptionalSchemaRefs(refs, schema.Not, prefix, visited)

	// OAS 3.1+ / JSON Schema Draft 2020-12 fields
	for _, s := range schema.PrefixItems {
		collectSchemaRefsRecursive(refs, s, prefix, visited)
	}
	appendOptionalSchemaRefs(refs, schema.Contains, prefix, visited)
	appendOptionalSchemaRefs(refs, schema.PropertyNames, prefix, visited)
	for _, depSchema := range schema.DependentSchemas {
		collectSchemaRefsRecursive(refs, depSchema, prefix, visited)
	}

	// JSON Schema 2020-12 unevaluated keywords, also schema-or-bool
	collectPolymorphicSchemaRefs(refs, schema.UnevaluatedProperties, prefix, visited)
	collectPolymorphicSchemaRefs(refs, schema.UnevaluatedItems, prefix, visited)

	// JSON Schema 2020-12 content keywords
	appendOptionalSchemaRefs(refs, schema.ContentSchema, prefix, visited)

	// Conditional schemas (OAS 3.1+)
	appendOptionalSchemaRefs(refs, schema.If, prefix, visited)
	appendOptionalSchemaRefs(refs, schema.Then, prefix, visited)
	appendOptionalSchemaRefs(refs, schema.Else, prefix, visited)

	// $defs (OAS 3.1+)
	for _, defSchema := range schema.Defs {
		collectSchemaRefsRecursive(refs, defSchema, prefix, visited)
	}

	// Pattern properties
	for _, propSchema := range schema.PatternProperties {
		collectSchemaRefsRecursive(refs, propSchema, prefix, visited)
	}

	// Discriminator mapping values are references
	if schema.Discriminator != nil {
		for _, mappingRef := range schema.Discriminator.Mapping {
			*refs = appendDiscriminatorTargetNames(*refs, mappingRef, prefix)
		}
		// defaultMapping (OAS 3.2+) keeps its target alive exactly as a mapping
		// value does; omitting it would let pruning delete the fallback schema.
		*refs = appendDiscriminatorTargetNames(*refs, schema.Discriminator.DefaultMapping, prefix)
	}
}

// appendDiscriminatorTargetNames appends the schema names a discriminator mapping
// or defaultMapping value could denote.
//
// A discriminator names its target in either of two spellings — a full pointer
// ("#/components/schemas/Dog") or a bare schema name ("Dog") — and the spec calls
// the second one out explicitly: "The behavior of a `mapping` value or
// `defaultMapping` value that is both a valid schema name and a valid relative URI
// reference is implementation-defined, but it is RECOMMENDED that it be treated as
// a schema name."
//
// [appendSchemaNames] alone only recognizes the pointer spelling, because it also
// serves real $ref values where a bare name must not match. Used on its own here
// it made pruning delete a schema referenced only by a bare mapping value, which
// is a legal document losing content. The bare value is appended as one more
// candidate rather than resolved: every caller looks each name up in the schemas
// map before acting, so a value that names no schema costs a lookup and no more —
// the same contract appendSchemaNames documents for its own candidates.
func appendDiscriminatorTargetNames(dst []string, value, prefix string) []string {
	if value == "" {
		return dst
	}
	dst = appendSchemaNames(dst, value, prefix)

	// A pointer spelling was already expanded above, so only a bare name is left
	// to add. "#" is the only reliable marker of a pointer: a bare schema name may
	// legitimately contain "/", and this repo's own generic-name fixer produces
	// exactly that shape, e.g. "Dog[example.com/pkg.Breed]". Excluding "/" here
	// made pruning delete schemas referenced only by such a name.
	if !strings.Contains(value, "#") && !slices.Contains(dst, value) {
		dst = append(dst, value)
	}
	return dst
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

	// AdditionalOperations (custom HTTP methods) are OAS 3.2+
	if version >= parser.OASVersion320 && len(pathItem.AdditionalOperations) > 0 {
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
//
// claimed carries the names earlier renames already took; the caller adds each
// resolved name to it. See [isNameAvailable] for why that set is separate from
// pendingRenames.
func resolveNameCollision(
	newName string,
	schemas map[string]*parser.Schema,
	pendingRenames map[string]string,
	claimed map[string]bool,
) string {
	// Check if the name is available
	if isNameAvailable(newName, schemas, pendingRenames, claimed) {
		return newName
	}

	// Find a unique name by appending a numeric suffix
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s%d", newName, i)
		if isNameAvailable(candidate, schemas, pendingRenames, claimed) {
			return candidate
		}
	}
}

// isNameAvailable checks if a name is available for use.
// A name is available if:
//  1. No earlier rename has already claimed it, AND
//  2. It doesn't exist in schemas, OR it exists but is being renamed away
//     (in pendingRenames as a key)
//
// The claimed check is what stops two schemas from being renamed to the same
// thing. Distinct names can transform to one result — "Resp[a/b.Pet]" and
// "Resp[a.b.Pet]" both reduce to "RespOfa.b.Pet" — and the second rename would
// otherwise be handed a name the first already took, so applying them in turn
// overwrites one schema with the other and silently drops it.
//
// claimed is a set rather than a scan of pendingRenames' values because this
// runs once per candidate name: deriving it here would make assigning n names
// quadratic, which is measurable on the code-first specs that produce hundreds
// of generic schemas at once.
func isNameAvailable(
	name string,
	schemas map[string]*parser.Schema,
	pendingRenames map[string]string,
	claimed map[string]bool,
) bool {
	if claimed[name] {
		return false
	}

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

// isComponentsEmpty returns true if all component fields are nil or empty.
// This is used to determine if the entire components object should be removed.
// Specification extensions (x-* fields in Extra) are also checked to preserve
// any custom extensions that users may have added to the components object.
func isComponentsEmpty(comp *parser.Components) bool {
	if comp == nil {
		return true
	}
	return len(comp.Schemas) == 0 &&
		len(comp.Responses) == 0 &&
		len(comp.Parameters) == 0 &&
		len(comp.Examples) == 0 &&
		len(comp.RequestBodies) == 0 &&
		len(comp.Headers) == 0 &&
		len(comp.SecuritySchemes) == 0 &&
		len(comp.Links) == 0 &&
		len(comp.Callbacks) == 0 &&
		len(comp.CallbackRefs) == 0 &&
		len(comp.PathItems) == 0 &&
		len(comp.Extra) == 0
}

// collectRefsFromMap extracts schema references from a raw map[string]any.
// This handles polymorphic schema fields (Items, AdditionalProperties, etc.) that may
// remain as untyped maps after YAML/JSON unmarshaling. These fields are declared as
// `any` in parser.Schema to support both *Schema and bool values per the OAS spec.
func collectRefsFromMap(refs *[]string, m map[string]any, prefix string) {
	// Check for direct $ref
	if refStr, ok := m["$ref"].(string); ok {
		*refs = appendSchemaNames(*refs, refStr, prefix)
	}

	// Check nested properties
	if props, ok := m["properties"].(map[string]any); ok {
		for _, propVal := range props {
			if propMap, ok := propVal.(map[string]any); ok {
				collectRefsFromMap(refs, propMap, prefix)
			}
		}
	}

	// Check items
	if items, ok := m["items"].(map[string]any); ok {
		collectRefsFromMap(refs, items, prefix)
	}

	// Check additionalProperties
	if addProps, ok := m["additionalProperties"].(map[string]any); ok {
		collectRefsFromMap(refs, addProps, prefix)
	}

	// Check allOf, anyOf, oneOf
	for _, key := range []string{"allOf", "anyOf", "oneOf"} {
		if arr, ok := m[key].([]any); ok {
			for _, item := range arr {
				if itemMap, ok := item.(map[string]any); ok {
					collectRefsFromMap(refs, itemMap, prefix)
				}
			}
		}
	}
}
