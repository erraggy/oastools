package walker

import (
	"fmt"

	"github.com/erraggy/oastools/internal/maputil"
	"github.com/erraggy/oastools/parser"
)

// walkOAS3 traverses an OAS 3.x document.
func (w *Walker) walkOAS3(doc *parser.OAS3Document, state *walkState) error {
	// Document root - typed handler first, then generic
	// Track whether to continue to children separately from stopping
	continueToChildren := true
	if w.onOAS3Document != nil {
		wc := state.buildContext("$")
		result := w.handleAction(w.onOAS3Document(wc, doc))
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

	// Servers
	for i, server := range doc.Servers {
		if w.stopped {
			return nil
		}
		if server != nil && w.onServer != nil {
			wc := state.buildContext(fmt.Sprintf("$.servers[%d]", i))
			w.handleAction(w.onServer(wc, server))
			releaseContext(wc)
		}
	}

	// Paths
	if doc.Paths != nil {
		if err := w.walkOAS3Paths(doc.Paths, "$.paths", state); err != nil {
			return err
		}
	}

	// Webhooks (OAS 3.1+)
	if doc.Webhooks != nil {
		if err := w.walkOAS3Webhooks(doc.Webhooks, "$.webhooks", state); err != nil {
			return err
		}
	}

	// Components
	if doc.Components != nil {
		compState := state.clone()
		compState.isComponent = true
		if err := w.walkOAS3Components(doc.Components, "$.components", compState); err != nil {
			return err
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
	if w.onOAS3DocumentPost != nil && !w.stopped {
		wc := state.buildContext("$")
		w.onOAS3DocumentPost(wc, doc)
		releaseContext(wc)
	}

	return nil
}

// walkOAS3Paths walks all paths in sorted order.
func (w *Walker) walkOAS3Paths(paths parser.Paths, basePath string, state *walkState) error {
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
			if err := w.walkOAS3PathItem(pathItem, itemPath, pathState); err != nil {
				return err
			}
		}
	}
	return nil
}

// walkOAS3Webhooks walks webhooks (OAS 3.1+).
func (w *Walker) walkOAS3Webhooks(webhooks map[string]*parser.PathItem, basePath string, state *walkState) error {
	for _, name := range maputil.SortedKeys(webhooks) {
		if w.stopped {
			return nil
		}
		pathItem := webhooks[name]
		if pathItem == nil {
			continue
		}

		itemPath := basePath + "['" + name + "']"

		// Webhook name goes in pathTemplate (it's the event name, similar to a path)
		webhookState := state.clone()
		webhookState.pathTemplate = name

		// PathItem handler for webhook
		continueToChildren := true
		if w.onPathItem != nil {
			wc := webhookState.buildContext(itemPath)
			continueToChildren = w.handleAction(w.onPathItem(wc, pathItem))
			releaseContext(wc)
			if w.stopped {
				return nil
			}
		}

		if continueToChildren {
			if err := w.walkOAS3PathItemOperations(pathItem, itemPath, webhookState); err != nil {
				return err
			}
		}
	}
	return nil
}

// walkOAS3PathItem walks a single PathItem.
func (w *Walker) walkOAS3PathItem(pathItem *parser.PathItem, basePath string, state *walkState) error {
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

	// Operations
	if err := w.walkOAS3PathItemOperations(pathItem, basePath, state); err != nil {
		return err
	}

	// Call post-visit handler after children (but before popParent)
	if w.onPathItemPost != nil && !w.stopped {
		wc := state.buildContext(basePath)
		w.onPathItemPost(wc, pathItem)
		releaseContext(wc)
	}

	return nil
}

// walkOAS3PathItemOperations walks all operations in a PathItem.
func (w *Walker) walkOAS3PathItemOperations(pathItem *parser.PathItem, basePath string, state *walkState) error {
	// Standard HTTP methods
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
		{"trace", pathItem.Trace},
		{"query", pathItem.Query}, // OAS 3.2+
	}

	for _, item := range ops {
		if w.stopped {
			return nil
		}
		if item.op != nil {
			opState := state.clone()
			opState.method = item.method
			if err := w.walkOAS3Operation(item.op, basePath+"."+item.method, opState); err != nil {
				return err
			}
		}
	}

	// Additional operations (OAS 3.2+)
	if pathItem.AdditionalOperations != nil {
		for _, method := range maputil.SortedKeys(pathItem.AdditionalOperations) {
			if w.stopped {
				return nil
			}
			op := pathItem.AdditionalOperations[method]
			if op != nil {
				opState := state.clone()
				opState.method = method
				if err := w.walkOAS3Operation(op, basePath+".additionalOperations."+method, opState); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// walkOAS3Operation walks a single Operation.
func (w *Walker) walkOAS3Operation(op *parser.Operation, basePath string, state *walkState) error {
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

	// RequestBody
	if op.RequestBody != nil {
		if err := w.walkOAS3RequestBody(op.RequestBody, basePath+".requestBody", state); err != nil {
			return err
		}
	}

	// Responses
	if op.Responses != nil {
		if err := w.walkOAS3Responses(op.Responses, basePath+".responses", state); err != nil {
			return err
		}
	}

	// Callbacks
	if op.Callbacks != nil {
		for _, name := range maputil.SortedKeys(op.Callbacks) {
			if w.stopped {
				return nil
			}
			callback := op.Callbacks[name]
			if callback != nil {
				if err := w.walkOAS3Callback(name, *callback, basePath+".callbacks['"+name+"']", state); err != nil {
					return err
				}
			}
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

// walkOAS3RequestBody walks a RequestBody.
func (w *Walker) walkOAS3RequestBody(reqBody *parser.RequestBody, basePath string, state *walkState) error {
	// Check for $ref
	if w.handleRef(reqBody.Ref, basePath, RefNodeRequestBody, state) == Stop {
		return nil
	}

	// RequestBody pre-visit handler
	continueToChildren := true
	if w.onRequestBody != nil {
		wc := state.buildContext(basePath)
		continueToChildren = w.handleAction(w.onRequestBody(wc, reqBody))
		releaseContext(wc)
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil // SkipChildren - don't call post handler
	}

	// Push request body as parent for nested nodes
	state.pushParent(reqBody, basePath)
	defer state.popParent()

	// Content
	if err := w.walkContent(reqBody.Content, basePath+".content", state); err != nil {
		return err
	}

	// Call post-visit handler after children (but before popParent)
	if w.onRequestBodyPost != nil && !w.stopped {
		wc := state.buildContext(basePath)
		w.onRequestBodyPost(wc, reqBody)
		releaseContext(wc)
	}

	return nil
}

// walkOAS3Responses walks Responses.
func (w *Walker) walkOAS3Responses(responses *parser.Responses, basePath string, state *walkState) error {
	// Default response
	if responses.Default != nil {
		respState := state.clone()
		respState.statusCode = "default"
		if err := w.walkOAS3Response(responses.Default, basePath+".default", respState); err != nil {
			return err
		}
	}

	// Status code responses - using Codes (not StatusCodes!)
	if responses.Codes != nil {
		for _, code := range maputil.SortedKeys(responses.Codes) {
			if w.stopped {
				return nil
			}
			resp := responses.Codes[code]
			if resp != nil {
				respState := state.clone()
				respState.statusCode = code
				if err := w.walkOAS3Response(resp, basePath+"['"+code+"']", respState); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// walkOAS3Response walks a single Response.
func (w *Walker) walkOAS3Response(resp *parser.Response, basePath string, state *walkState) error {
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

	// Content
	if resp.Content != nil {
		if err := w.walkContent(resp.Content, basePath+".content", state); err != nil {
			return err
		}
	}

	// Links
	if resp.Links != nil {
		for _, name := range maputil.SortedKeys(resp.Links) {
			if w.stopped {
				return nil
			}
			link := resp.Links[name]
			if link == nil {
				continue
			}

			linkPath := basePath + ".links['" + name + "']"
			linkState := state.clone()
			linkState.name = name

			// Check for $ref
			if w.handleRef(link.Ref, linkPath, RefNodeLink, linkState) == Stop {
				return nil
			}

			if w.onLink != nil {
				wc := linkState.buildContext(linkPath)
				w.handleAction(w.onLink(wc, link))
				releaseContext(wc)
			}
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

// walkOAS3Callback walks a Callback.
func (w *Walker) walkOAS3Callback(name string, callback parser.Callback, basePath string, state *walkState) error {
	// Callback pre-visit handler
	cbState := state.clone()
	cbState.name = name

	continueToChildren := true
	if w.onCallback != nil {
		wc := cbState.buildContext(basePath)
		continueToChildren = w.handleAction(w.onCallback(wc, callback))
		releaseContext(wc)
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil // SkipChildren - don't call post handler
	}

	// Note: We don't push callback as parent since it's a map type, not a struct pointer.
	// The PathItems within the callback will have their own parent tracking.

	// Callback is map[string]*PathItem
	for _, expr := range maputil.SortedKeys(callback) {
		if w.stopped {
			return nil
		}
		pathItem := callback[expr]
		if pathItem != nil {
			// Expression becomes the pathTemplate within the callback
			exprState := cbState.clone()
			exprState.pathTemplate = expr
			if err := w.walkOAS3PathItem(pathItem, basePath+"['"+expr+"']", exprState); err != nil {
				return err
			}
		}
	}

	// Call post-visit handler after children
	if w.onCallbackPost != nil && !w.stopped {
		wc := cbState.buildContext(basePath)
		w.onCallbackPost(wc, callback)
		releaseContext(wc)
	}

	return nil
}

// walkOAS3Components walks Components.
func (w *Walker) walkOAS3Components(components *parser.Components, basePath string, state *walkState) error {
	if err := w.walkComponentSchemas(components, basePath, state); err != nil {
		return err
	}
	if err := w.walkComponentResponses(components, basePath, state); err != nil {
		return err
	}
	if err := w.walkComponentParameters(components, basePath, state); err != nil {
		return err
	}
	if err := w.walkComponentRequestBodies(components, basePath, state); err != nil {
		return err
	}
	if err := w.walkComponentHeaders(components, basePath, state); err != nil {
		return err
	}
	if err := w.walkComponentSecuritySchemes(components, basePath, state); err != nil {
		return err
	}
	if err := w.walkComponentLinks(components, basePath, state); err != nil {
		return err
	}
	if err := w.walkComponentCallbacks(components, basePath, state); err != nil {
		return err
	}
	if err := w.walkComponentExamples(components, basePath, state); err != nil {
		return err
	}
	return w.walkComponentPathItems(components, basePath, state)
}

func (w *Walker) walkComponentSchemas(components *parser.Components, basePath string, state *walkState) error {
	if components.Schemas == nil {
		return nil
	}
	for _, name := range maputil.SortedKeys(components.Schemas) {
		if w.stopped {
			return nil
		}
		if schema := components.Schemas[name]; schema != nil {
			schemaState := state.clone()
			schemaState.name = name
			if err := w.walkSchema(schema, basePath+".schemas['"+name+"']", 0, schemaState); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Walker) walkComponentResponses(components *parser.Components, basePath string, state *walkState) error {
	if components.Responses == nil {
		return nil
	}
	for _, name := range maputil.SortedKeys(components.Responses) {
		if w.stopped {
			return nil
		}
		if resp := components.Responses[name]; resp != nil {
			respState := state.clone()
			respState.name = name
			// For component responses, statusCode is not set (it's a reusable response)
			if err := w.walkOAS3Response(resp, basePath+".responses['"+name+"']", respState); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Walker) walkComponentParameters(components *parser.Components, basePath string, state *walkState) error {
	if components.Parameters == nil {
		return nil
	}
	for _, name := range maputil.SortedKeys(components.Parameters) {
		if w.stopped {
			return nil
		}
		if param := components.Parameters[name]; param != nil {
			paramState := state.clone()
			paramState.name = name
			if err := w.walkParameter(param, basePath+".parameters['"+name+"']", paramState); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Walker) walkComponentRequestBodies(components *parser.Components, basePath string, state *walkState) error {
	if components.RequestBodies == nil {
		return nil
	}
	for _, name := range maputil.SortedKeys(components.RequestBodies) {
		if w.stopped {
			return nil
		}
		if rb := components.RequestBodies[name]; rb != nil {
			rbState := state.clone()
			rbState.name = name
			if err := w.walkOAS3RequestBody(rb, basePath+".requestBodies['"+name+"']", rbState); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Walker) walkComponentHeaders(components *parser.Components, basePath string, state *walkState) error {
	if components.Headers == nil {
		return nil
	}
	return w.walkHeaders(components.Headers, basePath+".headers", state)
}

func (w *Walker) walkComponentSecuritySchemes(components *parser.Components, basePath string, state *walkState) error {
	if components.SecuritySchemes == nil {
		return nil
	}
	for _, name := range maputil.SortedKeys(components.SecuritySchemes) {
		if w.stopped {
			return nil
		}
		ss := components.SecuritySchemes[name]
		if ss == nil {
			continue
		}

		ssPath := basePath + ".securitySchemes['" + name + "']"
		ssState := state.clone()
		ssState.name = name

		// Check for $ref
		if w.handleRef(ss.Ref, ssPath, RefNodeSecurityScheme, ssState) == Stop {
			return nil
		}

		if w.onSecurityScheme != nil {
			wc := ssState.buildContext(ssPath)
			w.handleAction(w.onSecurityScheme(wc, ss))
			releaseContext(wc)
		}
	}
	return nil
}

func (w *Walker) walkComponentLinks(components *parser.Components, basePath string, state *walkState) error {
	if components.Links == nil {
		return nil
	}
	for _, name := range maputil.SortedKeys(components.Links) {
		if w.stopped {
			return nil
		}
		link := components.Links[name]
		if link == nil {
			continue
		}

		linkPath := basePath + ".links['" + name + "']"
		linkState := state.clone()
		linkState.name = name

		// Check for $ref
		if w.handleRef(link.Ref, linkPath, RefNodeLink, linkState) == Stop {
			return nil
		}

		if w.onLink != nil {
			wc := linkState.buildContext(linkPath)
			w.handleAction(w.onLink(wc, link))
			releaseContext(wc)
		}
	}
	return nil
}

func (w *Walker) walkComponentCallbacks(components *parser.Components, basePath string, state *walkState) error {
	if components.Callbacks == nil {
		return nil
	}
	for _, name := range maputil.SortedKeys(components.Callbacks) {
		if w.stopped {
			return nil
		}
		if cb := components.Callbacks[name]; cb != nil {
			if err := w.walkOAS3Callback(name, *cb, basePath+".callbacks['"+name+"']", state); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Walker) walkComponentExamples(components *parser.Components, basePath string, state *walkState) error {
	if components.Examples == nil {
		return nil
	}
	for _, name := range maputil.SortedKeys(components.Examples) {
		if w.stopped {
			return nil
		}
		ex := components.Examples[name]
		if ex == nil {
			continue
		}

		exPath := basePath + ".examples['" + name + "']"
		exState := state.clone()
		exState.name = name

		// Check for $ref
		if w.handleRef(ex.Ref, exPath, RefNodeExample, exState) == Stop {
			return nil
		}

		if w.onExample != nil {
			wc := exState.buildContext(exPath)
			w.handleAction(w.onExample(wc, ex))
			releaseContext(wc)
		}
	}
	return nil
}

func (w *Walker) walkComponentPathItems(components *parser.Components, basePath string, state *walkState) error {
	if components.PathItems == nil {
		return nil
	}
	for _, name := range maputil.SortedKeys(components.PathItems) {
		if w.stopped {
			return nil
		}
		if pi := components.PathItems[name]; pi != nil {
			piState := state.clone()
			piState.name = name
			if err := w.walkOAS3PathItem(pi, basePath+".pathItems['"+name+"']", piState); err != nil {
				return err
			}
		}
	}
	return nil
}
