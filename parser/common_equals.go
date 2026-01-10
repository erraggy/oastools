package parser

// This file contains equality comparison functions for common OpenAPI types
// that are shared between OAS 2.0 and OAS 3.x specifications.
//
// Includes: Info, Contact, License, Tag, Server, and ServerVariable types.
//
// See also:
// - common.go: Type definitions for these structures

// =============================================================================
// Info type helpers
// =============================================================================

// equalInfo compares two *Info for equality.
func equalInfo(a, b *Info) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Title != b.Title {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if a.TermsOfService != b.TermsOfService {
		return false
	}
	if a.Version != b.Version {
		return false
	}
	if a.Summary != b.Summary {
		return false
	}
	if !equalContact(a.Contact, b.Contact) {
		return false
	}
	if !equalLicense(a.License, b.License) {
		return false
	}
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}
	return true
}

// equalContact compares two *Contact for equality.
func equalContact(a, b *Contact) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Name != b.Name {
		return false
	}
	if a.URL != b.URL {
		return false
	}
	if a.Email != b.Email {
		return false
	}
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}
	return true
}

// equalLicense compares two *License for equality.
func equalLicense(a, b *License) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Name != b.Name {
		return false
	}
	if a.URL != b.URL {
		return false
	}
	if a.Identifier != b.Identifier {
		return false
	}
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}
	return true
}

// =============================================================================
// Tag type helpers
// =============================================================================

// equalTag compares two *Tag for equality.
func equalTag(a, b *Tag) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Name != b.Name {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if !equalExternalDocs(a.ExternalDocs, b.ExternalDocs) {
		return false
	}
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}
	return true
}

// equalTagSlice compares two []*Tag slices for equality.
// Order-sensitive comparison. Nil and empty slices are considered equal.
func equalTagSlice(a, b []*Tag) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !equalTag(a[i], b[i]) {
			return false
		}
	}
	return true
}

// =============================================================================
// Server type helpers
// =============================================================================

// equalServer compares two *Server for equality.
func equalServer(a, b *Server) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.URL != b.URL {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if !equalServerVariableMap(a.Variables, b.Variables) {
		return false
	}
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}
	return true
}

// equalServerSlice compares two []*Server slices for equality.
// Order-sensitive comparison. Nil and empty slices are considered equal.
func equalServerSlice(a, b []*Server) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !equalServer(a[i], b[i]) {
			return false
		}
	}
	return true
}

// equalServerVariableMap compares two map[string]ServerVariable maps for equality.
// Note: ServerVariable is a VALUE type, not a pointer type.
// Nil and empty maps are considered equal.
func equalServerVariableMap(a, b map[string]ServerVariable) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !equalServerVariable(va, vb) {
			return false
		}
	}
	return true
}

// equalServerVariable compares two ServerVariable values for equality.
// ServerVariable is a value type, not a pointer.
func equalServerVariable(a, b ServerVariable) bool {
	if a.Default != b.Default {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if !equalStringSlice(a.Enum, b.Enum) {
		return false
	}
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}
	return true
}
