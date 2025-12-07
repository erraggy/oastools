// Package joiner provides joining for multiple OpenAPI Specification documents.
//
// The joiner merges multiple OAS documents of the same major version into a single
// document. It supports OAS 2.0 documents with other 2.0 documents, and all OAS 3.x
// versions together (3.0.x, 3.1.x, 3.2.x). It uses the version and format (JSON or YAML)
// from the first document as the result version and format, ensuring format consistency
// when writing output with WriteResult.
//
// # Quick Start
//
// Join files using functional options:
//
//	result, err := joiner.JoinWithOptions(
//		joiner.WithFilePaths([]string{"base.yaml", "ext.yaml"}),
//		joiner.WithPathStrategy(joiner.StrategyAcceptLeft),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	_ = joiner.WriteResult(result, "merged.yaml")
//
// Or use a full config with options:
//
//	config := joiner.DefaultConfig()
//	config.PathStrategy = joiner.StrategyAcceptLeft
//	result, err := joiner.JoinWithOptions(
//		joiner.WithFilePaths([]string{"base.yaml", "ext.yaml"}),
//		joiner.WithConfig(config),
//	)
//
// Or create a reusable Joiner instance:
//
//	j := joiner.New(joiner.DefaultConfig())
//	result1, _ := j.Join([]string{"api1-base.yaml", "api1-ext.yaml"})
//	result2, _ := j.Join([]string{"api2-base.yaml", "api2-ext.yaml"})
//	j.WriteResult(result1, "merged1.yaml")
//	j.WriteResult(result2, "merged2.yaml")
//
// # Collision Strategies
//
// Control how collisions between documents are handled:
//   - StrategyFailOnCollision: Fail on any collision (default)
//   - StrategyAcceptLeft: Keep value from first document
//   - StrategyAcceptRight: Keep value from last document
//   - StrategyFailOnPaths: Fail only on path collisions, allow schema merging
//
// Set strategies globally (DefaultStrategy) or per component type (PathStrategy,
// SchemaStrategy, ComponentStrategy). See the examples in example_test.go for
// configuration patterns.
//
// # Features and Limitations
//
// The joiner validates all input documents, prevents output file overwrites with
// restrictive 0600 permissions, deduplicates tags, and optionally merges arrays
// (servers, security, tags). It uses the info object from the first document;
// subsequent info sections are ignored.
//
// # External References
//
// The joiner preserves external $ref values but does NOT resolve or merge them.
// This is intentional to avoid ambiguity and maintain document structure.
//
// If your documents contain external references, you have two options:
//
//  1. Resolve references before joining:
//     Use parser.ParseWithOptions(parser.WithResolveRefs(true)) before joining
//
//  2. Keep external references and resolve after joining:
//     Join the documents, then parse the result with WithResolveRefs(true)
//
// Example with external references:
//
//	// Document 1: base.yaml
//	// paths:
//	//   /users:
//	//     get:
//	//       responses:
//	//         200:
//	//           schema:
//	//             $ref: "./schemas/user.yaml#/User"
//	//
//	// Document 2: extension.yaml
//	// paths:
//	//   /posts:
//	//     get:
//	//       responses:
//	//         200:
//	//           schema:
//	//             $ref: "./schemas/post.yaml#/Post"
//	//
//	// After joining, both $ref values are preserved in the merged document.
//	// Use parser.WithResolveRefs(true) to resolve them if needed.
//
// # Related Packages
//
// The joiner integrates with other oastools packages:
//   - [github.com/erraggy/oastools/parser] - Parse specifications before joining
//   - [github.com/erraggy/oastools/validator] - Validate documents before joining (required)
//   - [github.com/erraggy/oastools/fixer] - Fix common validation errors before joining
//   - [github.com/erraggy/oastools/converter] - Convert between OAS versions before joining
//   - [github.com/erraggy/oastools/differ] - Compare joined results with original documents
//   - [github.com/erraggy/oastools/generator] - Generate code from joined specifications
//   - [github.com/erraggy/oastools/builder] - Programmatically build specifications to join
package joiner
