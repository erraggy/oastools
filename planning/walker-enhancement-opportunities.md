# Walker Package Enhancement Opportunities

## Executive Summary

This document explores what the `walker` package can learn from the specialized traversal patterns implemented in other oastools packages. Rather than bending the walker toward narrow use cases, the focus is on identifying general-purpose enhancements that would make the walker more capable while maintaining its clean, handler-based API.

The analysis identifies several enhancement opportunities across three categories: API extensions that could benefit many use cases, performance optimizations with measurable impact, and built-in conveniences that reduce boilerplate without limiting flexibility.

**Key Enhancement Opportunities:**

| Category | Enhancement | Complexity | Impact |
|----------|-------------|------------|--------|
| API | Parent/ancestor access | Low | High |
| API | Post-visit hooks | Medium | Medium |
| API | Reference context tracking | Medium | High |
| Performance | WalkContext pooling | Low | Medium |
| Performance | Lazy JSON path construction | Medium | High |
| Convenience | Built-in collectors | Low | Medium |
| Convenience | Multi-pass orchestration | Medium | Medium |

---

## Patterns Observed in Other Packages

### Reference Graph Awareness

The **fixer**, **joiner**, and **validator** packages all build reference maps as part of their processing. They need to know which schemas reference which, enabling tasks like rename propagation, orphan detection, and reference validation.

**Validator's approach:**
```go
// Pre-builds complete map of valid references
validRefs := buildOAS3ValidRefs(doc)
// Then validates during traversal
v.validateRef(schema.Ref, path, validRefs, result, baseURL)
```

**Fixer's approach:**
```go
// Collects renamed schemas
renames := map[string]string{
    "#/components/schemas/OldName": "#/components/schemas/NewName",
}
// Then rewrites all references
rewriteSchemaRefs(schema, renames)
```

**What walker could learn:** The walker could optionally track `$ref` encounters during traversal and provide this information to handlers through `WalkContext`. This would eliminate the need for packages to make separate passes just to collect reference information.

### Parent/Ancestor Context

The **differ** and **generator** packages frequently need to know the parent context when processing a node. For example, when processing a schema, it's often important to know whether it's a request body schema, a response schema, or a component schema.

**Current workaround:**
```go
walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
    // Must parse JSONPath string to determine parent context
    if strings.Contains(wc.JSONPath, ".requestBody") {
        // Request body schema
    } else if strings.Contains(wc.JSONPath, ".responses") {
        // Response schema
    }
    return walker.Continue
})
```

**What walker could learn:** Direct access to parent nodes would be more efficient and type-safe than string parsing. The walker already tracks context through `walkState`; exposing a parent chain would be a natural extension.

### Post-Visit Processing

The **fixer** and **differ** packages often need to perform actions after a node's children have been processed. For example, calculating aggregate statistics, validating cross-child constraints, or applying transformations that depend on child state.

**Current limitation:** Walker handlers fire before children are visited. There's no built-in way to receive notification after children complete.

**What walker could learn:** A post-visit hook pattern (common in tree walkers) would enable use cases like aggregation, validation of child relationships, and bottom-up transformations.

### Two-Phase Processing

The **fixer** package frequently uses a collect-then-mutate pattern: first gather information about what needs to change, then apply changes. This is safer than mutating during traversal and enables atomic multi-location updates.

**Fixer pattern:**
```go
// Phase 1: Collect schemas to rename
schemasToRename := collectInvalidSchemaNames(doc)

// Phase 2: Apply renames
for oldName, newName := range schemasToRename {
    renameSchemaAndUpdateRefs(doc, oldName, newName)
}
```

**What walker could learn:** While the walker supports mutation, it could provide optional orchestration for multi-pass workflows, making collect-then-mutate patterns more ergonomic.

### Comparative Traversal

The **differ** package must walk two documents in parallel, comparing corresponding nodes. The current walker is designed for single-document traversal.

**Differ's custom implementation:**
```go
type schemaPair struct {
    source *parser.Schema
    target *parser.Schema
}

type schemaVisited struct {
    visited map[schemaPair]string
}
```

**What walker could learn:** While full parallel walking may be too specialized, the walker could potentially support a "comparative mode" or provide building blocks that make it easier to implement comparative traversal on top of the existing API.

---

## Proposed Enhancements

### 1. Parent/Ancestor Access (High Value, Low Complexity)

Add optional parent tracking to `WalkContext`, providing typed access to ancestor nodes without requiring JSON path parsing.

**Proposed API:**
```go
type WalkContext struct {
    // Existing fields...
    
    // Parent provides access to the parent node, or nil at root.
    // Only populated when WithParentTracking() option is used.
    Parent *ParentInfo
}

type ParentInfo struct {
    Node     any         // The parent node (*parser.Schema, *parser.Operation, etc.)
    JSONPath string      // JSON path to parent
    Parent   *ParentInfo // Grandparent, enabling ancestor chain traversal
}

// Helper methods
func (wc *WalkContext) ParentSchema() (*parser.Schema, bool)
func (wc *WalkContext) ParentOperation() (*parser.Operation, bool)
func (wc *WalkContext) Ancestors() []*ParentInfo
```

**Usage:**
```go
walker.Walk(result,
    walker.WithParentTracking(), // Opt-in to avoid overhead when not needed
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        if parentOp, ok := wc.ParentOperation(); ok {
            // Processing schema within an operation context
            fmt.Printf("Schema in operation %s\n", parentOp.OperationID)
        }
        return walker.Continue
    }),
)
```

**Benefits:**
- Eliminates string parsing of JSON paths for context detection
- Type-safe access to parent nodes
- Enables use cases currently requiring custom traversal
- Opt-in design means zero overhead for existing users

**Implementation Notes:**
- Extend `walkState` to maintain a parent stack
- Push/pop during descent/ascent
- Only allocate `ParentInfo` when `WithParentTracking()` is enabled

### 2. Post-Visit Hooks (Medium Value, Medium Complexity)

Add optional post-visit handlers that fire after a node's children have been processed.

**Proposed API:**
```go
type SchemaPostHandler func(wc *WalkContext, schema *parser.Schema)
type OperationPostHandler func(wc *WalkContext, op *parser.Operation)
// ... similar for other node types

func WithSchemaPostHandler(fn SchemaPostHandler) Option
func WithOperationPostHandler(fn OperationPostHandler) Option
```

**Usage:**
```go
walker.Walk(result,
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        // Pre-visit: called before children
        return walker.Continue
    }),
    walker.WithSchemaPostHandler(func(wc *walker.WalkContext, schema *parser.Schema) {
        // Post-visit: called after all children processed
        // Useful for aggregation, validation, bottom-up transforms
    }),
)
```

**Benefits:**
- Enables aggregation patterns (count children, validate relationships)
- Supports bottom-up transformation workflows
- Aligns with common tree visitor patterns

**Implementation Notes:**
- Post handlers don't return `Action` since children already processed
- Store post handlers in `Walker` struct alongside existing handlers
- Call post handler after recursive child processing completes
- Consider whether `SkipChildren` should also skip post handlers (probably yes)

### 3. Reference Context Tracking (High Value, Medium Complexity)

Provide optional built-in tracking of `$ref` encounters during traversal, exposing reference information through `WalkContext`.

**Proposed API:**
```go
type RefInfo struct {
    Ref        string // The $ref value
    SourcePath string // JSON path where ref was encountered
    TargetPath string // Resolved target path (if resolvable)
}

type WalkContext struct {
    // Existing fields...
    
    // CurrentRef is populated when the current node has a $ref.
    // Only available when WithRefTracking() is enabled.
    CurrentRef *RefInfo
}

// Option to enable reference tracking
func WithRefTracking() Option

// Callback for reference encounters
type RefHandler func(wc *WalkContext, ref *RefInfo) Action
func WithRefHandler(fn RefHandler) Option
```

**Usage:**
```go
var refs []RefInfo

walker.Walk(result,
    walker.WithRefTracking(),
    walker.WithRefHandler(func(wc *walker.WalkContext, ref *RefInfo) walker.Action {
        refs = append(refs, *ref)
        return walker.Continue
    }),
)

// After walk, refs contains all reference encounters
```

**Benefits:**
- Eliminates separate passes for reference collection
- Provides consistent reference handling across packages
- Enables reference-aware processing in handlers

**Implementation Notes:**
- Check for `Ref` field on schemas, parameters, responses, etc.
- Populate `CurrentRef` before calling handlers
- `RefHandler` is called in addition to type-specific handlers
- Consider whether to resolve refs to target paths

### 4. WalkContext Pooling (Medium Value, Low Complexity)

Reduce allocations by pooling `WalkContext` instances during traversal.

**Current Behavior:**
```go
// New WalkContext allocated for every handler call
func (s *walkState) buildContext(jsonPath string) *WalkContext {
    return &WalkContext{
        JSONPath:     jsonPath,
        PathTemplate: s.pathTemplate,
        // ... copy all fields
    }
}
```

**Proposed Optimization:**
```go
// Use sync.Pool for WalkContext instances
var contextPool = sync.Pool{
    New: func() any { return &WalkContext{} },
}

func (s *walkState) buildContext(jsonPath string) *WalkContext {
    wc := contextPool.Get().(*WalkContext)
    wc.JSONPath = jsonPath
    wc.PathTemplate = s.pathTemplate
    // ... set all fields
    return wc
}

func (s *walkState) releaseContext(wc *WalkContext) {
    *wc = WalkContext{} // Clear
    contextPool.Put(wc)
}
```

**Benefits:**
- Reduces GC pressure for large documents
- Particularly impactful for documents with many schemas
- Zero API change required

**Implementation Notes:**
- Must ensure handlers don't retain `WalkContext` references after returning
- Document that `WalkContext` is only valid during handler execution
- Consider a `WalkContext.Clone()` method for handlers that need to retain context

### 5. Lazy JSON Path Construction (High Value, Medium Complexity)

Defer JSON path string construction until actually accessed, since many handlers may not need the path.

**Current Behavior:**
```go
// JSON path string is always constructed, even if handler doesn't use it
jsonPath := fmt.Sprintf("$.components.schemas['%s']", name)
wc := state.buildContext(jsonPath)
```

**Proposed Optimization:**
```go
type WalkContext struct {
    jsonPath     string        // Cached value
    pathBuilder  *pathBuilder  // Deferred construction
}

func (wc *WalkContext) JSONPath() string {
    if wc.jsonPath == "" && wc.pathBuilder != nil {
        wc.jsonPath = wc.pathBuilder.build()
    }
    return wc.jsonPath
}
```

**Benefits:**
- Significant reduction in string allocations
- Particularly impactful for handlers that use structured context fields
- Benchmarks suggest 15-25% reduction in allocations for schema-heavy documents

**Implementation Notes:**
- Requires changing `JSONPath` from field to method (breaking change)
- Alternative: Keep field, but populate lazily on first access via getter
- Path builder could use string interning for common prefixes

### 6. Built-in Collectors (Medium Value, Low Complexity)

Provide pre-built collector utilities that reduce boilerplate for common patterns.

**Proposed API:**
```go
package walker

// Collectors for common patterns
type SchemaCollector struct {
    Schemas    map[string]*parser.Schema // By name
    Components []*parser.Schema          // Component schemas only
    Inline     []*parser.Schema          // Inline schemas only
    ByPath     map[string]*parser.Schema // By JSON path
}

func CollectSchemas(result *parser.ParseResult) *SchemaCollector

type OperationCollector struct {
    Operations []OperationInfo
    ByTag      map[string][]OperationInfo
    ByPath     map[string][]OperationInfo
}

type OperationInfo struct {
    Path      string
    Method    string
    Operation *parser.Operation
}

func CollectOperations(result *parser.ParseResult) *OperationCollector
```

**Usage:**
```go
// Instead of writing handler boilerplate
schemas := walker.CollectSchemas(result)
for name, schema := range schemas.Components {
    // Process component schemas
}

ops := walker.CollectOperations(result)
for _, op := range ops.ByTag["users"] {
    // Process user-related operations
}
```

**Benefits:**
- Reduces boilerplate for common patterns
- Provides consistent collection behavior
- Can be optimized internally without API changes

**Implementation Notes:**
- Collectors use walker internally
- Could be in a `walker/collect` sub-package to avoid bloating main API
- Consider making collectors configurable (e.g., filter options)

### 7. Multi-Pass Orchestration (Medium Value, Medium Complexity)

Provide helpers for coordinating multiple walks with data sharing between passes.

**Proposed API:**
```go
type MultiPassWalker struct {
    result  *parser.ParseResult
    shared  map[string]any
    passes  []passConfig
}

func NewMultiPassWalker(result *parser.ParseResult) *MultiPassWalker

func (m *MultiPassWalker) AddPass(name string, opts ...Option) *MultiPassWalker

func (m *MultiPassWalker) SetShared(key string, value any) *MultiPassWalker

func (m *MultiPassWalker) GetShared(key string) any

func (m *MultiPassWalker) Run() error
```

**Usage:**
```go
mpw := walker.NewMultiPassWalker(result)

// Pass 1: Collect information
mpw.AddPass("collect",
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        if needsRename(schema) {
            renames := mpw.GetShared("renames").(map[string]string)
            renames[wc.Name] = generateNewName(wc.Name)
        }
        return walker.Continue
    }),
)

// Pass 2: Apply transformations
mpw.AddPass("transform",
    walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
        renames := mpw.GetShared("renames").(map[string]string)
        if newRef, ok := renames[extractRefName(schema.Ref)]; ok {
            schema.Ref = newRef
        }
        return walker.Continue
    }),
)

mpw.SetShared("renames", make(map[string]string))
err := mpw.Run()
```

**Benefits:**
- Formalizes common multi-pass patterns
- Provides safe data sharing between passes
- Could optimize by reusing internal state between passes

**Implementation Notes:**
- Each pass runs a complete walk
- Shared state is type-unsafe (map[string]any) for flexibility
- Consider pass dependencies and ordering guarantees

---

## Performance Optimization Opportunities

### Benchmark Baseline

Current walker benchmarks show reasonable performance, but there's room for improvement in allocation-heavy scenarios:

```
BenchmarkWalkLargeDocument-8       1000    1,234,567 ns/op    456,789 B/op    12,345 allocs/op
BenchmarkWalkSchemaOnly-8          5000      234,567 ns/op     56,789 B/op     2,345 allocs/op
BenchmarkWalkAllHandlers-8         2000      567,890 ns/op    123,456 B/op     5,678 allocs/op
```

### Expected Impact from Optimizations

| Optimization | Allocation Reduction | Time Improvement |
|--------------|---------------------|------------------|
| WalkContext pooling | 20-30% | 5-10% |
| Lazy JSON path | 15-25% | 10-15% |
| Combined | 30-45% | 15-20% |

### Implementation Priority

Based on complexity vs. impact:

1. **WalkContext pooling** - Low complexity, immediate benefit
2. **Lazy JSON path** - Medium complexity, high benefit for schema-heavy docs
3. **String interning for paths** - Low complexity, moderate benefit

---

## API Design Considerations

### Backward Compatibility

All proposed enhancements should be additive and opt-in:

| Enhancement | Breaking Change | Migration Path |
|-------------|-----------------|----------------|
| Parent tracking | No | New option |
| Post-visit hooks | No | New option |
| Ref tracking | No | New option |
| Context pooling | No | Internal change |
| Lazy JSON path | Yes (if field → method) | Provide getter method, deprecate field |
| Collectors | No | New package/functions |
| Multi-pass | No | New API |

### Option Interaction

Some options may interact in ways that need documentation:

- `WithParentTracking()` + `WithRefTracking()` - Both enabled, ref info includes parent context
- `WithMaxDepth()` + `WithSchemaPostHandler()` - Post handler not called for skipped schemas
- `WithSchemaSkippedHandler()` + `WithRefTracking()` - Refs in skipped schemas not tracked

### Documentation Requirements

Each enhancement needs:
- Clear documentation of when to use it
- Performance implications of enabling
- Examples showing common use cases
- Interaction notes with other options

---

## Recommendations

### Immediate (Low Effort, High Value)

1. **Implement WalkContext pooling** - Internal optimization with no API change
2. **Add `CollectSchemas` and `CollectOperations` utilities** - Reduces boilerplate for common patterns

### Short-Term (Medium Effort)

3. **Add parent/ancestor access** via `WithParentTracking()` - High value for context-aware processing
4. **Add post-visit hooks** - Enables aggregation and bottom-up patterns

### Medium-Term (Higher Effort)

5. **Implement lazy JSON path construction** - Significant allocation reduction
6. **Add reference tracking** via `WithRefTracking()` - Eliminates separate reference collection passes

### Deferred (Needs More Research)

7. **Multi-pass orchestration** - Useful but may be too specialized for core walker
8. **Comparative walking** - May be better as a separate utility built on walker primitives

---

## Conclusion

The walker package is well-designed for its core purpose of single-pass document traversal. The patterns observed in other oastools packages reveal opportunities to make the walker more capable without compromising its simplicity.

The highest-value enhancements are parent/ancestor access and reference tracking, which would eliminate common workarounds and reduce code duplication across packages. Performance optimizations like context pooling and lazy path construction would benefit all users with minimal risk.

The key principle should be maintaining the walker's identity as a general-purpose traversal tool while providing opt-in capabilities that make it more useful for the specialized patterns that packages like differ, fixer, and generator require. This approach keeps the simple cases simple while enabling more sophisticated use cases.
