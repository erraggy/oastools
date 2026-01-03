package walker

import (
	"fmt"
	"sort"

	"github.com/erraggy/oastools/parser"
)

// walkOAS3 traverses an OAS 3.x document.
func (w *Walker) walkOAS3(doc *parser.OAS3Document) error {
	// Document root
	if w.onDocument != nil {
		if !w.handleAction(w.onDocument(doc, "$")) {
			if w.stopped {
				return nil
			}
		}
	}

	// Info
	if doc.Info != nil && w.onInfo != nil {
		if !w.handleAction(w.onInfo(doc.Info, "$.info")) {
			if w.stopped {
				return nil
			}
		}
	}

	// ExternalDocs (root level)
	if doc.ExternalDocs != nil && w.onExternalDocs != nil {
		if !w.handleAction(w.onExternalDocs(doc.ExternalDocs, "$.externalDocs")) {
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
			w.handleAction(w.onServer(server, fmt.Sprintf("$.servers[%d]", i)))
		}
	}

	// Paths
	if doc.Paths != nil {
		if err := w.walkOAS3Paths(doc.Paths, "$.paths"); err != nil {
			return err
		}
	}

	// Webhooks (OAS 3.1+)
	if doc.Webhooks != nil {
		if err := w.walkOAS3Webhooks(doc.Webhooks, "$.webhooks"); err != nil {
			return err
		}
	}

	// Components
	if doc.Components != nil {
		if err := w.walkOAS3Components(doc.Components, "$.components"); err != nil {
			return err
		}
	}

	// Tags
	for i, tag := range doc.Tags {
		if w.stopped {
			return nil
		}
		if tag != nil && w.onTag != nil {
			w.handleAction(w.onTag(tag, fmt.Sprintf("$.tags[%d]", i)))
		}
	}

	return nil
}

// walkOAS3Paths walks all paths in sorted order.
func (w *Walker) walkOAS3Paths(paths parser.Paths, basePath string) error {
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

		// Path handler
		continueToChildren := true
		if w.onPath != nil {
			continueToChildren = w.handleAction(w.onPath(pathTemplate, pathItem, itemPath))
			if w.stopped {
				return nil
			}
		}

		if continueToChildren {
			if err := w.walkOAS3PathItem(pathItem, itemPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// walkOAS3Webhooks walks webhooks (OAS 3.1+).
func (w *Walker) walkOAS3Webhooks(webhooks map[string]*parser.PathItem, basePath string) error {
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

		// PathItem handler for webhook
		continueToChildren := true
		if w.onPathItem != nil {
			continueToChildren = w.handleAction(w.onPathItem(pathItem, itemPath))
			if w.stopped {
				return nil
			}
		}

		if continueToChildren {
			if err := w.walkOAS3PathItemOperations(pathItem, itemPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// walkOAS3PathItem walks a single PathItem.
func (w *Walker) walkOAS3PathItem(pathItem *parser.PathItem, basePath string) error {
	// PathItem handler
	continueToChildren := true
	if w.onPathItem != nil {
		continueToChildren = w.handleAction(w.onPathItem(pathItem, basePath))
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
		if err := w.walkParameter(param, fmt.Sprintf("%s.parameters[%d]", basePath, i)); err != nil {
			return err
		}
	}

	// Operations
	return w.walkOAS3PathItemOperations(pathItem, basePath)
}

// walkOAS3PathItemOperations walks all operations in a PathItem.
func (w *Walker) walkOAS3PathItemOperations(pathItem *parser.PathItem, basePath string) error {
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
			if err := w.walkOAS3Operation(item.method, item.op, basePath+"."+item.method); err != nil {
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
				if err := w.walkOAS3Operation(method, op, basePath+".additionalOperations."+method); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// walkOAS3Operation walks a single Operation.
func (w *Walker) walkOAS3Operation(method string, op *parser.Operation, basePath string) error {
	// Operation handler
	continueToChildren := true
	if w.onOperation != nil {
		continueToChildren = w.handleAction(w.onOperation(method, op, basePath))
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil
	}

	// ExternalDocs
	if op.ExternalDocs != nil && w.onExternalDocs != nil {
		w.handleAction(w.onExternalDocs(op.ExternalDocs, basePath+".externalDocs"))
		if w.stopped {
			return nil
		}
	}

	// Parameters
	for i, param := range op.Parameters {
		if w.stopped {
			return nil
		}
		if err := w.walkParameter(param, fmt.Sprintf("%s.parameters[%d]", basePath, i)); err != nil {
			return err
		}
	}

	// RequestBody
	if op.RequestBody != nil {
		if err := w.walkOAS3RequestBody(op.RequestBody, basePath+".requestBody"); err != nil {
			return err
		}
	}

	// Responses
	if op.Responses != nil {
		if err := w.walkOAS3Responses(op.Responses, basePath+".responses"); err != nil {
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
				if err := w.walkOAS3Callback(name, *callback, basePath+".callbacks['"+name+"']"); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// walkOAS3RequestBody walks a RequestBody.
func (w *Walker) walkOAS3RequestBody(reqBody *parser.RequestBody, basePath string) error {
	continueToChildren := true
	if w.onRequestBody != nil {
		continueToChildren = w.handleAction(w.onRequestBody(reqBody, basePath))
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil
	}

	// Content
	return w.walkContent(reqBody.Content, basePath+".content")
}

// walkOAS3Responses walks Responses.
func (w *Walker) walkOAS3Responses(responses *parser.Responses, basePath string) error {
	// Default response
	if responses.Default != nil {
		if err := w.walkOAS3Response("default", responses.Default, basePath+".default"); err != nil {
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
				if err := w.walkOAS3Response(code, resp, basePath+"['"+code+"']"); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// walkOAS3Response walks a single Response.
func (w *Walker) walkOAS3Response(statusCode string, resp *parser.Response, basePath string) error {
	continueToChildren := true
	if w.onResponse != nil {
		continueToChildren = w.handleAction(w.onResponse(statusCode, resp, basePath))
		if w.stopped {
			return nil
		}
	}

	if !continueToChildren {
		return nil
	}

	// Headers
	if resp.Headers != nil {
		if err := w.walkHeaders(resp.Headers, basePath+".headers"); err != nil {
			return err
		}
	}

	// Content
	if resp.Content != nil {
		if err := w.walkContent(resp.Content, basePath+".content"); err != nil {
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
				w.handleAction(w.onLink(name, link, basePath+".links['"+name+"']"))
			}
		}
	}

	return nil
}

// walkOAS3Callback walks a Callback.
func (w *Walker) walkOAS3Callback(name string, callback parser.Callback, basePath string) error {
	// Callback handler
	continueToChildren := true
	if w.onCallback != nil {
		continueToChildren = w.handleAction(w.onCallback(name, callback, basePath))
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
			if err := w.walkOAS3PathItem(pathItem, basePath+"['"+expr+"']"); err != nil {
				return err
			}
		}
	}

	return nil
}

// walkOAS3Components walks Components.
func (w *Walker) walkOAS3Components(components *parser.Components, basePath string) error {
	if err := w.walkComponentSchemas(components, basePath); err != nil {
		return err
	}
	if err := w.walkComponentResponses(components, basePath); err != nil {
		return err
	}
	if err := w.walkComponentParameters(components, basePath); err != nil {
		return err
	}
	if err := w.walkComponentRequestBodies(components, basePath); err != nil {
		return err
	}
	if err := w.walkComponentHeaders(components, basePath); err != nil {
		return err
	}
	if err := w.walkComponentSecuritySchemes(components, basePath); err != nil {
		return err
	}
	if err := w.walkComponentLinks(components, basePath); err != nil {
		return err
	}
	if err := w.walkComponentCallbacks(components, basePath); err != nil {
		return err
	}
	if err := w.walkComponentExamples(components, basePath); err != nil {
		return err
	}
	return w.walkComponentPathItems(components, basePath)
}

func (w *Walker) walkComponentSchemas(components *parser.Components, basePath string) error {
	if components.Schemas == nil {
		return nil
	}
	for _, name := range sortedMapKeys(components.Schemas) {
		if w.stopped {
			return nil
		}
		if schema := components.Schemas[name]; schema != nil {
			if err := w.walkSchema(schema, basePath+".schemas['"+name+"']", 0); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Walker) walkComponentResponses(components *parser.Components, basePath string) error {
	if components.Responses == nil {
		return nil
	}
	for _, name := range sortedMapKeys(components.Responses) {
		if w.stopped {
			return nil
		}
		if resp := components.Responses[name]; resp != nil {
			if err := w.walkOAS3Response(name, resp, basePath+".responses['"+name+"']"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Walker) walkComponentParameters(components *parser.Components, basePath string) error {
	if components.Parameters == nil {
		return nil
	}
	for _, name := range sortedMapKeys(components.Parameters) {
		if w.stopped {
			return nil
		}
		if param := components.Parameters[name]; param != nil {
			if err := w.walkParameter(param, basePath+".parameters['"+name+"']"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Walker) walkComponentRequestBodies(components *parser.Components, basePath string) error {
	if components.RequestBodies == nil {
		return nil
	}
	for _, name := range sortedMapKeys(components.RequestBodies) {
		if w.stopped {
			return nil
		}
		if rb := components.RequestBodies[name]; rb != nil {
			if err := w.walkOAS3RequestBody(rb, basePath+".requestBodies['"+name+"']"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Walker) walkComponentHeaders(components *parser.Components, basePath string) error {
	if components.Headers == nil {
		return nil
	}
	return w.walkHeaders(components.Headers, basePath+".headers")
}

func (w *Walker) walkComponentSecuritySchemes(components *parser.Components, basePath string) error {
	if components.SecuritySchemes == nil {
		return nil
	}
	for _, name := range sortedMapKeys(components.SecuritySchemes) {
		if w.stopped {
			return nil
		}
		if ss := components.SecuritySchemes[name]; ss != nil && w.onSecurityScheme != nil {
			w.handleAction(w.onSecurityScheme(name, ss, basePath+".securitySchemes['"+name+"']"))
		}
	}
	return nil
}

func (w *Walker) walkComponentLinks(components *parser.Components, basePath string) error {
	if components.Links == nil {
		return nil
	}
	for _, name := range sortedMapKeys(components.Links) {
		if w.stopped {
			return nil
		}
		if link := components.Links[name]; link != nil && w.onLink != nil {
			w.handleAction(w.onLink(name, link, basePath+".links['"+name+"']"))
		}
	}
	return nil
}

func (w *Walker) walkComponentCallbacks(components *parser.Components, basePath string) error {
	if components.Callbacks == nil {
		return nil
	}
	for _, name := range sortedMapKeys(components.Callbacks) {
		if w.stopped {
			return nil
		}
		if cb := components.Callbacks[name]; cb != nil {
			if err := w.walkOAS3Callback(name, *cb, basePath+".callbacks['"+name+"']"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Walker) walkComponentExamples(components *parser.Components, basePath string) error {
	if components.Examples == nil {
		return nil
	}
	for _, name := range sortedMapKeys(components.Examples) {
		if w.stopped {
			return nil
		}
		if ex := components.Examples[name]; ex != nil && w.onExample != nil {
			w.handleAction(w.onExample(name, ex, basePath+".examples['"+name+"']"))
		}
	}
	return nil
}

func (w *Walker) walkComponentPathItems(components *parser.Components, basePath string) error {
	if components.PathItems == nil {
		return nil
	}
	for _, name := range sortedMapKeys(components.PathItems) {
		if w.stopped {
			return nil
		}
		if pi := components.PathItems[name]; pi != nil {
			if err := w.walkOAS3PathItem(pi, basePath+".pathItems['"+name+"']"); err != nil {
				return err
			}
		}
	}
	return nil
}
