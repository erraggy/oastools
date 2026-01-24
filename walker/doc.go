// Package walker provides a document traversal API for OpenAPI specifications.
//
// The walker enables single-pass traversal of OAS 2.0, 3.0.x, 3.1.x, and 3.2.0 documents,
// allowing handlers to receive and optionally mutate nodes. This is useful for analysis,
// transformation, and validation tasks that need to inspect or modify multiple parts
// of a specification in a consistent way.
//
// # Quick Start
//
// Walk a document and collect all operation IDs:
//
//	result, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
//
//	var operationIDs []string
//	err := walker.Walk(result,
//	    walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
//	        operationIDs = append(operationIDs, op.OperationID)
//	        return walker.Continue
//	    }),
//	)
//
// # Flow Control
//
// Handlers return an [Action] to control traversal:
//
//   - [Continue]: continue traversing children and siblings normally
//   - [SkipChildren]: skip all children of the current node, continue with siblings
//   - [Stop]: stop the entire walk immediately
//
// Example using SkipChildren to avoid internal paths:
//
//	walker.Walk(result,
//	    walker.WithPathHandler(func(wc *walker.WalkContext, pathItem *parser.PathItem) walker.Action {
//	        if strings.HasPrefix(wc.PathTemplate, "/internal") {
//	            return walker.SkipChildren
//	        }
//	        return walker.Continue
//	    }),
//	)
//
// # Handler Types
//
// The walker provides typed handlers for all major OAS node types:
//
//   - [DocumentHandler]: root OAS2Document or OAS3Document
//   - [InfoHandler]: API metadata
//   - [ServerHandler]: server definitions (OAS 3.x only)
//   - [TagHandler]: tag definitions
//   - [PathHandler]: path entries with template string
//   - [PathItemHandler]: path item containing operations
//   - [OperationHandler]: individual HTTP operations
//   - [ParameterHandler]: parameters at path or operation level
//   - [RequestBodyHandler]: request body definitions (OAS 3.x only)
//   - [ResponseHandler]: response definitions
//   - [SchemaHandler]: all schemas including nested schemas
//   - [SecuritySchemeHandler]: security scheme definitions
//   - [HeaderHandler]: header definitions
//   - [MediaTypeHandler]: media type definitions (OAS 3.x only)
//   - [LinkHandler]: link definitions (OAS 3.x only)
//   - [CallbackHandler]: callback definitions (OAS 3.x only)
//   - [ExampleHandler]: example definitions
//   - [ExternalDocsHandler]: external documentation references
//
// # Post-Visit Handlers
//
// Post-visit handlers are called after a node's children have been processed,
// enabling bottom-up processing patterns like aggregation:
//
//   - [WithSchemaPostHandler]: Called after schema children processed
//   - [WithOperationPostHandler]: Called after operation children processed
//   - [WithPathItemPostHandler]: Called after path item children processed
//   - [WithResponsePostHandler]: Called after response children processed
//   - [WithRequestBodyPostHandler]: Called after request body children processed
//   - [WithCallbackPostHandler]: Called after callback children processed
//   - [WithOAS2DocumentPostHandler]: Called after all OAS 2.0 document children processed
//   - [WithOAS3DocumentPostHandler]: Called after all OAS 3.x document children processed
//
// Post handlers are not called if the pre-visit handler returned SkipChildren or Stop.
//
// # Parent Tracking
//
// Enable parent tracking to access ancestor nodes during traversal:
//
//	walker.Walk(result,
//	    walker.WithParentTracking(),
//	    walker.WithSchemaHandler(func(wc *walker.WalkContext, s *parser.Schema) walker.Action {
//	        if op, ok := wc.ParentOperation(); ok {
//	            // Access containing operation
//	        }
//	        return walker.Continue
//	    }),
//	)
//
// Helper methods: [WalkContext.ParentSchema], [WalkContext.ParentOperation],
// [WalkContext.ParentPathItem], [WalkContext.ParentResponse], [WalkContext.ParentRequestBody],
// [WalkContext.Ancestors], [WalkContext.Depth].
//
// # Reference Tracking
//
// Use [WithRefHandler] to receive callbacks when $ref values are encountered:
//
//	walker.Walk(result,
//	    walker.WithRefHandler(func(wc *walker.WalkContext, ref *walker.RefInfo) walker.Action {
//	        fmt.Printf("Found ref: %s at %s\n", ref.Ref, ref.SourcePath)
//	        return walker.Continue
//	    }),
//	)
//
// For polymorphic schema fields that may contain map[string]any instead of *Schema
// (such as Items, AdditionalItems, AdditionalProperties, UnevaluatedItems, and
// UnevaluatedProperties), use [WithMapRefTracking] to also detect $ref values in
// those map structures.
//
// # Mutation Support
//
// Handlers receive pointers to the actual nodes, so mutations are applied directly:
//
//	walker.Walk(result,
//	    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
//	        if schema.Extra == nil {
//	            schema.Extra = make(map[string]any)
//	        }
//	        schema.Extra["x-processed"] = true
//	        return walker.Continue
//	    }),
//	)
//
// # WalkContext
//
// Every handler receives a [WalkContext] as its first parameter, providing
// contextual information about the current node:
//
//   - JSONPath: Full JSON path to the node (always populated)
//   - PathTemplate: URL path template when in $.paths scope
//   - Method: HTTP method when in operation scope (e.g., "get", "post")
//   - StatusCode: Status code when in response scope (e.g., "200", "default")
//   - Name: Map key for named items (headers, schemas, etc.)
//   - IsComponent: True when in components/definitions section
//
// Example JSON paths:
//
//	$.info                              // Info object
//	$.paths['/pets/{petId}']            // Path entry
//	$.paths['/pets'].get                // Operation
//	$.components.schemas['Pet']         // Schema
//
// Use helper methods like [WalkContext.InOperationScope] and
// [WalkContext.InResponseScope] for scope checks.
//
// # Context Propagation
//
// Pass a [context.Context] for cancellation and timeout support:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	walker.Walk(result,
//	    walker.WithContext(ctx),
//	    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
//	        if wc.Context().Err() != nil {
//	            return walker.Stop
//	        }
//	        return walker.Continue
//	    }),
//	)
//
// # Performance Considerations
//
// The walker uses the Parse-Once pattern. Always prefer passing a pre-parsed
// [parser.ParseResult] rather than re-parsing:
//
//	// Good: parse once, walk multiple times
//	result, _ := parser.ParseWithOptions(parser.WithFilePath("api.yaml"))
//	walker.Walk(result, handlers1...)
//	walker.Walk(result, handlers2...)
//
// # Built-in Collectors
//
// For common collection patterns, the walker provides pre-built helpers that
// reduce boilerplate:
//
//   - [CollectSchemas]: Returns a [SchemaCollector] with all schemas indexed by name
//   - [CollectOperations]: Returns an [OperationCollector] with all operations indexed by operationId
//
// Example:
//
//	schemas, err := walker.CollectSchemas(result)
//	for name, info := range schemas.ByName {
//	    fmt.Printf("Schema %s: %d properties\n", name, len(info.Schema.Properties))
//	}
//
//	ops, err := walker.CollectOperations(result)
//	for _, info := range ops.All {
//	    fmt.Printf("%s %s -> %s\n", info.Method, info.PathTemplate, info.Operation.OperationID)
//	}
//
// Each [OperationInfo] provides Method, PathTemplate, JSONPath, and the full [parser.Operation].
// Each [SchemaInfo] provides Name, JSONPath, IsComponent, and the full [parser.Schema].
//
// # Schema Cycle Detection
//
// The walker automatically detects circular schema references and avoids infinite loops.
// Use [WithMaxSchemaDepth] to limit recursion depth for deeply nested schemas (default: 100).
//
// # Related Packages
//
//   - [github.com/erraggy/oastools/parser] - Parse specifications before walking
//   - [github.com/erraggy/oastools/validator] - Validate OAS documents
//   - [github.com/erraggy/oastools/fixer] - Automatically fix common issues
//   - [github.com/erraggy/oastools/converter] - Convert between OAS versions
package walker
