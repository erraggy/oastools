// Copyright 2024 Erraggy
// SPDX-License-Identifier: MIT

package pathutil

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
	return RefPrefixSchemas + name
}

// DefinitionRef builds "#/definitions/{name}" (OAS 2.0).
func DefinitionRef(name string) string {
	return RefPrefixDefinitions + name
}

// ParameterRef builds the appropriate parameter ref.
// If oas2 is true, returns "#/parameters/{name}", otherwise "#/components/parameters/{name}".
func ParameterRef(name string, oas2 bool) string {
	if oas2 {
		return RefPrefixParameters + name
	}
	return RefPrefixParameters3 + name
}

// ResponseRef builds the appropriate response ref.
// If oas2 is true, returns "#/responses/{name}", otherwise "#/components/responses/{name}".
func ResponseRef(name string, oas2 bool) string {
	if oas2 {
		return RefPrefixResponses + name
	}
	return RefPrefixResponses3 + name
}

// SecuritySchemeRef builds the appropriate security scheme ref.
// If oas2 is true, returns "#/securityDefinitions/{name}", otherwise "#/components/securitySchemes/{name}".
func SecuritySchemeRef(name string, oas2 bool) string {
	if oas2 {
		return RefPrefixSecurityDefinitions + name
	}
	return RefPrefixSecuritySchemes + name
}

// HeaderRef builds "#/components/headers/{name}" (OAS 3.x only).
func HeaderRef(name string) string {
	return RefPrefixHeaders + name
}

// RequestBodyRef builds "#/components/requestBodies/{name}" (OAS 3.x only).
func RequestBodyRef(name string) string {
	return RefPrefixRequestBodies + name
}

// ExampleRef builds "#/components/examples/{name}" (OAS 3.x only).
func ExampleRef(name string) string {
	return RefPrefixExamples + name
}

// LinkRef builds "#/components/links/{name}" (OAS 3.x only).
func LinkRef(name string) string {
	return RefPrefixLinks + name
}

// CallbackRef builds "#/components/callbacks/{name}" (OAS 3.x only).
func CallbackRef(name string) string {
	return RefPrefixCallbacks + name
}

// PathItemRef builds "#/components/pathItems/{name}" (OAS 3.1+ only).
func PathItemRef(name string) string {
	return RefPrefixPathItems + name
}
