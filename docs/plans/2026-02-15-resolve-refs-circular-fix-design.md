# Design: Fix resolve_refs for documents with circular references

**Issue:** #326
**Date:** 2026-02-15
**Status:** Approved

## Problem

When `parser.WithResolveRefs(true)` is used on a spec with circular `$ref`s, the parser discards all resolution work, not just the circular parts. 4 of 10 corpus specs (DigitalOcean, Discord, MS Graph, Stripe) are affected.

### Root cause

After the resolver walks `rawData` in-place (resolving non-circular refs, leaving circular refs as `$ref` strings), the parser must re-serialize to `[]byte` for version-specific parsing. When `hasCircularRefs` is true, both parse paths skip re-serialization and fall back to the original unmodified `data` bytes:

```go
if p.ResolveRefs && !hasCircularRefs {
    parseData, err = yaml.Marshal(rawData)  // skipped when circular
} else if hasCircularRefs {
    parseData = data  // throws away all resolution work
}
```

### Why the guard is unnecessary

The resolver's `deepCopyJSONValue` prevents Go pointer cycles. Circular refs are left as `$ref` strings, not actual pointer cycles. Empirical testing confirms `yaml.Marshal` and `json.Marshal` both succeed after resolution on specs with circular refs:

| Spec | Circular? | yaml.Marshal | json.Marshal |
|------|:---------:|:------------:|:------------:|
| circular-schema.yaml | Yes | OK (1 KB) | OK (514 B) |
| DigitalOcean (2.5 MB) | Yes | OK (107 MB) | N/A |

## Design

### Approach: Remove the hasCircularRefs guard

The simplest fix. The resolver already handles circular refs correctly; the bug is entirely in the parser's overly conservative guard.

### Changes

**`parseBytesWithBaseDirAndURL` (~line 1007):**
- Change `if p.ResolveRefs && !hasCircularRefs` to `if p.ResolveRefs`
- Remove the `else if hasCircularRefs` fallback branch
- When `hasCircularRefs` is true and marshal succeeds, emit an updated warning

**`parseJSONFastPath` (~line 1094):**
- Same change, using `json.Marshal` instead of `yaml.Marshal`

**Warning message:**
- Old: "Circular references detected. Using original document structure. Some references may not be fully resolved."
- New: "Circular references detected. Non-circular references resolved normally. Circular references remain as $ref pointers."

### Tests

New file `parser/resolver_circular_test.go`:

1. **`TestResolveRefs_CircularRefsPreservesNonCircularResolution`** — Parse `circular-schema.yaml` with resolve_refs. Verify non-circular refs are resolved while circular refs remain as $ref pointers.

2. **`TestResolveRefs_CircularRefsWarningMessage`** — Verify updated warning message.

3. **`TestResolveRefs_CircularRefsJSONFastPath`** — Same as test 1 with JSON input to exercise `parseJSONFastPath`.

4. **`TestResolveRefs_NonCircularRefsUnaffected`** — Regression guard: parse petstore with resolve_refs, verify refs still resolve.

## Out of scope

- Size blowup from ref inlining (tracked in #328, pre-existing for non-circular specs)
- MCP walker integration tests (separate concern)
- Two-pass resolution or cycle-safe marshaling (unnecessary given the deep-copy approach)
