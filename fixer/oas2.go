// oas2.go contains OAS 2.0 (Swagger) specific fix implementations

package fixer

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/erraggy/oastools/parser"
)

// fixOAS2 applies fixes to an OAS 2.0 document
func (f *Fixer) fixOAS2(parseResult parser.ParseResult, result *FixResult) (*FixResult, error) {
	// Extract the OAS 2.0 document from the generic Document field
	srcDoc, ok := parseResult.OAS2Document()
	if !ok {
		return nil, fmt.Errorf("fixer: expected *parser.OAS2Document, got %T", parseResult.Document)
	}

	// Deep copy the document to avoid mutating the original
	doc, err := deepCopyOAS2Document(srcDoc)
	if err != nil {
		return nil, fmt.Errorf("fixer: failed to copy document: %w", err)
	}

	// Apply enabled fixes in order:
	// 1. Missing path parameters (existing)
	if f.isFixEnabled(FixTypeMissingPathParameter) {
		f.fixMissingPathParametersOAS2(doc, result)
	}

	// 2. Rename invalid schema names (must happen BEFORE pruning)
	if f.isFixEnabled(FixTypeRenamedGenericSchema) {
		f.fixInvalidSchemaNamesOAS2(doc, result)
	}

	// 3. Prune unused schemas
	if f.isFixEnabled(FixTypePrunedUnusedSchema) {
		f.pruneUnusedSchemasOAS2(doc, result)
	}

	// 4. Prune empty paths
	if f.isFixEnabled(FixTypePrunedEmptyPath) {
		f.pruneEmptyPaths(doc.Paths, result, parser.OASVersion20)
	}

	// Update result
	result.Document = doc
	result.FixCount = len(result.Fixes)

	return result, nil
}

// fixMissingPathParametersOAS2 adds missing path parameters to an OAS 2.0 document.
// Fixes are applied in sorted order (by path, method, parameter name) for deterministic output.
func (f *Fixer) fixMissingPathParametersOAS2(doc *parser.OAS2Document, result *FixResult) {
	if doc.Paths == nil {
		return
	}

	// Sort path patterns for deterministic order
	pathPatterns := make([]string, 0, len(doc.Paths))
	for pathPattern := range doc.Paths {
		pathPatterns = append(pathPatterns, pathPattern)
	}
	sort.Strings(pathPatterns)

	for _, pathPattern := range pathPatterns {
		pathItem := doc.Paths[pathPattern]
		if pathItem == nil {
			continue
		}

		// Extract parameters from path template
		pathParams := extractPathParameters(pathPattern)
		if len(pathParams) == 0 {
			continue
		}

		// Get operations for this path
		operations := parser.GetOperations(pathItem, parser.OASVersion20)

		// Sort methods for deterministic order
		methods := make([]string, 0, len(operations))
		for method := range operations {
			methods = append(methods, method)
		}
		sort.Strings(methods)

		for _, method := range methods {
			op := operations[method]
			if op == nil {
				continue
			}

			// Collect declared path parameters from PathItem and Operation
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

			// Sort parameter names for deterministic order
			paramNames := make([]string, 0, len(pathParams))
			for paramName := range pathParams {
				paramNames = append(paramNames, paramName)
			}
			sort.Strings(paramNames)

			// Find missing parameters
			for _, paramName := range paramNames {
				if declaredParams[paramName] {
					continue
				}

				// Create the missing parameter
				paramType := "string"
				paramFormat := ""
				if f.InferTypes {
					paramType, paramFormat = inferParameterType(paramName)
				}

				newParam := &parser.Parameter{
					Name:     paramName,
					In:       parser.ParamInPath,
					Required: true, // Path parameters are always required
					Type:     paramType,
				}
				if paramFormat != "" {
					newParam.Format = paramFormat
				}

				// Add to operation parameters
				op.Parameters = append(op.Parameters, newParam)

				// Record the fix
				jsonPath := fmt.Sprintf("paths.%s.%s.parameters", pathPattern, method)
				description := fmt.Sprintf("Added missing path parameter '%s' (type: %s", paramName, paramType)
				if paramFormat != "" {
					description += fmt.Sprintf(", format: %s", paramFormat)
				}
				description += ")"

				fix := Fix{
					Type:        FixTypeMissingPathParameter,
					Path:        jsonPath,
					Description: description,
					Before:      nil,
					After:       newParam,
				}
				f.populateFixLocation(&fix)
				result.Fixes = append(result.Fixes, fix)
			}
		}
	}
}

// fixInvalidSchemaNamesOAS2 renames schemas with invalid characters (e.g., generic type names
// like "Response[User]") to valid names in an OAS 2.0 document.
// This must run BEFORE pruning to ensure refs are updated correctly.
func (f *Fixer) fixInvalidSchemaNamesOAS2(doc *parser.OAS2Document, result *FixResult) {
	if len(doc.Definitions) == 0 {
		return
	}

	// Build rename map: old name -> new name
	renames := make(map[string]string)
	for name := range doc.Definitions {
		if hasInvalidSchemaNameChars(name) {
			newName := transformSchemaName(name, f.GenericNamingConfig)
			newName = resolveNameCollision(newName, doc.Definitions, renames)
			renames[name] = newName
		}
	}

	if len(renames) == 0 {
		return
	}

	// Build ref renames map for rewriting $refs
	// OAS 2.0 uses #/definitions/ prefix
	refRenames := make(map[string]string)
	for oldName, newName := range renames {
		oldRef := "#/definitions/" + oldName
		newRef := "#/definitions/" + newName
		refRenames[oldRef] = newRef

		// Also add URL-encoded version for refs that might be encoded
		encodedOldRef := "#/definitions/" + url.PathEscape(oldName)
		if encodedOldRef != oldRef {
			refRenames[encodedOldRef] = newRef
		}
	}

	// Apply renames to definitions map
	for oldName, newName := range renames {
		schema := doc.Definitions[oldName]
		delete(doc.Definitions, oldName)
		doc.Definitions[newName] = schema

		// Record the fix
		fix := Fix{
			Type:        FixTypeRenamedGenericSchema,
			Path:        fmt.Sprintf("definitions.%s", oldName),
			Description: fmt.Sprintf("renamed schema '%s' to '%s'", oldName, newName),
			Before:      oldName,
			After:       newName,
		}
		f.populateFixLocation(&fix)
		result.Fixes = append(result.Fixes, fix)
	}

	// Rewrite all $refs in definitions
	for _, schema := range doc.Definitions {
		rewriteSchemaRefs(schema, refRenames)
	}

	// Rewrite $refs in global parameters
	for _, param := range doc.Parameters {
		if param != nil && param.Schema != nil {
			rewriteSchemaRefs(param.Schema, refRenames)
		}
	}

	// Rewrite $refs in global responses
	for _, resp := range doc.Responses {
		if resp != nil && resp.Schema != nil {
			rewriteSchemaRefs(resp.Schema, refRenames)
		}
	}

	// Rewrite $refs in paths
	for _, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		// Path-level parameters
		for _, param := range pathItem.Parameters {
			if param != nil && param.Schema != nil {
				rewriteSchemaRefs(param.Schema, refRenames)
			}
		}

		// Operations
		ops := parser.GetOperations(pathItem, parser.OASVersion20)
		for _, op := range ops {
			if op == nil {
				continue
			}

			// Operation parameters
			for _, param := range op.Parameters {
				if param != nil && param.Schema != nil {
					rewriteSchemaRefs(param.Schema, refRenames)
				}
			}

			// Responses
			if op.Responses != nil {
				if op.Responses.Default != nil && op.Responses.Default.Schema != nil {
					rewriteSchemaRefs(op.Responses.Default.Schema, refRenames)
				}
				for _, resp := range op.Responses.Codes {
					if resp != nil && resp.Schema != nil {
						rewriteSchemaRefs(resp.Schema, refRenames)
					}
				}
			}
		}
	}
}

// pruneUnusedSchemasOAS2 removes schemas from definitions that are not referenced
// anywhere in the document.
func (f *Fixer) pruneUnusedSchemasOAS2(doc *parser.OAS2Document, result *FixResult) {
	if len(doc.Definitions) == 0 {
		return
	}

	// Collect all refs in the document
	collector := NewRefCollector()
	collector.CollectOAS2(doc)

	// Build the set of referenced schemas (including transitive refs)
	referenced := buildReferencedSchemaSet(collector, doc.Definitions, parser.OASVersion20)

	// Sort schema names for deterministic order
	schemaNames := make([]string, 0, len(doc.Definitions))
	for name := range doc.Definitions {
		schemaNames = append(schemaNames, name)
	}
	sort.Strings(schemaNames)

	// Delete unreferenced schemas
	for _, name := range schemaNames {
		if !referenced[name] {
			delete(doc.Definitions, name)

			fix := Fix{
				Type:        FixTypePrunedUnusedSchema,
				Path:        fmt.Sprintf("definitions.%s", name),
				Description: fmt.Sprintf("removed unreferenced schema '%s'", name),
				Before:      name,
				After:       nil,
			}
			f.populateFixLocation(&fix)
			result.Fixes = append(result.Fixes, fix)
		}
	}

	// Set definitions to nil if empty after pruning
	if len(doc.Definitions) == 0 {
		doc.Definitions = nil
	}
}
