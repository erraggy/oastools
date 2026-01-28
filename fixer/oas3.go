// oas3.go contains OAS 3.x specific fix implementations

package fixer

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// fixOAS3 applies fixes to an OAS 3.x document
func (f *Fixer) fixOAS3(parseResult parser.ParseResult, result *FixResult) (*FixResult, error) {
	// Extract the OAS 3.x document from the generic Document field
	srcDoc, ok := parseResult.OAS3Document()
	if !ok {
		return nil, fmt.Errorf("fixer: expected *parser.OAS3Document, got %T", parseResult.Document)
	}

	var doc *parser.OAS3Document
	if f.MutableInput {
		// Caller owns the document, mutate directly
		doc = srcDoc
	} else {
		// Deep copy the document to avoid mutating the original
		var err error
		doc, err = deepCopyOAS3Document(srcDoc)
		if err != nil {
			return nil, fmt.Errorf("fixer: failed to copy document: %w", err)
		}
	}

	// Apply fixes using shared pipeline
	f.applyFixPipeline(doc, result, oas3Pipeline)

	return result, nil
}

// fixMissingPathParametersOAS3 adds missing path parameters to an OAS 3.x document.
// Fixes are applied in sorted order (by path, method, parameter name) for deterministic output.
func (f *Fixer) fixMissingPathParametersOAS3(doc *parser.OAS3Document, result *FixResult) {
	f.fixMissingPathParameters(doc.Paths, doc.OASVersion, result)
}

// fixInvalidSchemaNamesOAS3 renames schemas with invalid characters (like generic types)
// to valid names. This must happen BEFORE pruning since pruning relies on ref collection
// which needs valid refs.
func (f *Fixer) fixInvalidSchemaNamesOAS3(doc *parser.OAS3Document, result *FixResult) {
	if doc.Components == nil || len(doc.Components.Schemas) == 0 {
		return
	}

	// Rename invalid schemas and get the ref rename map
	refRenames := f.renameInvalidSchemas(doc.Components.Schemas, doc, result)
	if len(refRenames) == 0 {
		return
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
		if param == nil {
			continue
		}
		if param.Schema != nil {
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
		if param == nil {
			continue
		}
		if param.Schema != nil {
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
}

// pruneUnusedSchemasOAS3 removes schemas that are not referenced anywhere in the document.
// This uses transitive reference analysis to ensure schemas that are only referenced
// by other schemas (not directly by operations) are not incorrectly pruned.
func (f *Fixer) pruneUnusedSchemasOAS3(doc *parser.OAS3Document, result *FixResult) {
	if doc.Components == nil || len(doc.Components.Schemas) == 0 {
		return
	}

	// Collect all refs in the document
	collector := NewRefCollector()
	collector.CollectOAS3(doc)

	// Prune unreferenced schemas
	f.pruneSchemas(doc.Components.Schemas, collector, doc, result)

	// Set schemas to nil if all were pruned
	if len(doc.Components.Schemas) == 0 {
		doc.Components.Schemas = nil
	}

	// Set components to nil if all fields are empty after pruning
	if isComponentsEmpty(doc.Components) {
		doc.Components = nil
	}
}
