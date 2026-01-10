package parser

// This file contains equality comparison functions for path-related OpenAPI types.
//
// Includes: Paths, PathItem, Operation, Response, Callback, Link, MediaType,
// Example, and Encoding types.
//
// See also:
// - paths.go: Type definitions for these structures

// =============================================================================
// Path type helpers
// =============================================================================

// equalPaths compares two Paths (map[string]*PathItem) for equality.
// Nil and empty maps are considered equal.
func equalPaths(a, b Paths) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !equalPathItem(va, vb) {
			return false
		}
	}
	return true
}

// equalPathItem compares two *PathItem for equality.
func equalPathItem(a, b *PathItem) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// String fields
	if a.Ref != b.Ref {
		return false
	}
	if a.Summary != b.Summary {
		return false
	}
	if a.Description != b.Description {
		return false
	}

	// Operations
	if !equalOperation(a.Get, b.Get) {
		return false
	}
	if !equalOperation(a.Put, b.Put) {
		return false
	}
	if !equalOperation(a.Post, b.Post) {
		return false
	}
	if !equalOperation(a.Delete, b.Delete) {
		return false
	}
	if !equalOperation(a.Options, b.Options) {
		return false
	}
	if !equalOperation(a.Head, b.Head) {
		return false
	}
	if !equalOperation(a.Patch, b.Patch) {
		return false
	}
	if !equalOperation(a.Trace, b.Trace) {
		return false
	}
	if !equalOperation(a.Query, b.Query) {
		return false
	}

	// Slices
	if !equalServerSlice(a.Servers, b.Servers) {
		return false
	}
	if !equalParameterSlice(a.Parameters, b.Parameters) {
		return false
	}

	// Maps
	if !equalOperationMap(a.AdditionalOperations, b.AdditionalOperations) {
		return false
	}

	// Extensions
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}

	return true
}

// equalPathItemMap compares two map[string]*PathItem maps for equality.
// Used for Webhooks and Components.PathItems.
// Nil and empty maps are considered equal.
func equalPathItemMap(a, b map[string]*PathItem) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !equalPathItem(va, vb) {
			return false
		}
	}
	return true
}

// =============================================================================
// Operation type helpers
// =============================================================================

// equalOperation compares two *Operation for equality.
func equalOperation(a, b *Operation) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Boolean fields (cheapest)
	if a.Deprecated != b.Deprecated {
		return false
	}

	// String fields
	if a.Summary != b.Summary {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if a.OperationID != b.OperationID {
		return false
	}

	// String slices
	if !equalStringSlice(a.Tags, b.Tags) {
		return false
	}
	if !equalStringSlice(a.Consumes, b.Consumes) {
		return false
	}
	if !equalStringSlice(a.Produces, b.Produces) {
		return false
	}
	if !equalStringSlice(a.Schemes, b.Schemes) {
		return false
	}

	// Struct pointers
	if !equalExternalDocs(a.ExternalDocs, b.ExternalDocs) {
		return false
	}
	if !equalRequestBody(a.RequestBody, b.RequestBody) {
		return false
	}
	if !equalResponses(a.Responses, b.Responses) {
		return false
	}

	// Slices
	if !equalParameterSlice(a.Parameters, b.Parameters) {
		return false
	}
	if !equalSecurityRequirementSlice(a.Security, b.Security) {
		return false
	}
	if !equalServerSlice(a.Servers, b.Servers) {
		return false
	}

	// Maps
	if !equalCallbackMap(a.Callbacks, b.Callbacks) {
		return false
	}

	// Extensions
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}

	return true
}

// equalOperationMap compares two map[string]*Operation maps for equality.
// Used for PathItem.AdditionalOperations.
// Nil and empty maps are considered equal.
func equalOperationMap(a, b map[string]*Operation) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !equalOperation(va, vb) {
			return false
		}
	}
	return true
}

// =============================================================================
// Response type helpers
// =============================================================================

// equalResponses compares two *Responses for equality.
func equalResponses(a, b *Responses) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if !equalResponse(a.Default, b.Default) {
		return false
	}
	if !equalResponseMap(a.Codes, b.Codes) {
		return false
	}
	return true
}

// equalResponse compares two *Response for equality.
func equalResponse(a, b *Response) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// String fields
	if a.Ref != b.Ref {
		return false
	}
	if a.Description != b.Description {
		return false
	}

	// Struct pointers (OAS 2.0)
	if !a.Schema.Equals(b.Schema) {
		return false
	}

	// Maps
	if !equalHeaderMap(a.Headers, b.Headers) {
		return false
	}
	if !equalMediaTypeMap(a.Content, b.Content) {
		return false
	}
	if !equalLinkMap(a.Links, b.Links) {
		return false
	}
	if !equalMapStringAny(a.Examples, b.Examples) {
		return false
	}

	// Extensions
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}

	return true
}

// equalResponseMap compares two map[string]*Response maps for equality.
// Nil and empty maps are considered equal.
func equalResponseMap(a, b map[string]*Response) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !equalResponse(va, vb) {
			return false
		}
	}
	return true
}

// =============================================================================
// Callback type helpers
// =============================================================================

// equalCallback compares two *Callback for equality.
// Callback is a named type for map[string]*PathItem.
func equalCallback(a, b *Callback) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return equalPathItemMap(map[string]*PathItem(*a), map[string]*PathItem(*b))
}

// equalCallbackMap compares two map[string]*Callback maps for equality.
// Nil and empty maps are considered equal.
func equalCallbackMap(a, b map[string]*Callback) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !equalCallback(va, vb) {
			return false
		}
	}
	return true
}

// =============================================================================
// Link type helpers
// =============================================================================

// equalLink compares two *Link for equality.
func equalLink(a, b *Link) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// String fields
	if a.Ref != b.Ref {
		return false
	}
	if a.OperationRef != b.OperationRef {
		return false
	}
	if a.OperationID != b.OperationID {
		return false
	}
	if a.Description != b.Description {
		return false
	}

	// Any field
	if !equalJSONValue(a.RequestBody, b.RequestBody) {
		return false
	}

	// Struct pointer
	if !equalServer(a.Server, b.Server) {
		return false
	}

	// Maps
	if !equalMapStringAny(a.Parameters, b.Parameters) {
		return false
	}

	// Extensions
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}

	return true
}

// equalLinkMap compares two map[string]*Link maps for equality.
// Nil and empty maps are considered equal.
func equalLinkMap(a, b map[string]*Link) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !equalLink(va, vb) {
			return false
		}
	}
	return true
}

// =============================================================================
// MediaType type helpers
// =============================================================================

// equalMediaType compares two *MediaType for equality.
func equalMediaType(a, b *MediaType) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Struct pointer
	if !a.Schema.Equals(b.Schema) {
		return false
	}

	// Any field
	if !equalJSONValue(a.Example, b.Example) {
		return false
	}

	// Maps
	if !equalExampleMap(a.Examples, b.Examples) {
		return false
	}
	if !equalEncodingMap(a.Encoding, b.Encoding) {
		return false
	}

	// Extensions
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}

	return true
}

// equalMediaTypeMap compares two map[string]*MediaType maps for equality.
// Nil and empty maps are considered equal.
func equalMediaTypeMap(a, b map[string]*MediaType) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !equalMediaType(va, vb) {
			return false
		}
	}
	return true
}

// =============================================================================
// Example type helpers
// =============================================================================

// equalExample compares two *Example for equality.
func equalExample(a, b *Example) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// String fields
	if a.Ref != b.Ref {
		return false
	}
	if a.Summary != b.Summary {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if a.ExternalValue != b.ExternalValue {
		return false
	}

	// Any field
	if !equalJSONValue(a.Value, b.Value) {
		return false
	}

	// Extensions
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}

	return true
}

// equalExampleMap compares two map[string]*Example maps for equality.
// Nil and empty maps are considered equal.
func equalExampleMap(a, b map[string]*Example) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !equalExample(va, vb) {
			return false
		}
	}
	return true
}

// =============================================================================
// Encoding type helpers
// =============================================================================

// equalEncoding compares two *Encoding for equality.
func equalEncoding(a, b *Encoding) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Boolean fields (cheapest)
	if a.AllowReserved != b.AllowReserved {
		return false
	}

	// String fields
	if a.ContentType != b.ContentType {
		return false
	}
	if a.Style != b.Style {
		return false
	}

	// Pointer fields
	if !equalBoolPtr(a.Explode, b.Explode) {
		return false
	}

	// Maps
	if !equalHeaderMap(a.Headers, b.Headers) {
		return false
	}

	// Extensions
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}

	return true
}

// equalEncodingMap compares two map[string]*Encoding maps for equality.
// Nil and empty maps are considered equal.
func equalEncodingMap(a, b map[string]*Encoding) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !equalEncoding(va, vb) {
			return false
		}
	}
	return true
}
