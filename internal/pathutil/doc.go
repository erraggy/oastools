// Copyright 2024 Erraggy
// SPDX-License-Identifier: MIT

// Package pathutil provides efficient path building utilities for OpenAPI
// document traversal.
//
// The primary type is [PathBuilder], which uses push/pop semantics to build
// paths incrementally without allocating intermediate strings. This is
// particularly useful in recursive traversal where paths are built on each
// recursive call but only used when reporting errors or differences.
//
// # PathBuilder Usage
//
// Use [Get] to obtain a pooled PathBuilder, and [Put] to return it:
//
//	path := pathutil.Get()
//	defer pathutil.Put(path)
//
//	path.Push("properties")
//	path.Push(propName)
//	// ... recurse ...
//	path.Pop()
//	path.Pop()
//
//	// Only call String() when needed (e.g., reporting an error)
//	if hasError {
//	    return fmt.Errorf("error at %s", path.String())
//	}
//
// Array indices are supported via [PathBuilder.PushIndex]:
//
//	path.Push("items")
//	path.PushIndex(0)  // produces "items[0]"
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
