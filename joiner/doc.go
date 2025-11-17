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
// Join files with a config:
//
//	config := joiner.DefaultConfig()
//	config.PathStrategy = joiner.StrategyAcceptLeft
//	result, err := joiner.Join([]string{"base.yaml", "ext.yaml"}, config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	_ = joiner.WriteResult(result, "merged.yaml")
//
// Or create a reusable Joiner instance:
//
//	j := joiner.New(config)
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
// subsequent info sections are ignored. External $ref values are preserved but
// not merged across documents.
package joiner
