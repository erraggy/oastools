package fixer

import (
	"fmt"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Duplicate OperationId Fix Tests
// =============================================================================

// TestFixDuplicateOperationIds tests fixing duplicate operationIds
func TestFixDuplicateOperationIds(t *testing.T) {
	tests := []struct {
		name          string
		template      string
		yaml          string
		expectedFixes int
		checkFixes    func(t *testing.T, fixes []Fix, doc any)
	}{
		// Test case 1: No duplicates
		{
			name:     "no duplicates",
			template: "",
			yaml: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "200":
          description: Success
  /items:
    get:
      operationId: getItems
      responses:
        "200":
          description: Success
`,
			expectedFixes: 0,
			checkFixes: func(t *testing.T, fixes []Fix, doc any) {
				assert.Empty(t, fixes)
			},
		},

		// Test case 2: Simple duplicate
		// Note: Paths are sorted alphabetically, so /items is processed before /users
		{
			name:     "simple duplicate",
			template: "",
			yaml: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getData
      responses:
        "200":
          description: Success
  /items:
    get:
      operationId: getData
      responses:
        "200":
          description: Success
`,
			expectedFixes: 1,
			checkFixes: func(t *testing.T, fixes []Fix, doc any) {
				require.Len(t, fixes, 1)
				assert.Equal(t, FixTypeDuplicateOperationId, fixes[0].Type)
				assert.Equal(t, "getData", fixes[0].Before)
				assert.Equal(t, "getData2", fixes[0].After)
				assert.Contains(t, fixes[0].Description, "renamed duplicate operationId")
				// /items is processed first (alphabetically), so /users is the duplicate
				assert.Contains(t, fixes[0].Description, "first occurrence at GET /items")

				// Verify document was modified
				// /items is seen first (alphabetically), so /users gets renamed
				oas3Doc := doc.(*parser.OAS3Document)
				assert.Equal(t, "getData2", oas3Doc.Paths["/users"].Get.OperationID)
				assert.Equal(t, "getData", oas3Doc.Paths["/items"].Get.OperationID)
			},
		},

		// Test case 3: Triple duplicate
		// Note: Paths are sorted alphabetically: /items, /posts, /users
		{
			name:     "triple duplicate",
			template: "",
			yaml: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: fetch
      responses:
        "200":
          description: Success
  /items:
    get:
      operationId: fetch
      responses:
        "200":
          description: Success
  /posts:
    get:
      operationId: fetch
      responses:
        "200":
          description: Success
`,
			expectedFixes: 2,
			checkFixes: func(t *testing.T, fixes []Fix, doc any) {
				require.Len(t, fixes, 2)

				// First fix: fetch -> fetch2 (for /posts, second alphabetically)
				assert.Equal(t, "fetch", fixes[0].Before)
				assert.Equal(t, "fetch2", fixes[0].After)

				// Second fix: fetch -> fetch3 (for /users, third alphabetically)
				assert.Equal(t, "fetch", fixes[1].Before)
				assert.Equal(t, "fetch3", fixes[1].After)

				// Verify document was modified
				// Alphabetically: /items (original), /posts (fetch2), /users (fetch3)
				oas3Doc := doc.(*parser.OAS3Document)
				assert.Equal(t, "fetch", oas3Doc.Paths["/items"].Get.OperationID)
				assert.Equal(t, "fetch2", oas3Doc.Paths["/posts"].Get.OperationID)
				assert.Equal(t, "fetch3", oas3Doc.Paths["/users"].Get.OperationID)
			},
		},

		// Test case 4: Method template
		// Note: /items comes before /users alphabetically
		{
			name:     "method template",
			template: "{operationId}_{method}",
			yaml: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: manage
      responses:
        "200":
          description: Success
  /items:
    post:
      operationId: manage
      responses:
        "200":
          description: Success
`,
			expectedFixes: 1,
			checkFixes: func(t *testing.T, fixes []Fix, doc any) {
				require.Len(t, fixes, 1)
				assert.Equal(t, "manage", fixes[0].Before)
				// /items is first, /users is duplicate, so it gets renamed with GET method
				assert.Equal(t, "manage_get", fixes[0].After)

				oas3Doc := doc.(*parser.OAS3Document)
				assert.Equal(t, "manage", oas3Doc.Paths["/items"].Post.OperationID)
				assert.Equal(t, "manage_get", oas3Doc.Paths["/users"].Get.OperationID)
			},
		},

		// Test case 5: Path template
		// Note: /items comes before /users alphabetically
		{
			name:     "path template",
			template: "{method}_{path}",
			yaml: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: list
      responses:
        "200":
          description: Success
  /items:
    get:
      operationId: list
      responses:
        "200":
          description: Success
`,
			expectedFixes: 1,
			checkFixes: func(t *testing.T, fixes []Fix, doc any) {
				require.Len(t, fixes, 1)
				assert.Equal(t, "list", fixes[0].Before)
				// /items is first, /users is duplicate, so it gets renamed with "users" path
				assert.Equal(t, "get_users", fixes[0].After)

				oas3Doc := doc.(*parser.OAS3Document)
				assert.Equal(t, "list", oas3Doc.Paths["/items"].Get.OperationID)
				assert.Equal(t, "get_users", oas3Doc.Paths["/users"].Get.OperationID)
			},
		},

		// Test case 6: Template collision falls back to numeric
		// Note: Paths sorted alphabetically: /items, /posts, /users
		{
			name:     "template collision falls back to numeric",
			template: "{operationId}_{method}",
			yaml: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getData
      responses:
        "200":
          description: Success
  /items:
    get:
      operationId: getData
      responses:
        "200":
          description: Success
  /posts:
    get:
      operationId: getData
      responses:
        "200":
          description: Success
`,
			expectedFixes: 2,
			checkFixes: func(t *testing.T, fixes []Fix, doc any) {
				require.Len(t, fixes, 2)

				// First duplicate (/posts): getData -> getData_get
				assert.Equal(t, "getData", fixes[0].Before)
				assert.Equal(t, "getData_get", fixes[0].After)

				// Second duplicate (/users): collision with getData_get, falls back to numeric
				assert.Equal(t, "getData", fixes[1].Before)
				assert.Equal(t, "getData_get3", fixes[1].After)

				// Alphabetically: /items (original), /posts (getData_get), /users (getData_get3)
				oas3Doc := doc.(*parser.OAS3Document)
				assert.Equal(t, "getData", oas3Doc.Paths["/items"].Get.OperationID)
				assert.Equal(t, "getData_get", oas3Doc.Paths["/posts"].Get.OperationID)
				assert.Equal(t, "getData_get3", oas3Doc.Paths["/users"].Get.OperationID)
			},
		},

		// Test case 7: Empty operationId skipped
		{
			name:     "empty operationId skipped",
			template: "",
			yaml: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: ""
      responses:
        "200":
          description: Success
  /items:
    get:
      operationId: ""
      responses:
        "200":
          description: Success
`,
			expectedFixes: 0,
			checkFixes: func(t *testing.T, fixes []Fix, doc any) {
				assert.Empty(t, fixes)
			},
		},

		// Test case 8: Mixed duplicates and uniques
		// Note: Paths sorted alphabetically: /items, /posts, /users
		{
			name:     "mixed duplicates and uniques",
			template: "",
			yaml: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "200":
          description: Success
    post:
      operationId: createUser
      responses:
        "200":
          description: Success
  /items:
    get:
      operationId: getUsers
      responses:
        "200":
          description: Success
    post:
      operationId: createItem
      responses:
        "200":
          description: Success
  /posts:
    get:
      operationId: getPosts
      responses:
        "200":
          description: Success
`,
			expectedFixes: 1,
			checkFixes: func(t *testing.T, fixes []Fix, doc any) {
				require.Len(t, fixes, 1)
				assert.Equal(t, "getUsers", fixes[0].Before)
				assert.Equal(t, "getUsers2", fixes[0].After)

				// /items is first, so /users.get is the duplicate
				oas3Doc := doc.(*parser.OAS3Document)
				assert.Equal(t, "getUsers", oas3Doc.Paths["/items"].Get.OperationID)
				assert.Equal(t, "createItem", oas3Doc.Paths["/items"].Post.OperationID)
				assert.Equal(t, "getPosts", oas3Doc.Paths["/posts"].Get.OperationID)
				assert.Equal(t, "getUsers2", oas3Doc.Paths["/users"].Get.OperationID)
				assert.Equal(t, "createUser", oas3Doc.Paths["/users"].Post.OperationID)
			},
		},

		// Test case 9: Different methods same path
		{
			name:     "different methods same path",
			template: "",
			yaml: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: manageUsers
      responses:
        "200":
          description: Success
    post:
      operationId: manageUsers
      responses:
        "200":
          description: Success
    put:
      operationId: manageUsers
      responses:
        "200":
          description: Success
`,
			expectedFixes: 2,
			checkFixes: func(t *testing.T, fixes []Fix, doc any) {
				require.Len(t, fixes, 2)

				oas3Doc := doc.(*parser.OAS3Document)
				assert.Equal(t, "manageUsers", oas3Doc.Paths["/users"].Get.OperationID)
				assert.Equal(t, "manageUsers2", oas3Doc.Paths["/users"].Put.OperationID)
				assert.Equal(t, "manageUsers3", oas3Doc.Paths["/users"].Post.OperationID)
			},
		},

		// Test case 10: Nil operation handling
		{
			name:     "nil operations skipped",
			template: "",
			yaml: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "200":
          description: Success
  /items: {}
`,
			expectedFixes: 0,
			checkFixes: func(t *testing.T, fixes []Fix, doc any) {
				assert.Empty(t, fixes)
			},
		},

		// Test case 11: Tag template
		// Note: /items comes before /users alphabetically
		{
			name:     "tag template",
			template: "{operationId}_{tag}_{n}",
			yaml: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: list
      tags:
        - Users
      responses:
        "200":
          description: Success
  /items:
    get:
      operationId: list
      tags:
        - Items
      responses:
        "200":
          description: Success
`,
			expectedFixes: 1,
			checkFixes: func(t *testing.T, fixes []Fix, doc any) {
				require.Len(t, fixes, 1)
				assert.Equal(t, "list", fixes[0].Before)
				// /items is first, /users is duplicate
				// n=2 so the {n} part should be "2", tag is "Users"
				assert.Equal(t, "list_Users_2", fixes[0].After)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(tt.yaml)))
			require.NoError(t, err)

			f := New()
			f.EnabledFixes = []FixType{FixTypeDuplicateOperationId}
			if tt.template != "" {
				f.OperationIdNamingConfig.Template = tt.template
			}

			result, err := f.FixParsed(*parseResult)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedFixes, len(result.Fixes))

			if tt.checkFixes != nil {
				tt.checkFixes(t, result.Fixes, result.Document)
			}
		})
	}
}

// =============================================================================
// OAS Version Tests
// =============================================================================

// TestFixDuplicateOperationIds_OAS2 tests fixing duplicate operationIds in OAS 2.0 documents
func TestFixDuplicateOperationIds_OAS2(t *testing.T) {
	// Note: /items comes before /users alphabetically
	yaml := `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getData
      responses:
        "200":
          description: Success
  /items:
    get:
      operationId: getData
      responses:
        "200":
          description: Success
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(yaml)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypeDuplicateOperationId}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	assert.Equal(t, 1, len(result.Fixes))
	assert.Equal(t, "getData", result.Fixes[0].Before)
	assert.Equal(t, "getData2", result.Fixes[0].After)

	// Verify OAS 2.0 document was modified
	// /items is first (alphabetically), so /users gets renamed
	oas2Doc := result.Document.(*parser.OAS2Document)
	assert.Equal(t, "getData", oas2Doc.Paths["/items"].Get.OperationID)
	assert.Equal(t, "getData2", oas2Doc.Paths["/users"].Get.OperationID)
}

// TestFixDuplicateOperationIds_OAS31Webhooks tests that OAS 3.1+ webhooks share operationId namespace with paths.
// Per the OAS spec: "The id MUST be unique among all operations described in the API."
func TestFixDuplicateOperationIds_OAS31Webhooks(t *testing.T) {
	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: handleEvent
      responses:
        "200":
          description: Success
webhooks:
  userCreated:
    post:
      operationId: handleEvent
      responses:
        "200":
          description: Success
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(yaml)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypeDuplicateOperationId}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// One fix should be applied - paths and webhooks share the same operationId namespace per OAS spec
	// "paths:/users" sorts before "webhooks:userCreated", so the webhook is the duplicate
	assert.Equal(t, 1, len(result.Fixes))
	assert.Equal(t, "handleEvent", result.Fixes[0].Before)
	assert.Equal(t, "handleEvent2", result.Fixes[0].After)
	assert.Contains(t, result.Fixes[0].Path, "webhooks.")

	// Verify: path keeps original, webhook is renamed
	oas3Doc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "handleEvent", oas3Doc.Paths["/users"].Get.OperationID)
	assert.Equal(t, "handleEvent2", oas3Doc.Webhooks["userCreated"].Post.OperationID)
}

// TestFixDuplicateOperationIds_OAS31WebhooksDuplicates tests duplicates among webhook operations
func TestFixDuplicateOperationIds_OAS31WebhooksDuplicates(t *testing.T) {
	yaml := `
openapi: "3.1.0"
info:
  title: Test API
  version: "1.0"
paths: {}
webhooks:
  userCreated:
    post:
      operationId: handleEvent
      responses:
        "200":
          description: Success
  orderCreated:
    post:
      operationId: handleEvent
      responses:
        "200":
          description: Success
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(yaml)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypeDuplicateOperationId}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// One fix should be applied for the duplicate within webhooks
	assert.Equal(t, 1, len(result.Fixes))
	assert.Equal(t, "handleEvent", result.Fixes[0].Before)
	assert.Equal(t, "handleEvent2", result.Fixes[0].After)
	assert.Contains(t, result.Fixes[0].Path, "webhooks.")

	// Verify document was modified
	oas3Doc := result.Document.(*parser.OAS3Document)
	// One should be original, one should be renamed (order depends on map iteration)
	opIds := []string{
		oas3Doc.Webhooks["userCreated"].Post.OperationID,
		oas3Doc.Webhooks["orderCreated"].Post.OperationID,
	}
	assert.Contains(t, opIds, "handleEvent")
	assert.Contains(t, opIds, "handleEvent2")
}

// TestFixDuplicateOperationIds_OAS30NoWebhooks tests that OAS 3.0.x doesn't process webhooks
func TestFixDuplicateOperationIds_OAS30NoWebhooks(t *testing.T) {
	// OAS 3.0.x doesn't have webhooks, so even if there's a webhooks field, it should be ignored
	// Note: /items comes before /users alphabetically
	yaml := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getData
      responses:
        "200":
          description: Success
  /items:
    get:
      operationId: getData
      responses:
        "200":
          description: Success
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(yaml)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypeDuplicateOperationId}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	assert.Equal(t, 1, len(result.Fixes))

	// /items is first (alphabetically), so /users gets renamed
	oas3Doc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "getData", oas3Doc.Paths["/items"].Get.OperationID)
	assert.Equal(t, "getData2", oas3Doc.Paths["/users"].Get.OperationID)
}

// =============================================================================
// Dry Run Tests
// =============================================================================

// TestFixDuplicateOperationIds_DryRun tests that dry run doesn't modify the document
func TestFixDuplicateOperationIds_DryRun(t *testing.T) {
	// Note: /items comes before /users alphabetically
	yaml := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getData
      responses:
        "200":
          description: Success
  /items:
    get:
      operationId: getData
      responses:
        "200":
          description: Success
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(yaml)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypeDuplicateOperationId}
	f.DryRun = true

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Fixes should still be reported
	assert.Equal(t, 1, len(result.Fixes))
	assert.Equal(t, "getData", result.Fixes[0].Before)
	assert.Equal(t, "getData2", result.Fixes[0].After)

	// But document should NOT be modified - both should still be getData
	oas3Doc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, "getData", oas3Doc.Paths["/items"].Get.OperationID)
	assert.Equal(t, "getData", oas3Doc.Paths["/users"].Get.OperationID) // Still getData, not getData2
}

// =============================================================================
// Configuration Option Tests
// =============================================================================

// TestWithOperationIdNamingConfig tests the WithOperationIdNamingConfig option
func TestWithOperationIdNamingConfig(t *testing.T) {
	// Note: /items comes before /users alphabetically
	// Using a path with segments to test PathSeparator: /api/v1/users
	yaml := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /api/v1/users:
    get:
      operationId: list
      responses:
        "200":
          description: Success
  /api/v1/items:
    get:
      operationId: list
      responses:
        "200":
          description: Success
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(yaml)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeDuplicateOperationId),
		WithOperationIdNamingConfig(OperationIdNamingConfig{
			Template:      "{method}-{path}",
			PathSeparator: "-",
		}),
	)
	require.NoError(t, err)

	assert.Equal(t, 1, len(result.Fixes))
	assert.Equal(t, "list", result.Fixes[0].Before)
	// /api/v1/items is first alphabetically, /api/v1/users is duplicate
	// PathSeparator "-" makes /api/v1/users become "api-v1-users"
	// Template "{method}-{path}" produces "get-api-v1-users"
	assert.Equal(t, "get-api-v1-users", result.Fixes[0].After)
}

// TestWithOperationIdNamingConfig_InvalidTemplate tests that invalid templates are rejected
func TestWithOperationIdNamingConfig_InvalidTemplate(t *testing.T) {
	yaml := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths: {}
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(yaml)))
	require.NoError(t, err)

	_, err = FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeDuplicateOperationId),
		WithOperationIdNamingConfig(OperationIdNamingConfig{
			Template: "{invalid}",
		}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown placeholder {invalid}")
}

// TestWithOperationIdNamingConfig_EmptyTemplate tests that empty templates are rejected
func TestWithOperationIdNamingConfig_EmptyTemplate(t *testing.T) {
	yaml := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths: {}
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(yaml)))
	require.NoError(t, err)

	_, err = FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeDuplicateOperationId),
		WithOperationIdNamingConfig(OperationIdNamingConfig{
			Template: "",
		}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

// =============================================================================
// Default Configuration Tests
// =============================================================================

// TestDefaultOperationIdNamingConfig tests the default configuration values
func TestDefaultOperationIdNamingConfig(t *testing.T) {
	config := DefaultOperationIdNamingConfig()

	assert.Equal(t, "{operationId}{n}", config.Template)
	assert.Equal(t, "_", config.PathSeparator)
	assert.Equal(t, "_", config.TagSeparator)
}

// =============================================================================
// Integration Tests
// =============================================================================

// TestFixDuplicateOperationIds_WithAllFixTypes tests that duplicate operationId fix works with other fixes enabled
func TestFixDuplicateOperationIds_WithAllFixTypes(t *testing.T) {
	yaml := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users/{userId}:
    get:
      operationId: getData
      responses:
        "200":
          description: Success
  /items/{itemId}:
    get:
      operationId: getData
      responses:
        "200":
          description: Success
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(yaml)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{
		FixTypeDuplicateOperationId,
		FixTypeMissingPathParameter,
	}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Should have both types of fixes
	fixTypes := make(map[FixType]int)
	for _, fix := range result.Fixes {
		fixTypes[fix.Type]++
	}

	// 1 duplicate operationId fix + 2 missing path parameter fixes (userId and itemId)
	assert.Equal(t, 1, fixTypes[FixTypeDuplicateOperationId])
	assert.Equal(t, 2, fixTypes[FixTypeMissingPathParameter])
}

// TestFixDuplicateOperationIds_NotEnabledByDefault tests that fix is not enabled by default
func TestFixDuplicateOperationIds_NotEnabledByDefault(t *testing.T) {
	yaml := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getData
      responses:
        "200":
          description: Success
  /items:
    get:
      operationId: getData
      responses:
        "200":
          description: Success
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(yaml)))
	require.NoError(t, err)

	f := New() // Default fixer, only FixTypeMissingPathParameter is enabled

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// No duplicate operationId fixes should be applied
	for _, fix := range result.Fixes {
		assert.NotEqual(t, FixTypeDuplicateOperationId, fix.Type)
	}
}

// =============================================================================
// Edge Case and Performance Tests
// =============================================================================

// TestResolveOperationIdCollision_Terminates tests that collision resolution terminates
func TestResolveOperationIdCollision_Terminates(t *testing.T) {
	// Create a fixer with default config
	f := New()
	f.EnabledFixes = []FixType{FixTypeDuplicateOperationId}

	// Pre-populate assigned map with many colliding names
	assigned := make(map[string]bool)
	for i := 2; i <= 100; i++ {
		assigned[fmt.Sprintf("getUser%d", i)] = true
	}

	ctx := OperationContext{
		OperationId: "getUser",
		Method:      "get",
		Path:        "/users",
	}

	// Should find a unique name (getUser101)
	result := f.resolveOperationIdCollision(ctx, assigned)
	assert.Equal(t, "getUser101", result)
	assert.False(t, assigned[result], "Result should not already be assigned")
}

// TestFixDuplicateOperationIds_LargeNumberOfDuplicates tests handling of many duplicates
func TestFixDuplicateOperationIds_LargeNumberOfDuplicates(t *testing.T) {
	// Build a spec with 100 operations all having the same operationId
	var pathsYaml string
	for i := 0; i < 100; i++ {
		pathsYaml += fmt.Sprintf(`  /path%03d:
    get:
      operationId: duplicate
      responses:
        "200":
          description: Success
`, i)
	}

	yaml := fmt.Sprintf(`
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
%s`, pathsYaml)

	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(yaml)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypeDuplicateOperationId}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Should have 99 fixes (first one is not a duplicate)
	assert.Equal(t, 99, len(result.Fixes))

	// Verify all operationIds are now unique
	oas3Doc := result.Document.(*parser.OAS3Document)
	seen := make(map[string]bool)
	for path, pathItem := range oas3Doc.Paths {
		if pathItem.Get != nil && pathItem.Get.OperationID != "" {
			opId := pathItem.Get.OperationID
			assert.False(t, seen[opId], "Duplicate operationId found: %s at %s", opId, path)
			seen[opId] = true
		}
	}
	assert.Equal(t, 100, len(seen), "Should have 100 unique operationIds")
}

// TestFixDuplicateOperationIds_UnicodeOperationId tests handling of unicode operationIds
func TestFixDuplicateOperationIds_UnicodeOperationId(t *testing.T) {
	yaml := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: "getUsuarios"
      responses:
        "200":
          description: Success
  /items:
    get:
      operationId: "getUsuarios"
      responses:
        "200":
          description: Success
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(yaml)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypeDuplicateOperationId}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Should have 1 fix
	assert.Equal(t, 1, len(result.Fixes))
	assert.Equal(t, "getUsuarios", result.Fixes[0].Before)
	assert.Equal(t, "getUsuarios2", result.Fixes[0].After)
}

// TestFixDuplicateOperationIds_ChineseOperationId tests handling of Chinese operationIds
func TestFixDuplicateOperationIds_ChineseOperationId(t *testing.T) {
	yaml := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: "获取用户"
      responses:
        "200":
          description: Success
  /items:
    get:
      operationId: "获取用户"
      responses:
        "200":
          description: Success
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(yaml)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypeDuplicateOperationId}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Should have 1 fix
	assert.Equal(t, 1, len(result.Fixes))
	assert.Equal(t, "获取用户", result.Fixes[0].Before)
	assert.Equal(t, "获取用户2", result.Fixes[0].After)
}

// TestExpandOperationIdTemplate_EmptyTagSeparator tests default TagSeparator when empty
func TestExpandOperationIdTemplate_EmptyTagSeparator(t *testing.T) {
	ctx := OperationContext{
		OperationId: "getUser",
		Method:      "get",
		Path:        "/users",
		Tags:        []string{"users", "admin"},
	}

	config := OperationIdNamingConfig{
		Template:      "{tags}",
		PathSeparator: "_",
		TagSeparator:  "", // Empty should default to "_"
	}

	result := expandOperationIdTemplate(config.Template, ctx, 1, config)
	assert.Equal(t, "users_admin", result, "Empty TagSeparator should default to underscore")
}
