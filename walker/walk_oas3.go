package walker

import (
	"fmt"
	"sort"

	"github.com/erraggy/oastools/parser"
)

// walkOAS3 traverses an OAS 3.x document.
func (w *Walker) walkOAS3(doc *parser.OAS3Document, state *walkState) error {
	// Document root - typed handler first, then generic
	// Track whether to continue to children separately from stopping
	continueToChildren := true
	if w.onOAS3Document != nil {
		wc := state.buildContext("$")
		if !w.handleAction(w.onOAS3Document(wc, doc)) {
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

	// Servers
	for i, server := range doc.Servers {
		if w.stopped {
			return nil
		}
		if server != nil && w.onServer != nil {
			wc := state.buildContext(fmt.Sprintf("$.servers[%d]", i))
			w.handleAction(w.onServer(wc, server))
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
		}
	}

	return nil
}

// walkOAS3Paths walks all paths in sorted order.
func (w *Walker) walkOAS3Paths(paths parser.Paths, basePath string, state *walkState) error {
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
			if err := w.walkOAS3PathItem(pathItem, itemPath, pathState); err != nil {
				return err
			}
		}
	}
	return nil
}

// walkOAS3Webhooks walks webhooks (OAS 3.1+).
func (w *Walker) walkOAS3Webhooks(webhooks map[string]*parser.PathItem, basePath string, state *walkState) error {
	webhookKeys := make([]string, 0, len(webhooks))
	for k := range webhooks {
		webhookKeys = append(webhookKeys, k)
	}
	sort.Strings(webhookKeys)

	for _, name := range webhookKeys {
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

	// Operations
	return w.walkOAS3PathItemOperations(pathItem, basePath, state)
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
		addOpKeys := make([]string, 0, len(pathItem.AdditionalOperations))
		for k := range pathItem.AdditionalOperations {
			addOpKeys = append(addOpKeys, k)
		}
		sort.Strings(addOpKeys)

		for _, method := range addOpKeys {
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
		callbackKeys := make([]string, 0, len(op.Callbacks))
		for k := range op.Callbacks {
			callbackKeys = append(callbackKeys, k)
		}
		sort.Strings(callbackKeys)

		for _, name := range callbackKeys {
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

	return nil
}

// walkOAS3RequestBody walks a RequestBody.
func (w *Walker) walkOAS3RequestBody(reqBody *parser.RequestBody, basePath string, state *walkState) error {
	continueToChildren := true
	if w.onRequestBody != nil {
		wc := state.buildContext(basePath)
		continueToChildren = w.handleAction(w.onRequestBody(wc, reqBody))
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil
	}

	// Content
	return w.walkContent(reqBody.Content, basePath+".content", state)
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

	// Content
	if resp.Content != nil {
		if err := w.walkContent(resp.Content, basePath+".content", state); err != nil {
			return err
		}
	}

	// Links
	if resp.Links != nil {
		linkKeys := make([]string, 0, len(resp.Links))
		for k := range resp.Links {
			linkKeys = append(linkKeys, k)
		}
		sort.Strings(linkKeys)

		for _, name := range linkKeys {
			if w.stopped {
				return nil
			}
			link := resp.Links[name]
			if link != nil && w.onLink != nil {
				linkState := state.clone()
				linkState.name = name
				wc := linkState.buildContext(basePath + ".links['" + name + "']")
				w.handleAction(w.onLink(wc, link))
			}
		}
	}

	return nil
}

// walkOAS3Callback walks a Callback.
func (w *Walker) walkOAS3Callback(name string, callback parser.Callback, basePath string, state *walkState) error {
	// Callback handler
	cbState := state.clone()
	cbState.name = name

	continueToChildren := true
	if w.onCallback != nil {
		wc := cbState.buildContext(basePath)
		continueToChildren = w.handleAction(w.onCallback(wc, callback))
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil
	}

	// Callback is map[string]*PathItem
	exprKeys := make([]string, 0, len(callback))
	for k := range callback {
		exprKeys = append(exprKeys, k)
	}
	sort.Strings(exprKeys)

	for _, expr := range exprKeys {
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
	for _, name := range sortedMapKeys(components.Schemas) {
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
	for _, name := range sortedMapKeys(components.Responses) {
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
	for _, name := range sortedMapKeys(components.Parameters) {
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
	for _, name := range sortedMapKeys(components.RequestBodies) {
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
	for _, name := range sortedMapKeys(components.SecuritySchemes) {
		if w.stopped {
			return nil
		}
		if ss := components.SecuritySchemes[name]; ss != nil && w.onSecurityScheme != nil {
			ssState := state.clone()
			ssState.name = name
			wc := ssState.buildContext(basePath + ".securitySchemes['" + name + "']")
			w.handleAction(w.onSecurityScheme(wc, ss))
		}
	}
	return nil
}

func (w *Walker) walkComponentLinks(components *parser.Components, basePath string, state *walkState) error {
	if components.Links == nil {
		return nil
	}
	for _, name := range sortedMapKeys(components.Links) {
		if w.stopped {
			return nil
		}
		if link := components.Links[name]; link != nil && w.onLink != nil {
			linkState := state.clone()
			linkState.name = name
			wc := linkState.buildContext(basePath + ".links['" + name + "']")
			w.handleAction(w.onLink(wc, link))
		}
	}
	return nil
}

func (w *Walker) walkComponentCallbacks(components *parser.Components, basePath string, state *walkState) error {
	if components.Callbacks == nil {
		return nil
	}
	for _, name := range sortedMapKeys(components.Callbacks) {
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
	for _, name := range sortedMapKeys(components.Examples) {
		if w.stopped {
			return nil
		}
		if ex := components.Examples[name]; ex != nil && w.onExample != nil {
			exState := state.clone()
			exState.name = name
			wc := exState.buildContext(basePath + ".examples['" + name + "']")
			w.handleAction(w.onExample(wc, ex))
		}
	}
	return nil
}

func (w *Walker) walkComponentPathItems(components *parser.Components, basePath string, state *walkState) error {
	if components.PathItems == nil {
		return nil
	}
	for _, name := range sortedMapKeys(components.PathItems) {
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
