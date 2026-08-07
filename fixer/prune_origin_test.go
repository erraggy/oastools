package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPruneIsolatedSchemaIsland(t *testing.T) {
	// Scenario: Schema A references Schema B.
	// Neither A nor B is referenced from any entry point (paths, etc.).
	// Before the fix, A might have been considered "referenced" because it's in the collector's
	// RefsByType[RefTypeSchema] map, and B would be kept because A refs it.
	// Now, since A's only origin is "components.schemas.A", it should be skipped in seeding.

	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /health:
    get:
      responses:
        "200":
          description: OK
components:
  schemas:
    SchemaA:
      $ref: "#/components/schemas/SchemaB"
    SchemaB:
      type: string
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix with pruning enabled
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, 2, result.FixCount, "Both SchemaA and SchemaB should be pruned")

	if doc.Components != nil {
		assert.NotContains(t, doc.Components.Schemas, "SchemaA")
		assert.NotContains(t, doc.Components.Schemas, "SchemaB")
	}
}

func TestPruneSchemaRefFromEntrypoint(t *testing.T) {
	// Scenario: Schema A is referenced from a Path.
	// Schema A references Schema B.
	// Both should be kept.

	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/SchemaA"
components:
  schemas:
    SchemaA:
      $ref: "#/components/schemas/SchemaB"
    SchemaB:
      type: string
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix with pruning enabled
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, 0, result.FixCount, "No schemas should be pruned")

	require.NotNil(t, doc.Components)
	assert.Contains(t, doc.Components.Schemas, "SchemaA")
	assert.Contains(t, doc.Components.Schemas, "SchemaB")
}

func TestPruneOAS2_Issue_474_Regression(t *testing.T) {
	// Issue #474 describes this spec in its reproduction case
	// This spec contains only a single _referenced_ schema (Pet),
	// but it contains 2 additional schemas that only reference
	// one another: Order<=>Customer
	spec := `{
  "swagger": "2.0",
  "info": {"title": "Petstore", "version": "1.0.0"},
  "paths": {
    "/pets": {
      "get": {
        "operationId": "listPets",
        "responses": {
          "200": {"description": "OK", "schema": {"$ref": "#/definitions/Pet"}}
        }
      }
    }
  },
  "definitions": {
    "Pet": {
      "type": "object",
      "properties": {"name": {"type": "string"}}
    },
    "Order": {
      "type": "object",
      "properties": {"customer": {"$ref": "#/definitions/Customer"}}
    },
    "Customer": {
      "type": "object",
      "properties": {"lastOrder": {"$ref": "#/definitions/Order"}}
    },
    "Tag": {
      "type": "object",
      "properties": {"label": {"type": "string"}}
    }
  }
}`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	require.True(t, parseResult.IsOAS2(), "expected OAS 2.0 document")

	// Apply pruning
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	f.MutableInput = true

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	schemas := result.ToParseResult().AsAccessor().GetSchemas()

	require.Len(t, schemas, 1)
	require.Contains(t, schemas, "Pet")
}
