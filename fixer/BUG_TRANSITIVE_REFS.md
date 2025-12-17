# Bug Report: Transitive Schema References Not Followed During Pruning

## Summary

When using `oastools fix -prune-all`, schemas that are transitively referenced via array `items` are incorrectly pruned. This causes the generated Go code to have undefined type errors.

## Reproduction

```bash
oastools fix -infer -prune-all -fix-schema-names -generic-naming of -q spec.json | \
  oastools generate -client -server -p myapi -o ./output -

cd ./output
go mod init myapi
go build ./...
# Results in "undefined: SomeType" errors
```

## Root Cause

The bug is in the parser's YAML/JSON unmarshaling. When parsing OAS 2.0 (and likely OAS 3.x) documents, the `Schema.Items` field is parsed as `map[string]interface{}` instead of `*parser.Schema`.

### Evidence

```go
// In a parsed OAS 2.0 document with:
//   properties:
//     items:
//       type: array
//       items:
//         $ref: '#/definitions/SomeType'

schema := doc.Definitions["MySchema"]
itemsProperty := schema.Properties["items"]

fmt.Printf("Items type: %T\n", itemsProperty.Items)
// Output: Items type: map[string]interface {}   <-- WRONG!
// Expected: Items type: *parser.Schema
```

### Why This Causes Pruning Failures

The pruning logic in `fixer/prune.go` uses type assertions to follow nested schema references:

```go
// collectSchemaRefsRecursive in prune.go
if schema.Items != nil {
    if items, ok := schema.Items.(*parser.Schema); ok {  // ← FAILS because Items is map[string]any
        refs = append(refs, collectSchemaRefsRecursive(items, prefix, visited)...)
    }
}
```

Since `Items` is actually `map[string]interface{}`, the type assertion silently fails, and nested `$ref` values are never collected.

### The Result

1. **Direct refs work**: Schemas referenced directly from operations are collected correctly
2. **Nested refs fail**: Schemas referenced via `properties.*.items.$ref` are NOT collected
3. **Incorrect pruning**: The transitive closure is incomplete, so transitively referenced schemas are marked as "unreferenced"
4. **Undefined types**: Generated code references types that were pruned

## Affected Fields

The same issue likely affects other `any`-typed fields in `parser.Schema`:

| Field | Type | Status |
|-------|------|--------|
| `Items` | `any` | ❌ Confirmed broken |
| `AdditionalProperties` | `any` | ⚠️ Likely broken |
| `AdditionalItems` | `any` | ⚠️ Likely broken |

## Test Cases

See `fixer/prune_transitive_test.go` for comprehensive test cases:

### TestPruneOAS2_TransitiveItemsRefs
Verifies schemas referenced via `items.$ref` are not pruned.

### TestPruneOAS2_DeeplyNestedItemsRefs
Verifies deeply nested chains (4+ levels) of items refs work.

### TestPruneOAS3_TransitiveItemsRefs
Verifies the same issue in OAS 3.x documents.

### TestPruneOAS2_AdditionalPropertiesRefs
Verifies schemas referenced via `additionalProperties.$ref` are not pruned.

### TestPruneOAS2_AllOfRefs
Verifies schemas referenced via `allOf` composition are not pruned.

### TestCollectSchemaRefs_ItemsAsMap
Diagnostic test that confirms the root cause - Items being parsed as map.

## Example Failing Spec

```yaml
swagger: "2.0"
info:
  title: Example API
  version: "1.0.0"
basePath: /v1
paths:
  /items:
    get:
      operationId: getItems
      responses:
        '200':
          description: OK
          schema:
            $ref: '#/definitions/ItemList'
definitions:
  ItemList:
    type: object
    properties:
      items:
        type: array
        items:
          $ref: '#/definitions/Item'    # ← This ref is NOT followed!
  Item:
    type: object
    properties:
      id:
        type: string
```

After `oastools fix -prune-all`:
- ✅ `ItemList` is kept (directly referenced)
- ❌ `Item` is INCORRECTLY pruned (transitive ref via items not followed)

## Proposed Fixes

### Option 1: Fix the Parser (Recommended)

Modify the parser's unmarshaling to properly convert nested schemas from raw maps to `*parser.Schema` types. This is the correct fix as it ensures all downstream code receives properly typed data.

Location: `parser/schema.go` or `parser/parser.go` - custom unmarshal logic for Schema

### Option 2: Fix the Fixer (Workaround)

Add fallback handling in `collectSchemaRefsRecursive` to extract `$ref` from `map[string]interface{}`:

```go
if schema.Items != nil {
    if items, ok := schema.Items.(*parser.Schema); ok {
        refs = append(refs, collectSchemaRefsRecursive(items, prefix, visited)...)
    } else if itemsMap, ok := schema.Items.(map[string]interface{}); ok {
        // Fallback: extract $ref from raw map
        if ref, ok := itemsMap["$ref"].(string); ok {
            if name := extractSchemaName(ref, prefix); name != "" {
                refs = append(refs, name)
            }
        }
    }
}
```

This is a workaround that addresses the symptom but not the root cause.

### Option 3: Both

Fix the parser for correctness, and add the fixer fallback for defense-in-depth.

## Files Involved

- `parser/schema.go` - Schema struct definition with `any`-typed fields
- `parser/parser.go` - Parsing logic that creates raw maps instead of typed structs
- `fixer/prune.go` - `collectSchemaRefsRecursive()` with failing type assertions
- `fixer/refs.go` - `RefCollector.collectSchemaRefs()` with same issue

## Related Issues

This bug affects any code that relies on type assertions for `Schema.Items`, `Schema.AdditionalProperties`, etc. The generator may have similar issues.
