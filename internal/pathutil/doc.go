// Copyright 2024 Erraggy
// SPDX-License-Identifier: MIT

// Package pathutil provides reference building, escaping, and path
// sanitization utilities for OpenAPI document traversal.
//
// # Reference Builders
//
// The package also provides functions for building JSON Pointer references
// to OpenAPI components:
//
//	ref := pathutil.SchemaRef("Pet")      // "#/components/schemas/Pet"
//	ref := pathutil.DefinitionRef("Pet")  // "#/definitions/Pet"
//
// These use simple string concatenation which Go optimizes well for two
// operands, avoiding the overhead of fmt.Sprintf.
//
// Version-aware helpers handle OAS 2.0 vs 3.x differences:
//
//	ref := pathutil.ParameterRef("limit", true)   // "#/parameters/limit" (OAS 2.0)
//	ref := pathutil.ParameterRef("limit", false)  // "#/components/parameters/limit" (OAS 3.x)
//
// # Reference Token Escaping
//
// Component names are not always safe to drop into a JSON Pointer as-is: RFC
// 6901 gives "~" and "/" special meaning, so a component legitimately named
// "pet/summary" must be referenced as "#/definitions/pet~1summary".
// [EscapeRefToken] and [UnescapeRefToken] convert a single pointer token in
// each direction:
//
//	tok := pathutil.EscapeRefToken("pet/summary")   // "pet~1summary"
//	name := pathutil.UnescapeRefToken("pet~1summary") // "pet/summary"
//
// Escape per token, never across a whole reference — escaping the full string
// would rewrite the "/" separators that give the pointer its structure.
//
// # Output Path Sanitization
//
// [SanitizeOutputPath] validates and cleans output file paths for security.
// It rejects directory traversal ("..") and symlinks:
//
//	safe, err := pathutil.SanitizeOutputPath(userProvidedPath)
//	if err != nil {
//	    return err // path traversal or symlink detected
//	}
package pathutil
