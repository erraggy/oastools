package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Pruning Tests
// =============================================================================

// TestPruneUnusedSchemasOAS3 tests removing orphaned schemas in OAS 3.x
func TestPruneUnusedSchemasOAS3(t *testing.T) {
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
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
    OrphanedSchema:
      type: object
      properties:
        unused:
          type: string
    AnotherOrphan:
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
	assert.Equal(t, 2, result.FixCount) // 2 orphaned schemas removed

	// User should remain (referenced)
	assert.Contains(t, doc.Components.Schemas, "User")

	// Orphaned schemas should be removed
	assert.NotContains(t, doc.Components.Schemas, "OrphanedSchema")
	assert.NotContains(t, doc.Components.Schemas, "AnotherOrphan")
}

// TestPruneUnusedSchemasOAS2 tests removing orphaned schemas in OAS 2.0
func TestPruneUnusedSchemasOAS2(t *testing.T) {
	spec := `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      produces:
        - application/json
      responses:
        "200":
          description: Success
          schema:
            $ref: "#/definitions/User"
definitions:
  User:
    type: object
    properties:
      id:
        type: integer
  UnusedDefinition:
    type: object
    properties:
      orphan:
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
	doc := result.Document.(*parser.OAS2Document)
	assert.Equal(t, 1, result.FixCount)

	// User should remain (referenced)
	assert.Contains(t, doc.Definitions, "User")

	// Orphaned definition should be removed
	assert.NotContains(t, doc.Definitions, "UnusedDefinition")
}

// TestPruneTransitiveReferencesPreserved tests that transitive refs are preserved
func TestPruneTransitiveReferencesPreserved(t *testing.T) {
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
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/UserResponse"
components:
  schemas:
    UserResponse:
      type: object
      properties:
        user:
          $ref: "#/components/schemas/User"
    User:
      type: object
      properties:
        address:
          $ref: "#/components/schemas/Address"
    Address:
      type: object
      properties:
        city:
          type: string
    Orphan:
      type: object
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
	assert.Equal(t, 1, result.FixCount) // Only Orphan removed

	// All transitively referenced schemas should remain
	assert.Contains(t, doc.Components.Schemas, "UserResponse")
	assert.Contains(t, doc.Components.Schemas, "User")
	assert.Contains(t, doc.Components.Schemas, "Address")

	// Orphan should be removed
	assert.NotContains(t, doc.Components.Schemas, "Orphan")
}

// TestPruneCircularReferencesHandled tests that circular refs don't cause infinite loops
func TestPruneCircularReferencesHandled(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /nodes:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Node"
components:
  schemas:
    Node:
      type: object
      properties:
        children:
          type: array
          items:
            $ref: "#/components/schemas/Node"
        parent:
          $ref: "#/components/schemas/Node"
    Orphan:
      type: object
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix with pruning enabled - should not hang on circular refs
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, 1, result.FixCount) // Only Orphan removed

	// Node (with circular refs) should remain
	assert.Contains(t, doc.Components.Schemas, "Node")

	// Orphan should be removed
	assert.NotContains(t, doc.Components.Schemas, "Orphan")
}

// TestPruneEmptyPaths tests removing paths with no operations
func TestPruneEmptyPaths(t *testing.T) {
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
          description: Success
  /empty:
    parameters:
      - name: id
        in: query
        schema:
          type: string
  /also-empty: {}
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix with path pruning enabled
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedEmptyPath}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, 2, result.FixCount) // Two empty paths removed

	// /users should remain (has operations)
	assert.Contains(t, doc.Paths, "/users")

	// Empty paths should be removed
	assert.NotContains(t, doc.Paths, "/empty")
	assert.NotContains(t, doc.Paths, "/also-empty")
}

// TestPruneEmptyPathsOAS2 tests removing empty paths in OAS 2.0
func TestPruneEmptyPathsOAS2(t *testing.T) {
	spec := `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /items:
    get:
      operationId: getItems
      responses:
        "200":
          description: Success
  /empty-path:
    parameters:
      - name: filter
        in: query
        type: string
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix with path pruning enabled
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedEmptyPath}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS2Document)
	assert.Equal(t, 1, result.FixCount)

	// /items should remain
	assert.Contains(t, doc.Paths, "/items")

	// Empty path should be removed
	assert.NotContains(t, doc.Paths, "/empty-path")
}

// TestPruneAllSchemasWhenNoneReferenced tests that components becomes nil when all schemas are pruned
func TestPruneAllSchemasWhenNoneReferenced(t *testing.T) {
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
    UnusedSchema:
      type: object
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, 1, result.FixCount)

	// Components should be nil when all schemas are pruned (and no other components exist)
	assert.Nil(t, doc.Components)
}

// TestPrunePartialSchemasKeepsComponents tests that components is retained when some schemas remain
func TestPrunePartialSchemasKeepsComponents(t *testing.T) {
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
                $ref: '#/components/schemas/User'
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
    UnusedSchema:
      type: object
      properties:
        unused:
          type: string
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, 1, result.FixCount)

	// Components should still exist with the User schema
	require.NotNil(t, doc.Components)
	require.NotNil(t, doc.Components.Schemas)
	assert.Contains(t, doc.Components.Schemas, "User")
	assert.NotContains(t, doc.Components.Schemas, "UnusedSchema")
}

// TestPruneWithNilComponents tests pruning when components is nil
func TestPruneWithNilComponents(t *testing.T) {
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
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix - should not panic with nil components
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert - no fixes since no schemas
	assert.Equal(t, 0, result.FixCount)
}

// TestPruneAllSchemasButKeepOtherComponents tests that components is retained when schemas
// are all pruned but other component fields exist (e.g., securitySchemes)
func TestPruneAllSchemasButKeepOtherComponents(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /health:
    get:
      security:
        - bearerAuth: []
      responses:
        "200":
          description: OK
components:
  schemas:
    UnusedSchema:
      type: object
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix - prune unused schemas
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, 1, result.FixCount, "should prune 1 unused schema")

	// Components should still exist (has securitySchemes)
	require.NotNil(t, doc.Components, "components should be retained when securitySchemes exist")
	assert.Nil(t, doc.Components.Schemas, "schemas should be nil after pruning all")
	assert.NotNil(t, doc.Components.SecuritySchemes, "securitySchemes should be retained")
	assert.Contains(t, doc.Components.SecuritySchemes, "bearerAuth")
}
