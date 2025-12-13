// Package oastools provides tools for parsing, validating, fixing, converting, joining,
// comparing, generating code from, and building OpenAPI Specification (OAS) documents from OAS 2.0 through OAS 3.2.0.
//
// The library consists of ten packages:
//
//   - [github.com/erraggy/oastools/parser] - Parse OpenAPI specifications from YAML or JSON
//   - [github.com/erraggy/oastools/validator] - Validate OpenAPI specifications against their declared version
//   - [github.com/erraggy/oastools/fixer] - Automatically fix common validation errors
//   - [github.com/erraggy/oastools/converter] - Convert OpenAPI specifications between different OAS versions
//   - [github.com/erraggy/oastools/joiner] - Join multiple OpenAPI specifications into a single document
//   - [github.com/erraggy/oastools/overlay] - Apply OpenAPI Overlay transformations with JSONPath targeting
//   - [github.com/erraggy/oastools/differ] - Compare OpenAPI specifications and detect breaking changes
//   - [github.com/erraggy/oastools/generator] - Generate idiomatic Go code for API clients and server stubs
//   - [github.com/erraggy/oastools/builder] - Programmatically construct OpenAPI specifications with reflection-based schema generation
//   - [github.com/erraggy/oastools/oaserrors] - Structured error types for programmatic handling
//
// For installation, CLI usage, and examples, see: https://github.com/erraggy/oastools
//
// For detailed API documentation and examples, see the individual package pages.
package oastools
