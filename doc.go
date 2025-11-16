// Package oastools provides comprehensive tools for working with OpenAPI Specification (OAS) documents.
//
// oastools offers four main packages for parsing, validating, converting, and joining OpenAPI
// specifications across all major versions from OAS 2.0 (Swagger) through OAS 3.2.0.
//
// # Overview
//
// The library consists of four primary packages:
//
//   - parser: Parse and analyze OpenAPI specifications
//   - validator: Validate OpenAPI specifications against their declared version
//   - converter: Convert OpenAPI specifications between different versions
//   - joiner: Join multiple OpenAPI specifications into a single document
//
// All packages support the following OpenAPI Specification versions:
//   - OAS 2.0 (Swagger): https://spec.openapis.org/oas/v2.0.html
//   - OAS 3.0.x (3.0.0 - 3.0.4): https://spec.openapis.org/oas/v3.0.0.html
//   - OAS 3.1.x (3.1.0 - 3.1.2): https://spec.openapis.org/oas/v3.1.0.html
//   - OAS 3.2.0: https://spec.openapis.org/oas/v3.2.0.html
//
// # Installation
//
// Install the library using go get:
//
//	go get github.com/erraggy/oastools
//
// # Quick Start
//
// Parse an OpenAPI specification:
//
//	import "github.com/erraggy/oastools/parser"
//
//	p := parser.New()
//	result, err := p.Parse("openapi.yaml")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Version: %s\n", result.Version)
//
// Validate an OpenAPI specification:
//
//	import "github.com/erraggy/oastools/validator"
//
//	v := validator.New()
//	result, err := v.Validate("openapi.yaml")
//	if err != nil {
//		log.Fatal(err)
//	}
//	if !result.Valid {
//		fmt.Printf("Found %d errors\n", result.ErrorCount)
//	}
//
// Join multiple OpenAPI specifications:
//
//	import "github.com/erraggy/oastools/joiner"
//
//	j := joiner.New(joiner.DefaultConfig())
//	result, err := j.Join([]string{"base.yaml", "extensions.yaml"})
//	if err != nil {
//		log.Fatal(err)
//	}
//	err = j.WriteResult(result, "merged.yaml")
//
// # Parser Package
//
// The parser package provides functionality to parse OpenAPI specification files
// in YAML or JSON format. It supports external reference resolution, version
// detection, and structural validation.
//
// Key features:
//   - Multi-format support (YAML, JSON)
//   - External reference resolution ($ref)
//   - Path traversal protection
//   - Operation ID uniqueness checking
//   - Memory-efficient caching
//
// Example:
//
//	p := parser.New()
//	p.ValidateStructure = true
//	p.ResolveRefs = true
//
//	result, err := p.Parse("openapi.yaml")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Type assertion to access version-specific fields
//	if doc, ok := result.Document.(*parser.OAS3Document); ok {
//		fmt.Printf("Title: %s\n", doc.Info.Title)
//		fmt.Printf("Paths: %d\n", len(doc.Paths))
//	}
//
// See the parser package documentation for more details.
//
// # Validator Package
//
// The validator package validates OpenAPI specifications against their declared
// version's requirements. It performs comprehensive structural, format, and
// semantic validation.
//
// Key features:
//   - Multi-version validation
//   - Structural validation (required fields, formats)
//   - Semantic validation (operation IDs, path parameters)
//   - Schema validation (JSON Schema)
//   - Security validation
//   - Best practice warnings (optional)
//   - Strict mode for additional checks
//
// Example:
//
//	v := validator.New()
//	v.StrictMode = true
//	v.IncludeWarnings = true
//
//	result, err := v.Validate("openapi.yaml")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	for _, verr := range result.Errors {
//		fmt.Printf("%s: %s\n", verr.Path, verr.Message)
//		if verr.SpecRef != "" {
//			fmt.Printf("  See: %s\n", verr.SpecRef)
//		}
//	}
//
// See the validator package documentation for more details.
//
// # Converter Package
//
// The converter package converts OpenAPI specifications between different OAS versions.
// It performs best-effort conversion while tracking issues and incompatibilities.
//
// Key features:
//   - OAS 2.0 ↔ OAS 3.x conversion
//   - OAS 3.x → OAS 3.y version updates
//   - Best-effort conversion preserving maximum information
//   - Detailed issue tracking with severity levels (Info, Warning, Critical)
//   - Security scheme conversion
//   - Schema conversion across versions
//
// Example:
//
//	c := converter.New()
//	result, err := c.Convert("swagger.yaml", "3.0.3")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if result.HasCriticalIssues() {
//		fmt.Printf("Conversion completed with %d critical issue(s):\n", result.CriticalCount)
//		for _, issue := range result.Issues {
//			if issue.Severity == converter.SeverityCritical {
//				fmt.Printf("  %s\n", issue.String())
//			}
//		}
//	}
//
// See the converter package documentation for more details.
//
// # Joiner Package
//
// The joiner package enables merging multiple OpenAPI specification documents
// into a single unified document. It provides flexible collision resolution
// strategies and supports all OAS versions.
//
// Key features:
//   - Flexible collision strategies (accept-left, accept-right, fail)
//   - Component-specific strategies (paths, schemas, components)
//   - Array merging (servers, security, tags)
//   - Tag deduplication
//   - Version compatibility checking
//   - Detailed collision reporting
//
// Example:
//
//	config := joiner.JoinerConfig{
//		PathStrategy:      joiner.StrategyFailOnCollision,
//		SchemaStrategy:    joiner.StrategyAcceptLeft,
//		ComponentStrategy: joiner.StrategyAcceptLeft,
//		MergeArrays:       true,
//		DeduplicateTags:   true,
//	}
//
//	j := joiner.New(config)
//	result, err := j.Join([]string{"base.yaml", "ext1.yaml", "ext2.yaml"})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	err = j.WriteResult(result, "merged.yaml")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// See the joiner package documentation for more details.
//
// # Common Workflows
//
// Validate before processing:
//
//	// Validate first
//	v := validator.New()
//	vResult, err := v.Validate("api.yaml")
//	if err != nil || !vResult.Valid {
//		log.Fatal("Invalid OpenAPI specification")
//	}
//
//	// Then parse
//	p := parser.New()
//	pResult, err := p.Parse("api.yaml")
//	if err != nil {
//		log.Fatal(err)
//	}
//	// Process the parsed document...
//
// Join and validate:
//
//	// Join multiple specs
//	j := joiner.New(joiner.DefaultConfig())
//	jResult, err := j.Join([]string{"base.yaml", "ext.yaml"})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Write to temp file
//	tempFile := "merged-temp.yaml"
//	err = j.WriteResult(jResult, tempFile)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Validate the joined result
//	v := validator.New()
//	vResult, err := v.Validate(tempFile)
//	if err != nil || !vResult.Valid {
//		log.Fatal("Joined specification is invalid")
//	}
//
// Parse multiple versions:
//
//	p := parser.New()
//	files := []string{"api-v2.yaml", "api-v3.0.yaml", "api-v3.1.yaml"}
//
//	for _, file := range files {
//		result, err := p.Parse(file)
//		if err != nil {
//			log.Printf("Failed to parse %s: %v", file, err)
//			continue
//		}
//		fmt.Printf("%s: OAS %s\n", file, result.Version)
//	}
//
// # Security Considerations
//
// All packages implement security best practices:
//
//   - Path traversal protection: External references are restricted to the base
//     directory and subdirectories
//   - Resource limits: Maximum cached documents (default: 1000) and schema
//     nesting depth (default: 100) to prevent resource exhaustion
//   - Input validation: All user-provided values are validated before processing
//   - File permissions: Output files are created with restrictive permissions (0600)
//   - No remote references: HTTP(S) URLs in $ref are not currently supported,
//     limiting attack surface
//
// # Limitations
//
// Current limitations across all packages:
//
//   - HTTP(S) references: Remote URL references in $ref are not supported; only
//     local file references are allowed
//   - External reference resolution: The joiner preserves $ref values as-is and
//     does not resolve or merge referenced content across documents
//   - Cross-version joining: Cannot join OAS 2.0 with OAS 3.x documents
//   - Custom extensions: OpenAPI extension fields (x-*) are preserved but not
//     validated or merged with custom logic
//
// # Performance Tips
//
// For best performance:
//
//   - Disable reference resolution if not needed (parser.ResolveRefs = false)
//   - Disable warnings if not needed (validator.IncludeWarnings = false)
//   - Use appropriate collision strategies for joining (accept-left/right vs fail)
//   - Cache validation results for unchanged documents
//   - Process documents concurrently when possible (packages are not goroutine-safe,
//     create separate instances for concurrent use)
//
// # Error Handling
//
// All packages follow consistent error handling patterns:
//
//   - File I/O errors: Returned directly (e.g., os.ErrNotExist)
//   - Parse errors: Returned with context about what failed
//   - Validation errors: Collected in ValidationResult.Errors (not returned as error)
//   - Join errors: Returned with detailed collision information
//
// Always check both the error return value and any error/warning collections
// in result objects.
//
// # Version Compatibility
//
// This library is designed to be backward compatible within major versions.
// The public API follows semantic versioning:
//
//   - Major version changes may include breaking API changes
//   - Minor version changes add functionality in a backward-compatible manner
//   - Patch version changes include backward-compatible bug fixes
//
// When upgrading, check the CHANGELOG for any breaking changes or new features.
//
// # Command-Line Interface
//
// In addition to the library packages, oastools provides a command-line interface:
//
//	# Validate a spec
//	oastools validate openapi.yaml
//
//	# Parse a spec
//	oastools parse openapi.yaml
//
//	# Convert between versions
//	oastools convert -t 3.0.3 swagger.yaml -o openapi.yaml
//
//	# Join multiple specs
//	oastools join -o merged.yaml base.yaml extensions.yaml
//
// Install the CLI:
//
//	go install github.com/erraggy/oastools/cmd/oastools@latest
//
// # Additional Resources
//
//   - GitHub Repository: https://github.com/erraggy/oastools
//   - OpenAPI Specification: https://spec.openapis.org
//   - JSON Schema Specification: https://json-schema.org
//   - Go Package Documentation: https://pkg.go.dev/github.com/erraggy/oastools
//
// # License
//
// This library is released under the MIT License. See the LICENSE file in the
// repository for full details.
package oastools
