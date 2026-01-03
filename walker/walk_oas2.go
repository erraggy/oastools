package walker

import (
	"fmt"
	"sort"

	"github.com/erraggy/oastools/parser"
)

// walkOAS2 traverses an OAS 2.0 document.
func (w *Walker) walkOAS2(doc *parser.OAS2Document) error {
	// Document root - typed handler first, then generic
	// Track whether to continue to children separately from stopping
	continueToChildren := true
	if w.onOAS2Document != nil {
		if !w.handleAction(w.onOAS2Document(doc, "$")) {
			if w.stopped {
				return nil
			}
			continueToChildren = false
		}
	}
	// Generic handler is called even if typed handler returned SkipChildren
	// but not if it returned Stop
	if w.onDocument != nil {
		if !w.handleAction(w.onDocument(doc, "$")) {
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

	// Paths
	if doc.Paths != nil {
		if err := w.walkOAS2Paths(doc.Paths, "$.paths"); err != nil {
			return err
		}
	}

	// Definitions (schemas)
	if doc.Definitions != nil {
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
				if err := w.walkSchema(schema, "$.definitions['"+name+"']", 0); err != nil {
					return err
				}
			}
		}
	}

	// Parameters (reusable)
	if doc.Parameters != nil {
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
				if err := w.walkParameter(param, "$.parameters['"+name+"']"); err != nil {
					return err
				}
			}
		}
	}

	// Responses (reusable)
	if doc.Responses != nil {
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
				if err := w.walkOAS2Response(name, resp, "$.responses['"+name+"']"); err != nil {
					return err
				}
			}
		}
	}

	// SecurityDefinitions
	if doc.SecurityDefinitions != nil {
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
				w.handleAction(w.onSecurityScheme(name, ss, "$.securityDefinitions['"+name+"']"))
			}
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

// walkOAS2Paths walks all paths in sorted order.
func (w *Walker) walkOAS2Paths(paths parser.Paths, basePath string) error {
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
			if err := w.walkOAS2PathItem(pathItem, itemPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// walkOAS2PathItem walks a single PathItem.
func (w *Walker) walkOAS2PathItem(pathItem *parser.PathItem, basePath string) error {
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
			if err := w.walkOAS2Operation(item.method, item.op, basePath+"."+item.method); err != nil {
				return err
			}
		}
	}

	return nil
}

// walkOAS2Operation walks a single Operation.
func (w *Walker) walkOAS2Operation(method string, op *parser.Operation, basePath string) error {
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

	// Responses
	if op.Responses != nil {
		if err := w.walkOAS2Responses(op.Responses, basePath+".responses"); err != nil {
			return err
		}
	}

	return nil
}

// walkOAS2Responses walks Responses.
func (w *Walker) walkOAS2Responses(responses *parser.Responses, basePath string) error {
	// Default response
	if responses.Default != nil {
		if err := w.walkOAS2Response("default", responses.Default, basePath+".default"); err != nil {
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
				if err := w.walkOAS2Response(code, resp, basePath+"['"+code+"']"); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// walkOAS2Response walks a single Response (OAS 2.0 style).
func (w *Walker) walkOAS2Response(statusCode string, resp *parser.Response, basePath string) error {
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

	// Schema (OAS 2.0 uses schema directly, not content)
	if resp.Schema != nil {
		if err := w.walkSchema(resp.Schema, basePath+".schema", 0); err != nil {
			return err
		}
	}

	return nil
}
