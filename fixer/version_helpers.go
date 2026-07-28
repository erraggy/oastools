// version_helpers.go provides version-agnostic helpers for OAS 2.0 and 3.x fixes

package fixer

import (
	"fmt"
	"sort"

	"github.com/erraggy/oastools/internal/paramutil"
	"github.com/erraggy/oastools/internal/pathutil"
	"github.com/erraggy/oastools/parser"
)

// schemaPathPrefix returns the JSON path prefix for schemas based on OAS version.
func schemaPathPrefix(version parser.OASVersion) string {
	if version == parser.OASVersion20 {
		return "definitions"
	}
	return "components.schemas"
}

// buildRefRenameMap creates a map of old refs to new refs for schema renames.
//
// Each rename registers two keys, matching the two-step lookup in
// [lookupRenamedRef]: the name exactly as the document spells it, and the name
// with both escaping conventions reversed. The exact key is what lets a
// component genuinely named "Foo%20Bar[Pet]" match a $ref spelling it the same
// way; the decoded key is what lets any of the encodings a generator might emit
// match a name that carries none of them. Registering only one of the two loses
// whichever case it does not cover.
//
// Exact keys are registered first and never overwritten, so a decoded key that
// would collide with another rename's exact spelling loses to it.
//
// The decoded pass runs in sorted order because two names can share a decoded
// form without either being it — "A%2FB[Pet]" and "A~1B[Pet]" both reduce to
// "A/B[Pet]" — and only one of them can claim that key. Ranging over the map
// would hand it to a different rename on each run, so a $ref spelled that way
// would resolve to a different schema run to run.
//
// Values are ready to write into a $ref, so the new name is escaped per RFC
// 6901: a name containing "/" or "~" must build the pointer that reaches the
// renamed component.
func buildRefRenameMap(renames map[string]string, accessor parser.DocumentAccessor) map[string]string {
	prefix := accessor.SchemaRefPrefix()
	refRenames := make(map[string]string, len(renames)*2)

	oldNames := make([]string, 0, len(renames))
	for oldName, newName := range renames {
		refRenames[prefix+oldName] = prefix + pathutil.EscapeRefToken(newName)
		oldNames = append(oldNames, oldName)
	}
	sort.Strings(oldNames)

	for _, oldName := range oldNames {
		decoded := prefix + pathutil.DecodeRefToken(oldName)
		if _, taken := refRenames[decoded]; !taken {
			refRenames[decoded] = prefix + pathutil.EscapeRefToken(renames[oldName])
		}
	}

	return refRenames
}

// createMissingPathParameter creates a new path parameter for the given OAS version.
// OAS 2.0 uses Type/Format directly, OAS 3.x uses Schema.
func createMissingPathParameter(paramName, paramType, paramFormat string, isOAS2 bool) *parser.Parameter {
	param := &parser.Parameter{
		Name:     paramName,
		In:       parser.ParamInPath,
		Required: true, // Path parameters are always required
	}

	if isOAS2 {
		param.Type = paramType
		if paramFormat != "" {
			param.Format = paramFormat
		}
	} else {
		schema := &parser.Schema{Type: paramType}
		if paramFormat != "" {
			schema.Format = paramFormat
		}
		param.Schema = schema
	}

	return param
}

// buildMissingParamDescription creates a description for a missing path parameter fix.
func buildMissingParamDescription(paramName, paramType, paramFormat string) string {
	desc := fmt.Sprintf("Added missing path parameter '%s' (type: %s", paramName, paramType)
	if paramFormat != "" {
		desc += fmt.Sprintf(", format: %s", paramFormat)
	}
	desc += ")"
	return desc
}

// findMissingPathParams finds path parameters declared in the path template but missing from the operation.
// Returns a sorted list of missing parameter names for deterministic output.
//
// Declared parameters may be $refs into the document's reusable parameter
// definitions, so resolver is used to look through them. When a $ref cannot be
// resolved no parameters are reported missing: adding one that the reference
// already declares would produce a duplicate name+location, which is worse than
// leaving the spec as-is.
func findMissingPathParams(pathPattern string, pathItem *parser.PathItem, op *parser.Operation, resolver paramutil.Resolver) []string {
	pathParams := extractPathParameters(pathPattern)
	if len(pathParams) == 0 {
		return nil
	}

	declaredParams, complete := resolver.DeclaredPathParams(pathItem.Parameters, op.Parameters)
	if !complete {
		return nil
	}

	// Collect missing params (pre-allocate with max possible capacity)
	missing := make([]string, 0, len(pathParams))
	for paramName := range pathParams {
		if !declaredParams[paramName] {
			missing = append(missing, paramName)
		}
	}

	// Sort for deterministic output
	sort.Strings(missing)
	return missing
}

// pruneSchemas removes unreferenced schemas from a schema map and returns the fixes.
// The schemas map is modified in place.
// The accessor parameter provides version-agnostic access to schema reference prefixes.
func (f *Fixer) pruneSchemas(
	schemas map[string]*parser.Schema,
	collector *RefCollector,
	accessor parser.DocumentAccessor,
	result *FixResult,
) {
	if len(schemas) == 0 {
		return
	}

	// Build the set of transitively referenced schemas
	referenced := buildReferencedSchemaSet(collector, schemas, accessor)

	// Sort schema names for deterministic output
	schemaNames := make([]string, 0, len(schemas))
	for name := range schemas {
		schemaNames = append(schemaNames, name)
	}
	sort.Strings(schemaNames)

	// Remove unreferenced schemas
	pathPrefix := schemaPathPrefix(accessor.GetVersion())
	for _, name := range schemaNames {
		if !referenced[name] {
			delete(schemas, name)

			fix := Fix{
				Type:        FixTypePrunedUnusedSchema,
				Path:        fmt.Sprintf("%s.%s", pathPrefix, name),
				Description: fmt.Sprintf("removed unreferenced schema '%s'", name),
				Before:      name,
				After:       nil,
			}
			f.populateFixLocation(&fix)
			result.Fixes = append(result.Fixes, fix)
		}
	}
}

// renameInvalidSchemas renames schemas with invalid characters and returns the ref rename map.
// The schemas map is modified in place.
// The accessor parameter provides version-agnostic access to schema reference prefixes.
func (f *Fixer) renameInvalidSchemas(
	schemas map[string]*parser.Schema,
	accessor parser.DocumentAccessor,
	result *FixResult,
) map[string]string {
	if len(schemas) == 0 {
		return nil
	}

	// Sort the candidates before assigning new names. Two names can transform to
	// the same result — "Resp[a/b.Pet]" and "Resp[a.b.Pet]" both reduce to
	// "RespOfa.b.Pet" — and the loser takes the numeric suffix
	// resolveNameCollision appends. Ranging over the map instead would hand that
	// suffix to a different schema on each run.
	candidates := make([]string, 0, len(schemas))
	for name := range schemas {
		if hasInvalidSchemaNameChars(name) {
			candidates = append(candidates, name)
		}
	}
	sort.Strings(candidates)

	// Build rename map: old name -> new name.
	//
	// No candidate can transform to itself — it was selected for containing a
	// character transformSchemaName always removes — so every candidate yields a
	// real rename and none needs a no-op guard here.
	renames := make(map[string]string, len(candidates))
	claimed := make(map[string]bool, len(candidates))
	for _, name := range candidates {
		newName := resolveNameCollision(
			transformSchemaName(name, f.GenericNamingConfig), schemas, renames, claimed)
		renames[name] = newName
		claimed[newName] = true
	}

	if len(renames) == 0 {
		return nil
	}

	// Already sorted, so the fixes are recorded in a deterministic order.
	oldNames := candidates

	// Apply renames to schemas map and record fixes
	pathPrefix := schemaPathPrefix(accessor.GetVersion())
	for _, oldName := range oldNames {
		newName := renames[oldName]
		schema := schemas[oldName]
		delete(schemas, oldName)
		schemas[newName] = schema

		fix := Fix{
			Type:        FixTypeRenamedGenericSchema,
			Path:        fmt.Sprintf("%s.%s", pathPrefix, oldName),
			Description: fmt.Sprintf("renamed schema '%s' to '%s'", oldName, newName),
			Before:      oldName,
			After:       newName,
		}
		f.populateFixLocation(&fix)
		result.Fixes = append(result.Fixes, fix)
	}

	// Build and return ref renames map
	return buildRefRenameMap(renames, accessor)
}
