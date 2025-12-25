// version_helpers.go provides version-agnostic helpers for OAS 2.0 and 3.x fixes

package fixer

import (
	"fmt"
	"net/url"
	"sort"

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
// It handles both URL-encoded and non-encoded refs.
func buildRefRenameMap(renames map[string]string, accessor parser.DocumentAccessor) map[string]string {
	prefix := accessor.SchemaRefPrefix()
	refRenames := make(map[string]string, len(renames)*2)

	for oldName, newName := range renames {
		oldRef := prefix + oldName
		newRef := prefix + newName
		refRenames[oldRef] = newRef

		// Add URL-encoded version for refs that might be encoded
		encodedOldRef := prefix + url.PathEscape(oldName)
		if encodedOldRef != oldRef {
			refRenames[encodedOldRef] = newRef
		}
	}

	return refRenames
}

// collectDeclaredPathParams collects declared path parameters from both PathItem and Operation.
func collectDeclaredPathParams(pathItem *parser.PathItem, op *parser.Operation) map[string]bool {
	declaredParams := make(map[string]bool)

	// PathItem-level parameters
	for _, param := range pathItem.Parameters {
		if param != nil && param.In == parser.ParamInPath {
			declaredParams[param.Name] = true
		}
	}

	// Operation-level parameters (override PathItem params)
	for _, param := range op.Parameters {
		if param != nil && param.In == parser.ParamInPath {
			declaredParams[param.Name] = true
		}
	}

	return declaredParams
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
func findMissingPathParams(pathPattern string, pathItem *parser.PathItem, op *parser.Operation) []string {
	pathParams := extractPathParameters(pathPattern)
	if len(pathParams) == 0 {
		return nil
	}

	declaredParams := collectDeclaredPathParams(pathItem, op)

	// Collect missing params
	var missing []string
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

	// Build rename map: old name -> new name
	renames := make(map[string]string)
	for name := range schemas {
		if hasInvalidSchemaNameChars(name) {
			newName := transformSchemaName(name, f.GenericNamingConfig)
			newName = resolveNameCollision(newName, schemas, renames)
			renames[name] = newName
		}
	}

	if len(renames) == 0 {
		return nil
	}

	// Sort old names for deterministic processing order
	oldNames := make([]string, 0, len(renames))
	for oldName := range renames {
		oldNames = append(oldNames, oldName)
	}
	sort.Strings(oldNames)

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
