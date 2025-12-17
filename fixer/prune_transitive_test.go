// prune_transitive_test.go tests the transitive reference tracking in schema pruning.
// These tests verify that schemas referenced via nested structures (items, properties, etc.)
// are not incorrectly pruned.
package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPruneOAS2_TransitiveItemsRefs verifies that schemas referenced via array items
// are not pruned. This is a regression test for a bug where Items fields parsed as
// map[string]interface{} instead of *parser.Schema, causing the type assertion in
// collectSchemaRefsRecursive to silently fail.
//
// Bug: When parsing OAS 2.0, schema.Items with a $ref is unmarshaled as map[string]any
// instead of *parser.Schema. The pruning code does:
//
//	if items, ok := schema.Items.(*parser.Schema); ok { ... }
//
// This type assertion fails, so nested refs in items are never followed.
func TestPruneOAS2_TransitiveItemsRefs(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: Walrus API
  version: "1.0.0"
basePath: /v1
paths:
  /pelicans:
    post:
      operationId: createPelican
      tags:
        - pelicans
      parameters:
        - name: body
          in: body
          required: true
          schema:
            $ref: '#/definitions/PelicanRequest'
      responses:
        '200':
          description: OK
          schema:
            $ref: '#/definitions/PelicanResponse'
definitions:
  PelicanRequest:
    type: object
    description: Request to create a pelican
    properties:
      feathers:
        type: array
        items:
          $ref: '#/definitions/Feather'
      beak_size:
        type: string
  PelicanResponse:
    type: object
    description: Response containing pelican data
    properties:
      pelicans:
        type: array
        items:
          $ref: '#/definitions/Pelican'
      metadata:
        $ref: '#/definitions/Metadata'
  Pelican:
    type: object
    properties:
      id:
        type: string
      name:
        type: string
  Feather:
    type: object
    properties:
      color:
        type: string
      length:
        type: integer
  Metadata:
    type: object
    properties:
      total:
        type: integer
      page:
        type: integer
  OrphanedMarmot:
    type: object
    description: This schema is not referenced anywhere and should be pruned
    properties:
      whiskers:
        type: string
`

	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	_, ok := parseResult.OAS2Document()
	require.True(t, ok, "expected OAS 2.0 document")

	// Apply pruning
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	fixedDoc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok)
	require.NotNil(t, fixedDoc.Definitions)

	// These schemas should NOT be pruned - they are transitively referenced
	expectedSchemas := []string{
		"PelicanRequest",  // directly referenced in operation parameter
		"PelicanResponse", // directly referenced in operation response
		"Pelican",         // referenced via PelicanResponse.pelicans.items.$ref
		"Feather",         // referenced via PelicanRequest.feathers.items.$ref
		"Metadata",        // referenced via PelicanResponse.metadata.$ref
	}

	for _, name := range expectedSchemas {
		_, exists := fixedDoc.Definitions[name]
		assert.True(t, exists, "schema %q should NOT have been pruned - it is transitively referenced", name)
	}

	// OrphanedMarmot SHOULD be pruned
	_, orphanExists := fixedDoc.Definitions["OrphanedMarmot"]
	assert.False(t, orphanExists, "schema OrphanedMarmot SHOULD have been pruned - it is not referenced")
}

// TestPruneOAS2_DeeplyNestedItemsRefs verifies that schemas referenced via deeply
// nested array items chains are not pruned.
func TestPruneOAS2_DeeplyNestedItemsRefs(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: Badger API
  version: "1.0.0"
basePath: /v1
paths:
  /burrows:
    get:
      operationId: listBurrows
      tags:
        - burrows
      responses:
        '200':
          description: OK
          schema:
            $ref: '#/definitions/BurrowList'
definitions:
  BurrowList:
    type: object
    properties:
      burrows:
        type: array
        items:
          $ref: '#/definitions/Burrow'
  Burrow:
    type: object
    properties:
      tunnels:
        type: array
        items:
          $ref: '#/definitions/Tunnel'
  Tunnel:
    type: object
    properties:
      chambers:
        type: array
        items:
          $ref: '#/definitions/Chamber'
  Chamber:
    type: object
    properties:
      contents:
        type: array
        items:
          $ref: '#/definitions/Acorn'
  Acorn:
    type: object
    properties:
      species:
        type: string
      weight:
        type: number
`

	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	fixedDoc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok)

	// All schemas should be kept - they form a transitive chain
	expectedSchemas := []string{"BurrowList", "Burrow", "Tunnel", "Chamber", "Acorn"}
	for _, name := range expectedSchemas {
		_, exists := fixedDoc.Definitions[name]
		assert.True(t, exists, "schema %q should NOT have been pruned - it is part of transitive chain", name)
	}
}

// TestPruneOAS3_TransitiveItemsRefs verifies the same issue in OAS 3.x documents.
func TestPruneOAS3_TransitiveItemsRefs(t *testing.T) {
	spec := `openapi: "3.0.3"
info:
  title: Otter API
  version: "1.0.0"
paths:
  /otters:
    post:
      operationId: createOtter
      tags:
        - otters
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/OtterRequest'
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/OtterResponse'
components:
  schemas:
    OtterRequest:
      type: object
      properties:
        paws:
          type: array
          items:
            $ref: '#/components/schemas/Paw'
        tail_length:
          type: integer
    OtterResponse:
      type: object
      properties:
        otters:
          type: array
          items:
            $ref: '#/components/schemas/Otter'
        stats:
          $ref: '#/components/schemas/Statistics'
    Otter:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
    Paw:
      type: object
      properties:
        webbed:
          type: boolean
        claws:
          type: integer
    Statistics:
      type: object
      properties:
        count:
          type: integer
    OrphanedSeagull:
      type: object
      description: Not referenced anywhere
      properties:
        wingspan:
          type: number
`

	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	doc, ok := parseResult.OAS3Document()
	require.True(t, ok, "expected OAS 3.x document")
	require.NotNil(t, doc.Components)

	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	fixedDoc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok)
	require.NotNil(t, fixedDoc.Components)
	require.NotNil(t, fixedDoc.Components.Schemas)

	// These schemas should NOT be pruned
	expectedSchemas := []string{
		"OtterRequest",
		"OtterResponse",
		"Otter",
		"Paw",
		"Statistics",
	}

	for _, name := range expectedSchemas {
		_, exists := fixedDoc.Components.Schemas[name]
		assert.True(t, exists, "schema %q should NOT have been pruned - it is transitively referenced", name)
	}

	// OrphanedSeagull SHOULD be pruned
	_, orphanExists := fixedDoc.Components.Schemas["OrphanedSeagull"]
	assert.False(t, orphanExists, "schema OrphanedSeagull SHOULD have been pruned")
}

// TestPruneOAS2_AdditionalPropertiesRefs verifies that schemas referenced via
// additionalProperties are not pruned.
func TestPruneOAS2_AdditionalPropertiesRefs(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: Hedgehog API
  version: "1.0.0"
basePath: /v1
paths:
  /hedgehogs:
    get:
      operationId: getHedgehogs
      responses:
        '200':
          description: OK
          schema:
            $ref: '#/definitions/HedgehogMap'
definitions:
  HedgehogMap:
    type: object
    additionalProperties:
      $ref: '#/definitions/Hedgehog'
  Hedgehog:
    type: object
    properties:
      spines:
        type: integer
      habitat:
        $ref: '#/definitions/Habitat'
  Habitat:
    type: object
    properties:
      terrain:
        type: string
      temperature:
        type: number
`

	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	fixedDoc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok)

	// All schemas should be kept
	for _, name := range []string{"HedgehogMap", "Hedgehog", "Habitat"} {
		_, exists := fixedDoc.Definitions[name]
		assert.True(t, exists, "schema %q should NOT have been pruned", name)
	}
}

// TestPruneOAS2_AllOfRefs verifies that schemas referenced via allOf composition
// are not pruned.
func TestPruneOAS2_AllOfRefs(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: Salamander API
  version: "1.0.0"
basePath: /v1
paths:
  /salamanders:
    get:
      operationId: getSalamanders
      responses:
        '200':
          description: OK
          schema:
            $ref: '#/definitions/FireSalamander'
definitions:
  FireSalamander:
    allOf:
      - $ref: '#/definitions/Amphibian'
      - $ref: '#/definitions/FireBreather'
      - type: object
        properties:
          spots:
            type: integer
  Amphibian:
    type: object
    properties:
      moisture_level:
        type: number
      gills:
        type: boolean
  FireBreather:
    type: object
    properties:
      flame_color:
        type: string
      temperature:
        type: integer
`

	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	fixedDoc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok)

	// All schemas should be kept - allOf creates transitive references
	for _, name := range []string{"FireSalamander", "Amphibian", "FireBreather"} {
		_, exists := fixedDoc.Definitions[name]
		assert.True(t, exists, "schema %q should NOT have been pruned - referenced via allOf", name)
	}
}

// TestCollectSchemaRefs_ItemsAsMap demonstrates the root cause: Items parsed as map
// instead of *parser.Schema.
func TestCollectSchemaRefs_ItemsAsMap(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: Wombat API
  version: "1.0.0"
basePath: /v1
paths:
  /wombats:
    get:
      operationId: listWombats
      responses:
        '200':
          description: OK
          schema:
            $ref: '#/definitions/WombatList'
definitions:
  WombatList:
    type: object
    properties:
      wombats:
        type: array
        items:
          $ref: '#/definitions/Wombat'
  Wombat:
    type: object
    properties:
      name:
        type: string
`

	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	doc, ok := parseResult.OAS2Document()
	require.True(t, ok)

	// Verify the bug: Items is parsed as map[string]interface{}, not *parser.Schema
	wombatListSchema := doc.Definitions["WombatList"]
	require.NotNil(t, wombatListSchema)

	wombatsProperty := wombatListSchema.Properties["wombats"]
	require.NotNil(t, wombatsProperty)
	require.NotNil(t, wombatsProperty.Items, "Items should not be nil")

	// This is the bug: Items should be *parser.Schema but is actually map[string]interface{}
	_, isSchema := wombatsProperty.Items.(*parser.Schema)
	_, isMap := wombatsProperty.Items.(map[string]interface{})

	// Document the current (buggy) behavior
	t.Logf("Items type: %T", wombatsProperty.Items)
	t.Logf("Items is *parser.Schema: %v", isSchema)
	t.Logf("Items is map[string]interface{}: %v", isMap)

	// This assertion documents the bug - when fixed, this should change
	if isMap && !isSchema {
		t.Log("BUG CONFIRMED: Items with $ref is parsed as map[string]interface{} instead of *parser.Schema")
		t.Log("This causes collectSchemaRefsRecursive to miss nested refs")
	}

	// Now verify the pruning behavior is broken
	collector := NewRefCollector()
	collector.CollectOAS2(doc)

	// Check what refs were collected
	schemaRefs := collector.RefsByType[RefTypeSchema]
	t.Logf("Schema refs collected: %v", schemaRefs)

	// The ref to Wombat via items should be collected, but isn't due to the bug
	wombatRef := "#/definitions/Wombat"
	if !schemaRefs[wombatRef] {
		t.Logf("BUG CONFIRMED: Ref %q was NOT collected from WombatList.wombats.items", wombatRef)
	}
}
