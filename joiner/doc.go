// Package joiner provides joining for multiple OpenAPI Specification documents.
//
// The joiner merges multiple OAS documents of the same major version into a single
// document. It supports OAS 2.0 documents with other 2.0 documents, and all OAS 3.x
// versions together (3.0.x, 3.1.x, 3.2.x). It uses the version and format (JSON or YAML)
// from the first document as the result version and format, ensuring format consistency
// when writing output with WriteResult.
//
// # Configuration
//
// Always use [DefaultConfig] to create a [JoinerConfig] instance. Direct struct
// instantiation (e.g., JoinerConfig{}) is not recommended as it leaves required
// fields at their zero values, which may cause unexpected behavior:
//
//   - NamespacePrefix will be nil (causes nil map panics if accessed)
//   - RenameTemplate will be empty (falls back to default, but unclear intent)
//   - All strategies default to empty string (treated as StrategyFailOnCollision)
//
// Correct usage:
//
//	config := joiner.DefaultConfig()           // Always start with defaults
//	config.PathStrategy = joiner.StrategyAcceptLeft  // Then customize as needed
//
// Incorrect usage:
//
//	config := joiner.JoinerConfig{}            // Zero values - avoid this!
//	config.PathStrategy = joiner.StrategyAcceptLeft
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
//   - StrategyRenameLeft: Rename left schema, keep right under original name
//   - StrategyRenameRight: Rename right schema, keep left under original name
//   - StrategyDeduplicateEquivalent: Merge structurally identical schemas
//
// Set strategies globally (DefaultStrategy) or per component type (PathStrategy,
// SchemaStrategy, ComponentStrategy). The rename and deduplicate strategies provide
// advanced collision handling with automatic reference rewriting.
//
// # Advanced Collision Handling
//
// The rename strategies preserve both colliding schemas by renaming one and
// automatically updating all references throughout the merged document:
//
//	config := joiner.DefaultConfig()
//	config.SchemaStrategy = joiner.StrategyRenameRight
//	config.RenameTemplate = "{{.Name}}_{{.Source}}"
//	result, err := joiner.JoinWithOptions(
//		joiner.WithFilePaths([]string{"users-api.yaml", "billing-api.yaml"}),
//		joiner.WithConfig(config),
//	)
//
// The deduplicate strategy uses semantic equivalence detection to merge
// structurally identical schemas while failing on true structural conflicts:
//
//	config := joiner.DefaultConfig()
//	config.SchemaStrategy = joiner.StrategyDeduplicateEquivalent
//	config.EquivalenceMode = "deep"
//	result, err := joiner.JoinWithOptions(
//		joiner.WithFilePaths([]string{"base.yaml", "ext.yaml"}),
//		joiner.WithConfig(config),
//	)
//
// See the examples in example_test.go for more configuration patterns.
//
// # Operation-Aware Renaming
//
// When renaming schemas to resolve collisions, you can enable operation context
// to trace schemas back to their originating operations. This provides rich context
// (Path, Method, OperationID, Tags, UsageType) for generating meaningful names.
//
// Enable via option:
//
//	result, err := joiner.JoinWithOptions(
//	    joiner.WithFilePaths([]string{"users-api.yaml", "billing-api.yaml"}),
//	    joiner.WithSchemaStrategy(joiner.StrategyRenameRight),
//	    joiner.WithRenameTemplate("{{.Name}}_{{.OperationID}}"),
//	    joiner.WithOperationContext(true),
//	)
//
// Or via config:
//
//	config := joiner.DefaultConfig()
//	config.SchemaStrategy = joiner.StrategyRenameRight
//	config.OperationContext = true
//	config.RenameTemplate = "{{.Name}}_{{pathResource .Path | pascalCase}}"
//
// # RenameContext Fields
//
// The [RenameContext] struct provides comprehensive context for rename templates.
//
// Core fields (always available):
//
//	Name    string  // Original schema name
//	Source  string  // Source file name (sanitized, no extension)
//	Index   int     // Document index (0-based)
//
// Operation context (requires WithOperationContext(true)):
//
//	Path        string    // API path: "/users/{id}"
//	Method      string    // HTTP method: "get", "post"
//	OperationID string    // Operation ID if defined
//	Tags        []string  // Tags from primary operation
//	UsageType   string    // "request", "response", "parameter", "header", "callback"
//	StatusCode  string    // Response status code (for response usage)
//	ParamName   string    // Parameter name (for parameter usage)
//	MediaType   string    // Media type: "application/json"
//
// Aggregate context (for schemas referenced by multiple operations):
//
//	AllPaths        []string  // All referencing paths
//	AllMethods      []string  // All methods (deduplicated)
//	AllOperationIDs []string  // All operation IDs (non-empty only)
//	AllTags         []string  // All tags (deduplicated)
//	RefCount        int       // Total operation references
//	PrimaryResource string    // Extracted resource name from path
//	IsShared        bool      // True if referenced by multiple operations
//
// # Template Functions
//
// The following template functions are available in rename templates.
//
// Path functions:
//
//	pathSegment(path, index)  // Extract nth segment (negative = from end)
//	pathResource(path)        // First non-parameter segment
//	pathLast(path)            // Last non-parameter segment
//	pathClean(path)           // Sanitize for naming: "/users/{id}" -> "users_id"
//
// Tag functions:
//
//	firstTag(tags)            // First tag or empty string
//	joinTags(tags, sep)       // Join tags with separator
//	hasTag(tags, tag)         // Check if tag is present
//
// Case transformation functions:
//
//	pascalCase(s)             // "user_name" -> "UserName"
//	camelCase(s)              // "user_name" -> "userName"
//	snakeCase(s)              // "UserName" -> "user_name"
//	kebabCase(s)              // "UserName" -> "user-name"
//
// Conditional helpers:
//
//	default(value, fallback)  // Return fallback if value is empty
//	coalesce(values...)       // Return first non-empty value
//
// Example templates:
//
//	// Use operation ID or fall back to source file
//	"{{.Name}}_{{coalesce .OperationID .Source}}"
//
//	// Use primary resource in PascalCase
//	"{{.Name}}_{{pathResource .Path | pascalCase}}"
//
//	// Include first tag if available
//	"{{.Name}}_{{default (firstTag .Tags) .Source}}"
//
//	// Clean path for naming
//	"{{.Name}}_{{pathClean .Path}}"
//
// # Primary Operation Policy
//
// When a schema is referenced by multiple operations, the policy determines
// which operation provides the primary context values (Path, Method, etc.).
//
//	config := joiner.DefaultConfig()
//	config.PrimaryOperationPolicy = joiner.PolicyMostSpecific
//
// Or via option:
//
//	joiner.WithPrimaryOperationPolicy(joiner.PolicyMostSpecific)
//
// Available policies:
//   - PolicyFirstEncountered: Use the first operation found during traversal (default)
//   - PolicyMostSpecific: Prefer operations with operationId, then those with tags
//   - PolicyAlphabetical: Sort by path+method and use alphabetically first
//
// # Overlay Integration
//
// Apply overlays during the join process for pre-processing inputs or post-processing results:
//
//	result, err := joiner.JoinWithOptions(
//	    joiner.WithFilePaths([]string{"base.yaml", "ext.yaml"}),
//	    joiner.WithPreJoinOverlayFile("normalize.yaml"),   // Applied to each input
//	    joiner.WithPostJoinOverlayFile("enhance.yaml"),    // Applied to merged result
//	)
//
// Pre-join overlays are applied to each input document before merging.
// Post-join overlays are applied to the final merged result.
//
// # Semantic Schema Deduplication
//
// After merging, the joiner can automatically identify and consolidate structurally
// identical schemas across all input documents. This reduces document size when multiple
// APIs happen to define equivalent types with different names.
//
// Enable via option:
//
//	result, err := joiner.JoinWithOptions(
//	    joiner.WithFilePaths([]string{"users-api.yaml", "orders-api.yaml"}),
//	    joiner.WithSemanticDeduplication(true),
//	)
//
// Or via config:
//
//	config := joiner.DefaultConfig()
//	config.SemanticDeduplication = true
//	j := joiner.New(config)
//	result, _ := j.Join([]string{"api1.yaml", "api2.yaml"})
//
// When schemas from different documents are structurally equivalent (same type, properties,
// constraints), they are consolidated into a single canonical schema (alphabetically first
// name). All $ref references throughout the merged document are automatically rewritten.
//
// This differs from the StrategyDeduplicateEquivalent collision strategy which only
// handles same-named collisions. Semantic deduplication works across all schemas
// regardless of their original names.
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
// # Structured Warnings
//
// The joiner provides structured warnings through the [JoinWarning] type, which includes
// detailed context about non-fatal issues encountered during document joining. Each warning
// has a category, source location, and optional context data for programmatic handling.
//
// Access structured warnings from the result:
//
//	result, _ := joiner.JoinWithOptions(
//	    joiner.WithFilePaths([]string{"base.yaml", "ext.yaml"}),
//	)
//	for _, w := range result.StructuredWarnings {
//	    fmt.Printf("[%s] %s\n", w.Category, w.Message)
//	    if w.HasLocation() {
//	        fmt.Printf("  at %s\n", w.Location())
//	    }
//	}
//
// Filter warnings by category or severity:
//
//	pathCollisions := result.StructuredWarnings.ByCategory(joiner.WarnPathCollision)
//	criticalWarnings := result.StructuredWarnings.BySeverity(severity.SeverityWarning)
//
// Warning categories include:
//   - WarnVersionMismatch: Documents have different minor versions
//   - WarnPathCollision: Path collision was resolved by strategy
//   - WarnSchemaCollision: Schema/definition collision was resolved
//   - WarnWebhookCollision: Webhook collision was resolved
//   - WarnSchemaRenamed: Schema was renamed due to collision
//   - WarnSchemaDeduplicated: Schema was deduplicated (structurally equivalent)
//   - WarnNamespacePrefixed: Namespace prefix was applied
//   - WarnMetadataOverride: Metadata was overridden (host, basePath)
//   - WarnGenericSourceName: Document has a generic source name (e.g., "ParseBytes.yaml")
//
// For backward compatibility, warnings are also available as []string via result.Warnings.
//
// # Pre-Parsed Documents and Source Names
//
// When using [Joiner.JoinParsed] with pre-parsed documents (the recommended path
// for high performance), ensure each document has a meaningful SourcePath set.
// The joiner uses SourcePath in collision reports and warnings. Without meaningful
// names, collision reports show unhelpful text like "ParseBytes.yaml vs ParseBytes.yaml".
//
// Set meaningful source names before joining:
//
//	// Fetch and parse specs from your services
//	specs := make([]parser.ParseResult, 0, len(services))
//	for name, data := range serviceSpecs {
//	    result, _ := parser.ParseWithOptions(parser.WithBytes(data))
//	    result.SourcePath = name  // Set meaningful name for collision reports
//	    specs = append(specs, *result)
//	}
//
//	// Collision reports now show "users-api vs billing-api"
//	joined, _ := joiner.JoinParsed(specs)
//
// Alternatively, use parser.WithSourceName when parsing:
//
//	result, _ := parser.ParseWithOptions(
//	    parser.WithBytes(data),
//	    parser.WithSourceName("users-api"),
//	)
//
// The joiner emits an info-level [WarnGenericSourceName] warning when documents have
// generic source names (empty, "ParseBytes.*", "ParseReader.*") to help identify this issue.
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
//   - [github.com/erraggy/oastools/overlay] - Apply overlay transformations during join
package joiner
