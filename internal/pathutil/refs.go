// Copyright 2024 Erraggy
// SPDX-License-Identifier: MIT

package pathutil

import "strings"

// EscapeRefToken escapes a component name for use as a single JSON Pointer
// token, per RFC 6901: "~" becomes "~0" and "/" becomes "~1".
//
// Order matters — "~" must be escaped first, or the "~" introduced by escaping
// "/" would itself be escaped and produce "~01".
//
// OAS 2.0 places no charset constraint on the keys of the root-level
// parameters, definitions, and responses objects, so a name containing either
// character is legitimate and must be escaped for a document to reference it.
// Building a reference by raw concatenation instead produces a pointer that
// names a different location and therefore never matches.
func EscapeRefToken(name string) string {
	// Fast path: the overwhelming majority of component names need no escaping.
	if !strings.ContainsAny(name, "~/") {
		return name
	}
	name = strings.ReplaceAll(name, "~", "~0")
	return strings.ReplaceAll(name, "/", "~1")
}

// UnescapeRefToken reverses [EscapeRefToken], recovering a component name from
// a single JSON Pointer token.
//
// Use it anywhere a reference is split back into a name — renaming, pruning, or
// resolving a ref to its definition — or names containing "/" or "~" are
// corrupted. Order is the inverse of escaping: "~1" must be unescaped before
// "~0", or the "~" recovered from "~0" could combine with a following "1".
//
// Mirrors unescapeJSONPointer in the parser's resolver, which has always
// implemented this for reference resolution.
func UnescapeRefToken(token string) string {
	// Fast path: a token with no "~" cannot carry an escape sequence.
	if !strings.Contains(token, "~") {
		return token
	}
	token = strings.ReplaceAll(token, "~1", "/")
	return strings.ReplaceAll(token, "~0", "~")
}

// OAS 2.0 reference prefixes
const (
	RefPrefixDefinitions         = "#/definitions/"
	RefPrefixParameters          = "#/parameters/"
	RefPrefixResponses           = "#/responses/"
	RefPrefixSecurityDefinitions = "#/securityDefinitions/"
)

// OAS 3.x reference prefixes
const (
	RefPrefixSchemas         = "#/components/schemas/"
	RefPrefixParameters3     = "#/components/parameters/"
	RefPrefixResponses3      = "#/components/responses/"
	RefPrefixExamples        = "#/components/examples/"
	RefPrefixRequestBodies   = "#/components/requestBodies/"
	RefPrefixHeaders         = "#/components/headers/"
	RefPrefixSecuritySchemes = "#/components/securitySchemes/"
	RefPrefixLinks           = "#/components/links/"
	RefPrefixCallbacks       = "#/components/callbacks/"
	RefPrefixPathItems       = "#/components/pathItems/"
)

// SchemaRef builds "#/components/schemas/{name}" (OAS 3.x).
func SchemaRef(name string) string {
	return RefPrefixSchemas + EscapeRefToken(name)
}

// DefinitionRef builds "#/definitions/{name}" (OAS 2.0).
func DefinitionRef(name string) string {
	return RefPrefixDefinitions + EscapeRefToken(name)
}

// ParameterRef builds the appropriate parameter ref.
// If oas2 is true, returns "#/parameters/{name}", otherwise "#/components/parameters/{name}".
func ParameterRef(name string, oas2 bool) string {
	if oas2 {
		return RefPrefixParameters + EscapeRefToken(name)
	}
	return RefPrefixParameters3 + EscapeRefToken(name)
}

// ResponseRef builds the appropriate response ref.
// If oas2 is true, returns "#/responses/{name}", otherwise "#/components/responses/{name}".
func ResponseRef(name string, oas2 bool) string {
	if oas2 {
		return RefPrefixResponses + EscapeRefToken(name)
	}
	return RefPrefixResponses3 + EscapeRefToken(name)
}

// SecuritySchemeRef builds the appropriate security scheme ref.
// If oas2 is true, returns "#/securityDefinitions/{name}", otherwise "#/components/securitySchemes/{name}".
func SecuritySchemeRef(name string, oas2 bool) string {
	if oas2 {
		return RefPrefixSecurityDefinitions + EscapeRefToken(name)
	}
	return RefPrefixSecuritySchemes + EscapeRefToken(name)
}

// HeaderRef builds "#/components/headers/{name}" (OAS 3.x only).
func HeaderRef(name string) string {
	return RefPrefixHeaders + EscapeRefToken(name)
}

// RequestBodyRef builds "#/components/requestBodies/{name}" (OAS 3.x only).
func RequestBodyRef(name string) string {
	return RefPrefixRequestBodies + EscapeRefToken(name)
}

// ExampleRef builds "#/components/examples/{name}" (OAS 3.x only).
func ExampleRef(name string) string {
	return RefPrefixExamples + EscapeRefToken(name)
}

// LinkRef builds "#/components/links/{name}" (OAS 3.x only).
func LinkRef(name string) string {
	return RefPrefixLinks + EscapeRefToken(name)
}

// CallbackRef builds "#/components/callbacks/{name}" (OAS 3.x only).
func CallbackRef(name string) string {
	return RefPrefixCallbacks + EscapeRefToken(name)
}

// PathItemRef builds "#/components/pathItems/{name}" (OAS 3.1+ only).
func PathItemRef(name string) string {
	return RefPrefixPathItems + EscapeRefToken(name)
}
