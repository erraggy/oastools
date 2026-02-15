# Corpus Bug Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix three bugs discovered during plugin corpus testing: generator type collision, converter formData passthrough, and converter downconversion gaps.

**Architecture:** Three independent fixes across two packages. Bug 1 modifies the generator's server wrapper naming. Bugs 2 and 3 modify the converter's parameter and response handling. Each bug gets TDD treatment with synthetic tests.

**Tech Stack:** Go, testify (assert/require), parser package types, schemautil helpers.

---

### Task 1: Generator — Add `resolveWrapperName` helper

**Files:**
- Modify: `generator/security_gen_shared.go:60-80`

**Step 1: Write the failing test**

Add to `generator/server_test.go`:

```go
func TestServerWrapperTypeCollision(t *testing.T) {
	// Schema named "CreatePetRequest" should collide with createPet operation's wrapper
	spec := `openapi: "3.0.0"
info:
  title: Pet API
  version: "1.0.0"
paths:
  /pets:
    post:
      operationId: createPet
      responses:
        '201':
          description: Pet created
components:
  schemas:
    CreatePetRequest:
      type: object
      properties:
        name:
          type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "collision.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
	)
	require.NoError(t, err)

	serverFile := result.GetFile("server.go")
	require.NotNil(t, serverFile, "server.go not generated")

	content := string(serverFile.Content)
	// The wrapper should be renamed to avoid collision with the schema type
	assert.Contains(t, content, "type CreatePetInput struct", "wrapper should use Input suffix to avoid collision")
	assert.NotContains(t, content, "type CreatePetRequest struct {", "should not have duplicate CreatePetRequest type")
	// The interface signature should reference the renamed type
	assert.Contains(t, content, "req *CreatePetInput)", "interface should reference renamed wrapper")
	// The schema type should still exist
	typesFile := result.GetFile("types.go")
	require.NotNil(t, typesFile, "types.go not generated")
	assert.Contains(t, string(typesFile.Content), "type CreatePetRequest struct")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./generator/ -run TestServerWrapperTypeCollision -v`
Expected: FAIL — the wrapper is still named `CreatePetRequest`, causing a duplicate type.

**Step 3: Implement `resolveWrapperName`**

Add to `generator/security_gen_shared.go` (before `buildServerMethodSignature`):

```go
// wrapperSuffixes is the ordered list of suffixes to try when naming server wrapper types.
var wrapperSuffixes = []string{"Request", "Input", "Req"}

// resolveWrapperName picks a wrapper type name that doesn't collide with schema types.
// It tries {methodName}Request, then Input, then Req, then numeric fallback.
func resolveWrapperName(methodName string, schemaTypes map[string]bool) string {
	for _, suffix := range wrapperSuffixes {
		candidate := methodName + suffix
		if !schemaTypes[candidate] {
			return candidate
		}
	}
	// Numeric fallback
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%sRequest%d", methodName, i)
		if !schemaTypes[candidate] {
			return candidate
		}
	}
}
```

**Step 4: Update `buildServerMethodSignature` to accept schema types**

Change the signature and body in `generator/security_gen_shared.go:62`:

```go
func buildServerMethodSignature(path, method string, op *parser.Operation, responseType string, schemaTypes map[string]bool) string {
	var buf bytes.Buffer

	methodName := operationToMethodName(op, path, method)
	wrapperName := resolveWrapperName(methodName, schemaTypes)

	// Write comment - handle multiline descriptions properly
	if op.Summary != "" {
		buf.WriteString(formatMultilineComment(op.Summary, methodName, "\t"))
	} else if op.Description != "" {
		buf.WriteString(formatMultilineComment(op.Description, methodName, "\t"))
	}
	if op.Deprecated {
		buf.WriteString("\t// Deprecated: This operation is deprecated.\n")
	}

	buf.WriteString(fmt.Sprintf("\t%s(ctx context.Context, req *%s) (%s, error)\n", methodName, wrapperName, responseType))

	return buf.String()
}
```

**Step 5: Update all callers of `buildServerMethodSignature`**

In `generator/oas3_generator.go`, update `generateServerMethodSignature`:

```go
func (cg *oas3CodeGenerator) generateServerMethodSignature(path, method string, op *parser.Operation) string {
	return buildServerMethodSignature(path, method, op, cg.getResponseType(op), cg.generatedTypes)
}
```

In `generator/oas2_generator.go`, update `generateServerMethodSignature`:

```go
func (cg *oas2CodeGenerator) generateServerMethodSignature(path, method string, op *parser.Operation) string {
	return buildServerMethodSignature(path, method, op, cg.getResponseType(op), cg.generatedTypes)
}
```

**Step 6: Update `generateRequestType` in both generators**

In `generator/oas3_generator.go`, update `generateRequestType` (line 727):

```go
func (cg *oas3CodeGenerator) generateRequestType(path, method string, op *parser.Operation) string {
	var buf bytes.Buffer

	methodName := operationToMethodName(op, path, method)
	wrapperName := resolveWrapperName(methodName, cg.generatedTypes)

	buf.WriteString(fmt.Sprintf("// %s contains the request data for %s.\n", wrapperName, methodName))
	buf.WriteString(fmt.Sprintf("type %s struct {\n", wrapperName))
	// ... rest unchanged
```

In `generator/oas2_generator.go`, update `generateRequestType` (line 705):

```go
func (cg *oas2CodeGenerator) generateRequestType(path, method string, op *parser.Operation) string {
	var buf bytes.Buffer

	methodName := operationToMethodName(op, path, method)
	wrapperName := resolveWrapperName(methodName, cg.generatedTypes)

	buf.WriteString(fmt.Sprintf("// %s contains the request data for %s.\n", wrapperName, methodName))
	buf.WriteString(fmt.Sprintf("type %s struct {\n", wrapperName))
	// ... rest unchanged
```

**Step 7: Run test to verify it passes**

Run: `go test ./generator/ -run TestServerWrapperTypeCollision -v`
Expected: PASS

**Step 8: Run full generator test suite**

Run: `go test ./generator/ -v -count=1`
Expected: All tests pass. Existing tests use schemas like `Pet` that don't collide with `ListPetsRequest`/`CreatePetRequest`, so no naming changes.

**Step 9: Run go_diagnostics on modified files**

Run gopls diagnostics on all modified files.

**Step 10: Commit**

```
fix(generator): resolve server wrapper type collisions with schema names

When a schema name like "CreatePetRequest" collides with the server
wrapper struct for the "createPet" operation, use suffix cascade
(Input → Req → numeric) to disambiguate.
```

---

### Task 2: Generator — Test numeric fallback

**Files:**
- Modify: `generator/server_test.go`

**Step 1: Write the failing test (should pass immediately)**

```go
func TestServerWrapperTypeCollision_AllSuffixes(t *testing.T) {
	// Schema names covering all suffixes: Request, Input, Req
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /items:
    post:
      operationId: createItem
      responses:
        '201':
          description: Created
components:
  schemas:
    CreateItemRequest:
      type: object
    CreateItemInput:
      type: object
    CreateItemReq:
      type: object
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "all-suffixes.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithServer(true),
	)
	require.NoError(t, err)

	serverFile := result.GetFile("server.go")
	require.NotNil(t, serverFile)

	content := string(serverFile.Content)
	// All suffixes taken, should fall back to numeric
	assert.Contains(t, content, "type CreateItemRequest2 struct", "should use numeric fallback")
	assert.Contains(t, content, "req *CreateItemRequest2)", "interface should reference numeric fallback")
}
```

**Step 2: Run test**

Run: `go test ./generator/ -run TestServerWrapperTypeCollision_AllSuffixes -v`
Expected: PASS (the implementation from Task 1 already handles this)

**Step 3: Commit**

```
test(generator): add numeric fallback test for wrapper type collision
```

---

### Task 3: Converter — formData to requestBody (2.0 → 3.x)

**Files:**
- Modify: `converter/oas2_to_oas3.go:144-191`
- Test: `converter/oas2_to_oas3_test.go`

**Step 1: Write the failing test**

Add to `converter/oas2_to_oas3_test.go`:

```go
func TestConvertOAS2FormDataToRequestBody(t *testing.T) {
	tests := []struct {
		name              string
		operation         *parser.Operation
		doc               *parser.OAS2Document
		expectedMediaType string
		expectedProps     []string
		expectNoFormData  bool
	}{
		{
			name: "url-encoded form data",
			operation: &parser.Operation{
				OperationID: "createPet",
				Parameters: []*parser.Parameter{
					{Name: "name", In: "formData", Type: "string", Required: true},
					{Name: "age", In: "formData", Type: "integer"},
				},
				Responses: &parser.Responses{},
			},
			doc:               &parser.OAS2Document{},
			expectedMediaType: "application/x-www-form-urlencoded",
			expectedProps:     []string{"name", "age"},
			expectNoFormData:  true,
		},
		{
			name: "multipart form data with file",
			operation: &parser.Operation{
				OperationID: "uploadFile",
				Parameters: []*parser.Parameter{
					{Name: "file", In: "formData", Type: "file"},
					{Name: "description", In: "formData", Type: "string"},
				},
				Responses: &parser.Responses{},
			},
			doc:               &parser.OAS2Document{},
			expectedMediaType: "multipart/form-data",
			expectedProps:     []string{"file", "description"},
			expectNoFormData:  true,
		},
		{
			name: "mixed formData and query params",
			operation: &parser.Operation{
				OperationID: "createWithQuery",
				Parameters: []*parser.Parameter{
					{Name: "name", In: "formData", Type: "string"},
					{Name: "limit", In: "query", Type: "integer"},
				},
				Responses: &parser.Responses{},
			},
			doc:               &parser.OAS2Document{},
			expectedMediaType: "application/x-www-form-urlencoded",
			expectedProps:     []string{"name"},
			expectNoFormData:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Converter{}
			result := &ConversionResult{}
			converted := c.convertOAS2OperationToOAS3(tt.operation, tt.doc, result, "paths./test.post")

			// Should have requestBody
			require.NotNil(t, converted.RequestBody, "expected requestBody from formData params")

			// Check media type
			require.Contains(t, converted.RequestBody.Content, tt.expectedMediaType)
			mediaType := converted.RequestBody.Content[tt.expectedMediaType]
			require.NotNil(t, mediaType.Schema, "expected schema in media type")

			// Check properties
			for _, prop := range tt.expectedProps {
				assert.Contains(t, mediaType.Schema.Properties, prop,
					"expected property %s in requestBody schema", prop)
			}

			// No formData params should remain
			if tt.expectNoFormData {
				for _, param := range converted.Parameters {
					assert.NotEqual(t, "formData", param.In,
						"formData params should be removed after conversion")
				}
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./converter/ -run TestConvertOAS2FormDataToRequestBody -v`
Expected: FAIL — `converted.RequestBody` is nil.

**Step 3: Implement formData conversion**

Add to `converter/oas2_to_oas3.go`, after the existing body-param block (after line 188):

```go
	// Convert formData parameters to requestBody
	hasFormData := false
	for _, param := range src.Parameters {
		if param != nil && param.In == "formData" {
			hasFormData = true
			break
		}
	}

	if hasFormData {
		if dst.RequestBody != nil {
			// Shouldn't happen per spec (body and formData are mutually exclusive)
			c.addIssueWithContext(result, opPath,
				"Operation has both body and formData parameters",
				"OAS 2.0 spec forbids this; formData parameters ignored")
		} else {
			dst.RequestBody = c.convertOAS2FormDataToRequestBody(src, doc)
			// Remove formData parameters from the parameters list
			filteredParams := make([]*parser.Parameter, 0, len(dst.Parameters))
			for _, param := range dst.Parameters {
				if param != nil && param.In != "formData" {
					filteredParams = append(filteredParams, param)
				}
			}
			dst.Parameters = filteredParams
		}
	}
```

Add the helper function after `convertOAS2RequestBody`:

```go
// convertOAS2FormDataToRequestBody converts OAS 2.0 formData parameters to OAS 3.x requestBody.
func (c *Converter) convertOAS2FormDataToRequestBody(src *parser.Operation, doc *parser.OAS2Document) *parser.RequestBody {
	// Collect formData parameters
	var formDataParams []*parser.Parameter
	hasFile := false
	for _, param := range src.Parameters {
		if param != nil && param.In == "formData" {
			formDataParams = append(formDataParams, param)
			if param.Type == "file" {
				hasFile = true
			}
		}
	}

	if len(formDataParams) == 0 {
		return nil
	}

	// Build schema from formData parameters
	schema := &parser.Schema{
		Type:       "object",
		Properties: make(map[string]*parser.Schema),
	}

	var required []string
	for _, param := range formDataParams {
		propSchema := &parser.Schema{}
		if param.Type == "file" {
			propSchema.Type = "string"
			propSchema.Format = "binary"
		} else {
			propSchema.Type = param.Type
			propSchema.Format = param.Format
		}
		if param.Description != "" {
			propSchema.Description = param.Description
		}
		schema.Properties[param.Name] = propSchema
		if param.Required {
			required = append(required, param.Name)
		}
	}
	if len(required) > 0 {
		schema.Required = required
	}

	// Determine content type
	contentType := "application/x-www-form-urlencoded"
	if hasFile {
		contentType = "multipart/form-data"
	} else {
		// Check consumes for explicit media type
		consumes := c.getConsumes(src, doc)
		for _, ct := range consumes {
			if ct == "multipart/form-data" {
				contentType = ct
				break
			}
		}
	}

	return &parser.RequestBody{
		Content: map[string]*parser.MediaType{
			contentType: {Schema: schema},
		},
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./converter/ -run TestConvertOAS2FormDataToRequestBody -v`
Expected: PASS

**Step 5: Run full converter test suite**

Run: `go test ./converter/ -v -count=1`
Expected: All tests pass.

**Step 6: Run go_diagnostics**

Run gopls diagnostics on modified files.

**Step 7: Commit**

```
fix(converter): convert formData parameters to requestBody in 2.0→3.x

OAS 2.0 `in: "formData"` parameters are now properly converted to
`requestBody` with `application/x-www-form-urlencoded` or
`multipart/form-data` (when file params present).
```

---

### Task 4: Converter — Type inference fallback for 3.0 → 2.0 params

**Files:**
- Modify: `converter/helpers.go:76-117`
- Test: `converter/oas3_to_oas2_test.go`

**Step 1: Write the failing test**

Add to `converter/oas3_to_oas2_test.go`:

```go
func TestConvertOAS3ParameterToOAS2_TypeFallback(t *testing.T) {
	tests := []struct {
		name         string
		param        *parser.Parameter
		expectedType string
		expectIssue  bool
	}{
		{
			name: "allOf with concrete type",
			param: &parser.Parameter{
				Name: "filter",
				In:   "query",
				Schema: &parser.Schema{
					AllOf: []*parser.Schema{
						{Type: "string"},
						{Description: "filter constraint"},
					},
				},
			},
			expectedType: "string",
			expectIssue:  true,
		},
		{
			name: "oneOf with concrete types",
			param: &parser.Parameter{
				Name: "id",
				In:   "query",
				Schema: &parser.Schema{
					OneOf: []*parser.Schema{
						{Type: "string"},
						{Type: "integer"},
					},
				},
			},
			expectedType: "string",
			expectIssue:  true,
		},
		{
			name: "no type at all defaults to string",
			param: &parser.Parameter{
				Name:   "unknown",
				In:     "query",
				Schema: &parser.Schema{},
			},
			expectedType: "string",
			expectIssue:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Converter{}
			result := &ConversionResult{}
			converted := c.convertOAS3ParameterToOAS2(tt.param, result, "parameters.test")

			require.NotNil(t, converted)
			assert.Equal(t, tt.expectedType, converted.Type,
				"expected type %q for parameter %s", tt.expectedType, tt.param.Name)

			if tt.expectIssue {
				assert.NotEmpty(t, result.Issues, "expected conversion issue")
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./converter/ -run TestConvertOAS3ParameterToOAS2_TypeFallback -v`
Expected: FAIL — `converted.Type` is empty for allOf/oneOf schemas.

**Step 3: Implement type fallback**

In `converter/helpers.go`, after line 106 (inside `convertOAS3ParameterToOAS2`), add:

```go
		// Fallback: infer type from composite schemas
		if converted.Type == "" && param.In != "body" {
			inferred := inferTypeFromSchema(schema)
			if inferred != "" {
				converted.Type = inferred
				c.addIssueWithContext(result, path,
					fmt.Sprintf("Inferred type '%s' from composite schema", inferred),
					"OAS 2.0 requires explicit type for non-body parameters")
			} else {
				converted.Type = "string"
				c.addIssueWithContext(result, path,
					"Could not infer type from schema, defaulting to 'string'",
					"OAS 2.0 requires explicit type for non-body parameters")
			}
		}
```

Add the helper function (before `convertOAS2ResponseToOAS3Old`):

```go
// inferTypeFromSchema walks allOf/oneOf/anyOf to find a concrete type.
func inferTypeFromSchema(schema *parser.Schema) string {
	if schema == nil {
		return ""
	}
	// Check allOf first (most common for extending schemas)
	for _, sub := range schema.AllOf {
		if t := schemautil.GetPrimaryType(sub); t != "" {
			return t
		}
	}
	// Then oneOf/anyOf — use first concrete type
	for _, sub := range schema.OneOf {
		if t := schemautil.GetPrimaryType(sub); t != "" {
			return t
		}
	}
	for _, sub := range schema.AnyOf {
		if t := schemautil.GetPrimaryType(sub); t != "" {
			return t
		}
	}
	return ""
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./converter/ -run TestConvertOAS3ParameterToOAS2_TypeFallback -v`
Expected: PASS

**Step 5: Run go_diagnostics**

Run gopls diagnostics on modified files.

**Step 6: Commit**

```
fix(converter): infer parameter type from composite schemas in 3.0→2.0

When converting OAS 3.x parameters with allOf/oneOf/anyOf schemas to
OAS 2.0, extract the concrete type. Falls back to "string" with a
warning when no type can be inferred.
```

---

### Task 5: Converter — Inline component header refs in 3.0 → 2.0

**Files:**
- Modify: `converter/helpers.go:150-194`
- Modify: `converter/oas3_to_oas2.go:9-99`
- Test: `converter/oas3_to_oas2_test.go`

**Step 1: Write the failing test**

Add to `converter/oas3_to_oas2_test.go`:

```go
func TestConvertOAS3ToOAS2_InlineHeaderRefs(t *testing.T) {
	c := &Converter{}
	result := &ConversionResult{}

	src := parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion30,
		Document: &parser.OAS3Document{
			OpenAPI:    "3.0.0",
			OASVersion: parser.OASVersion30,
			Info:       &parser.Info{Title: "Test", Version: "1.0.0"},
			Paths: parser.Paths{
				"/test": &parser.PathItem{
					Get: &parser.Operation{
						OperationID: "getTest",
						Responses: &parser.Responses{
							Codes: map[string]*parser.Response{
								"200": {
									Description: "OK",
									Headers: map[string]*parser.Header{
										"X-Rate-Limit": {
											Ref: "#/components/headers/X-Rate-Limit",
										},
									},
								},
							},
						},
					},
				},
			},
			Components: &parser.Components{
				Headers: map[string]*parser.Header{
					"X-Rate-Limit": {
						Description: "Rate limit",
						Schema:      &parser.Schema{Type: "integer"},
					},
				},
			},
		},
	}

	err := c.convertOAS3ToOAS2(src, result)
	require.NoError(t, err)

	doc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok)

	// Check that the header ref was inlined
	resp := doc.Paths["/test"].Get.Responses.Codes["200"]
	require.NotNil(t, resp)
	require.Contains(t, resp.Headers, "X-Rate-Limit")

	header := resp.Headers["X-Rate-Limit"]
	assert.Empty(t, header.Ref, "ref should be resolved/cleared")
	assert.Equal(t, "Rate limit", header.Description)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./converter/ -run TestConvertOAS3ToOAS2_InlineHeaderRefs -v`
Expected: FAIL — the header ref is left unresolved.

**Step 3: Add `sourceHeaders` field to Converter**

In `converter/converter.go`, add a field to the `Converter` struct:

```go
type Converter struct {
	// ... existing fields ...
	// sourceHeaders holds OAS 3.x component headers during 3.0→2.0 conversion,
	// allowing response header refs to be inlined.
	sourceHeaders map[string]*parser.Header
}
```

**Step 4: Set sourceHeaders in `convertOAS3ToOAS2`**

In `converter/oas3_to_oas2.go`, after the `src.Components != nil` check (around line 29), store the source headers:

```go
	if src.Components != nil {
		// Store source headers for ref inlining during response conversion
		c.sourceHeaders = src.Components.Headers

		// ... existing schema/parameter/response/security conversion ...
	}
```

And clean up at the end of the function (before `return nil`):

```go
	c.sourceHeaders = nil
```

**Step 5: Update `convertOAS3ResponseToOAS2` to inline header refs**

In `converter/helpers.go`, update `convertOAS3ResponseToOAS2` (around line 156-158):

Replace `Headers: response.Headers` with header ref resolution:

```go
	converted := &parser.Response{
		Description: response.Description,
	}

	// Resolve header refs — OAS 2.0 has no components.headers
	if len(response.Headers) > 0 {
		converted.Headers = make(map[string]*parser.Header, len(response.Headers))
		for name, header := range response.Headers {
			if header != nil && header.Ref != "" && c.sourceHeaders != nil {
				resolved := c.resolveHeaderRef(header.Ref, result, path)
				if resolved != nil {
					converted.Headers[name] = resolved
					continue
				}
			}
			converted.Headers[name] = header
		}
	}
```

Add the header ref resolution helper:

```go
// resolveHeaderRef resolves a #/components/headers/* ref by inlining the header definition.
func (c *Converter) resolveHeaderRef(ref string, result *ConversionResult, path string) *parser.Header {
	const prefix = "#/components/headers/"
	if !strings.HasPrefix(ref, prefix) {
		return nil
	}

	name := ref[len(prefix):]
	header, ok := c.sourceHeaders[name]
	if !ok {
		c.addIssueWithContext(result, path,
			fmt.Sprintf("Unresolved header ref: %s", ref),
			"Header not found in components.headers")
		return nil
	}

	c.addIssue(result, path,
		fmt.Sprintf("Inlined component header ref %s", ref), SeverityInfo)

	// Return a copy without the ref
	inlined := *header
	inlined.Ref = ""
	return &inlined
}
```

Note: `strings` import may need to be added to helpers.go.

**Step 6: Run test to verify it passes**

Run: `go test ./converter/ -run TestConvertOAS3ToOAS2_InlineHeaderRefs -v`
Expected: PASS

**Step 7: Run full converter test suite**

Run: `go test ./converter/ -v -count=1`
Expected: All tests pass.

**Step 8: Run go_diagnostics**

Run gopls diagnostics on modified files.

**Step 9: Commit**

```
fix(converter): inline component header refs in 3.0→2.0 downconversion

OAS 2.0 has no components.headers section. Header refs like
#/components/headers/X-Rate-Limit are now resolved by looking up
the source OAS 3.x document and inlining the header definition.
```

---

### Task 6: Final verification

**Step 1: Run `make check`**

Run: `make check`
Expected: All checks pass (tests, lint, formatting).

**Step 2: Review all changes**

Run: `git log --oneline main..HEAD`
Expected: 4 commits — design doc, generator fix, generator test, converter formData, converter type fallback, converter header inlining.

**Step 3: Verify test count hasn't decreased**

Run: `go test ./... -count=1 2>&1 | grep -E "^ok|FAIL"`
Expected: All packages pass, no decrease in test count.
