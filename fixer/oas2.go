// oas2.go contains OAS 2.0 (Swagger) specific fix implementations

package fixer

import (
	"fmt"
	"log/slog"
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

	var doc *parser.OAS2Document
	if f.MutableInput {
		// Caller owns the document, mutate directly
		doc = srcDoc
	} else {
		// Deep copy the document to avoid mutating the original
		doc = srcDoc.DeepCopy()
	}

	// Apply fixes using shared pipeline
	f.applyFixPipeline(doc, result, oas2Pipeline)

	return result, nil
}

// fixMissingPathParametersOAS2 adds missing path parameters to an OAS 2.0 document.
// Fixes are applied in sorted order (by path, method, parameter name) for deterministic output.
func (f *Fixer) fixMissingPathParametersOAS2(doc *parser.OAS2Document, result *FixResult) {
	f.fixMissingPathParameters(doc.Paths, parser.OASVersion20, result)
}

// fixMissingPathParameters is the shared implementation for both OAS versions.
func (f *Fixer) fixMissingPathParameters(paths map[string]*parser.PathItem, version parser.OASVersion, result *FixResult) {
	if paths == nil {
		return
	}

	// Sort path patterns for deterministic order
	pathPatterns := make([]string, 0, len(paths))
	for pathPattern := range paths {
		pathPatterns = append(pathPatterns, pathPattern)
	}
	sort.Strings(pathPatterns)

	for _, pathPattern := range pathPatterns {
		pathItem := paths[pathPattern]
		if pathItem == nil {
			continue
		}

		// Get operations for this path
		operations := parser.GetOperations(pathItem, version)

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

			// Find missing parameters
			missingParams := findMissingPathParams(pathPattern, pathItem, op)
			for _, paramName := range missingParams {
				// Determine type
				paramType := "string"
				paramFormat := ""
				if f.InferTypes {
					paramType, paramFormat = inferParameterType(paramName)
				}

				// Create and add the parameter
				newParam := createMissingPathParameter(paramName, paramType, paramFormat, version == parser.OASVersion20)
				op.Parameters = append(op.Parameters, newParam)

				// Record the fix
				fix := Fix{
					Type:        FixTypeMissingPathParameter,
					Path:        fmt.Sprintf("paths.%s.%s.parameters", pathPattern, method),
					Description: buildMissingParamDescription(paramName, paramType, paramFormat),
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

	// Rename invalid schemas and get the ref rename map
	refRenames := f.renameInvalidSchemas(doc.Definitions, doc, result)
	if len(refRenames) == 0 {
		return
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
	for _, w := range collector.Warnings {
		slog.Warn("ref collection warning", "detail", w)
	}

	// Prune unreferenced schemas
	f.pruneSchemas(doc.Definitions, collector, doc, result)

	// Set definitions to nil if empty after pruning
	if len(doc.Definitions) == 0 {
		doc.Definitions = nil
	}
}
