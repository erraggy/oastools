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

// TestPruneOAS2_AnyOfOneOfRefs verifies that schemas referenced via anyOf/oneOf
// composition are not pruned when they appear as map[string]interface{}.
func TestPruneOAS2_AnyOfOneOfRefs(t *testing.T) {
	// Note: OAS 2.0 doesn't officially support anyOf/oneOf, but many parsers allow it
	// and the pruning logic should handle it correctly regardless
	spec := `openapi: "3.0.3"
info:
  title: Shape API
  version: "1.0.0"
paths:
  /shapes:
    get:
      operationId: getShapes
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ShapeUnion'
components:
  schemas:
    ShapeUnion:
      oneOf:
        - $ref: '#/components/schemas/Circle'
        - $ref: '#/components/schemas/Square'
      anyOf:
        - $ref: '#/components/schemas/Triangle'
    Circle:
      type: object
      properties:
        radius:
          type: number
    Square:
      type: object
      properties:
        side:
          type: number
    Triangle:
      type: object
      properties:
        base:
          type: number
        height:
          type: number
    OrphanedHexagon:
      type: object
      description: Not referenced anywhere
      properties:
        sides:
          type: integer
`

	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	fixedDoc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok)
	require.NotNil(t, fixedDoc.Components)
	require.NotNil(t, fixedDoc.Components.Schemas)

	// These schemas should NOT be pruned - they are referenced via oneOf/anyOf
	expectedSchemas := []string{
		"ShapeUnion", // directly referenced in operation response
		"Circle",     // referenced via ShapeUnion.oneOf
		"Square",     // referenced via ShapeUnion.oneOf
		"Triangle",   // referenced via ShapeUnion.anyOf
	}

	for _, name := range expectedSchemas {
		_, exists := fixedDoc.Components.Schemas[name]
		assert.True(t, exists, "schema %q should NOT have been pruned - it is transitively referenced", name)
	}

	// OrphanedHexagon SHOULD be pruned
	_, orphanExists := fixedDoc.Components.Schemas["OrphanedHexagon"]
	assert.False(t, orphanExists, "schema OrphanedHexagon SHOULD have been pruned")
}

// TestIsComponentsEmpty verifies the isComponentsEmpty helper function.
func TestIsComponentsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		comp     *parser.Components
		expected bool
	}{
		{
			name:     "nil components",
			comp:     nil,
			expected: true,
		},
		{
			name:     "empty components",
			comp:     &parser.Components{},
			expected: true,
		},
		{
			name: "only schemas",
			comp: &parser.Components{
				Schemas: map[string]*parser.Schema{"A": {}},
			},
			expected: false,
		},
		{
			name: "only responses",
			comp: &parser.Components{
				Responses: map[string]*parser.Response{"R": {}},
			},
			expected: false,
		},
		{
			name: "only parameters",
			comp: &parser.Components{
				Parameters: map[string]*parser.Parameter{"P": {}},
			},
			expected: false,
		},
		{
			name: "only examples",
			comp: &parser.Components{
				Examples: map[string]*parser.Example{"E": {}},
			},
			expected: false,
		},
		{
			name: "only requestBodies",
			comp: &parser.Components{
				RequestBodies: map[string]*parser.RequestBody{"RB": {}},
			},
			expected: false,
		},
		{
			name: "only headers",
			comp: &parser.Components{
				Headers: map[string]*parser.Header{"H": {}},
			},
			expected: false,
		},
		{
			name: "only securitySchemes",
			comp: &parser.Components{
				SecuritySchemes: map[string]*parser.SecurityScheme{"SS": {}},
			},
			expected: false,
		},
		{
			name: "only links",
			comp: &parser.Components{
				Links: map[string]*parser.Link{"L": {}},
			},
			expected: false,
		},
		{
			name: "only callbacks",
			comp: &parser.Components{
				Callbacks: map[string]*parser.Callback{"CB": {}},
			},
			expected: false,
		},
		{
			name: "only pathItems (OAS 3.1+)",
			comp: &parser.Components{
				PathItems: map[string]*parser.PathItem{"PI": {}},
			},
			expected: false,
		},
		{
			name: "only extra (specification extensions)",
			comp: &parser.Components{
				Extra: map[string]any{"x-custom": "value"},
			},
			expected: false,
		},
		{
			name: "multiple fields populated",
			comp: &parser.Components{
				Schemas:         map[string]*parser.Schema{"A": {}},
				SecuritySchemes: map[string]*parser.SecurityScheme{"SS": {}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isComponentsEmpty(tt.comp)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestPrunePreservesExtensionsInComponents verifies that specification extensions
// (x-* fields) in components prevent the components object from being removed.
func TestPrunePreservesExtensionsInComponents(t *testing.T) {
	spec := `openapi: "3.0.3"
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
  x-custom-extension: "important-value"
  schemas:
    UnusedSchema:
      type: object
`

	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	// Manually add the extension to components since YAML parsing may not preserve it
	doc, ok := parseResult.OAS3Document()
	require.True(t, ok)
	if doc.Components.Extra == nil {
		doc.Components.Extra = make(map[string]any)
	}
	doc.Components.Extra["x-custom-extension"] = "important-value"

	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	fixedDoc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok)

	// Components should be retained because of the x-* extension
	require.NotNil(t, fixedDoc.Components, "components should be retained when x-* extensions exist")
	assert.NotNil(t, fixedDoc.Components.Extra, "Extra field should be preserved")
	assert.Equal(t, "important-value", fixedDoc.Components.Extra["x-custom-extension"])

	// But the unused schema should still be pruned
	assert.Nil(t, fixedDoc.Components.Schemas, "schemas should be nil after pruning all")
}

// TestPruneOAS2_NestedPropertiesRefs verifies that schemas referenced via nested
// properties within a map[string]interface{} are not pruned.
func TestPruneOAS2_NestedPropertiesRefs(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: Nested Properties API
  version: "1.0.0"
basePath: /v1
paths:
  /containers:
    get:
      operationId: getContainers
      responses:
        '200':
          description: OK
          schema:
            $ref: '#/definitions/Container'
definitions:
  Container:
    type: object
    properties:
      metadata:
        type: object
        properties:
          owner:
            $ref: '#/definitions/Owner'
          tags:
            type: array
            items:
              $ref: '#/definitions/Tag'
  Owner:
    type: object
    properties:
      name:
        type: string
  Tag:
    type: object
    properties:
      key:
        type: string
      value:
        type: string
  OrphanedType:
    type: object
    description: Not referenced anywhere
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
	require.NotNil(t, fixedDoc.Definitions)

	// These schemas should NOT be pruned - they are referenced via nested properties
	expectedSchemas := []string{
		"Container", // directly referenced
		"Owner",     // referenced via Container.metadata.properties.owner.$ref
		"Tag",       // referenced via Container.metadata.properties.tags.items.$ref
	}

	for _, name := range expectedSchemas {
		_, exists := fixedDoc.Definitions[name]
		assert.True(t, exists, "schema %q should NOT have been pruned - it is transitively referenced via nested properties", name)
	}

	// OrphanedType SHOULD be pruned
	_, orphanExists := fixedDoc.Definitions["OrphanedType"]
	assert.False(t, orphanExists, "schema OrphanedType SHOULD have been pruned")
}

// TestPruneOAS2_AdditionalItemsRefs verifies that schemas referenced via additionalItems
// are not pruned when additionalItems is parsed as map[string]interface{}.
func TestPruneOAS2_AdditionalItemsRefs(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: AdditionalItems API
  version: "1.0.0"
basePath: /v1
paths:
  /tuples:
    get:
      operationId: getTuples
      responses:
        '200':
          description: OK
          schema:
            $ref: '#/definitions/TupleList'
definitions:
  TupleList:
    type: object
    properties:
      tuples:
        type: array
        additionalItems:
          $ref: '#/definitions/ExtraItem'
  ExtraItem:
    type: object
    properties:
      value:
        type: string
  OrphanedSchema:
    type: object
    description: Not referenced
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
	require.NotNil(t, fixedDoc.Definitions)

	// These schemas should NOT be pruned
	expectedSchemas := []string{
		"TupleList", // directly referenced
		"ExtraItem", // referenced via TupleList.tuples.additionalItems.$ref
	}

	for _, name := range expectedSchemas {
		_, exists := fixedDoc.Definitions[name]
		assert.True(t, exists, "schema %q should NOT have been pruned - it is referenced via additionalItems", name)
	}

	// OrphanedSchema SHOULD be pruned
	_, orphanExists := fixedDoc.Definitions["OrphanedSchema"]
	assert.False(t, orphanExists, "schema OrphanedSchema SHOULD have been pruned")
}

// TestPruneOAS2_InlineParameterItemsRefs verifies that schemas referenced via
// inline parameter schemas with array items are not pruned.
// This is a regression test for the incomplete v1.28.2 fix.
//
// BUG SCENARIO: The v1.28.2 fix only updated prune.go's collectSchemaRefsRecursive()
// to handle map[string]any fallback for Items/AdditionalProperties. However, the
// initial reference collection in refs.go's RefCollector.collectSchemaRefs() was NOT
// updated, causing refs in inline parameter schemas to be missed.
//
// The key difference from other tests is that the schema reference appears in an
// INLINE schema (inside a parameter body), not in a top-level definition. The
// RefCollector traverses inline schemas via collectSchemaRefs(), which was broken.
func TestPruneOAS2_InlineParameterItemsRefs(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: Walrus API
  version: "1.0.0"
basePath: /v1
paths:
  /walrus/aggregates:
    post:
      operationId: QueryWalrusAggregate
      parameters:
        - name: body
          in: body
          schema:
            type: array
            items:
              $ref: '#/definitions/WalrusAggregateQueryRequest'
      responses:
        '200':
          description: OK
          schema:
            $ref: '#/definitions/WalrusAggregatesResponse'
definitions:
  WalrusAggregateQueryRequest:
    type: object
    properties:
      date_ranges:
        type: array
        items:
          $ref: '#/definitions/WalrusDateRangeSpec'
      field:
        type: string
  WalrusDateRangeSpec:
    type: object
    properties:
      from:
        type: string
      to:
        type: string
  WalrusAggregatesResponse:
    type: object
    properties:
      resources:
        type: array
        items:
          $ref: '#/definitions/PelicanAggregatesResponse'
  PelicanAggregatesResponse:
    type: object
    properties:
      count:
        type: integer
  OrphanedSchema:
    type: object
    description: Not referenced anywhere
`

	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	doc, ok := parseResult.OAS2Document()
	require.True(t, ok, "expected OAS 2.0 document")

	// First, verify the bug scenario exists:
	// The inline parameter schema has items as map[string]any, not *parser.Schema
	op := doc.Paths["/walrus/aggregates"].Post
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)

	bodyParam := op.Parameters[0]
	require.NotNil(t, bodyParam.Schema, "body parameter should have inline schema")

	// This is the bug: items in inline schema is map[string]any, not *parser.Schema
	_, isMap := bodyParam.Schema.Items.(map[string]any)
	t.Logf("Inline parameter schema Items type: %T (is map: %v)", bodyParam.Schema.Items, isMap)

	// Now verify the fix: RefCollector should find refs in map[string]any items
	collector := NewRefCollector()
	collector.CollectOAS2(doc)

	schemaRefs := collector.RefsByType[RefTypeSchema]
	t.Logf("Schema refs collected: %v", schemaRefs)

	// The ref to WalrusAggregateQueryRequest via inline items MUST be collected
	walrusRef := "#/definitions/WalrusAggregateQueryRequest"
	require.True(t, schemaRefs[walrusRef],
		"RefCollector MUST find refs in inline parameter schema items (map[string]any) - ref: %s", walrusRef)

	// Apply pruning
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	fixedDoc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok)
	require.NotNil(t, fixedDoc.Definitions)

	// These schemas should NOT be pruned - they are transitively referenced
	// via the INLINE parameter schema's items.$ref
	expectedSchemas := []string{
		"WalrusAggregateQueryRequest", // referenced via inline parameter items.$ref
		"WalrusDateRangeSpec",         // referenced via WalrusAggregateQueryRequest.date_ranges.items.$ref
		"WalrusAggregatesResponse",    // directly referenced in response
		"PelicanAggregatesResponse",   // referenced via WalrusAggregatesResponse.resources.items.$ref
	}

	for _, name := range expectedSchemas {
		_, exists := fixedDoc.Definitions[name]
		assert.True(t, exists, "schema %q should NOT have been pruned - it is transitively referenced via inline parameter schema", name)
	}

	// OrphanedSchema SHOULD be pruned
	_, orphanExists := fixedDoc.Definitions["OrphanedSchema"]
	assert.False(t, orphanExists, "schema OrphanedSchema SHOULD have been pruned - it is not referenced")
}

// TestRefCollector_CollectRefsFromMap_AllPaths exercises the RefCollector's
// collectRefsFromMap method directly for coverage.
func TestRefCollector_CollectRefsFromMap_AllPaths(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected []string
	}{
		{
			name: "direct $ref",
			input: map[string]any{
				"$ref": "#/definitions/User",
			},
			expected: []string{"#/definitions/User"},
		},
		{
			name: "nested properties with $ref",
			input: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"owner": map[string]any{
						"$ref": "#/definitions/Owner",
					},
					"metadata": map[string]any{
						"type": "string",
					},
				},
			},
			expected: []string{"#/definitions/Owner"},
		},
		{
			name: "items with $ref",
			input: map[string]any{
				"type": "array",
				"items": map[string]any{
					"$ref": "#/definitions/Item",
				},
			},
			expected: []string{"#/definitions/Item"},
		},
		{
			name: "additionalProperties with $ref",
			input: map[string]any{
				"type": "object",
				"additionalProperties": map[string]any{
					"$ref": "#/definitions/Value",
				},
			},
			expected: []string{"#/definitions/Value"},
		},
		{
			name: "allOf with $refs",
			input: map[string]any{
				"allOf": []any{
					map[string]any{"$ref": "#/definitions/Base"},
					map[string]any{"$ref": "#/definitions/Extension"},
				},
			},
			expected: []string{"#/definitions/Base", "#/definitions/Extension"},
		},
		{
			name: "anyOf with $refs",
			input: map[string]any{
				"anyOf": []any{
					map[string]any{"$ref": "#/definitions/TypeA"},
					map[string]any{"$ref": "#/definitions/TypeB"},
				},
			},
			expected: []string{"#/definitions/TypeA", "#/definitions/TypeB"},
		},
		{
			name: "oneOf with $refs",
			input: map[string]any{
				"oneOf": []any{
					map[string]any{"$ref": "#/definitions/Option1"},
					map[string]any{"$ref": "#/definitions/Option2"},
				},
			},
			expected: []string{"#/definitions/Option1", "#/definitions/Option2"},
		},
		{
			name: "no refs",
			input: map[string]any{
				"type": "string",
			},
			expected: nil,
		},
		{
			name: "empty $ref is ignored",
			input: map[string]any{
				"$ref": "",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewRefCollector()
			collector.collectRefsFromMap(tt.input, "test.path")

			schemaRefs := collector.RefsByType[RefTypeSchema]
			if tt.expected == nil {
				assert.Empty(t, schemaRefs)
			} else {
				for _, expectedRef := range tt.expected {
					assert.True(t, schemaRefs[expectedRef], "expected ref %q to be collected", expectedRef)
				}
			}
		})
	}
}

// TestCollectRefsFromMap_AllPaths exercises all code paths in collectRefsFromMap directly.
func TestCollectRefsFromMap_AllPaths(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		prefix   string
		expected []string
	}{
		{
			name: "direct $ref",
			input: map[string]any{
				"$ref": "#/definitions/User",
			},
			prefix:   "#/definitions/",
			expected: []string{"User"},
		},
		{
			name: "nested properties with $ref",
			input: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"owner": map[string]any{
						"$ref": "#/definitions/Owner",
					},
					"metadata": map[string]any{
						"type": "string",
					},
				},
			},
			prefix:   "#/definitions/",
			expected: []string{"Owner"},
		},
		{
			name: "items with $ref",
			input: map[string]any{
				"type": "array",
				"items": map[string]any{
					"$ref": "#/definitions/Item",
				},
			},
			prefix:   "#/definitions/",
			expected: []string{"Item"},
		},
		{
			name: "additionalProperties with $ref",
			input: map[string]any{
				"type": "object",
				"additionalProperties": map[string]any{
					"$ref": "#/definitions/Value",
				},
			},
			prefix:   "#/definitions/",
			expected: []string{"Value"},
		},
		{
			name: "allOf with $refs",
			input: map[string]any{
				"allOf": []any{
					map[string]any{"$ref": "#/definitions/Base"},
					map[string]any{"$ref": "#/definitions/Extension"},
				},
			},
			prefix:   "#/definitions/",
			expected: []string{"Base", "Extension"},
		},
		{
			name: "anyOf with $refs",
			input: map[string]any{
				"anyOf": []any{
					map[string]any{"$ref": "#/definitions/TypeA"},
					map[string]any{"$ref": "#/definitions/TypeB"},
				},
			},
			prefix:   "#/definitions/",
			expected: []string{"TypeA", "TypeB"},
		},
		{
			name: "oneOf with $refs",
			input: map[string]any{
				"oneOf": []any{
					map[string]any{"$ref": "#/definitions/Option1"},
					map[string]any{"$ref": "#/definitions/Option2"},
				},
			},
			prefix:   "#/definitions/",
			expected: []string{"Option1", "Option2"},
		},
		{
			name: "no refs",
			input: map[string]any{
				"type": "string",
			},
			prefix:   "#/definitions/",
			expected: nil,
		},
		{
			name: "OAS3 prefix",
			input: map[string]any{
				"$ref": "#/components/schemas/Pet",
			},
			prefix:   "#/components/schemas/",
			expected: []string{"Pet"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectRefsFromMap(tt.input, tt.prefix)
			if tt.expected == nil {
				assert.Empty(t, result)
			} else {
				assert.ElementsMatch(t, tt.expected, result)
			}
		})
	}
}
