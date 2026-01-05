package joiner

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Integration Tests for Operation-Aware Renaming
// ============================================================================

// TestJoinWithOperationContext_OperationID tests that operation context renaming
// uses the operationId field to differentiate colliding schemas.
func TestJoinWithOperationContext_OperationID(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	// Join two specs with colliding "Response" schema
	result, err := JoinWithOptions(
		WithFilePaths(
			filepath.Join(testdataDir, "join-operation-context-users-3.0.yaml"),
			filepath.Join(testdataDir, "join-operation-context-orders-3.0.yaml"),
		),
		WithOperationContext(true),
		WithSchemaStrategy(StrategyRenameRight),
		WithRenameTemplate("{{.Name}}_{{.OperationID}}"),
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3 document")
	require.NotNil(t, doc.Components)
	require.NotNil(t, doc.Components.Schemas)

	// Verify original Response schema from first doc exists
	assert.NotNil(t, doc.Components.Schemas["Response"],
		"expected original 'Response' schema from first document")

	// Verify renamed Response from second doc uses operationId
	assert.NotNil(t, doc.Components.Schemas["Response_createOrder"],
		"expected renamed 'Response_createOrder' schema from second document")

	// Verify OrderRequest schema was not renamed (no collision)
	assert.NotNil(t, doc.Components.Schemas["OrderRequest"],
		"expected 'OrderRequest' schema to be present")

	// Verify we have paths from both documents
	assert.NotNil(t, doc.Paths["/users"], "expected /users path")
	assert.NotNil(t, doc.Paths["/users/{id}"], "expected /users/{id} path")
	assert.NotNil(t, doc.Paths["/orders"], "expected /orders path")

	// Verify collision count
	assert.Equal(t, 1, result.CollisionCount, "expected 1 collision (Response schema)")
}

// TestJoinWithOperationContext_PathResource tests that operation context renaming
// uses the path resource extraction to differentiate colliding schemas.
func TestJoinWithOperationContext_PathResource(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	// Join two specs with colliding "Response" schema
	result, err := JoinWithOptions(
		WithFilePaths(
			filepath.Join(testdataDir, "join-operation-context-users-3.0.yaml"),
			filepath.Join(testdataDir, "join-operation-context-orders-3.0.yaml"),
		),
		WithOperationContext(true),
		WithSchemaStrategy(StrategyRenameRight),
		WithRenameTemplate("{{pathResource .Path | pascalCase}}{{.Name}}"),
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3 document")
	require.NotNil(t, doc.Components)

	// Verify original Response schema from first doc exists
	assert.NotNil(t, doc.Components.Schemas["Response"],
		"expected original 'Response' schema from first document")

	// Verify renamed Response from second doc uses path resource
	assert.NotNil(t, doc.Components.Schemas["OrdersResponse"],
		"expected renamed 'OrdersResponse' schema from second document")
}

// TestJoinWithOperationContext_DeepRefs tests that schemas referenced through
// a chain of references inherit operation context from the originating operation.
func TestJoinWithOperationContext_DeepRefs(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	// Join two specs where ItemDetails is referenced through a chain:
	// Document 1: Operation -> ItemList -> Item -> ItemDetails
	// Document 2: Operation -> ProductList -> Product -> ItemDetails (collision)
	// The deep ref chain should be traced to resolve operation context.
	result, err := JoinWithOptions(
		WithFilePaths(
			filepath.Join(testdataDir, "join-deep-refs-3.0.yaml"),
			filepath.Join(testdataDir, "join-deep-refs-ext-3.0.yaml"),
		),
		WithOperationContext(true),
		WithSchemaStrategy(StrategyRenameRight),
		// Use coalesce to fall back through operation context fields
		WithRenameTemplate("{{.Name}}_{{coalesce .OperationID .PrimaryResource .Source}}"),
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3 document")
	require.NotNil(t, doc.Components)

	// Log all schema names for debugging
	schemaNames := make([]string, 0, len(doc.Components.Schemas))
	for name := range doc.Components.Schemas {
		schemaNames = append(schemaNames, name)
	}
	t.Logf("Generated schemas: %v", schemaNames)

	// Verify original ItemDetails from first doc exists
	assert.NotNil(t, doc.Components.Schemas["ItemDetails"],
		"expected original 'ItemDetails' schema from first document")

	// Verify renamed ItemDetails from second doc
	// The chain is: listProducts -> ProductList -> Product -> ItemDetails
	// With deep ref tracing, should use listProducts as OperationID
	// If OperationID is empty, coalesce falls back to PrimaryResource ("products")
	// or finally Source ("join_deep_refs_ext_3_0")
	foundRenamedItemDetails := false
	var renamedItemDetailsName string
	for name := range doc.Components.Schemas {
		if strings.HasPrefix(name, "ItemDetails_") && name != "ItemDetails" {
			foundRenamedItemDetails = true
			renamedItemDetailsName = name
			break
		}
	}
	require.True(t, foundRenamedItemDetails,
		"expected renamed ItemDetails schema with operation context")
	t.Logf("Renamed ItemDetails schema: %s", renamedItemDetailsName)

	// The renamed schema should use context from the deep ref chain
	// Could be ItemDetails_listProducts, ItemDetails_products, or ItemDetails_join_deep_refs_ext_3_0
	assert.True(t,
		renamedItemDetailsName == "ItemDetails_listProducts" ||
			renamedItemDetailsName == "ItemDetails_products" ||
			strings.HasPrefix(renamedItemDetailsName, "ItemDetails_join"),
		"expected renamed schema to use operation context from deep ref chain, got: %s", renamedItemDetailsName)

	// Verify the chain schemas are present
	assert.NotNil(t, doc.Components.Schemas["ItemList"], "expected ItemList schema")
	assert.NotNil(t, doc.Components.Schemas["Item"], "expected Item schema")
	assert.NotNil(t, doc.Components.Schemas["ProductList"], "expected ProductList schema")
	assert.NotNil(t, doc.Components.Schemas["Product"], "expected Product schema")

	// Verify we have paths from both documents
	assert.NotNil(t, doc.Paths["/items"], "expected /items path")
	assert.NotNil(t, doc.Paths["/products"], "expected /products path")
}

// TestJoinWithOperationContext_BackwardCompatible tests that existing behavior
// is preserved when WithOperationContext is false (the default).
func TestJoinWithOperationContext_BackwardCompatible(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	// Join with operation context disabled (default)
	result, err := JoinWithOptions(
		WithFilePaths(
			filepath.Join(testdataDir, "join-operation-context-users-3.0.yaml"),
			filepath.Join(testdataDir, "join-operation-context-orders-3.0.yaml"),
		),
		WithOperationContext(false),
		WithSchemaStrategy(StrategyRenameRight),
		WithRenameTemplate("{{.Name}}_{{.Source}}"),
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3 document")
	require.NotNil(t, doc.Components)

	// Verify original Response schema from first doc exists
	assert.NotNil(t, doc.Components.Schemas["Response"],
		"expected original 'Response' schema")

	// Verify renamed Response uses source file name (backward compatible behavior)
	// Should be Response_join_operation_context_orders_3_0 (sanitized source name)
	foundRenamed := false
	for name := range doc.Components.Schemas {
		if strings.HasPrefix(name, "Response_join") {
			foundRenamed = true
			break
		}
	}
	assert.True(t, foundRenamed, "expected renamed schema with source file pattern")
}

// TestJoinWithOperationContext_MultiOperationSchema tests schemas that are
// referenced by multiple operations and the policy selection.
func TestJoinWithOperationContext_MultiOperationSchema(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	// The Users API has Response referenced by both listUsers and getUser
	t.Run("PolicyFirstEncountered", func(t *testing.T) {
		result, err := JoinWithOptions(
			WithFilePaths(
				filepath.Join(testdataDir, "join-operation-context-users-3.0.yaml"),
				filepath.Join(testdataDir, "join-operation-context-orders-3.0.yaml"),
			),
			WithOperationContext(true),
			WithSchemaStrategy(StrategyRenameRight),
			WithPrimaryOperationPolicy(PolicyFirstEncountered),
			WithRenameTemplate("{{.Name}}_{{.OperationID}}"),
		)

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// The renamed schema should use the first encountered operation's ID
		assert.NotNil(t, doc.Components.Schemas["Response_createOrder"],
			"expected Response_createOrder from second document")
	})

	t.Run("PolicyAlphabetical", func(t *testing.T) {
		result, err := JoinWithOptions(
			WithFilePaths(
				filepath.Join(testdataDir, "join-operation-context-users-3.0.yaml"),
				filepath.Join(testdataDir, "join-operation-context-orders-3.0.yaml"),
			),
			WithOperationContext(true),
			WithSchemaStrategy(StrategyRenameRight),
			WithPrimaryOperationPolicy(PolicyAlphabetical),
			WithRenameTemplate("{{.Name}}_{{.OperationID}}"),
		)

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// The renamed schema should use alphabetically first operation
		// For orders API, the only operation is createOrder
		assert.NotNil(t, doc.Components.Schemas["Response_createOrder"],
			"expected Response_createOrder from second document")
	})

	t.Run("PolicyMostSpecific", func(t *testing.T) {
		result, err := JoinWithOptions(
			WithFilePaths(
				filepath.Join(testdataDir, "join-operation-context-users-3.0.yaml"),
				filepath.Join(testdataDir, "join-operation-context-orders-3.0.yaml"),
			),
			WithOperationContext(true),
			WithSchemaStrategy(StrategyRenameRight),
			WithPrimaryOperationPolicy(PolicyMostSpecific),
			WithRenameTemplate("{{.Name}}_{{.OperationID}}"),
		)

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// Should prefer operation with operationId (createOrder has one)
		assert.NotNil(t, doc.Components.Schemas["Response_createOrder"],
			"expected Response_createOrder from second document")
	})
}

// TestJoinWithOperationContext_DisabledByDefault tests that operation context
// fields are empty when not explicitly enabled.
func TestJoinWithOperationContext_DisabledByDefault(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	// Join without explicitly setting OperationContext
	result, err := JoinWithOptions(
		WithFilePaths(
			filepath.Join(testdataDir, "join-operation-context-users-3.0.yaml"),
			filepath.Join(testdataDir, "join-operation-context-orders-3.0.yaml"),
		),
		WithSchemaStrategy(StrategyRenameRight),
		// Note: not calling WithOperationContext at all
		WithRenameTemplate("{{.Name}}_{{default .OperationID .Source}}"),
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3 document")
	require.NotNil(t, doc.Components)

	// Verify the template fell back to Source since OperationID was empty
	foundRenamed := false
	for name := range doc.Components.Schemas {
		if strings.HasPrefix(name, "Response_join_operation") {
			foundRenamed = true
			break
		}
	}
	assert.True(t, foundRenamed,
		"expected renamed schema with source fallback (OperationID should be empty when operation context disabled)")
}

// TestJoinWithOperationContext_TemplateWithTags tests using tags in rename templates.
func TestJoinWithOperationContext_TemplateWithTags(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	result, err := JoinWithOptions(
		WithFilePaths(
			filepath.Join(testdataDir, "join-operation-context-users-3.0.yaml"),
			filepath.Join(testdataDir, "join-operation-context-orders-3.0.yaml"),
		),
		WithOperationContext(true),
		WithSchemaStrategy(StrategyRenameRight),
		WithRenameTemplate("{{firstTag .Tags}}_{{.Name}}"),
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3 document")
	require.NotNil(t, doc.Components)

	// Verify original Response schema from first doc exists
	assert.NotNil(t, doc.Components.Schemas["Response"],
		"expected original 'Response' schema")

	// Verify renamed Response uses first tag (Orders)
	assert.NotNil(t, doc.Components.Schemas["Orders_Response"],
		"expected renamed 'Orders_Response' schema using first tag")
}

// TestJoinWithOperationContext_TemplateWithUsageType tests using usage type in templates.
func TestJoinWithOperationContext_TemplateWithUsageType(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	result, err := JoinWithOptions(
		WithFilePaths(
			filepath.Join(testdataDir, "join-operation-context-users-3.0.yaml"),
			filepath.Join(testdataDir, "join-operation-context-orders-3.0.yaml"),
		),
		WithOperationContext(true),
		WithSchemaStrategy(StrategyRenameRight),
		WithRenameTemplate("{{.Name}}_{{.UsageType}}"),
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3 document")
	require.NotNil(t, doc.Components)

	// Verify original Response schema exists
	assert.NotNil(t, doc.Components.Schemas["Response"],
		"expected original 'Response' schema")

	// Verify renamed Response uses usage type (response)
	assert.NotNil(t, doc.Components.Schemas["Response_response"],
		"expected renamed 'Response_response' schema using usage type")
}

// TestJoinWithOperationContext_TemplateWithStatusCode tests using status code in templates.
func TestJoinWithOperationContext_TemplateWithStatusCode(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	result, err := JoinWithOptions(
		WithFilePaths(
			filepath.Join(testdataDir, "join-operation-context-users-3.0.yaml"),
			filepath.Join(testdataDir, "join-operation-context-orders-3.0.yaml"),
		),
		WithOperationContext(true),
		WithSchemaStrategy(StrategyRenameRight),
		WithRenameTemplate("{{.Name}}_{{.StatusCode}}"),
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3 document")
	require.NotNil(t, doc.Components)

	// Verify original Response schema exists
	assert.NotNil(t, doc.Components.Schemas["Response"],
		"expected original 'Response' schema")

	// Verify renamed Response uses status code (201 for createOrder)
	assert.NotNil(t, doc.Components.Schemas["Response_201"],
		"expected renamed 'Response_201' schema using status code")
}

// TestJoinWithOperationContext_AggregateFields tests that aggregate fields
// (AllPaths, AllOperationIDs, etc.) are populated when a schema is shared.
func TestJoinWithOperationContext_AggregateFields(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	// Using joinTags with AllTags demonstrates aggregate field access
	result, err := JoinWithOptions(
		WithFilePaths(
			filepath.Join(testdataDir, "join-operation-context-users-3.0.yaml"),
			filepath.Join(testdataDir, "join-operation-context-orders-3.0.yaml"),
		),
		WithOperationContext(true),
		WithSchemaStrategy(StrategyRenameRight),
		WithRenameTemplate("{{.Name}}_{{joinTags .AllTags \"_\"}}"),
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3 document")
	require.NotNil(t, doc.Components)

	// For the orders doc, Response is only referenced by one operation with tag "Orders"
	assert.NotNil(t, doc.Components.Schemas["Response_Orders"],
		"expected renamed schema with joined tags")
}

// TestJoinWithOperationContext_CoalesceTemplate tests using coalesce for fallbacks.
func TestJoinWithOperationContext_CoalesceTemplate(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	result, err := JoinWithOptions(
		WithFilePaths(
			filepath.Join(testdataDir, "join-operation-context-users-3.0.yaml"),
			filepath.Join(testdataDir, "join-operation-context-orders-3.0.yaml"),
		),
		WithOperationContext(true),
		WithSchemaStrategy(StrategyRenameRight),
		WithRenameTemplate("{{.Name}}_{{coalesce .OperationID .PrimaryResource .Source}}"),
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3 document")
	require.NotNil(t, doc.Components)

	// Should use OperationID since it's available (createOrder)
	assert.NotNil(t, doc.Components.Schemas["Response_createOrder"],
		"expected coalesce to pick OperationID as first non-empty value")
}

// TestJoinWithOperationContext_RenameLeft tests operation context with rename-left strategy.
func TestJoinWithOperationContext_RenameLeft(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	result, err := JoinWithOptions(
		WithFilePaths(
			filepath.Join(testdataDir, "join-operation-context-users-3.0.yaml"),
			filepath.Join(testdataDir, "join-operation-context-orders-3.0.yaml"),
		),
		WithOperationContext(true),
		WithSchemaStrategy(StrategyRenameLeft),
		WithRenameTemplate("{{.Name}}_{{.OperationID}}"),
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3 document")
	require.NotNil(t, doc.Components)

	// With rename-left, the original (left) schema gets renamed
	// The new (right) schema keeps the original name
	assert.NotNil(t, doc.Components.Schemas["Response"],
		"expected 'Response' schema (from second doc)")

	// The left schema should be renamed using its operation context
	// Note: For rename-left, we pass nil graph so it falls back to source-based naming
	foundRenamed := false
	for name := range doc.Components.Schemas {
		if strings.HasPrefix(name, "Response_") && name != "Response" {
			foundRenamed = true
			t.Logf("Found renamed schema: %s", name)
			break
		}
	}
	assert.True(t, foundRenamed, "expected original Response to be renamed")
}

// TestJoinWithOperationContext_CaseTransformations tests case transformation functions.
func TestJoinWithOperationContext_CaseTransformations(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	tests := []struct {
		name           string
		template       string
		expectedSchema string
	}{
		{
			name:           "pascalCase operationId",
			template:       "{{.OperationID | pascalCase}}{{.Name}}",
			expectedSchema: "CreateOrderResponse",
		},
		{
			name:           "camelCase operationId",
			template:       "{{.Name}}_{{.OperationID | camelCase}}",
			expectedSchema: "Response_createOrder", // createOrder is already camelCase
		},
		{
			name:           "snakeCase operationId",
			template:       "{{.Name}}_{{.OperationID | snakeCase}}",
			expectedSchema: "Response_create_order",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := JoinWithOptions(
				WithFilePaths(
					filepath.Join(testdataDir, "join-operation-context-users-3.0.yaml"),
					filepath.Join(testdataDir, "join-operation-context-orders-3.0.yaml"),
				),
				WithOperationContext(true),
				WithSchemaStrategy(StrategyRenameRight),
				WithRenameTemplate(tt.template),
			)

			require.NoError(t, err)
			require.NotNil(t, result)

			doc, ok := result.Document.(*parser.OAS3Document)
			require.True(t, ok)

			assert.NotNil(t, doc.Components.Schemas[tt.expectedSchema],
				"expected schema %q with template %q", tt.expectedSchema, tt.template)
		})
	}
}

// TestJoinWithOperationContext_PathCleanTemplate tests the pathClean function.
func TestJoinWithOperationContext_PathCleanTemplate(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	result, err := JoinWithOptions(
		WithFilePaths(
			filepath.Join(testdataDir, "join-operation-context-users-3.0.yaml"),
			filepath.Join(testdataDir, "join-operation-context-orders-3.0.yaml"),
		),
		WithOperationContext(true),
		WithSchemaStrategy(StrategyRenameRight),
		WithRenameTemplate("{{.Name}}_{{pathClean .Path}}"),
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3 document")
	require.NotNil(t, doc.Components)

	// pathClean on /orders should produce "orders"
	assert.NotNil(t, doc.Components.Schemas["Response_orders"],
		"expected renamed schema with cleaned path")
}

// TestJoinWithOperationContext_OAS2 tests operation context with OAS 2.0 documents.
func TestJoinWithOperationContext_OAS2(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	result, err := JoinWithOptions(
		WithFilePaths(
			filepath.Join(testdataDir, "join-collision-rename-base-2.0.yaml"),
			filepath.Join(testdataDir, "join-collision-rename-ext-2.0.yaml"),
		),
		WithOperationContext(true),
		WithSchemaStrategy(StrategyRenameRight),
		WithRenameTemplate("{{.Name}}_{{default .OperationID .Source}}"),
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	doc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok, "expected OAS2 document")
	require.NotNil(t, doc.Definitions)

	// Verify the original User definition exists
	assert.NotNil(t, doc.Definitions["User"], "expected User definition")

	// Verify a renamed definition exists
	foundRenamed := false
	for name := range doc.Definitions {
		if strings.HasPrefix(name, "User_") && name != "User" {
			foundRenamed = true
			break
		}
	}
	assert.True(t, foundRenamed, "expected renamed User definition")
}
