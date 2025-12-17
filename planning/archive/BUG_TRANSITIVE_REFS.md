# [FIXED] Transitive Schema References Not Followed During Pruning - Part 2

> **Status:** ✅ FIXED - Complete fix in v1.28.3
> **Original Fix:** v1.28.2 (PR #150) - `e711ed0` - INCOMPLETE
> **Complete Fix:** v1.28.3 - Updated `refs.go` to match `prune.go` fallback handling

---

## Summary

When using `oastools fix -prune-all`, schemas that are transitively referenced via array `items` in **inline schemas** (parameters, request bodies) are incorrectly pruned. This causes the generated Go code to have undefined type errors.

**The v1.28.2 fix was incomplete.** It only added `map[string]any` fallback handling to `prune.go`'s `collectSchemaRefsRecursive()`, but the same fix was NOT applied to `refs.go`'s `RefCollector.collectSchemaRefs()`.

## Current Failure

```bash
oastools fix -infer -prune-all -fix-schema-names -generic-naming of -q https://example.com/swagger/v1 | \
  oastools generate -client -server -p myapi -o ./output -

cd ./output
go mod init github.com/example/api
go build ./...
# Results in errors like:
# ./server_walrus.go:46:16: undefined: WalrusAggregateQueryRequest
# ./server.go:109:71: undefined: PelicanGroupItem
# ./server_otter.go:12:16: undefined: OtterAggregateQueryRequest
```

## Root Cause Analysis

### The Architecture

The fixer uses a two-phase approach to find all referenced schemas:

1. **Phase 1 - Initial Collection** (`refs.go`):
   - `RefCollector.CollectOAS2/3()` traverses the document
   - Calls `RefCollector.collectSchemaRefs()` for each schema encountered
   - Builds `RefsByType[RefTypeSchema]` with all directly-found refs

2. **Phase 2 - Transitive Closure** (`prune.go`):
   - `buildReferencedSchemaSet()` takes the Phase 1 refs as seeds
   - For each seed, calls `collectSchemaRefs()` (from `prune.go`) to find transitive refs
   - Expands the set until no new refs are found

### The Bug

**`refs.go` lines 501-519 were NOT updated in v1.28.2:**

```go
// AdditionalProperties (can be *Schema or bool)
if schema.AdditionalProperties != nil {
    if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
        c.collectSchemaRefs(addProps, fmt.Sprintf("%s.additionalProperties", path))
    }
    // ❌ NO FALLBACK FOR map[string]any
}

// Items (can be *Schema or bool in OAS 3.1+)
if schema.Items != nil {
    if items, ok := schema.Items.(*parser.Schema); ok {
        c.collectSchemaRefs(items, fmt.Sprintf("%s.items", path))
    }
    // ❌ NO FALLBACK FOR map[string]any
}
```

**While `prune.go` lines 114-140 WERE updated:**

```go
// AdditionalProperties (can be *Schema or bool)
if schema.AdditionalProperties != nil {
    if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
        refs = append(refs, collectSchemaRefsRecursive(addProps, prefix, visited)...)
    } else if addPropsMap, ok := schema.AdditionalProperties.(map[string]any); ok {
        // ✅ Fallback: extract refs from raw map (polymorphic field may remain as map)
        refs = append(refs, collectRefsFromMap(addPropsMap, prefix)...)
    }
}

// Items (can be *Schema or bool in OAS 3.1+)
if schema.Items != nil {
    if items, ok := schema.Items.(*parser.Schema); ok {
        refs = append(refs, collectSchemaRefsRecursive(items, prefix, visited)...)
    } else if itemsMap, ok := schema.Items.(map[string]any); ok {
        // ✅ Fallback: extract refs from raw map (polymorphic field may remain as map)
        refs = append(refs, collectRefsFromMap(itemsMap, prefix)...)
    }
}
```

### Why This Causes Failures

Consider this OAS 2.0 JSON spec:

```json
{
  "swagger": "2.0",
  "info": { "title": "Walrus API", "version": "1.0" },
  "basePath": "/v1",
  "paths": {
    "/walrus/aggregates": {
      "post": {
        "operationId": "QueryWalrusAggregate",
        "parameters": [{
          "name": "body",
          "in": "body",
          "schema": {
            "type": "array",
            "items": { "$ref": "#/definitions/WalrusAggregateQueryRequest" }
          }
        }],
        "responses": {
          "200": {
            "description": "OK",
            "schema": { "$ref": "#/definitions/WalrusAggregatesResponse" }
          }
        }
      }
    }
  },
  "definitions": {
    "WalrusAggregateQueryRequest": {
      "type": "object",
      "properties": {
        "date_ranges": {
          "type": "array",
          "items": { "$ref": "#/definitions/WalrusDateRangeSpec" }
        },
        "field": { "type": "string" }
      }
    },
    "WalrusDateRangeSpec": {
      "type": "object",
      "properties": {
        "from": { "type": "string" },
        "to": { "type": "string" }
      }
    },
    "WalrusAggregatesResponse": {
      "type": "object",
      "properties": {
        "resources": {
          "type": "array",
          "items": { "$ref": "#/definitions/PelicanAggregatesResponse" }
        }
      }
    },
    "PelicanAggregatesResponse": {
      "type": "object",
      "properties": { "count": { "type": "integer" } }
    }
  }
}
```

**Phase 1 (`refs.go`) collects:**
- `#/definitions/WalrusAggregatesResponse` (from response schema `$ref`)
- ❌ MISSES `#/definitions/WalrusAggregateQueryRequest` (inline `items` is `map[string]any`)

**Phase 2 (`prune.go`) expands:**
- From `WalrusAggregatesResponse` → finds `PelicanAggregatesResponse` (via `items` in definitions)
- But `WalrusAggregateQueryRequest` was never in the seed set!

**Result:**
- `WalrusAggregateQueryRequest` is pruned
- `WalrusDateRangeSpec` is pruned (was only reachable through the pruned schema)
- Generated code references these types → compile error

### The Key Insight

The bug only manifests when:
1. A schema reference exists in an **inline schema** (not in `definitions`/`components/schemas`)
2. That inline schema has `items` or `additionalProperties` as `map[string]any`
3. The referenced schema is not also referenced elsewhere that `RefCollector` can see

This is why the existing tests pass - they test schemas within `definitions` that reference each other. The `prune.go` fix handles transitive closure within definitions. But inline schemas in parameters/request bodies are handled by `refs.go`, which is broken.

## Diagnostic Evidence

Running `TestCollectSchemaRefs_ItemsAsMap` from `fixer/prune_transitive_test.go`:

```
=== RUN   TestCollectSchemaRefs_ItemsAsMap
    prune_transitive_test.go:495: Items type: map[string]interface {}
    prune_transitive_test.go:496: Items is *parser.Schema: false
    prune_transitive_test.go:497: Items is map[string]interface{}: true
    prune_transitive_test.go:501: BUG CONFIRMED: Items with $ref is parsed as map[string]interface{} instead of *parser.Schema
    prune_transitive_test.go:502: This causes collectSchemaRefsRecursive to miss nested refs
    prune_transitive_test.go:511: Schema refs collected: map[#/definitions/WombatList:true]
    prune_transitive_test.go:516: BUG CONFIRMED: Ref "#/definitions/Wombat" was NOT collected from WombatList.wombats.items
--- PASS: TestCollectSchemaRefs_ItemsAsMap (0.00s)
```

The test documents the bug but passes because it doesn't actually assert on the broken behavior.

## Verified Fix (Tested Against Original Case)

The fix has been **verified locally** against the original failing spec:

**Before fix:**
- `walrus.AggregateQueryRequest`: Referenced 2x, defined 0x (PRUNED)
- `types.GroupItem`: Referenced 1x, defined 0x (PRUNED)
- `go build` fails with undefined type errors

**After fix:**
- `walrus.AggregateQueryRequest`: Referenced 2x, defined 1x ✅
- `types.GroupItem`: Referenced 1x, defined 1x ✅
- `go build` succeeds ✅

## Affected Fields (Both files need fixing)

| Field | Type | prune.go | refs.go |
|-------|------|----------|---------|
| `Items` | `any` | ✅ Fixed | ❌ Missing |
| `AdditionalProperties` | `any` | ✅ Fixed | ❌ Missing |
| `AdditionalItems` | `any` | ✅ Fixed | ❌ Missing |

## Fix Plan

### Step 1: Add helper method to RefCollector

Add `collectRefsFromMap()` method to `RefCollector` in `refs.go` (mirrors the function in `prune.go`):

```go
// collectRefsFromMap extracts schema references from a raw map[string]any.
// This handles polymorphic schema fields (Items, AdditionalProperties, etc.) that may
// remain as untyped maps after YAML/JSON unmarshaling. These fields are declared as
// `any` in parser.Schema to support both *Schema and bool values per the OAS spec.
func (c *RefCollector) collectRefsFromMap(m map[string]any, path string) {
    // Check for direct $ref
    if refStr, ok := m["$ref"].(string); ok && refStr != "" {
        c.addRef(refStr, path, RefTypeSchema)
    }

    // Check nested properties
    if props, ok := m["properties"].(map[string]any); ok {
        for propName, propVal := range props {
            if propMap, ok := propVal.(map[string]any); ok {
                c.collectRefsFromMap(propMap, fmt.Sprintf("%s.properties.%s", path, propName))
            }
        }
    }

    // Check items
    if items, ok := m["items"].(map[string]any); ok {
        c.collectRefsFromMap(items, fmt.Sprintf("%s.items", path))
    }

    // Check additionalProperties
    if addProps, ok := m["additionalProperties"].(map[string]any); ok {
        c.collectRefsFromMap(addProps, fmt.Sprintf("%s.additionalProperties", path))
    }

    // Check allOf, anyOf, oneOf
    for _, key := range []string{"allOf", "anyOf", "oneOf"} {
        if arr, ok := m[key].([]any); ok {
            for i, item := range arr {
                if itemMap, ok := item.(map[string]any); ok {
                    c.collectRefsFromMap(itemMap, fmt.Sprintf("%s.%s[%d]", path, key, i))
                }
            }
        }
    }
}
```

### Step 2: Add fallbacks in collectSchemaRefs

Update `RefCollector.collectSchemaRefs()` to call the helper for `map[string]any` cases:

```go
// Items
if schema.Items != nil {
    if items, ok := schema.Items.(*parser.Schema); ok {
        c.collectSchemaRefs(items, fmt.Sprintf("%s.items", path))
    } else if itemsMap, ok := schema.Items.(map[string]any); ok {
        c.collectRefsFromMap(itemsMap, fmt.Sprintf("%s.items", path))
    }
}

// AdditionalProperties
if schema.AdditionalProperties != nil {
    if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
        c.collectSchemaRefs(addProps, fmt.Sprintf("%s.additionalProperties", path))
    } else if addPropsMap, ok := schema.AdditionalProperties.(map[string]any); ok {
        c.collectRefsFromMap(addPropsMap, fmt.Sprintf("%s.additionalProperties", path))
    }
}

// AdditionalItems
if schema.AdditionalItems != nil {
    if addItems, ok := schema.AdditionalItems.(*parser.Schema); ok {
        c.collectSchemaRefs(addItems, fmt.Sprintf("%s.additionalItems", path))
    } else if addItemsMap, ok := schema.AdditionalItems.(map[string]any); ok {
        c.collectRefsFromMap(addItemsMap, fmt.Sprintf("%s.additionalItems", path))
    }
}
```

### Step 3: Add regression test

Add a test case that uses inline parameter schemas with `items.$ref`:

```go
// TestPruneOAS2_InlineParameterItemsRefs verifies that schemas referenced via
// inline parameter schemas with array items are not pruned.
// This is a regression test for the incomplete v1.28.2 fix.
func TestPruneOAS2_InlineParameterItemsRefs(t *testing.T) {
    // Test that refs in inline parameter schemas are collected
    // This is the case that was missed in v1.28.2
}
```

## Files to Modify

1. `fixer/refs.go` - Add `collectRefsFromMap` method and fallback handling
2. `fixer/prune_transitive_test.go` - Add test for inline parameter schemas

## Timeline

- **v1.28.2** (2025-12-16): Partial fix - `prune.go` updated, `refs.go` missed
- **v1.28.3** (2025-12-16): Complete fix - `refs.go` updated with `collectRefsFromMap` method and fallback handling for `Items`, `AdditionalProperties`, and `AdditionalItems`
