// Package joiner provides OpenAPI Specification (OAS) joining functionality.
//
// This package enables merging multiple OpenAPI specification documents into a
// single unified document. It supports all OAS versions from 2.0 through 3.2.0
// and provides flexible collision resolution strategies for handling conflicts
// between documents.
//
// # Supported Versions
//
// The joiner supports combining documents of the same major version:
//   - OAS 2.0 (Swagger) documents can be joined with other 2.0 documents
//   - All OAS 3.x versions (3.0.x, 3.1.x, 3.2.x) can be joined together
//
// The resulting document will use the OpenAPI version declared in the first
// input document. While minor version mismatches (e.g., 3.0.3 + 3.1.0) are
// allowed, users should verify the joined document is valid for its declared
// version, as features from newer versions may be incompatible.
//
// See the OpenAPI Specification references:
//   - OAS 2.0: https://spec.openapis.org/oas/v2.0.html
//   - OAS 3.0.x: https://spec.openapis.org/oas/v3.0.0.html
//   - OAS 3.1.x: https://spec.openapis.org/oas/v3.1.0.html
//   - OAS 3.2.0: https://spec.openapis.org/oas/v3.2.0.html
//
// # Features
//
//   - Flexible collision resolution with configurable strategies
//   - Support for all major OpenAPI components (paths, schemas, parameters, etc.)
//   - Array merging for servers, security requirements, and tags
//   - Tag deduplication by name
//   - Detailed collision reporting and warnings
//   - Version compatibility validation
//
// # Collision Strategies
//
// When joining documents, the joiner handles collisions using configurable strategies:
//
//   - StrategyAcceptLeft: Keep the value from the first document (default for schemas/components)
//   - StrategyAcceptRight: Keep the value from the last document (overwrite)
//   - StrategyFailOnCollision: Return an error on any collision (default for all)
//   - StrategyFailOnPaths: Fail only on path collisions, allow schema collisions
//
// Different strategies can be set globally or for specific component types:
//
//   - PathStrategy: Controls collision handling for API paths and webhooks
//   - SchemaStrategy: Controls collision handling for schemas/definitions
//   - ComponentStrategy: Controls collision handling for other components
//     (parameters, responses, examples, request bodies, headers, security schemes, links, callbacks)
//
// # Security Considerations
//
// The joiner implements several security protections:
//
//   - File overwrite protection: Validates that the output path does not
//     overwrite any input files
//
//   - Restrictive permissions: Output files are created with 0600 permissions
//     (owner read/write only) to protect potentially sensitive API specifications
//
//   - Validation: All input documents are validated before joining to prevent
//     combining invalid or malformed specifications
//
//   - Resource limits: Inherits MaxCachedDocuments limit from the parser
//     (default: 1000) to prevent memory exhaustion
//
// # Basic Usage
//
// For simple, one-off joining, use the convenience function:
//
//	config := joiner.DefaultConfig()
//	config.PathStrategy = joiner.StrategyAcceptLeft
//	config.SchemaStrategy = joiner.StrategyAcceptLeft
//
//	result, err := joiner.Join([]string{"api-base.yaml", "api-extensions.yaml"}, config)
//	if err != nil {
//		log.Fatalf("Join failed: %v", err)
//	}
//
// For joining multiple sets of files with the same configuration, create a Joiner instance:
//
//	config := joiner.DefaultConfig()
//	config.PathStrategy = joiner.StrategyFailOnCollision
//	config.SchemaStrategy = joiner.StrategyAcceptLeft
//
//	j := joiner.New(config)
//	result1, err := j.Join([]string{"api1-base.yaml", "api1-ext.yaml"})
//	result2, err := j.Join([]string{"api2-base.yaml", "api2-ext.yaml"})
//
//	// Write results
//	j.WriteResult(result1, "merged-api1.yaml")
//	j.WriteResult(result2, "merged-api2.yaml")
//
// # Advanced Usage
//
// For more control over the joining process:
//
//	config := joiner.JoinerConfig{
//		DefaultStrategy:   joiner.StrategyFailOnCollision,
//		PathStrategy:      joiner.StrategyFailOnPaths,
//		SchemaStrategy:    joiner.StrategyAcceptLeft,
//		ComponentStrategy: joiner.StrategyAcceptRight,
//		DeduplicateTags:   true,
//		MergeArrays:       true,
//	}
//
//	result, err := joiner.Join([]string{"base.yaml", "ext1.yaml", "ext2.yaml"}, config)
//	if err != nil {
//		log.Fatalf("Join failed: %v", err)
//	}
//
//	// Check for warnings
//	if len(result.Warnings) > 0 {
//		for _, warning := range result.Warnings {
//			log.Printf("Warning: %s", warning)
//		}
//	}
//
//	// Report collision statistics
//	if result.CollisionCount > 0 {
//		log.Printf("Resolved %d collisions", result.CollisionCount)
//	}
//
// # Limitations
//
//   - Cross-version joining: Cannot join OAS 2.0 documents with OAS 3.x documents
//   - Info object: The info section from the first document is used; subsequent
//     info sections are ignored
//   - External references: $ref values in components are preserved as-is; the
//     joiner does not resolve or merge referenced content across documents
//   - OpenAPI extensions: Extension fields (x-*) are merged like other fields,
//     but custom merging logic for extensions is not supported
//
// # Performance Notes
//
// The joiner performs full validation of all input documents before joining,
// which provides safety but may impact performance for large documents. For
// better performance:
//   - Pre-validate documents if possible
//   - Use StrategyAcceptLeft or StrategyAcceptRight for schemas/components
//     to allow collisions without failing
//   - Disable array merging (MergeArrays: false) if not needed
//   - Disable tag deduplication (DeduplicateTags: false) if not needed
package joiner
