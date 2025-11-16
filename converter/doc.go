// Package converter provides OpenAPI Specification (OAS) version conversion functionality.
//
// This package converts OpenAPI specifications between different OAS versions while
// tracking conversion issues and incompatibilities. It performs best-effort conversion
// to preserve maximum information while clearly documenting any lossy transformations
// or features that cannot be converted.
//
// # Supported Conversions
//
// The converter supports the following conversion paths:
//   - OAS 2.0 (Swagger) → OAS 3.x (3.0.0 through 3.2.0)
//   - OAS 3.x → OAS 2.0
//   - OAS 3.x → OAS 3.y (version updates within 3.x family)
//
// All conversions are designed to be lossless where possible, but some features
// are specific to certain versions and cannot be fully converted. The converter
// tracks all such limitations as ConversionIssue items with appropriate severity levels.
//
// # Features
//
//   - Multi-version support: Convert between OAS 2.0 and all OAS 3.x versions
//   - Best-effort conversion: Preserves maximum information while documenting limitations
//   - Issue tracking: Detailed reporting of conversion issues with severity levels
//   - Schema conversion: Handles JSON schema differences between versions
//   - Security scheme conversion: Converts between different security definition formats
//   - Dual API pattern: Convenience functions and reusable converter instances
//
// # Conversion Severity Levels
//
// The converter provides three severity levels for issues:
//
//   - SeverityInfo: Informational messages about conversion choices
//   - SeverityWarning: Lossy conversions or best-effort transformations
//   - SeverityCritical: Features that cannot be converted (data loss)
//
// Critical issues indicate features that exist in the source version but have no
// equivalent in the target version. Warnings indicate lossy conversions where some
// information may be lost but a reasonable approximation can be made. Info messages
// provide additional context about conversion choices.
//
// # OAS 2.0 to OAS 3.x Conversion
//
// When converting from OAS 2.0 to OAS 3.x, the following transformations occur:
//
// Automatic Conversions:
//   - swagger/host/basePath/schemes → servers array
//   - definitions → components.schemas
//   - parameters → components.parameters
//   - responses → components.responses
//   - securityDefinitions → components.securitySchemes
//   - consumes/produces → requestBody.content and responses.content
//   - Body parameters → requestBody objects
//   - OAuth2 flows → OAS 3.x flow objects
//   - Basic authentication → HTTP security scheme
//
// Known Limitations:
//   - collectionFormat values may not map perfectly to style/explode parameters (Warning)
//   - allowEmptyValue was removed in OAS 3.0 (Warning)
//   - Some OAuth2 flow names changed between versions (converted automatically)
//
// # OAS 3.x to OAS 2.0 Conversion
//
// When converting from OAS 3.x to OAS 2.0, the following transformations occur:
//
// Automatic Conversions:
//   - First server → host/basePath/schemes
//   - components.schemas → definitions
//   - components.parameters → parameters
//   - components.responses → responses
//   - components.securitySchemes → securityDefinitions
//   - requestBody → body parameter with schema
//   - Media types from content → consumes/produces arrays
//   - HTTP basic scheme → basic authentication
//   - OAuth2 flows → single OAuth2 flow
//
// Known Limitations (Critical Issues):
//   - Webhooks cannot be represented in OAS 2.0 (OAS 3.1+)
//   - Callbacks cannot be converted
//   - Links cannot be converted
//   - TRACE HTTP method is not supported
//   - Cookie parameters (in: cookie) not supported in OAS 2.0
//   - OpenID Connect security schemes not supported
//   - Multiple servers (only first server is converted)
//
// Known Limitations (Warnings):
//   - Multiple media types in requestBody (uses first, lists others in consumes)
//   - Multiple OAuth2 flows (uses first, ignores others)
//   - Non-basic HTTP authentication schemes
//   - Server variables are removed
//   - Style/explode parameter settings
//   - Nullable schemas (OAS 3.0+)
//
// # OAS 3.x to OAS 3.y Conversion
//
// When converting between OAS 3.x versions (e.g., 3.0.3 to 3.1.0), the converter
// primarily updates the version string. While the 3.x family is generally compatible,
// minor versions can introduce new features:
//
//   - 3.1.0 added: webhooks, better JSON Schema alignment
//   - 3.2.0 added: additional features (check spec for details)
//
// The converter adds an informational message about the version update and reminds
// users to verify that features used are supported in the target version.
//
// # Basic Usage
//
// For simple, one-off conversion, use the convenience function:
//
//	result, err := converter.Convert("swagger.yaml", "3.0.3")
//	if err != nil {
//		log.Fatalf("Conversion failed: %v", err)
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
// For converting multiple files with the same configuration, create a Converter instance:
//
//	c := converter.New()
//	c.StrictMode = false
//	c.IncludeInfo = true
//
//	result1, err := c.Convert("api1.yaml", "3.0.3")
//	result2, err := c.Convert("api2.yaml", "3.0.3")
//
// # Advanced Usage
//
// Enable strict mode to fail on any issues (even warnings):
//
//	c := converter.New()
//	c.StrictMode = true
//	result, err := c.Convert("swagger.yaml", "3.0.3")
//	if err != nil {
//		// err will be non-nil if there are any warnings or critical issues
//		log.Fatalf("Strict conversion failed: %v", err)
//	}
//
// Convert an already-parsed document:
//
//	parseResult, _ := parser.Parse("openapi.yaml", false, true)
//	result, err := converter.ConvertParsed(*parseResult, "2.0")
//	if err != nil {
//		log.Fatalf("Conversion failed: %v", err)
//	}
//
// Suppress informational messages:
//
//	c := converter.New()
//	c.IncludeInfo = false
//	result, err := c.Convert("swagger.yaml", "3.0.3")
//	// result.Issues will only contain warnings and critical issues
//
// # Issue Reporting
//
// The ConversionResult contains detailed information about all issues encountered:
//
//	for _, issue := range result.Issues {
//		fmt.Printf("[%s] %s: %s\n", issue.Severity, issue.Path, issue.Message)
//		if issue.Context != "" {
//			fmt.Printf("    Context: %s\n", issue.Context)
//		}
//	}
//
// Issues are categorized by severity:
//
//	fmt.Printf("Conversion summary:\n")
//	fmt.Printf("  Info:     %d\n", result.InfoCount)
//	fmt.Printf("  Warnings: %d\n", result.WarningCount)
//	fmt.Printf("  Critical: %d\n", result.CriticalCount)
//
// # Writing Converted Documents
//
// Currently, converter doesn't include a WriteResult method. Use YAML marshal
// from the yaml.v3 package:
//
//	import "gopkg.in/yaml.v3"
//
//	data, err := yaml.Marshal(result.Document)
//	if err != nil {
//		log.Fatalf("Failed to marshal: %v", err)
//	}
//	os.WriteFile("converted.yaml", data, 0600)
//
// # Validation After Conversion
//
// Always validate the converted document to ensure it's valid for the target version:
//
//	// Convert the document
//	convResult, err := converter.Convert("swagger.yaml", "3.0.3")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Write to temporary file
//	tmpFile := "temp-converted.yaml"
//	data, _ := yaml.Marshal(convResult.Document)
//	os.WriteFile(tmpFile, data, 0600)
//
//	// Validate the converted document
//	valResult, err := validator.Validate(tmpFile, true, false)
//	if err != nil {
//		log.Fatal(err)
//	}
//	if !valResult.Valid {
//		fmt.Printf("Converted document has validation errors:\n")
//		for _, err := range valResult.Errors {
//			fmt.Printf("  %s\n", err.String())
//		}
//	}
//
// # Performance Notes
//
// The converter performs deep copying of document structures to avoid mutations.
// Performance considerations:
//
//   - Large documents: Deep copying is O(n) in document size
//   - Complex schemas: Nested schema conversion requires traversal
//   - Multiple conversions: Reuse Converter instances for better performance
//
// For better performance:
//   - Convert documents during build/deployment rather than runtime
//   - Reuse Converter instances when converting multiple documents
//   - Disable info messages (IncludeInfo: false) if not needed
//   - Cache conversion results for unchanged documents
//
// # Conversion Result Format
//
// The ConversionResult contains:
//
//   - Document: The converted document (*parser.OAS2Document or *parser.OAS3Document)
//   - SourceVersion: Detected source OAS version string
//   - SourceOASVersion: Enumerated source OAS version
//   - TargetVersion: Target OAS version string
//   - TargetOASVersion: Enumerated target OAS version
//   - Issues: All conversion issues with paths and descriptions
//   - InfoCount: Number of informational messages
//   - WarningCount: Number of warnings
//   - CriticalCount: Number of critical issues
//   - Success: True if conversion completed without critical issues
//
// Each ConversionIssue contains:
//
//   - Path: JSON path to the affected element (e.g., "paths./pets.get.parameters[0]")
//   - Message: Human-readable description of the issue
//   - Severity: Info, Warning, or Critical
//   - Field: Specific field name (optional)
//   - Value: The problematic value (optional)
//   - Context: Additional context or suggestions (optional)
//
// # Common Conversion Issues
//
// Critical issues (features that cannot be converted):
//   - "Webhooks are OAS 3.1+ only and cannot be converted to OAS 2.0"
//   - "Operation contains callbacks which are not supported in OAS 2.0"
//   - "Response contains links which are not supported in OAS 2.0"
//   - "Cookie parameters are not supported in OAS 2.0"
//   - "TRACE method is OAS 3.x only and cannot be converted to OAS 2.0"
//   - "OpenID Connect is OAS 3.x only and cannot be converted to OAS 2.0"
//
// Warnings (lossy conversions):
//   - "Multiple servers defined (N), using only the first one"
//   - "RequestBody has multiple media types (N), using first (application/json)"
//   - "Multiple OAuth2 flows defined (N), using only one"
//   - "Parameter uses collectionFormat 'X'"
//   - "Parameter uses 'allowEmptyValue'"
//   - "Schema uses 'nullable' which is OAS 3.0+"
//   - "Server variables are not supported in OAS 2.0"
//
// Info messages (conversions choices):
//   - "No host specified in OAS 2.0 document, using default server"
//   - "No servers defined in OAS 3.x document, using defaults"
//   - "Updated version from X to Y"
//   - "Source and target versions are the same (X), no conversion needed"
//
// # Limitations
//
//   - External references: The converter preserves $ref but does not follow or convert referenced documents
//   - Custom extensions: Extension fields (x-*) are preserved but not validated or converted
//   - Complex schemas: Some advanced JSON Schema keywords may not convert perfectly
//   - Specification evolution: Future OAS versions may introduce breaking changes requiring updates
//
// # Best Practices
//
//  1. Always validate converted documents using the validator package
//  2. Review critical issues and warnings before deploying converted specs
//  3. Test converted APIs to ensure behavior is preserved
//  4. Keep backups of original specifications
//  5. Use version control to track conversion history
//  6. Document any manual adjustments made after conversion
//  7. Consider the target version's feature set before converting
//
// # Version Selection
//
// When choosing a target version:
//
//   - Converting to 3.0.3: Most compatible with older tools and libraries
//   - Converting to 3.1.x: Better JSON Schema alignment, adds webhooks
//   - Converting to 2.0: For legacy tools that only support Swagger 2.0
//
// # Error Handling
//
// The converter returns errors for:
//
//   - Invalid source documents (parse errors)
//   - Invalid target version strings
//   - Unsupported conversion paths
//   - Document type mismatches
//   - Strict mode failures (when warnings/critical issues exist)
//
// The converter does NOT return errors for:
//
//   - Conversion issues (tracked in ConversionResult.Issues)
//   - Lossy conversions (tracked as warnings)
//   - Feature incompatibilities (tracked as critical issues)
//
// Check ConversionResult.Success and ConversionResult.Issues for conversion
// outcomes rather than relying solely on the error return value.
package converter
