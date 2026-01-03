package walker

import (
	"fmt"
	"sort"

	"github.com/erraggy/oastools/parser"
)

// walkOAS2 traverses an OAS 2.0 document.
func (w *Walker) walkOAS2(doc *parser.OAS2Document, state *walkState) error {
	// Document root - typed handler first, then generic
	// Track whether to continue to children separately from stopping
	continueToChildren := true
	if w.onOAS2Document != nil {
		wc := state.buildContext("$")
		if !w.handleAction(w.onOAS2Document(wc, doc)) {
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
		if !w.handleAction(w.onDocument(wc, doc)) {
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
		if !w.handleAction(w.onInfo(wc, doc.Info)) {
			if w.stopped {
				return nil
			}
		}
	}

	// ExternalDocs (root level)
	if doc.ExternalDocs != nil && w.onExternalDocs != nil {
		wc := state.buildContext("$.externalDocs")
		if !w.handleAction(w.onExternalDocs(wc, doc.ExternalDocs)) {
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

		defKeys := make([]string, 0, len(doc.Definitions))
		for k := range doc.Definitions {
			defKeys = append(defKeys, k)
		}
		sort.Strings(defKeys)

		for _, name := range defKeys {
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

		paramKeys := make([]string, 0, len(doc.Parameters))
		for k := range doc.Parameters {
			paramKeys = append(paramKeys, k)
		}
		sort.Strings(paramKeys)

		for _, name := range paramKeys {
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

		respKeys := make([]string, 0, len(doc.Responses))
		for k := range doc.Responses {
			respKeys = append(respKeys, k)
		}
		sort.Strings(respKeys)

		for _, name := range respKeys {
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

		ssKeys := make([]string, 0, len(doc.SecurityDefinitions))
		for k := range doc.SecurityDefinitions {
			ssKeys = append(ssKeys, k)
		}
		sort.Strings(ssKeys)

		for _, name := range ssKeys {
			if w.stopped {
				return nil
			}
			ss := doc.SecurityDefinitions[name]
			if ss != nil && w.onSecurityScheme != nil {
				sState := ssState.clone()
				sState.name = name
				wc := sState.buildContext("$.securityDefinitions['" + name + "']")
				w.handleAction(w.onSecurityScheme(wc, ss))
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
		}
	}

	return nil
}

// walkOAS2Paths walks all paths in sorted order.
func (w *Walker) walkOAS2Paths(paths parser.Paths, basePath string, state *walkState) error {
	pathKeys := sortedMapKeys(paths)
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
	// PathItem handler
	continueToChildren := true
	if w.onPathItem != nil {
		wc := state.buildContext(basePath)
		continueToChildren = w.handleAction(w.onPathItem(wc, pathItem))
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil
	}

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

	return nil
}

// walkOAS2Operation walks a single Operation.
func (w *Walker) walkOAS2Operation(op *parser.Operation, basePath string, state *walkState) error {
	// Operation handler
	continueToChildren := true
	if w.onOperation != nil {
		wc := state.buildContext(basePath)
		continueToChildren = w.handleAction(w.onOperation(wc, op))
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil
	}

	// ExternalDocs
	if op.ExternalDocs != nil && w.onExternalDocs != nil {
		wc := state.buildContext(basePath + ".externalDocs")
		w.handleAction(w.onExternalDocs(wc, op.ExternalDocs))
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
		codeKeys := make([]string, 0, len(responses.Codes))
		for k := range responses.Codes {
			codeKeys = append(codeKeys, k)
		}
		sort.Strings(codeKeys)

		for _, code := range codeKeys {
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
	continueToChildren := true
	if w.onResponse != nil {
		wc := state.buildContext(basePath)
		continueToChildren = w.handleAction(w.onResponse(wc, resp))
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil
	}

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

	return nil
}
