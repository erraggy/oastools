// oas3.go contains OAS 3.x specific fix implementations

package fixer

import (
	"fmt"
	"net/url"
	"sort"

	"github.com/erraggy/oastools/parser"
)

// fixOAS3 applies fixes to an OAS 3.x document
func (f *Fixer) fixOAS3(parseResult parser.ParseResult, result *FixResult) (*FixResult, error) {
	// Extract the OAS 3.x document from the generic Document field
	srcDoc, ok := parseResult.OAS3Document()
	if !ok {
		return nil, fmt.Errorf("fixer: expected *parser.OAS3Document, got %T", parseResult.Document)
	}

	// Deep copy the document to avoid mutating the original
	doc, err := deepCopyOAS3Document(srcDoc)
	if err != nil {
		return nil, fmt.Errorf("fixer: failed to copy document: %w", err)
	}

	// Apply enabled fixes in order:
	// 1. Missing path parameters (existing)
	if f.isFixEnabled(FixTypeMissingPathParameter) {
		f.fixMissingPathParametersOAS3(doc, result)
	}

	// 2. Rename invalid schema names (must happen BEFORE pruning)
	if f.isFixEnabled(FixTypeRenamedGenericSchema) {
		f.fixInvalidSchemaNamesOAS3(doc, result)
	}

	// 3. Prune unused schemas
	if f.isFixEnabled(FixTypePrunedUnusedSchema) {
		f.pruneUnusedSchemasOAS3(doc, result)
	}

	// 4. Prune empty paths
	if f.isFixEnabled(FixTypePrunedEmptyPath) {
		f.pruneEmptyPaths(doc.Paths, result, doc.OASVersion)
	}

	// Update result
	result.Document = doc
	result.FixCount = len(result.Fixes)

	return result, nil
}

// fixMissingPathParametersOAS3 adds missing path parameters to an OAS 3.x document.
// Fixes are applied in sorted order (by path, method, parameter name) for deterministic output.
func (f *Fixer) fixMissingPathParametersOAS3(doc *parser.OAS3Document, result *FixResult) {
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
		operations := parser.GetOperations(pathItem, doc.OASVersion)

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

				// OAS 3.x uses Schema for type definition
				schema := &parser.Schema{
					Type: paramType,
				}
				if paramFormat != "" {
					schema.Format = paramFormat
				}

				newParam := &parser.Parameter{
					Name:     paramName,
					In:       parser.ParamInPath,
					Required: true, // Path parameters are always required
					Schema:   schema,
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

// fixInvalidSchemaNamesOAS3 renames schemas with invalid characters (like generic types)
// to valid names. This must happen BEFORE pruning since pruning relies on ref collection
// which needs valid refs.
func (f *Fixer) fixInvalidSchemaNamesOAS3(doc *parser.OAS3Document, result *FixResult) {
	if doc.Components == nil || len(doc.Components.Schemas) == 0 {
		return
	}

	schemas := doc.Components.Schemas

	// Build rename map: old name -> new name
	// Only include schemas that have invalid characters
	pendingRenames := make(map[string]string)
	for name := range schemas {
		if hasInvalidSchemaNameChars(name) {
			newName := transformSchemaName(name, f.GenericNamingConfig)
			newName = resolveNameCollision(newName, schemas, pendingRenames)
			pendingRenames[name] = newName
		}
	}

	if len(pendingRenames) == 0 {
		return
	}

	// Build ref renames map with both encoded and non-encoded refs
	// Key: old ref path, Value: new ref path
	refRenames := make(map[string]string)
	const prefix = "#/components/schemas/"

	for oldName, newName := range pendingRenames {
		oldRef := prefix + oldName
		newRef := prefix + newName

		// Add non-encoded ref
		refRenames[oldRef] = newRef

		// Add URL-encoded ref (for names with special characters)
		encodedOldRef := prefix + url.PathEscape(oldName)
		if encodedOldRef != oldRef {
			refRenames[encodedOldRef] = newRef
		}
	}

	// Sort old names for deterministic processing order
	oldNames := make([]string, 0, len(pendingRenames))
	for oldName := range pendingRenames {
		oldNames = append(oldNames, oldName)
	}
	sort.Strings(oldNames)

	// Apply renames to schemas map
	for _, oldName := range oldNames {
		newName := pendingRenames[oldName]
		schema := schemas[oldName]
		delete(schemas, oldName)
		schemas[newName] = schema

		// Record fix
		fix := Fix{
			Type:        FixTypeRenamedGenericSchema,
			Path:        fmt.Sprintf("components.schemas.%s", oldName),
			Description: fmt.Sprintf("renamed schema '%s' to '%s'", oldName, newName),
			Before:      oldName,
			After:       newName,
		}
		f.populateFixLocation(&fix)
		result.Fixes = append(result.Fixes, fix)
	}

	// Rewrite all $refs in the document
	f.rewriteAllRefsOAS3(doc, refRenames)
}

// rewriteAllRefsOAS3 rewrites all $refs in an OAS 3.x document using the rename map.
func (f *Fixer) rewriteAllRefsOAS3(doc *parser.OAS3Document, refRenames map[string]string) {
	if len(refRenames) == 0 {
		return
	}

	// Rewrite refs in components
	if doc.Components != nil {
		// Schemas
		for _, schema := range doc.Components.Schemas {
			rewriteSchemaRefs(schema, refRenames)
		}

		// Parameters
		for _, param := range doc.Components.Parameters {
			if param == nil {
				continue
			}
			if param.Schema != nil {
				rewriteSchemaRefs(param.Schema, refRenames)
			}
			// Content schemas
			for _, mt := range param.Content {
				if mt != nil && mt.Schema != nil {
					rewriteSchemaRefs(mt.Schema, refRenames)
				}
			}
		}

		// Responses
		for _, resp := range doc.Components.Responses {
			f.rewriteResponseRefs(resp, refRenames)
		}

		// RequestBodies
		for _, reqBody := range doc.Components.RequestBodies {
			f.rewriteRequestBodyRefs(reqBody, refRenames)
		}

		// Headers
		for _, header := range doc.Components.Headers {
			if header == nil {
				continue
			}
			if header.Schema != nil {
				rewriteSchemaRefs(header.Schema, refRenames)
			}
			for _, mt := range header.Content {
				if mt != nil && mt.Schema != nil {
					rewriteSchemaRefs(mt.Schema, refRenames)
				}
			}
		}

		// Callbacks
		for _, callback := range doc.Components.Callbacks {
			if callback != nil {
				for _, pathItem := range *callback {
					f.rewritePathItemRefs(pathItem, refRenames)
				}
			}
		}

		// PathItems (OAS 3.1+)
		for _, pathItem := range doc.Components.PathItems {
			f.rewritePathItemRefs(pathItem, refRenames)
		}
	}

	// Rewrite refs in paths
	for _, pathItem := range doc.Paths {
		f.rewritePathItemRefs(pathItem, refRenames)
	}

	// Rewrite refs in webhooks (OAS 3.1+)
	for _, pathItem := range doc.Webhooks {
		f.rewritePathItemRefs(pathItem, refRenames)
	}
}

// rewritePathItemRefs rewrites $refs in a path item and all its operations.
func (f *Fixer) rewritePathItemRefs(pathItem *parser.PathItem, refRenames map[string]string) {
	if pathItem == nil {
		return
	}

	// Path-level parameters
	for _, param := range pathItem.Parameters {
		if param != nil && param.Schema != nil {
			rewriteSchemaRefs(param.Schema, refRenames)
		}
		for _, mt := range param.Content {
			if mt != nil && mt.Schema != nil {
				rewriteSchemaRefs(mt.Schema, refRenames)
			}
		}
	}

	// Operations
	operations := []*parser.Operation{
		pathItem.Get, pathItem.Put, pathItem.Post, pathItem.Delete,
		pathItem.Options, pathItem.Head, pathItem.Patch, pathItem.Trace, pathItem.Query,
	}

	for _, op := range operations {
		f.rewriteOperationRefs(op, refRenames)
	}
}

// rewriteOperationRefs rewrites $refs in an operation.
func (f *Fixer) rewriteOperationRefs(op *parser.Operation, refRenames map[string]string) {
	if op == nil {
		return
	}

	// Parameters
	for _, param := range op.Parameters {
		if param != nil && param.Schema != nil {
			rewriteSchemaRefs(param.Schema, refRenames)
		}
		for _, mt := range param.Content {
			if mt != nil && mt.Schema != nil {
				rewriteSchemaRefs(mt.Schema, refRenames)
			}
		}
	}

	// Request body
	f.rewriteRequestBodyRefs(op.RequestBody, refRenames)

	// Responses
	if op.Responses != nil {
		if op.Responses.Default != nil {
			f.rewriteResponseRefs(op.Responses.Default, refRenames)
		}
		for _, resp := range op.Responses.Codes {
			f.rewriteResponseRefs(resp, refRenames)
		}
	}

	// Callbacks
	for _, callback := range op.Callbacks {
		if callback != nil {
			for _, pathItem := range *callback {
				f.rewritePathItemRefs(pathItem, refRenames)
			}
		}
	}
}

// rewriteRequestBodyRefs rewrites $refs in a request body.
func (f *Fixer) rewriteRequestBodyRefs(reqBody *parser.RequestBody, refRenames map[string]string) {
	if reqBody == nil {
		return
	}

	for _, mt := range reqBody.Content {
		if mt != nil && mt.Schema != nil {
			rewriteSchemaRefs(mt.Schema, refRenames)
		}
	}
}

// rewriteResponseRefs rewrites $refs in a response.
func (f *Fixer) rewriteResponseRefs(resp *parser.Response, refRenames map[string]string) {
	if resp == nil {
		return
	}

	// Content schemas
	for _, mt := range resp.Content {
		if mt != nil && mt.Schema != nil {
			rewriteSchemaRefs(mt.Schema, refRenames)
		}
	}

	// Header schemas
	for _, header := range resp.Headers {
		if header != nil && header.Schema != nil {
			rewriteSchemaRefs(header.Schema, refRenames)
		}
		for _, mt := range header.Content {
			if mt != nil && mt.Schema != nil {
				rewriteSchemaRefs(mt.Schema, refRenames)
			}
		}
	}
}

// pruneUnusedSchemasOAS3 removes schemas that are not referenced anywhere in the document.
// This uses transitive reference analysis to ensure schemas that are only referenced
// by other schemas (not directly by operations) are not incorrectly pruned.
func (f *Fixer) pruneUnusedSchemasOAS3(doc *parser.OAS3Document, result *FixResult) {
	if doc.Components == nil || len(doc.Components.Schemas) == 0 {
		return
	}

	schemas := doc.Components.Schemas

	// Collect all refs in the document
	collector := NewRefCollector()
	collector.CollectOAS3(doc)

	// Build the set of transitively referenced schemas
	referenced := buildReferencedSchemaSet(collector, schemas, doc.OASVersion)

	// Sort schema names for deterministic output
	schemaNames := make([]string, 0, len(schemas))
	for name := range schemas {
		schemaNames = append(schemaNames, name)
	}
	sort.Strings(schemaNames)

	// Remove unreferenced schemas
	for _, name := range schemaNames {
		if !referenced[name] {
			delete(schemas, name)

			fix := Fix{
				Type:        FixTypePrunedUnusedSchema,
				Path:        fmt.Sprintf("components.schemas.%s", name),
				Description: fmt.Sprintf("removed unused schema '%s'", name),
				Before:      name,
				After:       nil,
			}
			f.populateFixLocation(&fix)
			result.Fixes = append(result.Fixes, fix)
		}
	}

	// Set schemas to nil if all were pruned
	if len(schemas) == 0 {
		doc.Components.Schemas = nil
	}

	// Set components to nil if all fields are empty after pruning
	if isComponentsEmpty(doc.Components) {
		doc.Components = nil
	}
}
