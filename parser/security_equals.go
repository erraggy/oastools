package parser

// This file contains equality comparison functions for security-related OpenAPI types.
//
// Includes: SecurityRequirement, SecurityScheme, OAuthFlows, and OAuthFlow types.
//
// See also:
// - security.go: Type definitions for these structures

// =============================================================================
// SecurityRequirement type helpers
// =============================================================================

// equalSecurityRequirement compares two SecurityRequirement values for equality.
// SecurityRequirement is map[string][]string.
func equalSecurityRequirement(a, b SecurityRequirement) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !equalStringSlice(va, vb) {
			return false
		}
	}
	return true
}

// equalSecurityRequirementSlice compares two []SecurityRequirement slices for equality.
// Order-sensitive comparison. Nil and empty slices are considered equal.
func equalSecurityRequirementSlice(a, b []SecurityRequirement) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !equalSecurityRequirement(a[i], b[i]) {
			return false
		}
	}
	return true
}

// =============================================================================
// SecurityScheme type helpers
// =============================================================================

// equalSecurityScheme compares two *SecurityScheme for equality.
func equalSecurityScheme(a, b *SecurityScheme) bool {
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
	if a.Type != b.Type {
		return false
	}
	if a.Description != b.Description {
		return false
	}
	if a.Name != b.Name {
		return false
	}
	if a.In != b.In {
		return false
	}
	if a.Scheme != b.Scheme {
		return false
	}
	if a.BearerFormat != b.BearerFormat {
		return false
	}
	if a.Flow != b.Flow {
		return false
	}
	if a.AuthorizationURL != b.AuthorizationURL {
		return false
	}
	if a.TokenURL != b.TokenURL {
		return false
	}
	if a.OpenIDConnectURL != b.OpenIDConnectURL {
		return false
	}

	// Struct pointers
	if !equalOAuthFlows(a.Flows, b.Flows) {
		return false
	}

	// Maps
	if !equalMapStringString(a.Scopes, b.Scopes) {
		return false
	}

	// Extensions
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}

	return true
}

// equalSecuritySchemeMap compares two map[string]*SecurityScheme maps for equality.
// Nil and empty maps are considered equal.
func equalSecuritySchemeMap(a, b map[string]*SecurityScheme) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !equalSecurityScheme(va, vb) {
			return false
		}
	}
	return true
}

// =============================================================================
// OAuthFlows type helpers
// =============================================================================

// equalOAuthFlows compares two *OAuthFlows for equality.
func equalOAuthFlows(a, b *OAuthFlows) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	if !equalOAuthFlow(a.Implicit, b.Implicit) {
		return false
	}
	if !equalOAuthFlow(a.Password, b.Password) {
		return false
	}
	if !equalOAuthFlow(a.ClientCredentials, b.ClientCredentials) {
		return false
	}
	if !equalOAuthFlow(a.AuthorizationCode, b.AuthorizationCode) {
		return false
	}

	// Extensions
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}

	return true
}

// equalOAuthFlow compares two *OAuthFlow for equality.
func equalOAuthFlow(a, b *OAuthFlow) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// String fields
	if a.AuthorizationURL != b.AuthorizationURL {
		return false
	}
	if a.TokenURL != b.TokenURL {
		return false
	}
	if a.RefreshURL != b.RefreshURL {
		return false
	}

	// Maps
	if !equalMapStringString(a.Scopes, b.Scopes) {
		return false
	}

	// Extensions
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}

	return true
}
