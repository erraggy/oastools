// oas3_schema_positions.go visits every position an OAS 3.x document can hold a
// Schema Object. Schemas are rewritten in place, so the walk only has to reach
// each one.

package converter

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// schemaVisitor receives each Schema Object with the path that locates it.
type schemaVisitor func(schema *parser.Schema, path string)

// forEachOAS3Schema visits the outermost Schema Object at every position in an
// OAS 3.x document. Subschemas are left to the passes, which walk them
// themselves.
func forEachOAS3Schema(doc *parser.OAS3Document, visit schemaVisitor) {
	if doc == nil {
		return
	}

	forEachComponentsSchema(doc.Components, visit)

	for pattern, item := range doc.Paths {
		forEachPathItemSchema(item, fmt.Sprintf("paths.%s", pattern), visit)
	}
	for name, item := range doc.Webhooks {
		forEachPathItemSchema(item, fmt.Sprintf("webhooks.%s", name), visit)
	}
}

func forEachComponentsSchema(comp *parser.Components, visit schemaVisitor) {
	if comp == nil {
		return
	}

	for name, schema := range comp.Schemas {
		visitSchema(schema, fmt.Sprintf("components.schemas.%s", name), visit)
	}
	for name, param := range comp.Parameters {
		forEachParameterSchema(param, fmt.Sprintf("components.parameters.%s", name), visit)
	}
	for name, header := range comp.Headers {
		forEachHeaderSchema(header, fmt.Sprintf("components.headers.%s", name), visit)
	}
	for name, body := range comp.RequestBodies {
		if body != nil {
			forEachContentSchema(body.Content, fmt.Sprintf("components.requestBodies.%s", name), visit)
		}
	}
	for name, resp := range comp.Responses {
		forEachResponseSchema(resp, fmt.Sprintf("components.responses.%s", name), visit)
	}
	for name, item := range comp.PathItems {
		forEachPathItemSchema(item, fmt.Sprintf("components.pathItems.%s", name), visit)
	}
	for name, media := range comp.MediaTypes {
		forEachMediaTypeSchema(media, fmt.Sprintf("components.mediaTypes.%s", name), visit)
	}
	for name, callback := range comp.Callbacks {
		forEachCallbackSchema(callback, fmt.Sprintf("components.callbacks.%s", name), visit)
	}
}

func forEachPathItemSchema(item *parser.PathItem, path string, visit schemaVisitor) {
	if item == nil {
		return
	}

	for i, param := range item.Parameters {
		forEachParameterSchema(param, fmt.Sprintf("%s.parameters[%d]", path, i), visit)
	}
	for _, op := range oas32PathItemOperations {
		forEachOperationSchema(op.get(item), fmt.Sprintf("%s.%s", path, op.method), visit)
	}
	for method, op := range item.AdditionalOperations {
		forEachOperationSchema(op, fmt.Sprintf("%s.additionalOperations.%s", path, method), visit)
	}
}

func forEachOperationSchema(op *parser.Operation, path string, visit schemaVisitor) {
	if op == nil {
		return
	}

	for i, param := range op.Parameters {
		forEachParameterSchema(param, fmt.Sprintf("%s.parameters[%d]", path, i), visit)
	}
	if op.RequestBody != nil {
		forEachContentSchema(op.RequestBody.Content, path+".requestBody", visit)
	}
	if op.Responses != nil {
		forEachResponseSchema(op.Responses.Default, path+".responses.default", visit)
		for code, resp := range op.Responses.Codes {
			forEachResponseSchema(resp, fmt.Sprintf("%s.responses.%s", path, code), visit)
		}
	}
	for name, callback := range op.Callbacks {
		forEachCallbackSchema(callback, fmt.Sprintf("%s.callbacks.%s", path, name), visit)
	}
}

// forEachCallbackSchema visits a Callback, which maps a runtime expression to a
// Path Item.
func forEachCallbackSchema(callback *parser.Callback, path string, visit schemaVisitor) {
	if callback == nil {
		return
	}
	for expression, item := range *callback {
		forEachPathItemSchema(item, fmt.Sprintf("%s.%s", path, expression), visit)
	}
}

func forEachResponseSchema(resp *parser.Response, path string, visit schemaVisitor) {
	if resp == nil {
		return
	}
	for name, header := range resp.Headers {
		forEachHeaderSchema(header, fmt.Sprintf("%s.headers.%s", path, name), visit)
	}
	forEachContentSchema(resp.Content, path, visit)
}

func forEachParameterSchema(param *parser.Parameter, path string, visit schemaVisitor) {
	if param == nil {
		return
	}
	visitSchema(param.Schema, path+".schema", visit)
	forEachContentSchema(param.Content, path, visit)
}

func forEachHeaderSchema(header *parser.Header, path string, visit schemaVisitor) {
	if header == nil {
		return
	}
	visitSchema(header.Schema, path+".schema", visit)
	forEachContentSchema(header.Content, path, visit)
}

func forEachContentSchema(content map[string]*parser.MediaType, path string, visit schemaVisitor) {
	for mediaType, media := range content {
		forEachMediaTypeSchema(media, fmt.Sprintf("%s.content.%s", path, mediaType), visit)
	}
}

func forEachMediaTypeSchema(media *parser.MediaType, path string, visit schemaVisitor) {
	if media == nil {
		return
	}
	visitSchema(media.Schema, path+".schema", visit)
	visitSchema(media.ItemSchema, path+".itemSchema", visit)
	for name, enc := range media.Encoding {
		forEachEncodingSchema(enc, fmt.Sprintf("%s.encoding.%s", path, name), visit, nil, 0)
	}
	forEachEncodingSchema(media.ItemEncoding, path+".itemEncoding", visit, nil, 0)
	for i, enc := range media.PrefixEncoding {
		forEachEncodingSchema(enc, fmt.Sprintf("%s.prefixEncoding[%d]", path, i), visit, nil, 0)
	}
}

// forEachEncodingSchema reaches the Header Objects an Encoding carries. The
// visited set and depth bound match detectOAS32EncodingFeatures, because Convert
// takes the caller's document rather than a parsed one.
func forEachEncodingSchema(enc *parser.Encoding, path string, visit schemaVisitor, visited map[*parser.Encoding]bool, depth int) {
	if enc == nil || depth > maxEncodingNestingDepth || visited[enc] {
		return
	}

	for name, header := range enc.Headers {
		forEachHeaderSchema(header, fmt.Sprintf("%s.headers.%s", path, name), visit)
	}

	// An encoding that does not nest cannot repeat, so it allocates no visited set.
	if len(enc.Encoding) == 0 && enc.ItemEncoding == nil && len(enc.PrefixEncoding) == 0 {
		return
	}
	if visited == nil {
		visited = make(map[*parser.Encoding]bool)
	}
	visited[enc] = true

	for name, nested := range enc.Encoding {
		forEachEncodingSchema(nested, fmt.Sprintf("%s.encoding.%s", path, name), visit, visited, depth+1)
	}
	forEachEncodingSchema(enc.ItemEncoding, path+".itemEncoding", visit, visited, depth+1)
	for i, nested := range enc.PrefixEncoding {
		forEachEncodingSchema(nested, fmt.Sprintf("%s.prefixEncoding[%d]", path, i), visit, visited, depth+1)
	}
}

func visitSchema(schema *parser.Schema, path string, visit schemaVisitor) {
	if schema == nil {
		return
	}
	visit(schema, path)
}
