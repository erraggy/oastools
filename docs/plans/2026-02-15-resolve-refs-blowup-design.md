# Design: Eliminate resolve_refs memory blowup

**Issue:** #328
**Date:** 2026-02-15
**Status:** Approved

## Problem

When `parser.WithResolveRefs(true)` is used, the parser inflates memory significantly due to two cascading effects:

1. **Deep-copy inflation:** The resolver's `deepCopyJSONValue` duplicates every `$ref` target. A schema referenced 40 times becomes 40 independent copies in the `map[string]any`.
2. **Re-marshal allocation:** The parser must re-serialize the inflated map to `[]byte` for version-specific parsing (`yaml.Unmarshal`/`json.Unmarshal` require bytes). This creates a second copy of the inflated data.

### Measurements (current)

| Spec | Input | Peak memory | Ratio |
|------|------:|------------:|------:|
| Petstore 3.0 | 3 KB | ~24 KB | 8x |
| GitHub API | 11.6 MB | ~320 MB | 28x |
| DigitalOcean | 2.5 MB | ~310 MB | 124x |

### Root cause chain

```
input []byte
  → yaml/json.Unmarshal → map[string]any (~original size)
  → resolver.ResolveAllRefs (deepCopyJSONValue) → inflated map (~10-40x)
  → yaml/json.Marshal → inflated []byte (~same as inflated map)
  → yaml/json.Unmarshal → typed struct (~same as inflated map)
```

Peak memory holds all four simultaneously.

## Design

Two complementary optimizations that together eliminate ~60% of peak memory.

### Optimization 1: Direct map-to-struct decoding (`decodeFromMap`)

Eliminate the re-marshal roundtrip entirely. Instead of `map → []byte → struct`, go directly `map → struct` using code-generated decoder methods.

#### Code generator (`cmd/gen-decode/`)

A Go code generator using `go/types` (via `golang.org/x/tools/go/packages`) to introspect all OAS struct types and produce `decodeFromMap` methods.

**Target identification:** Types with `Extra map[string]any` field tagged `json:"-"` (the consistent marker across all 31 OAS types).

**Field type patterns:**

| Field type | Generated code pattern |
|-----------|----------------------|
| `string` | `x.Field, _ = m["key"].(string)` |
| `bool` | `x.Field, _ = m["key"].(bool)` |
| `*bool` | Allocate, check `bool` assertion |
| `*int` | Convert from `float64` or `int` (JSON vs YAML) |
| `*float64` | Convert from `float64` or `int` |
| `[]string` | Iterate `[]any`, assert each to `string` |
| `[]*T` (OAS type) | Iterate `[]any`, create `*T`, call `decodeFromMap` |
| `map[string]*T` | Iterate `map[string]any`, create `*T`, call `decodeFromMap` |
| `*T` (OAS type) | Create `*T`, call `decodeFromMap` |
| `any` | Assign directly (preserves `map[string]any`, `bool`, etc.) |

**Extension handling:** Scan map keys for `x-` prefix, collect into `Extra`:

```go
func extractExtensionsFromMap(m map[string]any) map[string]any {
    var extra map[string]any
    for k, v := range m {
        if strings.HasPrefix(k, "x-") {
            if extra == nil {
                extra = make(map[string]any)
            }
            extra[k] = v
        }
    }
    return extra
}
```

**Special case — `Responses`:** Handle inline `Codes` field by iterating non-extension, non-known-field keys and validating as HTTP status codes (same logic as existing `UnmarshalJSON`).

**Generated file layout:**

```
parser/
  oas2_decode_gen.go        # OAS2Document
  oas3_decode_gen.go        # OAS3Document, Components
  common_decode_gen.go      # Info, Contact, License, etc.
  paths_decode_gen.go       # PathItem, Operation, Response, etc.
  schema_decode_gen.go      # Schema, Discriminator, XML
  parameters_decode_gen.go  # Parameter, Items, RequestBody, Header
  security_decode_gen.go    # SecurityScheme, OAuthFlows, OAuthFlow
  decode_helpers.go         # Hand-written: extractExtensionsFromMap, numeric conversion
```

**Trigger:** `go generate ./parser/` or `make generate`.

#### Integration in `parser.go`

Both parse paths (`parseBytesWithBaseDirAndURL` and `parseJSONFastPath`) get the same change:

```go
if p.ResolveRefs {
    doc, oasVersion, err = decodeDocumentFromMap(rawData, version)
} else {
    doc, oasVersion, err = p.parseVersionSpecific(data, version)  // unchanged
}
```

The `decodeDocumentFromMap` dispatch function (~20 lines) switches on OAS version:
- `OASVersion20` → `OAS2Document.decodeFromMap(data)`
- `OASVersion300+` → `OAS3Document.decodeFromMap(data)`

When `ResolveRefs` is false, the existing path is untouched.

### Optimization 2: Shallow refs in resolver

Reduce map inflation by sharing references instead of deep-copying.

#### Change in `resolver.go`

Add `ShallowCopy` field to `RefResolver`:

```go
type RefResolver struct {
    // ...existing fields...
    ShallowCopy bool
}
```

In `resolveRefsRecursive` (~line 468), the current deep copy:

```go
resolved := deepCopyJSONValue(target)
```

Becomes:

```go
var resolved any
if r.ShallowCopy {
    resolved = target
} else {
    resolved = deepCopyJSONValue(target)
}
```

#### When is shallow copy safe?

Shallow copy is safe when the resolved map is consumed read-only — specifically, when `decodeFromMap` creates independent struct copies from the map data. The parser sets `ShallowCopy = true` only when `ResolveRefs` is true.

#### Impact on `result.Data`

After resolution with shallow refs, `result.Data` contains shared sub-maps. Multiple paths point to the same underlying `map[string]any`. This is safe for read-only consumers (walkers, MCP tools, serialization). The `MutableInput` pattern establishes the precedent for this contract.

### Combined memory impact

```
input []byte
  → yaml/json.Unmarshal → map[string]any (~original size)
  → resolver.ResolveAllRefs (shallow copy) → map with shared refs (~1.5x original)
  → decodeFromMap → typed struct (~10-40x, irreducible)
```

| Spec | Before | After | Reduction |
|------|-------:|------:|----------:|
| GitHub API (11.6 MB) | ~320 MB | ~125 MB | **60%** |
| DigitalOcean (2.5 MB) | ~310 MB | ~105 MB | **66%** |

## Tests

### 1. Round-trip equivalence tests

For every test fixture, parse with both the existing `json.Unmarshal` path (ResolveRefs=false) and the new `decodeFromMap` path (ResolveRefs=true on a non-ref spec). Compare resulting Document structs field-by-field to ensure behavioral equivalence.

### 2. Generator freshness test

A test in `cmd/gen-decode/` that runs the generator and compares output with checked-in files. Fails if struct definitions changed without regenerating.

### 3. Shallow-ref safety tests

Parse with deep copy + decodeFromMap vs shallow copy + decodeFromMap. Compare resulting Documents to ensure identical typed output.

### 4. Existing tests as regression guard

All existing parser tests continue to pass unchanged.

## Out of scope

- Reducing the struct size itself (irreducible — resolved content must live somewhere)
- MCP-specific guardrails (can be added independently)
- Lazy resolution during struct construction (unnecessary given shallow refs)
- Third-party dependencies (no `mapstructure` or similar)

## Implementation phases

**Phase 1:** Code generator + generated decoders + `decodeDocumentFromMap` integration + `decode_helpers.go`. This eliminates the re-marshal `[]byte` allocation.

**Phase 2:** Shallow copy option in resolver + parser integration. This reduces map inflation.

Both phases are in the same PR but can be reviewed independently.
