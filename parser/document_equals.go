package parser

// This file contains equality methods for OAS2Document and OAS3Document.
// These methods enable semantic comparison of OpenAPI specifications at
// the document level.
//
// Field comparisons are ordered from cheapest to most expensive for early exit:
// 1. Boolean fields (single byte comparison)
// 2. Integer/enum fields (fixed size)
// 3. String fields (variable length)
// 4. Pointer fields (nil check + dereference)
// 5. Slices (length + iteration)
// 6. Maps (length + iteration)
//
// See also:
// - common_equals.go: Info, Contact, License, Tag, Server helpers
// - paths_equals.go: Path, Operation, Response, Callback, Link, MediaType helpers
// - parameters_equals.go: Parameter, Header, RequestBody, Items helpers
// - security_equals.go: SecurityRequirement, SecurityScheme, OAuth helpers

// Equals compares two OAS3Documents for structural equality.
// Returns true if both documents have identical content.
func (d *OAS3Document) Equals(other *OAS3Document) bool {
	if d == nil && other == nil {
		return true
	}
	if d == nil || other == nil {
		return false
	}

	// Group 1: Enum fields (cheapest - single value comparison)
	if d.OASVersion != other.OASVersion {
		return false
	}

	// Group 2: String fields
	if d.OpenAPI != other.OpenAPI {
		return false
	}
	if d.JSONSchemaDialect != other.JSONSchemaDialect {
		return false
	}
	if d.Self != other.Self {
		return false
	}

	// Group 3: Pointer fields
	if !equalInfo(d.Info, other.Info) {
		return false
	}
	if !equalExternalDocs(d.ExternalDocs, other.ExternalDocs) {
		return false
	}
	if !equalComponents(d.Components, other.Components) {
		return false
	}

	// Group 4: Slices
	if !equalServerSlice(d.Servers, other.Servers) {
		return false
	}
	if !equalSecurityRequirementSlice(d.Security, other.Security) {
		return false
	}
	if !equalTagSlice(d.Tags, other.Tags) {
		return false
	}

	// Group 5: Maps
	if !equalPaths(d.Paths, other.Paths) {
		return false
	}
	if !equalPathItemMap(d.Webhooks, other.Webhooks) {
		return false
	}

	// Group 6: Extensions
	if !equalMapStringAny(d.Extra, other.Extra) {
		return false
	}

	return true
}

// Equals compares two OAS2Documents for structural equality.
// Returns true if both documents have identical content.
func (d *OAS2Document) Equals(other *OAS2Document) bool {
	if d == nil && other == nil {
		return true
	}
	if d == nil || other == nil {
		return false
	}

	// Group 1: Enum fields (cheapest - single value comparison)
	if d.OASVersion != other.OASVersion {
		return false
	}

	// Group 2: String fields
	if d.Swagger != other.Swagger {
		return false
	}
	if d.Host != other.Host {
		return false
	}
	if d.BasePath != other.BasePath {
		return false
	}

	// Group 3: Pointer fields
	if !equalInfo(d.Info, other.Info) {
		return false
	}
	if !equalExternalDocs(d.ExternalDocs, other.ExternalDocs) {
		return false
	}

	// Group 4: String slices
	if !equalStringSlice(d.Schemes, other.Schemes) {
		return false
	}
	if !equalStringSlice(d.Consumes, other.Consumes) {
		return false
	}
	if !equalStringSlice(d.Produces, other.Produces) {
		return false
	}

	// Group 5: Other slices
	if !equalSecurityRequirementSlice(d.Security, other.Security) {
		return false
	}
	if !equalTagSlice(d.Tags, other.Tags) {
		return false
	}

	// Group 6: Maps
	if !equalPaths(d.Paths, other.Paths) {
		return false
	}
	if !equalSchemaMap(d.Definitions, other.Definitions) {
		return false
	}
	if !equalParameterMap(d.Parameters, other.Parameters) {
		return false
	}
	if !equalResponseMap(d.Responses, other.Responses) {
		return false
	}
	if !equalSecuritySchemeMap(d.SecurityDefinitions, other.SecurityDefinitions) {
		return false
	}

	// Group 7: Extensions
	if !equalMapStringAny(d.Extra, other.Extra) {
		return false
	}

	return true
}

// =============================================================================
// Components helper (OAS 3.0+)
// =============================================================================

// equalComponents compares two *Components for equality.
func equalComponents(a, b *Components) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Schema maps
	if !equalSchemaMap(a.Schemas, b.Schemas) {
		return false
	}

	// Response maps
	if !equalResponseMap(a.Responses, b.Responses) {
		return false
	}

	// Parameter maps
	if !equalParameterMap(a.Parameters, b.Parameters) {
		return false
	}

	// Example maps
	if !equalExampleMap(a.Examples, b.Examples) {
		return false
	}

	// RequestBody maps
	if !equalRequestBodyMap(a.RequestBodies, b.RequestBodies) {
		return false
	}

	// Header maps
	if !equalHeaderMap(a.Headers, b.Headers) {
		return false
	}

	// SecurityScheme maps
	if !equalSecuritySchemeMap(a.SecuritySchemes, b.SecuritySchemes) {
		return false
	}

	// Link maps
	if !equalLinkMap(a.Links, b.Links) {
		return false
	}

	// Callback maps
	if !equalCallbackMap(a.Callbacks, b.Callbacks) {
		return false
	}

	// PathItem maps (OAS 3.1+)
	if !equalPathItemMap(a.PathItems, b.PathItems) {
		return false
	}

	// MediaType maps (OAS 3.2+)
	if !equalMediaTypeMap(a.MediaTypes, b.MediaTypes) {
		return false
	}

	// Extensions
	if !equalMapStringAny(a.Extra, b.Extra) {
		return false
	}

	return true
}
