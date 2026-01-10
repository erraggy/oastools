package parser

// This file contains equality comparison functions for parameter-related OpenAPI types.
//
// Includes: Parameter, Header, RequestBody, and Items (OAS 2.0) types.
//
// See also:
// - parameters.go: Type definitions for these structures

// =============================================================================
// Parameter type helpers
// =============================================================================

// equalParameter compares two *Parameter for equality.
func equalParameter(a, b *Parameter) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Boolean fields (cheapest)
	if a.Required != b.Required {
		return false
	}
	if a.Deprecated != b.Deprecated {
		return false
	}
	if a.AllowReserved != b.AllowReserved {
		return false
	}
	if a.AllowEmptyValue != b.AllowEmptyValue {
		return false
	}
	if a.ExclusiveMaximum != b.ExclusiveMaximum {
		return false
	}
	if a.ExclusiveMinimum != b.ExclusiveMinimum {
		return false
	}
	if a.UniqueItems != b.UniqueItems {
		return false
	}

	// String fields
	if a.Ref != b.Ref {
		return false
	}
	if a.Name != b.Name {
		return false
	}
	if a.In != b.In {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if a.Style != b.Style {
		return false
	}
	if a.Type != b.Type {
		return false
	}
	if a.Format != b.Format {
		return false
	}
	if a.CollectionFormat != b.CollectionFormat {
		return false
	}
	if a.Pattern != b.Pattern {
		return false
	}

	// Pointer fields
	if !equalBoolPtr(a.Explode, b.Explode) {
		return false
	}
	if !equalFloat64Ptr(a.Maximum, b.Maximum) {
		return false
	}
	if !equalFloat64Ptr(a.Minimum, b.Minimum) {
		return false
	}
	if !equalFloat64Ptr(a.MultipleOf, b.MultipleOf) {
		return false
	}
	if !equalIntPtr(a.MaxLength, b.MaxLength) {
		return false
	}
	if !equalIntPtr(a.MinLength, b.MinLength) {
		return false
	}
	if !equalIntPtr(a.MaxItems, b.MaxItems) {
		return false
	}
	if !equalIntPtr(a.MinItems, b.MinItems) {
		return false
	}

	// Any fields
	if !equalJSONValue(a.Example, b.Example) {
		return false
	}
	if !equalJSONValue(a.Default, b.Default) {
		return false
	}
	if !equalAnySlice(a.Enum, b.Enum) {
		return false
	}

	// Struct pointers
	if !a.Schema.Equals(b.Schema) {
		return false
	}
	if !equalItems(a.Items, b.Items) {
		return false
	}

	// Maps
	if !equalExampleMap(a.Examples, b.Examples) {
		return false
	}
	if !equalMediaTypeMap(a.Content, b.Content) {
		return false
	}

	// Extensions
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}

	return true
}

// equalParameterSlice compares two []*Parameter slices for equality.
// Order-sensitive comparison. Nil and empty slices are considered equal.
func equalParameterSlice(a, b []*Parameter) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !equalParameter(a[i], b[i]) {
			return false
		}
	}
	return true
}

// equalParameterMap compares two map[string]*Parameter maps for equality.
// Nil and empty maps are considered equal.
func equalParameterMap(a, b map[string]*Parameter) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !equalParameter(va, vb) {
			return false
		}
	}
	return true
}

// =============================================================================
// Header type helpers
// =============================================================================

// equalHeader compares two *Header for equality.
func equalHeader(a, b *Header) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Boolean fields (cheapest)
	if a.Required != b.Required {
		return false
	}
	if a.Deprecated != b.Deprecated {
		return false
	}
	if a.ExclusiveMaximum != b.ExclusiveMaximum {
		return false
	}
	if a.ExclusiveMinimum != b.ExclusiveMinimum {
		return false
	}
	if a.UniqueItems != b.UniqueItems {
		return false
	}

	// String fields
	if a.Ref != b.Ref {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if a.Style != b.Style {
		return false
	}
	if a.Type != b.Type {
		return false
	}
	if a.Format != b.Format {
		return false
	}
	if a.CollectionFormat != b.CollectionFormat {
		return false
	}
	if a.Pattern != b.Pattern {
		return false
	}

	// Pointer fields
	if !equalBoolPtr(a.Explode, b.Explode) {
		return false
	}
	if !equalFloat64Ptr(a.Maximum, b.Maximum) {
		return false
	}
	if !equalFloat64Ptr(a.Minimum, b.Minimum) {
		return false
	}
	if !equalFloat64Ptr(a.MultipleOf, b.MultipleOf) {
		return false
	}
	if !equalIntPtr(a.MaxLength, b.MaxLength) {
		return false
	}
	if !equalIntPtr(a.MinLength, b.MinLength) {
		return false
	}
	if !equalIntPtr(a.MaxItems, b.MaxItems) {
		return false
	}
	if !equalIntPtr(a.MinItems, b.MinItems) {
		return false
	}

	// Any fields
	if !equalJSONValue(a.Example, b.Example) {
		return false
	}
	if !equalJSONValue(a.Default, b.Default) {
		return false
	}
	if !equalAnySlice(a.Enum, b.Enum) {
		return false
	}

	// Struct pointers
	if !a.Schema.Equals(b.Schema) {
		return false
	}
	if !equalItems(a.Items, b.Items) {
		return false
	}

	// Maps
	if !equalExampleMap(a.Examples, b.Examples) {
		return false
	}
	if !equalMediaTypeMap(a.Content, b.Content) {
		return false
	}

	// Extensions
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}

	return true
}

// equalHeaderMap compares two map[string]*Header maps for equality.
// Nil and empty maps are considered equal.
func equalHeaderMap(a, b map[string]*Header) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !equalHeader(va, vb) {
			return false
		}
	}
	return true
}

// =============================================================================
// RequestBody type helpers
// =============================================================================

// equalRequestBody compares two *RequestBody for equality.
func equalRequestBody(a, b *RequestBody) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Boolean fields (cheapest)
	if a.Required != b.Required {
		return false
	}

	// String fields
	if a.Ref != b.Ref {
		return false
	}
	if a.Description != b.Description {
		return false
	}

	// Maps
	if !equalMediaTypeMap(a.Content, b.Content) {
		return false
	}

	// Extensions
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}

	return true
}

// equalRequestBodyMap compares two map[string]*RequestBody maps for equality.
// Nil and empty maps are considered equal.
func equalRequestBodyMap(a, b map[string]*RequestBody) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !equalRequestBody(va, vb) {
			return false
		}
	}
	return true
}

// =============================================================================
// Items type helpers (OAS 2.0)
// =============================================================================

// equalItems compares two *Items for equality (OAS 2.0).
func equalItems(a, b *Items) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Boolean fields (cheapest)
	if a.ExclusiveMaximum != b.ExclusiveMaximum {
		return false
	}
	if a.ExclusiveMinimum != b.ExclusiveMinimum {
		return false
	}
	if a.UniqueItems != b.UniqueItems {
		return false
	}

	// String fields
	if a.Type != b.Type {
		return false
	}
	if a.Format != b.Format {
		return false
	}
	if a.CollectionFormat != b.CollectionFormat {
		return false
	}
	if a.Pattern != b.Pattern {
		return false
	}

	// Pointer fields
	if !equalFloat64Ptr(a.Maximum, b.Maximum) {
		return false
	}
	if !equalFloat64Ptr(a.Minimum, b.Minimum) {
		return false
	}
	if !equalFloat64Ptr(a.MultipleOf, b.MultipleOf) {
		return false
	}
	if !equalIntPtr(a.MaxLength, b.MaxLength) {
		return false
	}
	if !equalIntPtr(a.MinLength, b.MinLength) {
		return false
	}
	if !equalIntPtr(a.MaxItems, b.MaxItems) {
		return false
	}
	if !equalIntPtr(a.MinItems, b.MinItems) {
		return false
	}

	// Any fields
	if !equalJSONValue(a.Default, b.Default) {
		return false
	}
	if !equalAnySlice(a.Enum, b.Enum) {
		return false
	}

	// Recursive Items
	if !equalItems(a.Items, b.Items) {
		return false
	}

	// Extensions
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}

	return true
}
