// Package converter provides version conversion for OpenAPI Specification documents.
//
// The converter supports OAS 2.0 ↔ OAS 3.x conversions, performing best-effort
// conversion with detailed issue tracking. Features converted include servers,
// schemas, parameters, security schemes, and request/response bodies. The converter
// preserves the input file format (JSON or YAML) in the ConversionResult.SourceFormat
// field, allowing tools to maintain format consistency when writing output.
//
// # Quick Start
//
// Convert a file using functional options:
//
//	result, err := converter.ConvertWithOptions(
//		converter.WithFilePath("swagger.yaml"),
//		converter.WithTargetVersion("3.0.3"),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	if result.HasCriticalIssues() {
//		fmt.Printf("%d critical issue(s)\n", result.CriticalCount)
//	}
//
// Or use a reusable Converter instance:
//
//	c := converter.New()
//	c.StrictMode = false
//	result1, _ := c.Convert("api1.yaml", "3.0.3")
//	result2, _ := c.Convert("api2.yaml", "3.0.3")
//
// # Conversion Issues
//
// The converter tracks three severity levels: Info (conversion choices), Warning
// (lossy conversions), and Critical (features that cannot be converted). Some
// OAS 3.x features (webhooks, callbacks, links, TRACE method) cannot convert to
// OAS 2.0. Some OAS 2.0 features (collectionFormat, allowEmptyValue) may not map
// perfectly to OAS 3.x. See the examples in example_test.go for handling issues.
//
// # Converting with the Validator Package
//
// Always validate converted documents for the target version:
//
//	convResult, _ := converter.ConvertWithOptions(
//		converter.WithFilePath("swagger.yaml"),
//		converter.WithTargetVersion("3.0.3"),
//	)
//	data, _ := yaml.Marshal(convResult.Document)
//	tmpFile := "temp.yaml"
//	os.WriteFile(tmpFile, data, 0600)
//	valResult, _ := validator.ValidateWithOptions(
//		validator.WithFilePath(tmpFile),
//		validator.WithIncludeWarnings(true),
//	)
//	if !valResult.Valid {
//		fmt.Printf("Conversion produced invalid document\n")
//	}
//
// See the exported ConversionResult and ConversionIssue types for complete details.
//
// # Related Packages
//
// Conversion integrates with other oastools packages:
//   - [github.com/erraggy/oastools/parser] - Parse specifications before conversion
//   - [github.com/erraggy/oastools/validator] - Validate converted specifications
//   - [github.com/erraggy/oastools/fixer] - Fix common validation errors in specifications
//   - [github.com/erraggy/oastools/joiner] - Join converted specifications
//   - [github.com/erraggy/oastools/differ] - Compare original and converted specifications
//   - [github.com/erraggy/oastools/generator] - Generate code from converted specifications
//   - [github.com/erraggy/oastools/builder] - Programmatically build specifications
package converter
