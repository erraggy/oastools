package walker

import (
	"fmt"

	"github.com/erraggy/oastools/internal/maputil"
	"github.com/erraggy/oastools/parser"
)

// walkOAS2 traverses an OAS 2.0 document.
func (w *Walker) walkOAS2(doc *parser.OAS2Document, state *walkState) error {
	// Document root - typed handler first, then generic
	// Track whether to continue to children separately from stopping
	continueToChildren := true
	if w.onOAS2Document != nil {
		wc := state.buildContext("$")
		result := w.handleAction(w.onOAS2Document(wc, doc))
		releaseContext(wc)
		if !result {
			if w.stopped {
				return nil
			}
			continueToChildren = false
		}
	}
	// Generic handler is called even if typed handler returned SkipChildren
	// but not if it returned Stop
	if w.onDocument != nil {
		wc := state.buildContext("$")
		result := w.handleAction(w.onDocument(wc, doc))
		releaseContext(wc)
		if !result {
			if w.stopped {
				return nil
			}
			continueToChildren = false
		}
	}

	if !continueToChildren {
		return nil
	}

	// Info
	if doc.Info != nil && w.onInfo != nil {
		wc := state.buildContext("$.info")
		result := w.handleAction(w.onInfo(wc, doc.Info))
		releaseContext(wc)
		if !result {
			if w.stopped {
				return nil
			}
		}
	}

	// ExternalDocs (root level)
	if doc.ExternalDocs != nil && w.onExternalDocs != nil {
		wc := state.buildContext("$.externalDocs")
		result := w.handleAction(w.onExternalDocs(wc, doc.ExternalDocs))
		releaseContext(wc)
		if !result {
			if w.stopped {
				return nil
			}
		}
	}

	// Paths
	if doc.Paths != nil {
		if err := w.walkOAS2Paths(doc.Paths, "$.paths", state); err != nil {
			return err
		}
	}

	// Definitions (schemas) - these are components in OAS 2.0
	if doc.Definitions != nil {
		defState := state.clone()
		defState.isComponent = true

		for _, name := range maputil.SortedKeys(doc.Definitions) {
			if w.stopped {
				return nil
			}
			schema := doc.Definitions[name]
			if schema != nil {
				schemaState := defState.clone()
				schemaState.name = name
				if err := w.walkSchema(schema, "$.definitions['"+name+"']", 0, schemaState); err != nil {
					return err
				}
			}
		}
	}

	// Parameters (reusable) - these are components in OAS 2.0
	if doc.Parameters != nil {
		paramState := state.clone()
		paramState.isComponent = true

		for _, name := range maputil.SortedKeys(doc.Parameters) {
			if w.stopped {
				return nil
			}
			param := doc.Parameters[name]
			if param != nil {
				pState := paramState.clone()
				pState.name = name
				if err := w.walkParameter(param, "$.parameters['"+name+"']", pState); err != nil {
					return err
				}
			}
		}
	}

	// Responses (reusable) - these are components in OAS 2.0
	if doc.Responses != nil {
		respState := state.clone()
		respState.isComponent = true

		for _, name := range maputil.SortedKeys(doc.Responses) {
			if w.stopped {
				return nil
			}
			resp := doc.Responses[name]
			if resp != nil {
				rState := respState.clone()
				rState.name = name
				if err := w.walkOAS2Response(resp, "$.responses['"+name+"']", rState); err != nil {
					return err
				}
			}
		}
	}

	// SecurityDefinitions - these are components in OAS 2.0
	if doc.SecurityDefinitions != nil {
		ssState := state.clone()
		ssState.isComponent = true

		for _, name := range maputil.SortedKeys(doc.SecurityDefinitions) {
			if w.stopped {
				return nil
			}
			ss := doc.SecurityDefinitions[name]
			if ss == nil {
				continue
			}

			ssPath := "$.securityDefinitions['" + name + "']"
			sState := ssState.clone()
			sState.name = name

			// Check for $ref
			if w.handleRef(ss.Ref, ssPath, RefNodeSecurityScheme, sState) == Stop {
				return nil
			}

			if w.onSecurityScheme != nil {
				wc := sState.buildContext(ssPath)
				w.handleAction(w.onSecurityScheme(wc, ss))
				releaseContext(wc)
			}
		}
	}

	// Tags
	for i, tag := range doc.Tags {
		if w.stopped {
			return nil
		}
		if tag != nil && w.onTag != nil {
			wc := state.buildContext(fmt.Sprintf("$.tags[%d]", i))
			w.handleAction(w.onTag(wc, tag))
			releaseContext(wc)
		}
	}

	// Call document post handler after all children have been processed
	if w.onOAS2DocumentPost != nil && !w.stopped {
		wc := state.buildContext("$")
		w.onOAS2DocumentPost(wc, doc)
		releaseContext(wc)
	}

	return nil
}

// walkOAS2Paths walks all paths in sorted order.
func (w *Walker) walkOAS2Paths(paths parser.Paths, basePath string, state *walkState) error {
	pathKeys := maputil.SortedKeys(paths)
	for _, pathTemplate := range pathKeys {
		if w.stopped {
			return nil
		}
		pathItem := paths[pathTemplate]
		if pathItem == nil {
			continue
		}

		itemPath := basePath + "['" + pathTemplate + "']"

		// Create state with pathTemplate set
		pathState := state.clone()
		pathState.pathTemplate = pathTemplate

		// Path handler
		continueToChildren := true
		if w.onPath != nil {
			wc := pathState.buildContext(itemPath)
			continueToChildren = w.handleAction(w.onPath(wc, pathItem))
			releaseContext(wc)
			if w.stopped {
				return nil
			}
		}

		if continueToChildren {
			if err := w.walkOAS2PathItem(pathItem, itemPath, pathState); err != nil {
				return err
			}
		}
	}
	return nil
}

// walkOAS2PathItem walks a single PathItem.
func (w *Walker) walkOAS2PathItem(pathItem *parser.PathItem, basePath string, state *walkState) error {
	// Check for $ref
	if w.handleRef(pathItem.Ref, basePath, RefNodePathItem, state) == Stop {
		return nil
	}

	// PathItem pre-visit handler
	continueToChildren := true
	if w.onPathItem != nil {
		wc := state.buildContext(basePath)
		continueToChildren = w.handleAction(w.onPathItem(wc, pathItem))
		releaseContext(wc)
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil // SkipChildren - don't call post handler
	}

	// Push path item as parent for nested nodes
	state.pushParent(pathItem, basePath)
	defer state.popParent()

	// PathItem-level parameters
	for i, param := range pathItem.Parameters {
		if w.stopped {
			return nil
		}
		if err := w.walkParameter(param, fmt.Sprintf("%s.parameters[%d]", basePath, i), state); err != nil {
			return err
		}
	}

	// Operations (OAS 2.0 doesn't have trace or query)
	ops := []struct {
		method string
		op     *parser.Operation
	}{
		{"get", pathItem.Get},
		{"put", pathItem.Put},
		{"post", pathItem.Post},
		{"delete", pathItem.Delete},
		{"options", pathItem.Options},
		{"head", pathItem.Head},
		{"patch", pathItem.Patch},
	}

	for _, item := range ops {
		if w.stopped {
			return nil
		}
		if item.op != nil {
			opState := state.clone()
			opState.method = item.method
			if err := w.walkOAS2Operation(item.op, basePath+"."+item.method, opState); err != nil {
				return err
			}
		}
	}

	// Call post-visit handler after children (but before popParent)
	if w.onPathItemPost != nil && !w.stopped {
		wc := state.buildContext(basePath)
		w.onPathItemPost(wc, pathItem)
		releaseContext(wc)
	}

	return nil
}

// walkOAS2Operation walks a single Operation.
func (w *Walker) walkOAS2Operation(op *parser.Operation, basePath string, state *walkState) error {
	// Operation pre-visit handler
	continueToChildren := true
	if w.onOperation != nil {
		wc := state.buildContext(basePath)
		continueToChildren = w.handleAction(w.onOperation(wc, op))
		releaseContext(wc)
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil // SkipChildren - don't call post handler
	}

	// Push operation as parent for nested nodes
	state.pushParent(op, basePath)
	defer state.popParent()

	// ExternalDocs
	if op.ExternalDocs != nil && w.onExternalDocs != nil {
		wc := state.buildContext(basePath + ".externalDocs")
		w.handleAction(w.onExternalDocs(wc, op.ExternalDocs))
		releaseContext(wc)
		if w.stopped {
			return nil
		}
	}

	// Parameters
	for i, param := range op.Parameters {
		if w.stopped {
			return nil
		}
		if err := w.walkParameter(param, fmt.Sprintf("%s.parameters[%d]", basePath, i), state); err != nil {
			return err
		}
	}

	// Responses
	if op.Responses != nil {
		if err := w.walkOAS2Responses(op.Responses, basePath+".responses", state); err != nil {
			return err
		}
	}

	// Call post-visit handler after children (but before popParent)
	if w.onOperationPost != nil && !w.stopped {
		wc := state.buildContext(basePath)
		w.onOperationPost(wc, op)
		releaseContext(wc)
	}

	return nil
}

// walkOAS2Responses walks Responses.
func (w *Walker) walkOAS2Responses(responses *parser.Responses, basePath string, state *walkState) error {
	// Default response
	if responses.Default != nil {
		respState := state.clone()
		respState.statusCode = "default"
		if err := w.walkOAS2Response(responses.Default, basePath+".default", respState); err != nil {
			return err
		}
	}

	// Status code responses - using Codes
	if responses.Codes != nil {
		for _, code := range maputil.SortedKeys(responses.Codes) {
			if w.stopped {
				return nil
			}
			resp := responses.Codes[code]
			if resp != nil {
				respState := state.clone()
				respState.statusCode = code
				if err := w.walkOAS2Response(resp, basePath+"['"+code+"']", respState); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// walkOAS2Response walks a single Response (OAS 2.0 style).
func (w *Walker) walkOAS2Response(resp *parser.Response, basePath string, state *walkState) error {
	// Check for $ref
	if w.handleRef(resp.Ref, basePath, RefNodeResponse, state) == Stop {
		return nil
	}

	// Response pre-visit handler
	continueToChildren := true
	if w.onResponse != nil {
		wc := state.buildContext(basePath)
		continueToChildren = w.handleAction(w.onResponse(wc, resp))
		releaseContext(wc)
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil // SkipChildren - don't call post handler
	}

	// Push response as parent for nested nodes
	state.pushParent(resp, basePath)
	defer state.popParent()

	// Headers
	if resp.Headers != nil {
		if err := w.walkHeaders(resp.Headers, basePath+".headers", state); err != nil {
			return err
		}
	}

	// Schema (OAS 2.0 uses schema directly, not content)
	if resp.Schema != nil {
		schemaState := state.clone()
		schemaState.name = "" // Clear name for nested schemas
		if err := w.walkSchema(resp.Schema, basePath+".schema", 0, schemaState); err != nil {
			return err
		}
	}

	// Call post-visit handler after children (but before popParent)
	if w.onResponsePost != nil && !w.stopped {
		wc := state.buildContext(basePath)
		w.onResponsePost(wc, resp)
		releaseContext(wc)
	}

	return nil
}
