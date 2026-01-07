// Package fixer provides automatic fixes for common OpenAPI Specification validation errors.
//
// The fixer analyzes OAS documents and applies fixes for issues that would cause
// validation failures. It supports both OAS 2.0 and OAS 3.x documents. The fixer
// preserves the input file format (JSON or YAML) in the FixResult.SourceFormat
// field, allowing tools to maintain format consistency when writing output.
//
// # Quick Start
//
// Fix a file using functional options:
//
//	result, err := fixer.FixWithOptions(
//		fixer.WithFilePath("openapi.yaml"),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Applied %d fixes\n", result.FixCount)
//
// Or use a reusable Fixer instance:
//
//	f := fixer.New()
//	f.InferTypes = true // Infer parameter types from naming conventions
//	result1, _ := f.Fix("api1.yaml")
//	result2, _ := f.Fix("api2.yaml")
//
// # Supported Fixes
//
// The fixer currently supports the following automatic fixes:
//
//   - Missing path parameters (FixTypeMissingPathParameter): Adds Parameter objects
//     for path template variables that are not declared in the operation's parameters
//     list. For example, if a path is "/users/{userId}" but the operation doesn't
//     declare a "userId" path parameter, the fixer adds one with type "string" (or
//     inferred type if enabled).
//
//   - Invalid schema names (FixTypeRenamedGenericSchema): Renames schemas with names
//     containing characters that require URL encoding in $ref values. This commonly
//     occurs with code generators that produce generic type names like "Response[User]".
//     The fixer transforms these using configurable naming strategies.
//
//   - Unused schemas (FixTypePrunedUnusedSchema): Removes schema definitions that are
//     not referenced anywhere in the document. Useful for cleaning up orphaned schemas.
//
//   - Empty paths (FixTypePrunedEmptyPath): Removes path items that have no HTTP
//     operations defined (e.g., paths with only parameters but no get/post/etc).
//
// # Default Behavior
//
// For performance, only FixTypeMissingPathParameter is enabled by default.
// The schema renaming and pruning fixes involve expensive operations (walking
// all references, computing unused schemas) that can significantly slow down
// processing of large specifications.
//
// To enable additional fixes:
//
//	// Enable specific fixes via CLI flags
//	oastools fix --prune-unused api.yaml
//	oastools fix --rename-generics --prune-unused api.yaml
//
//	// Enable specific fixes programmatically
//	result, err := fixer.FixWithOptions(
//		fixer.WithFilePath("api.yaml"),
//		fixer.WithEnabledFixes(
//			fixer.FixTypeMissingPathParameter,
//			fixer.FixTypeRenamedGenericSchema,
//			fixer.FixTypePrunedUnusedSchema,
//		),
//	)
//
//	// Enable ALL fixes (backward compatible with pre-v1.28.1)
//	f := fixer.New()
//	f.EnabledFixes = []fixer.FixType{} // empty slice enables all
//	result, _ := f.Fix("api.yaml")
//
// # Generic Naming Strategies
//
// When fixing invalid schema names, the following strategies are available:
//
//   - GenericNamingUnderscore: Response[User] → Response_User_
//   - GenericNamingOf: Response[User] → ResponseOfUser
//   - GenericNamingFor: Response[User] → ResponseForUser
//   - GenericNamingFlattened: Response[User] → ResponseUser
//   - GenericNamingDot: Response[User] → Response.User
//
// Configure using WithGenericNaming() or WithGenericNamingConfig() options.
//
// # Type Inference
//
// When InferTypes is enabled (--infer flag in CLI), the fixer uses naming conventions
// to determine parameter types:
//
//   - Names ending in "id", "Id", or "ID" -> integer
//   - Names containing "uuid" or "guid" -> string with format "uuid"
//   - All other names -> string
//
// # Pipeline Usage
//
// The fixer is designed to work in a pipeline with other oastools commands:
//
//	# Fix and validate
//	oastools fix api.yaml | oastools validate -q -
//
//	# Fix and save
//	oastools fix api.yaml -o fixed.yaml
//
// # Chaining with Other Packages
//
// Use [FixResult.ToParseResult] to convert the fix result for use with other packages:
//
//	// Fix a specification
//	fixResult, _ := fixer.FixWithOptions(
//	    fixer.WithFilePath("api.yaml"),
//	)
//
//	// Validate the fixed result
//	v := validator.New()
//	validationResult, _ := v.ValidateParsed(*fixResult.ToParseResult())
//
//	// Or convert to a different version
//	c := converter.New()
//	convResult, _ := c.ConvertParsed(*fixResult.ToParseResult(), "3.1.0")
//
//	// Or diff against another document
//	diffResult, _ := differ.DiffWithOptions(
//	    differ.WithSourceParsed(*fixResult.ToParseResult()),
//	    differ.WithTargetFilePath("production.yaml"),
//	)
//
// The returned [parser.ParseResult] uses the source version (the version of the
// fixed document). Errors and Warnings are empty slices since fixes are
// informational, not validation errors.
//
// # Related Packages
//
// The fixer integrates with other oastools packages:
//   - [github.com/erraggy/oastools/parser] - Parse specifications before fixing
//   - [github.com/erraggy/oastools/validator] - Validate specifications (use to see errors)
//   - [github.com/erraggy/oastools/converter] - Convert between OAS versions
//   - [github.com/erraggy/oastools/joiner] - Join multiple specifications
//   - [github.com/erraggy/oastools/differ] - Compare specifications
//   - [github.com/erraggy/oastools/generator] - Generate code from specifications
//   - [github.com/erraggy/oastools/builder] - Programmatically build specifications
package fixer
