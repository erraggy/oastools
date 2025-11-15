// Package validator provides OpenAPI Specification (OAS) validation functionality.
//
// This package validates OpenAPI specifications across multiple versions against
// their respective specification requirements. It performs structural validation,
// format checking, and semantic analysis to ensure API specifications are correct
// and conformant.
//
// # Supported Versions
//
// The validator supports all official OpenAPI Specification releases:
//   - OAS 2.0 (Swagger): https://spec.openapis.org/oas/v2.0.html
//   - OAS 3.0.x (3.0.0 through 3.0.4): https://spec.openapis.org/oas/v3.0.0.html
//   - OAS 3.1.x (3.1.0 through 3.1.2): https://spec.openapis.org/oas/v3.1.0.html
//   - OAS 3.2.0: https://spec.openapis.org/oas/v3.2.0.html
//
// All schema definitions are validated against JSON Schema Specification Draft 2020-12:
// https://www.ietf.org/archive/id/draft-bhutton-json-schema-01.html
//
// Release candidate versions (e.g., 3.0.0-rc0) are detected but not officially supported.
//
// # Features
//
//   - Multi-version support: Validates OAS 2.0 through OAS 3.2.0
//   - Structural validation: Ensures required fields are present and properly formatted
//   - Format validation: Validates URLs, emails, media types, HTTP status codes
//   - Semantic validation: Checks operation ID uniqueness, path parameter consistency
//   - Schema validation: Validates JSON schemas including type constraints and nested structures
//   - Security validation: Validates security schemes and requirements
//   - Best practice warnings: Optional recommendations for better API design
//   - Strict mode: Additional validation beyond specification requirements
//
// # Validation Levels
//
// The validator provides two severity levels for issues:
//
//   - SeverityError: Specification violations that make the document invalid
//   - SeverityWarning: Best practice violations or recommendations (optional)
//
// Warnings can be suppressed by setting IncludeWarnings to false. Strict mode
// can be enabled to perform additional validation beyond spec requirements.
//
// # Validation Rules
//
// The validator checks numerous aspects of OpenAPI documents:
//
// Info Object:
//   - Required fields: title, version
//   - Format: Valid URLs for contact/license, valid email format
//
// Paths:
//   - Path patterns must start with "/"
//   - Path templates must be well-formed (no empty braces, nested braces, etc.)
//   - Path parameters must be declared and match template variables
//
// Operations:
//   - Operation IDs must be unique across the entire document
//   - HTTP status codes must be valid (100-599 or wildcard patterns like "2XX")
//   - Media types must follow RFC 2045/2046 format
//   - Request bodies must have at least one media type (OAS 3.x)
//
// Parameters:
//   - Path parameters must have required: true
//   - Body parameters must have a schema (OAS 2.0)
//   - Non-body parameters must have a type (OAS 2.0)
//   - Parameters must have either schema or content (OAS 3.x)
//
// Schemas:
//   - Array schemas must have 'items' defined
//   - minLength/maxLength must be consistent
//   - minimum/maximum must be consistent
//   - Required fields must exist in properties
//   - Enum values must match schema type
//
// Security:
//   - Security requirements must reference defined security schemes
//   - Security schemes must have required fields for their type
//   - OAuth2 flows must have required URLs
//
// # Security Considerations
//
// The validator implements several protections:
//
//   - Resource limits: Maximum schema nesting depth (100) to prevent stack overflow
//   - Cycle detection: Prevents infinite loops in circular schema references
//   - Format validation: Uses standard library parsing for URLs, media types, etc.
//   - Input validation: All user-provided values are validated before processing
//
// # Basic Usage
//
// For simple, one-off validation, use the convenience function:
//
//	result, err := validator.Validate("openapi.yaml", true, false)
//	if err != nil {
//		log.Fatalf("Validation failed: %v", err)
//	}
//
//	if !result.Valid {
//		fmt.Printf("Found %d error(s):\n", result.ErrorCount)
//		for _, err := range result.Errors {
//			fmt.Printf("  %s\n", err.String())
//		}
//	}
//
// For validating multiple files with the same configuration, create a Validator instance:
//
//	v := validator.New()
//	v.StrictMode = true
//	v.IncludeWarnings = true
//
//	result1, err := v.Validate("api1.yaml")
//	result2, err := v.Validate("api2.yaml")
//
// # Advanced Usage
//
// Enable strict mode with warnings:
//
//	result, err := validator.Validate("openapi.yaml", true, true)
//	if err != nil {
//		log.Fatalf("Validation failed: %v", err)
//	}
//
//	// Process errors
//	for _, verr := range result.Errors {
//		fmt.Printf("ERROR: %s at %s\n", verr.Message, verr.Path)
//		if verr.SpecRef != "" {
//			fmt.Printf("  See: %s\n", verr.SpecRef)
//		}
//	}
//
//	// Process warnings
//	for _, warn := range result.Warnings {
//		fmt.Printf("WARNING: %s at %s\n", warn.Message, warn.Path)
//	}
//
// Suppress warnings for production:
//
//	result, err := validator.Validate("openapi.yaml", false, false)
//	if err != nil {
//		log.Fatalf("Validation failed: %v", err)
//	}
//
//	// Only errors will be reported
//	if !result.Valid {
//		log.Printf("Validation failed with %d errors", result.ErrorCount)
//	}
//
// # Strict Mode
//
// When StrictMode is enabled, the validator performs additional checks:
//
//   - Warns about non-standard HTTP status codes (not defined in RFCs)
//   - Warns if operations don't have at least one successful (2XX) response
//   - May add additional best practice validations in future versions
//
// # Validation Output
//
// The ValidationResult contains:
//
//   - Valid: Boolean indicating if the document is valid (no errors)
//   - Version: Detected OpenAPI version string (e.g., "3.0.3")
//   - OASVersion: Enumerated OAS version for programmatic use
//   - Errors: Slice of validation errors with paths and spec references
//   - Warnings: Slice of validation warnings (if IncludeWarnings is true)
//   - ErrorCount: Total number of errors
//   - WarningCount: Total number of warnings
//
// Each ValidationError contains:
//
//   - Path: JSON path to the problematic field (e.g., "paths./pets.get.responses")
//   - Message: Human-readable error description
//   - SpecRef: URL to the relevant section of the OAS specification
//   - Severity: Error or Warning
//   - Field: Specific field name that has the issue (optional)
//   - Value: The problematic value (optional)
//
// # Performance Notes
//
// The validator performs comprehensive validation which may be resource-intensive
// for large documents. Performance considerations:
//
//   - Schema validation: Deep schemas with many levels of nesting will take longer
//   - Cycle detection: Maintains a visited map for each schema tree traversal
//   - Operation ID checking: Requires scanning all operations in the document
//   - Parameter consistency: Requires comparing path templates with parameter definitions
//
// For better performance:
//   - Validate documents during development rather than at runtime
//   - Disable warnings (IncludeWarnings: false) if not needed
//   - Disable strict mode if the additional checks aren't required
//   - Cache validation results for unchanged documents
//
// # Error Path Format
//
// Validation errors include a Path field that uses JSON path notation to identify
// the location of the issue:
//
//   - "info.title" - Missing or invalid title in info object
//   - "paths./pets.get.responses" - Issue with responses in GET /pets
//   - "components.schemas.Pet.properties.name" - Issue with Pet.name property
//   - "paths./users/{id}.get.parameters[0]" - Issue with first parameter
//
// # Common Validation Errors
//
// Missing required fields:
//   - "Info object must have a title"
//   - "Info object must have a version"
//   - "Response must have a description"
//
// Format validation:
//   - "Path must start with '/'"
//   - "Invalid HTTP status code: 999"
//   - "Invalid URL format: not-a-url"
//   - "Invalid email format: invalid@@email"
//
// Parameter issues:
//   - "Path parameters must have required: true"
//   - "Path template references parameter '{id}' but it is not declared"
//   - "Body parameter must have a schema"
//
// Schema validation:
//   - "Array schema must have 'items' defined"
//   - "minLength (10) cannot be greater than maxLength (5)"
//   - "Required field 'name' not found in properties"
//
// # Limitations
//
//   - External references: The validator validates the structure but does not
//     follow or validate external $ref links
//   - Custom validators: No support for custom validation rules or plugins
//   - Schema keywords: Some advanced JSON Schema keywords may not be fully validated
//   - OpenAPI extensions: Extension fields (x-*) are preserved but not validated
package validator
