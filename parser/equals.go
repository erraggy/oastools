package parser

// This file contains helper functions for equality comparison of OAS-typed fields.
// These helpers understand the OAS specification semantics for fields that use
// interface{}/any types but have well-defined possible types per the spec.
// The functions mirror the patterns in deepcopy_helpers.go.

import (
	"log/slog"
	"maps"
	"reflect"
	"slices"
)

// equalFloat64Ptr compares two *float64 pointers for equality.
// Both nil returns true, both non-nil with equal values returns true.
func equalFloat64Ptr(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// equalIntPtr compares two *int pointers for equality.
// Both nil returns true, both non-nil with equal values returns true.
func equalIntPtr(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// equalBoolPtr compares two *bool pointers for equality.
// Both nil returns true, both non-nil with equal values returns true.
func equalBoolPtr(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// equalStringSlice compares two string slices for equality.
// Order-sensitive comparison. Nil and empty slices are considered equal.
func equalStringSlice(a, b []string) bool {
	return slices.Equal(a, b)
}

// equalAnySlice compares two []any slices for equality.
// Uses reflect.DeepEqual for element comparison. Nil and empty slices are considered equal.
func equalAnySlice(a, b []any) bool {
	return slices.EqualFunc(a, b, reflect.DeepEqual)
}

// equalMapStringAny compares two map[string]any maps for equality.
// Uses reflect.DeepEqual for value comparison. Nil and empty maps are considered equal.
func equalMapStringAny(a, b map[string]any) bool {
	return maps.EqualFunc(a, b, reflect.DeepEqual)
}

// equalMapStringBool compares two map[string]bool maps for equality.
// Used for Schema.Vocabulary field. Nil and empty maps are considered equal.
func equalMapStringBool(a, b map[string]bool) bool {
	return maps.Equal(a, b)
}

// equalMapStringStringSlice compares two map[string][]string maps for equality.
// Used for Schema.DependentRequired field. Nil and empty maps are considered equal.
func equalMapStringStringSlice(a, b map[string][]string) bool {
	return maps.EqualFunc(a, b, slices.Equal)
}

// equalSchemaType handles Schema.Type which can be:
// - string (OAS 2.0, 3.0, 3.1)
// - []string (OAS 3.1+ for type arrays like ["string", "null"])
// - []any (YAML may unmarshal as []any instead of []string)
func equalSchemaType(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch ta := a.(type) {
	case string:
		tb, ok := b.(string)
		if !ok {
			return false
		}
		return ta == tb
	case []string:
		tb, ok := b.([]string)
		if !ok {
			return false
		}
		return equalStringSlice(ta, tb)
	case []any:
		tb, ok := b.([]any)
		if !ok {
			return false
		}
		return equalAnySlice(ta, tb)
	default:
		// Unknown type, fall back to reflect.DeepEqual
		return reflect.DeepEqual(a, b)
	}
}

// equalSchemaOrBool handles fields that can be *Schema or bool:
// - Schema.Items (OAS 3.1+: bool for additionalItems semantics)
// - Schema.AdditionalProperties
// - Schema.AdditionalItems
// - Schema.UnevaluatedItems
// - Schema.UnevaluatedProperties
func equalSchemaOrBool(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch ta := a.(type) {
	case bool:
		tb, ok := b.(bool)
		if !ok {
			return false
		}
		return ta == tb
	case *Schema:
		tb, ok := b.(*Schema)
		if !ok {
			return false
		}
		return ta.Equals(tb)
	default:
		// Unknown type, fall back to reflect.DeepEqual
		return reflect.DeepEqual(a, b)
	}
}

// equalBoolOrNumber handles ExclusiveMinimum/ExclusiveMaximum:
// - bool (OAS 2.0, 3.0)
// - float64/int (OAS 3.1+ JSON Schema Draft 2020-12)
func equalBoolOrNumber(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch ta := a.(type) {
	case bool:
		tb, ok := b.(bool)
		if !ok {
			return false
		}
		return ta == tb
	case float64:
		tb, ok := b.(float64)
		if !ok {
			return false
		}
		return ta == tb
	case int:
		tb, ok := b.(int)
		if !ok {
			return false
		}
		return ta == tb
	case int64:
		tb, ok := b.(int64)
		if !ok {
			return false
		}
		return ta == tb
	default:
		// Unknown type, fall back to reflect.DeepEqual
		return reflect.DeepEqual(a, b)
	}
}

// equalJSONValue compares arbitrary JSON-compatible values recursively.
// Handles Default, Example, Const, and other fields that can hold
// arbitrary JSON values.
func equalJSONValue(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch ta := a.(type) {
	case string:
		tb, ok := b.(string)
		return ok && ta == tb
	case bool:
		tb, ok := b.(bool)
		return ok && ta == tb
	case float64:
		tb, ok := b.(float64)
		return ok && ta == tb
	case int:
		tb, ok := b.(int)
		return ok && ta == tb
	case int64:
		tb, ok := b.(int64)
		return ok && ta == tb
	case float32:
		tb, ok := b.(float32)
		return ok && ta == tb
	case int32:
		tb, ok := b.(int32)
		return ok && ta == tb
	case int16:
		tb, ok := b.(int16)
		return ok && ta == tb
	case int8:
		tb, ok := b.(int8)
		return ok && ta == tb
	case uint:
		tb, ok := b.(uint)
		return ok && ta == tb
	case uint64:
		tb, ok := b.(uint64)
		return ok && ta == tb
	case uint32:
		tb, ok := b.(uint32)
		return ok && ta == tb
	case uint16:
		tb, ok := b.(uint16)
		return ok && ta == tb
	case uint8:
		tb, ok := b.(uint8)
		return ok && ta == tb
	case []any:
		tb, ok := b.([]any)
		if !ok {
			return false
		}
		if len(ta) != len(tb) {
			return false
		}
		for i := range ta {
			if !equalJSONValue(ta[i], tb[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		tb, ok := b.(map[string]any)
		if !ok {
			return false
		}
		if len(ta) != len(tb) {
			return false
		}
		for k, va := range ta {
			vb, exists := tb[k]
			if !exists {
				return false
			}
			if !equalJSONValue(va, vb) {
				return false
			}
		}
		return true
	default:
		// Unknown type - could be custom types in extensions
		// Fall back to reflect.DeepEqual
		return reflect.DeepEqual(a, b)
	}
}

// =============================================================================
// ParseResult equality methods
// =============================================================================

// Equals compares two ParseResults for semantic equality.
// It returns true if both results represent the same OpenAPI specification,
// ignoring runtime metadata like LoadTime and SourcePath.
//
// Equality is determined by:
//   - Version and OASVersion match
//   - Document contents are structurally equal
//
// Fields explicitly NOT compared (runtime metadata):
//   - SourcePath, SourceFormat (how it was loaded)
//   - LoadTime, SourceSize (runtime metrics)
//   - Errors, Warnings (parse diagnostics)
//   - Data (raw map; Document is the canonical representation)
//   - SourceMap (debugging aid)
//   - Stats (derived from Document)
func (pr *ParseResult) Equals(other *ParseResult) bool {
	if pr == nil && other == nil {
		return true
	}
	if pr == nil || other == nil {
		return false
	}

	// Group 1: Enum fields (cheapest - single value comparison)
	if pr.OASVersion != other.OASVersion {
		return false
	}

	// Group 2: String fields
	if pr.Version != other.Version {
		return false
	}

	// Group 3: Document comparison (type switch + recursive)
	return equalDocument(pr.Document, other.Document)
}

// DocumentEquals compares only the Document field, ignoring version.
// This is useful when comparing documents that may have been converted
// between versions but should have equivalent content.
func (pr *ParseResult) DocumentEquals(other *ParseResult) bool {
	if pr == nil && other == nil {
		return true
	}
	if pr == nil || other == nil {
		return false
	}

	return equalDocument(pr.Document, other.Document)
}

// equalDocument compares two Document fields for equality.
// Handles *OAS3Document and *OAS2Document with full semantic comparison.
//
// WARNING: Unknown document types (e.g., future OAS4Document) will fall back
// to reflect.DeepEqual, which may not provide optimal comparison semantics.
// When adding support for new OAS versions, add a case here with proper Equals() method.
//
// Set equalDocumentDebug = true during development to log unknown type fallbacks.
func equalDocument(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch docA := a.(type) {
	case *OAS3Document:
		docB, ok := b.(*OAS3Document)
		if !ok {
			return false
		}
		return docA.Equals(docB)
	case *OAS2Document:
		docB, ok := b.(*OAS2Document)
		if !ok {
			return false
		}
		return docA.Equals(docB)
	default:
		// Unknown document type - log warning and fall back to reflect.DeepEqual.
		// This should only happen with future OAS versions (e.g., OAS4).
		// When adding support for new versions, add an explicit case above.
		slog.Warn("equalDocument: unknown document type, using reflect.DeepEqual fallback",
			"type_a", reflect.TypeOf(a).String(),
			"type_b", reflect.TypeOf(b).String())
		return reflect.DeepEqual(a, b)
	}
}
