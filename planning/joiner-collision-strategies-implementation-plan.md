# Implementation Plan: Advanced Collision Strategies for the Joiner Package

This plan provides a single-phase implementation roadmap for enhancing the `joiner` package with sophisticated collision handling strategies, including schema renaming with automatic reference rewriting. The implementation is scoped for completion in a single Claude Code session.

---

## Executive Summary

The current joiner implementation offers four collision strategies (`accept-left`, `accept-right`, `fail`, `fail-on-paths`) that operate on a binary keep-or-discard model. Large-scale use cases with hundreds of source documents and common domain names like "Statement" require more nuanced control: the ability to rename colliding schemas while automatically updating all references throughout the merged document. This enhancement introduces rename-based strategies, namespace prefixing, semantic equivalence detection, and comprehensive collision reporting.

---

## Scope and Objectives

The implementation delivers five interconnected capabilities:

1. **Rename-based collision strategies** (`rename-left` and `rename-right`) that preserve both colliding schemas by renaming one and rewriting all its references.

2. **Configurable namespace prefixing** that can apply source-derived prefixes to schema names either on collision or universally.

3. **Semantic equivalence detection** that identifies structurally identical schemas and deduplicates them rather than treating them as conflicts.

4. **Reference rewriting engine** capable of traversing all `$ref` locations including discriminator mappings in both shorthand and full-path formats.

5. **Enhanced collision reporting** that provides structural comparison and actionable resolution guidance.

---

## Architecture Overview

The implementation introduces three new source files and modifies four existing files within the `joiner` package. The reference rewriting logic is extracted into a dedicated `rewriter.go` file to maintain single-responsibility principles and enable reuse. Configuration extends the existing `JoinerConfig` struct with new fields while preserving backward compatibility through zero-value defaults that maintain current behavior.

```
joiner/
├── joiner.go          (modified: new strategies, config fields)
├── oas2.go            (modified: integrate rewriter calls)
├── oas3.go            (modified: integrate rewriter calls)
├── rewriter.go        (new: reference rewriting engine)
├── equivalence.go     (new: semantic schema comparison)
├── collision.go       (new: collision report structures)
├── joiner_test.go     (modified: new strategy tests)
├── rewriter_test.go   (new: reference rewriting tests)
├── equivalence_test.go (new: equivalence detection tests)
└── collision_test.go  (new: collision reporting tests)
```

---

## Detailed Implementation Specifications

### New Collision Strategies

The `CollisionStrategy` type gains three new constants. `StrategyRenameLeft` keeps the right-side schema under the original name and renames the left-side schema with a configurable suffix, updating all references in the left document's contributed content. `StrategyRenameRight` performs the inverse operation, keeping left and renaming right. `StrategyDeduplicateEquivalent` triggers semantic comparison and merges structurally identical schemas regardless of source, failing only on true structural conflicts.

```go
const (
    StrategyRenameLeft            CollisionStrategy = "rename-left"
    StrategyRenameRight           CollisionStrategy = "rename-right"
    StrategyDeduplicateEquivalent CollisionStrategy = "deduplicate"
)
```

### Configuration Extensions

The `JoinerConfig` struct receives new fields controlling rename behavior. `RenameTemplate` specifies the naming pattern for renamed schemas using Go template syntax with variables `{{.Name}}`, `{{.Source}}`, `{{.Index}}`, and `{{.Suffix}}`. The default template is `{{.Name}}_{{.Source}}` which produces names like `Statement_billing`. `NamespacePrefix` enables source-based prefixing for all schemas from a document regardless of collision status when `AlwaysApplyPrefix` is true. `EquivalenceMode` controls the depth of structural comparison: `shallow` compares only top-level properties while `deep` performs recursive field-by-field analysis.

```go
type JoinerConfig struct {
    // Existing fields preserved...
    
    RenameTemplate     string            // Go template for renamed schema names
    NamespacePrefix    map[string]string // Source path → prefix mapping
    AlwaysApplyPrefix  bool              // Apply prefix even without collision
    EquivalenceMode    string            // "shallow", "deep", or "none"
    CollisionReport    bool              // Generate detailed collision analysis
}
```

### Reference Rewriting Engine

The `rewriter.go` file implements the core traversal and update logic. The `SchemaRewriter` struct maintains a mapping from old reference paths to new paths and tracks visited nodes to handle circular references safely. The primary method `RewriteRefs` accepts a document (either `*parser.OAS2Document` or `*parser.OAS3Document`) and applies all registered renames.

The traversal covers every location where `$ref` values can appear in an OpenAPI document. For schemas, this includes `properties`, `additionalProperties`, `items`, `allOf`, `anyOf`, `oneOf`, `not`, `if`, `then`, `else`, `prefixItems`, `contains`, `propertyNames`, `dependentSchemas`, and `$defs`. Beyond schemas, the rewriter processes parameter references, response references, request body references, header references, callback references, link references, and path item references.

Discriminator mappings require special handling because they accept both full `$ref` paths (`#/components/schemas/Dog`) and bare schema names (`Dog`). The rewriter maintains a parallel mapping of bare names to detect and update shorthand references. When a schema named `Statement` is renamed to `Statement_billing`, both `#/components/schemas/Statement` references and bare `Statement` discriminator values update to their new forms.

```go
type SchemaRewriter struct {
    refMap      map[string]string // "#/components/schemas/Old" → "#/components/schemas/New"
    bareNameMap map[string]string // "Old" → "New" for discriminator shorthand
    visited     map[uintptr]bool  // Circular reference protection
}

func (r *SchemaRewriter) RegisterRename(oldName, newName string, version parser.OASVersion) {
    oldRef := schemaRefPath(oldName, version)
    newRef := schemaRefPath(newName, version)
    r.refMap[oldRef] = newRef
    r.bareNameMap[oldName] = newName
}

func (r *SchemaRewriter) RewriteDocument(doc any) error {
    // Type switch for OAS2/OAS3, then recursive traversal
}
```

### Semantic Equivalence Detection

The `equivalence.go` file implements structural comparison of schemas. Two schemas are considered equivalent when they have identical `type`, matching `properties` with recursively equivalent sub-schemas, equivalent `items` for array types, matching `required` arrays (order-independent), identical `enum` values, matching composition (`allOf`/`anyOf`/`oneOf`) with recursively equivalent members, and equivalent `additionalProperties` configuration.

The comparison deliberately ignores `description`, `title`, `example`, `deprecated`, and extension fields (`x-*`) as these represent documentation rather than structure. The `EquivalenceResult` struct captures the comparison outcome with detailed field-by-field differences when schemas are not equivalent.

```go
type EquivalenceResult struct {
    Equivalent  bool
    Differences []SchemaDifference
}

type SchemaDifference struct {
    Path        string // JSON path to differing element
    LeftValue   any
    RightValue  any
    Description string
}

func CompareSchemas(left, right *parser.Schema, mode string) EquivalenceResult
```

### Collision Reporting

The `collision.go` file defines structures for detailed collision analysis. `CollisionReport` captures all collision events during a join operation with sufficient detail to understand the conflict and choose resolution strategies. Each `CollisionEvent` records the schema name, source files, the strategy applied, structural differences if schemas were compared, and the resolution outcome.

```go
type CollisionReport struct {
    TotalCollisions    int
    ResolvedByRename   int
    ResolvedByDedup    int
    ResolvedByAccept   int
    FailedCollisions   int
    Events             []CollisionEvent
}

type CollisionEvent struct {
    SchemaName      string
    LeftSource      string
    RightSource     string
    Strategy        CollisionStrategy
    Resolution      string // "renamed", "deduplicated", "kept-left", "kept-right", "failed"
    NewName         string // For rename resolutions
    Differences     []SchemaDifference // When equivalence was checked
}
```

### Integration with Existing Merge Logic

The `mergeSchemas` function in `oas3.go` (and corresponding `mergeOAS2Definitions` in `oas2.go`) gains logic branches for the new strategies. When `StrategyRenameLeft` or `StrategyRenameRight` is active and a collision occurs, the merge function generates the new name using the template, registers the rename with the rewriter, adds the renamed schema to the target map, and records the collision event. After all documents are merged, a final rewriting pass applies all accumulated renames to the complete merged document.

The `StrategyDeduplicateEquivalent` branch first invokes `CompareSchemas`. If schemas are equivalent, the collision resolves by keeping one copy (the left by convention) and recording the deduplication. If schemas differ structurally, the behavior falls back to the configured `DefaultStrategy` or fails if no fallback is specified.

### CLI Integration

The `cmd/oastools/commands/join.go` file gains new flags for the enhanced strategies. The `--schema-strategy` flag accepts the new strategy values. Additional flags include `--rename-template` for customizing rename patterns, `--namespace-prefix` accepting `source=prefix` pairs, `--always-prefix` boolean, `--equivalence-mode` accepting `shallow`/`deep`/`none`, and `--collision-report` to output detailed analysis.

```bash
oastools join \
  --schema-strategy rename-right \
  --rename-template "{{.Name}}_{{.Source}}" \
  --namespace-prefix "users-api.yaml=Users" \
  --namespace-prefix "billing-api.yaml=Billing" \
  --equivalence-mode deep \
  --collision-report \
  -o merged.yaml \
  users-api.yaml billing-api.yaml payments-api.yaml
```

---

## Testing Strategy

The test suite covers four categories:

**Unit tests for the reference rewriter** verify correct handling of all `$ref` locations, discriminator mapping formats, circular references, and OAS 2.0 versus 3.x path differences.

**Unit tests for equivalence detection** cover positive matches, structural differences at various depths, and edge cases like empty schemas and composition hierarchies.

**Integration tests** exercise end-to-end join operations with the new strategies using purpose-built test fixtures that include colliding schemas, discriminator usage, and circular references.

**Corpus tests** using the existing integration test infrastructure validate behavior against real-world specifications from the corpus.

Test fixtures reside in `testdata/` with descriptive names: `join-collision-rename-3.0.yaml`, `join-discriminator-3.0.yaml`, `join-circular-refs-3.0.yaml`, and `join-equivalent-schemas-3.0.yaml`.

---

## File Manifest

### New Files to Create

| File | Lines (est.) | Purpose |
|------|--------------|---------|
| `joiner/rewriter.go` | 350 | Reference rewriting engine with traversal logic |
| `joiner/equivalence.go` | 200 | Semantic schema comparison |
| `joiner/collision.go` | 100 | Collision report structures |
| `joiner/rewriter_test.go` | 400 | Rewriter unit tests |
| `joiner/equivalence_test.go` | 250 | Equivalence detection tests |
| `joiner/collision_test.go` | 150 | Collision report tests |
| `testdata/join-collision-rename-3.0.yaml` | 80 | Test fixture |
| `testdata/join-discriminator-3.0.yaml` | 60 | Test fixture |
| `testdata/join-circular-refs-3.0.yaml` | 50 | Test fixture |
| `testdata/join-equivalent-schemas-3.0.yaml` | 70 | Test fixture |

### Files to Modify

| File | Changes |
|------|---------|
| `joiner/joiner.go` | Add strategy constants, extend `JoinerConfig`, add functional options |
| `joiner/oas3.go` | Integrate rewriter in `mergeSchemas`, add new strategy branches |
| `joiner/oas2.go` | Integrate rewriter in `mergeOAS2Definitions`, add new strategy branches |
| `joiner/joiner_test.go` | Add integration tests for new strategies |
| `joiner/doc.go` | Update package documentation |
| `cmd/oastools/commands/join.go` | Add new CLI flags |
| `docs/cli-reference.md` | Document new flags and strategies |
| `docs/developer-guide.md` | Add examples for new strategies |

---

## Implementation Sequence

The implementation proceeds in dependency order:

1. Begin with the collision report structures in `collision.go` as these have no dependencies.

2. Continue with the equivalence detection in `equivalence.go` which depends only on `parser.Schema`.

3. Implement the reference rewriter in `rewriter.go` which depends on parser types.

4. Extend `joiner.go` with new constants, config fields, and functional options.

5. Integrate the new logic into `oas3.go` and `oas2.go`.

6. Add CLI flags in `join.go`.

7. Create test fixtures, then implement all test files.

8. Conclude with documentation updates.

---

## Risk Considerations

**Discriminator mapping edge case** presents the highest risk of subtle bugs due to the dual format support. The implementation must handle both `#/components/schemas/Name` and bare `Name` references, including cases where the same discriminator uses both formats.

**Circular reference handling** requires careful visited-node tracking to prevent infinite loops without incorrectly skipping legitimate repeated references in different contexts.

**Performance with very large specifications** (thousands of schemas) could degrade if the rewriter performs naive repeated traversals. The implementation should accumulate all renames before performing a single traversal pass.

**Memory allocation during deep equivalence comparison** should use early-exit patterns to avoid unnecessary recursion into subtrees that have already shown differences.

---

## Success Criteria

The implementation succeeds when:

1. All existing joiner tests continue to pass (backward compatibility).

2. New strategies correctly rename schemas and update all references including discriminator mappings.

3. Semantic equivalence correctly identifies structurally identical schemas across a variety of schema complexities.

4. The corpus integration tests pass with new strategies applied to real-world specifications.

5. `make check` completes without errors or warnings.
