package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOAS2JsonWithAllFixerOptions tests the exact use case from issue #149:
// OAS 2.0 in JSON format with --infer, --prune-all, generic naming, and missing path params
func TestOAS2JsonWithAllFixerOptions(t *testing.T) {
	// JSON format (not YAML) with OAS 2.0
	specJSON := `{
  "swagger": "2.0",
  "info": {
    "title": "Test API",
    "version": "1.0"
  },
  "paths": {
    "/users/{userId}/posts/{postId}": {
      "get": {
        "operationId": "getUserPost",
        "produces": ["application/json"],
        "responses": {
          "200": {
            "description": "Success",
            "schema": {
              "$ref": "#/definitions/Response[Post]"
            }
          }
        }
      }
    }
  },
  "definitions": {
    "Response[Post]": {
      "type": "object",
      "properties": {
        "data": {
          "$ref": "#/definitions/Post"
        }
      }
    },
    "Post": {
      "type": "object",
      "properties": {
        "id": {
          "type": "integer"
        },
        "title": {
          "type": "string"
        }
      }
    },
    "UnusedSchema": {
      "type": "object",
      "properties": {
        "orphan": {
          "type": "string"
        }
      }
    }
  }
}`

	// Parse JSON
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(specJSON)))
	require.NoError(t, err)
	assert.Equal(t, parser.SourceFormatJSON, parseResult.SourceFormat, "should detect JSON format")

	// Fix with ALL options enabled (matching the user's use case)
	f := New()
	f.InferTypes = true // --infer flag
	f.EnabledFixes = []FixType{
		FixTypeMissingPathParameter, // fix missing params
		FixTypeRenamedGenericSchema, // generic naming
		FixTypePrunedUnusedSchema,   // --prune-all
		FixTypePrunedEmptyPath,      // --prune-all
	}
	f.GenericNamingConfig.Strategy = GenericNamingOf // _of_ strategy

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert - should have multiple fixes
	assert.True(t, result.HasFixes(), "should have applied fixes")

	// Verify all fix types were applied
	fixTypes := make(map[FixType]int)
	for _, fix := range result.Fixes {
		fixTypes[fix.Type]++
	}

	// Should have fixed:
	// 1. Missing path parameters (userId, postId)
	assert.Equal(t, 2, fixTypes[FixTypeMissingPathParameter], "should fix 2 missing path params")

	// 2. Renamed generic schema name (Response[Post] -> ResponseOfPost)
	assert.Equal(t, 1, fixTypes[FixTypeRenamedGenericSchema], "should rename 1 generic schema")

	// 3. Pruned unused schema (UnusedSchema)
	assert.Equal(t, 1, fixTypes[FixTypePrunedUnusedSchema], "should prune 1 unused schema")

	// Verify the fixed document
	doc := result.Document.(*parser.OAS2Document)

	// Check that generic schema was renamed
	assert.Contains(t, doc.Definitions, "ResponseOfPost", "should have renamed schema")
	assert.NotContains(t, doc.Definitions, "Response[Post]", "should not have original generic name")

	// Check that unused schema was removed
	assert.NotContains(t, doc.Definitions, "UnusedSchema", "should have pruned unused schema")

	// Check that used schemas remain
	assert.Contains(t, doc.Definitions, "Post", "should keep referenced schema")

	// Check that path parameters were added with inferred types
	pathItem := doc.Paths["/users/{userId}/posts/{postId}"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Get)
	require.Len(t, pathItem.Get.Parameters, 2, "should have 2 path parameters")

	// Find the parameters by name
	paramsByName := make(map[string]*parser.Parameter)
	for _, p := range pathItem.Get.Parameters {
		paramsByName[p.Name] = p
	}

	// userId should be inferred as integer (--infer flag)
	require.Contains(t, paramsByName, "userId")
	assert.Equal(t, "integer", paramsByName["userId"].Type, "userId should be inferred as integer")
	assert.Equal(t, "path", paramsByName["userId"].In)
	assert.True(t, paramsByName["userId"].Required)

	// postId should be inferred as integer (--infer flag)
	require.Contains(t, paramsByName, "postId")
	assert.Equal(t, "integer", paramsByName["postId"].Type, "postId should be inferred as integer")
	assert.Equal(t, "path", paramsByName["postId"].In)
	assert.True(t, paramsByName["postId"].Required)

	// Check that the ref was rewritten to the new name
	resp200 := pathItem.Get.Responses.Codes["200"]
	assert.Equal(t, "#/definitions/ResponseOfPost", resp200.Schema.Ref,
		"ref should be rewritten to new schema name")

	// Verify the output would be JSON (not YAML)
	assert.Equal(t, parser.SourceFormatJSON, result.SourceFormat,
		"should preserve JSON format for output")
}

// BenchmarkFix benchmarks fixing a spec with missing path parameters
func BenchmarkFix(b *testing.B) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
  /projects/{projectId}/tasks/{taskId}:
    get:
      operationId: getTask
      responses:
        '200':
          description: Success
    put:
      operationId: updateTask
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		f := New()
		_, err := f.FixParsed(*parseResult)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Note: BenchmarkCorpus_Fix has been moved to corpus_bench_test.go
// Run with: go test -tags=corpus -bench=BenchmarkCorpus ./fixer/...
