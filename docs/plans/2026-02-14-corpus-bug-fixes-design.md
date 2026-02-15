# Corpus Bug Fixes Design

> Fixes for three bugs discovered during plugin corpus testing (2026-02-14).

## Bug 1: Generator Server Wrapper Type Collision

### Problem

Server mode generates wrapper structs named `{MethodName}Request` (e.g., `BanUserFromGuildRequest`). When the OAS document has a schema with the same name, Go compilation fails with a type redeclaration error.

Discovered with discord-openapi.json — 5 type conflicts.

### Root Cause

- `security_gen_shared.go:77` unconditionally names the wrapper `{MethodName}Request`
- `oas3_generator.go:733` generates `type {MethodName}Request struct`
- No cross-check between schema type names and wrapper struct names
- `generatedTypes` map tracks schema names but wrapper names are generated independently

### Fix

Add a `resolveWrapperName` function that checks against known schema type names:

1. Try `{MethodName}Request`
2. If collision → try `{MethodName}Input`
3. If collision → try `{MethodName}Req`
4. If all collide → numeric fallback: `{MethodName}Request2`, `Request3`, etc.

Pass the `generatedTypes` map (already populated during schema collection) to `buildServerMethodSignature` and `generateRequestType`.

### Files

- `generator/security_gen_shared.go` — new `resolveWrapperName`, updated `buildServerMethodSignature`
- `generator/oas3_generator.go` — `generateRequestType` + `generateServerMethodSignature` pass schema names
- `generator/oas2_generator.go` — same for OAS 2.0 path

### Tests

Synthetic spec with schema `CreatePetRequest` and operation `createPet` → wrapper becomes `CreatePetInput`. Second test with schemas covering all suffixes → numeric fallback.

---

## Bug 2: Converter formData Not Converted (2.0 → 3.x)

### Problem

`in: "formData"` parameters pass through unchanged during 2.0 → 3.x conversion. They remain as parameters instead of being converted to `requestBody` — invalid in OAS 3.x.

Discovered with petstore 2.0 — 4 validation errors in converted output.

### Root Cause

`convertOAS2OperationToOAS3` (oas2_to_oas3.go:170) only checks for `in: "body"` parameters. The `convertOAS2ParameterToOAS3` helper copies `In` field as-is, so `"formData"` passes through unchanged.

### Fix

1. After the existing body-parameter handling in `convertOAS2OperationToOAS3`, add a second pass for `in: "formData"` params
2. Collect all formData params from the source operation
3. Build a schema: each formData param → property (name, type, format)
4. Determine content type:
   - Any param has `type: "file"` → `multipart/form-data`
   - Otherwise → `application/x-www-form-urlencoded`
5. Create `requestBody` with the schema wrapped in the appropriate media type
6. Filter out `in: "formData"` params from `dst.Parameters`
7. If operation already has a `requestBody` from a body param (shouldn't happen per spec), emit warning and skip

### Files

- `converter/oas2_to_oas3.go` — new `convertOAS2FormDataToRequestBody` helper, updated `convertOAS2OperationToOAS3`

### Tests

- Synthetic spec with `in: "formData"` params including `type: "file"` → verify `multipart/form-data` requestBody
- Synthetic spec with `in: "formData"` params without file → verify `application/x-www-form-urlencoded` requestBody
- Both verify formData params removed from converted parameters list

---

## Bug 3: Converter Incomplete 3.0 → 2.0 Downconversion

### Problem A: Missing `type` Field

When `GetPrimaryType()` returns empty for complex schemas (`allOf`, `oneOf`, etc.), converted OAS 2.0 non-body parameters lack the required `type` field.

### Problem B: Unresolvable Header Refs

`#/components/headers/*` refs have no OAS 2.0 equivalent. The ref rewriter has no mapping for them, so they're left unchanged → validation errors.

Discovered with nws 3.0 → 2.0 — 189 validation errors.

### Fix A: Type Inference Fallback

In `convertOAS3ParameterToOAS2` (helpers.go), after existing type extraction:

1. If `converted.Type` is still empty for a non-body param:
   - Walk schema's `allOf`/`oneOf`/`anyOf` to find a concrete type
   - If found → use it, emit `SeverityInfo`
   - If not found → default to `"string"`, emit `SeverityWarning` ("defaulting to string")

### Fix B: Header Ref Inlining

During `convertOAS3ResponseToOAS2`, when processing headers with `$ref: "#/components/headers/..."`:

1. Extract the header name from the ref path
2. Look up the header in source OAS3 document's `Components.Headers`
3. Inline the resolved header definition directly
4. Emit `SeverityInfo` ("inlined component header ref")

This requires threading source `Components.Headers` to the response converter. OAS 2.0 has no reusable header definition section (response headers are always inline), so inlining is the only option.

### Files

- `converter/helpers.go` — type fallback logic in `convertOAS3ParameterToOAS2`
- `converter/oas3_to_oas2.go` — pass source headers to `convertOAS3ResponseToOAS2`, inline header refs

### Tests

- Synthetic spec with `allOf` parameter schema → verify type is inferred
- Synthetic spec with `$ref: "#/components/headers/X-Rate-Limit"` in response → verify header is inlined and ref is resolved

---

## Implementation Notes

- Bug 1 and Bug 2+3 are in different packages (`generator` vs `converter`) — can be developed in parallel
- All tests are synthetic (minimal test specs), no corpus file dependencies
- Each bug gets its own commit with conventional commit message (`fix(generator):`, `fix(converter):`)