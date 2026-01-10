package fixer

import (
	"fmt"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Template Parsing Tests
// =============================================================================

// TestParseOperationIdNamingTemplate tests validation of operationId naming templates
func TestParseOperationIdNamingTemplate(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		expectErr bool
		errMsg    string
	}{
		// Valid templates
		{
			name:      "default template",
			template:  "{operationId}{n}",
			expectErr: false,
		},
		{
			name:      "method and path template",
			template:  "{method}_{path}",
			expectErr: false,
		},
		{
			name:      "operationId and tag template",
			template:  "{operationId}_{tag}",
			expectErr: false,
		},
		{
			name:      "all tags template",
			template:  "{operationId}_{tags}_{n}",
			expectErr: false,
		},
		{
			name:      "plain text only",
			template:  "operation",
			expectErr: false,
		},
		{
			name:      "mixed text and placeholders",
			template:  "api_{method}_{operationId}_v{n}",
			expectErr: false,
		},
		{
			name:      "single placeholder operationId",
			template:  "{operationId}",
			expectErr: false,
		},
		{
			name:      "single placeholder method",
			template:  "{method}",
			expectErr: false,
		},
		{
			name:      "single placeholder path",
			template:  "{path}",
			expectErr: false,
		},
		{
			name:      "single placeholder tag",
			template:  "{tag}",
			expectErr: false,
		},
		{
			name:      "single placeholder tags",
			template:  "{tags}",
			expectErr: false,
		},
		{
			name:      "single placeholder n",
			template:  "{n}",
			expectErr: false,
		},
		{
			name:      "empty template",
			template:  "",
			expectErr: true,
			errMsg:    "cannot be empty",
		},

		// Invalid templates
		{
			name:      "unknown placeholder invalid",
			template:  "{invalid}",
			expectErr: true,
			errMsg:    "unknown placeholder {invalid}",
		},
		{
			name:      "unknown placeholder unknown",
			template:  "{unknown}",
			expectErr: true,
			errMsg:    "unknown placeholder {unknown}",
		},
		{
			name:      "mixed valid and invalid",
			template:  "{operationId}_{invalid}_{n}",
			expectErr: true,
			errMsg:    "unknown placeholder {invalid}",
		},
		{
			name:      "unknown placeholder foo",
			template:  "{foo}",
			expectErr: true,
			errMsg:    "unknown placeholder {foo}",
		},

		// Valid templates with modifiers
		{
			name:      "placeholder with pascal modifier",
			template:  "{operationId:pascal}",
			expectErr: false,
		},
		{
			name:      "placeholder with camel modifier",
			template:  "{operationId:camel}",
			expectErr: false,
		},
		{
			name:      "placeholder with snake modifier",
			template:  "{path:snake}",
			expectErr: false,
		},
		{
			name:      "placeholder with kebab modifier",
			template:  "{tag:kebab}",
			expectErr: false,
		},
		{
			name:      "placeholder with upper modifier",
			template:  "{method:upper}",
			expectErr: false,
		},
		{
			name:      "placeholder with lower modifier",
			template:  "{operationId:lower}",
			expectErr: false,
		},
		{
			name:      "mixed placeholders with and without modifiers",
			template:  "{operationId:pascal}_{method:upper}_{n}",
			expectErr: false,
		},
		{
			name:      "all modifiers valid",
			template:  "{operationId:pascal}_{operationId:camel}_{path:snake}_{tag:kebab}_{method:upper}_{n:lower}",
			expectErr: false,
		},

		// Invalid modifiers
		{
			name:      "unknown modifier invalid",
			template:  "{operationId:invalid}",
			expectErr: true,
			errMsg:    "unknown modifier :invalid",
		},
		{
			name:      "unknown modifier PASCAL (case sensitive)",
			template:  "{operationId:PASCAL}",
			expectErr: true,
			errMsg:    "unknown modifier :PASCAL",
		},
		{
			name:      "valid placeholder with unknown modifier",
			template:  "{method:screaming}",
			expectErr: true,
			errMsg:    "unknown modifier :screaming",
		},
		{
			name:      "mixed valid and invalid modifier",
			template:  "{operationId:pascal}_{method:invalid}",
			expectErr: true,
			errMsg:    "unknown modifier :invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseOperationIdNamingTemplate(tt.template)
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Template Expansion Tests
// =============================================================================

// TestExpandOperationIdTemplate tests template expansion with various placeholders
func TestExpandOperationIdTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		ctx      OperationContext
		n        int
		config   OperationIdNamingConfig
		expected string
	}{
		// Test {operationId} placeholder
		{
			name:     "operationId placeholder",
			template: "{operationId}",
			ctx: OperationContext{
				OperationId: "getUser",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "getUser",
		},

		// Test {method} placeholder
		{
			name:     "method placeholder",
			template: "{method}",
			ctx: OperationContext{
				Method: "get",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "get",
		},
		{
			name:     "method placeholder post",
			template: "{method}",
			ctx: OperationContext{
				Method: "post",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "post",
		},

		// Test {path} placeholder
		{
			name:     "path placeholder simple",
			template: "{path}",
			ctx: OperationContext{
				Path: "/users",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "users",
		},
		{
			name:     "path placeholder with param",
			template: "{path}",
			ctx: OperationContext{
				Path: "/users/{id}",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "users_id",
		},
		{
			name:     "path placeholder nested",
			template: "{path}",
			ctx: OperationContext{
				Path: "/users/{userId}/posts/{postId}",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "users_userId_posts_postId",
		},

		// Test {tag} placeholder (first tag only)
		{
			name:     "tag placeholder with single tag",
			template: "{tag}",
			ctx: OperationContext{
				Tags: []string{"users"},
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "users",
		},
		{
			name:     "tag placeholder with multiple tags",
			template: "{tag}",
			ctx: OperationContext{
				Tags: []string{"users", "admin"},
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "users",
		},
		{
			name:     "tag placeholder with no tags",
			template: "{tag}",
			ctx: OperationContext{
				Tags: []string{},
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "",
		},
		{
			name:     "tag placeholder with nil tags",
			template: "{tag}",
			ctx: OperationContext{
				Tags: nil,
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "",
		},

		// Test {tags} placeholder (all tags joined)
		{
			name:     "tags placeholder with multiple tags",
			template: "{tags}",
			ctx: OperationContext{
				Tags: []string{"users", "admin", "api"},
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "users_admin_api",
		},
		{
			name:     "tags placeholder with single tag",
			template: "{tags}",
			ctx: OperationContext{
				Tags: []string{"users"},
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "users",
		},
		{
			name:     "tags placeholder custom separator",
			template: "{tags}",
			ctx: OperationContext{
				Tags: []string{"users", "admin"},
			},
			n: 1,
			config: OperationIdNamingConfig{
				Template:     "{tags}",
				TagSeparator: "-",
			},
			expected: "users-admin",
		},

		// Test {n} placeholder
		{
			name:     "n placeholder n=1 is empty",
			template: "{operationId}{n}",
			ctx: OperationContext{
				OperationId: "getUser",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "getUser",
		},
		{
			name:     "n placeholder n=2",
			template: "{operationId}{n}",
			ctx: OperationContext{
				OperationId: "getUser",
			},
			n:        2,
			config:   DefaultOperationIdNamingConfig(),
			expected: "getUser2",
		},
		{
			name:     "n placeholder n=3",
			template: "{operationId}{n}",
			ctx: OperationContext{
				OperationId: "getUser",
			},
			n:        3,
			config:   DefaultOperationIdNamingConfig(),
			expected: "getUser3",
		},
		{
			name:     "n placeholder n=10",
			template: "{operationId}{n}",
			ctx: OperationContext{
				OperationId: "getUser",
			},
			n:        10,
			config:   DefaultOperationIdNamingConfig(),
			expected: "getUser10",
		},
		{
			name:     "n placeholder n=0 is empty",
			template: "{operationId}{n}",
			ctx: OperationContext{
				OperationId: "getUser",
			},
			n:        0,
			config:   DefaultOperationIdNamingConfig(),
			expected: "getUser",
		},

		// Test combined templates
		{
			name:     "combined operationId method n",
			template: "{operationId}_{method}_{n}",
			ctx: OperationContext{
				OperationId: "getUser",
				Method:      "get",
			},
			n:        2,
			config:   DefaultOperationIdNamingConfig(),
			expected: "getUser_get_2",
		},
		{
			name:     "combined method path",
			template: "{method}_{path}",
			ctx: OperationContext{
				Method: "get",
				Path:   "/users/{id}",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "get_users_id",
		},
		{
			name:     "combined operationId tag n",
			template: "{operationId}_{tag}_{n}",
			ctx: OperationContext{
				OperationId: "createItem",
				Tags:        []string{"items"},
			},
			n:        3,
			config:   DefaultOperationIdNamingConfig(),
			expected: "createItem_items_3",
		},
		{
			name:     "complex template",
			template: "api_{tag}_{method}_{path}_v{n}",
			ctx: OperationContext{
				OperationId: "getUser",
				Method:      "get",
				Path:        "/users/{id}",
				Tags:        []string{"users"},
			},
			n:        2,
			config:   DefaultOperationIdNamingConfig(),
			expected: "api_users_get_users_id_v2",
		},

		// Test all placeholders at once
		{
			name:     "all placeholders",
			template: "{operationId}_{method}_{path}_{tag}_{tags}_{n}",
			ctx: OperationContext{
				OperationId: "getUser",
				Method:      "get",
				Path:        "/api/v1",
				Tags:        []string{"users", "admin"},
			},
			n:        5,
			config:   DefaultOperationIdNamingConfig(),
			expected: "getUser_get_api_v1_users_users_admin_5",
		},

		// Test modifiers
		{
			name:     "operationId with pascal modifier",
			template: "{operationId:pascal}",
			ctx: OperationContext{
				OperationId: "get_user",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "GetUser",
		},
		{
			name:     "operationId with camel modifier",
			template: "{operationId:camel}",
			ctx: OperationContext{
				OperationId: "GetUser",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "getUser",
		},
		{
			name:     "method with upper modifier",
			template: "{method:upper}",
			ctx: OperationContext{
				Method: "get",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "GET",
		},
		{
			name:     "method with lower modifier",
			template: "{method:lower}",
			ctx: OperationContext{
				Method: "GET",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "get",
		},
		{
			name:     "path with snake modifier",
			template: "{path:snake}",
			ctx: OperationContext{
				Path: "/usersProfile/{userId}",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "users_profile_user_id",
		},
		{
			name:     "path with kebab modifier",
			template: "{path:kebab}",
			ctx: OperationContext{
				Path: "/usersProfile/{userId}",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "users-profile-user-id",
		},
		{
			name:     "tag with pascal modifier",
			template: "{tag:pascal}",
			ctx: OperationContext{
				Tags: []string{"user-profile"},
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "UserProfile",
		},
		{
			name:     "combined modifiers",
			template: "{operationId:pascal}_{method:upper}_{n}",
			ctx: OperationContext{
				OperationId: "get_user",
				Method:      "post",
			},
			n:        2,
			config:   DefaultOperationIdNamingConfig(),
			expected: "GetUser_POST_2",
		},
		{
			name:     "modifier with no effect on already formatted value",
			template: "{operationId:upper}",
			ctx: OperationContext{
				OperationId: "GETUSER",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "GETUSER",
		},
		{
			name:     "empty value with modifier",
			template: "{tag:pascal}",
			ctx: OperationContext{
				Tags: []string{},
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "",
		},
		{
			name:     "n with modifier (edge case)",
			template: "{n:upper}",
			ctx:      OperationContext{},
			n:        2,
			config:   DefaultOperationIdNamingConfig(),
			expected: "2", // Numbers unaffected by upper
		},
		{
			name:     "multiple same placeholder different modifiers",
			template: "{operationId:pascal}_{operationId:snake}",
			ctx: OperationContext{
				OperationId: "getUserProfile",
			},
			n:        1,
			config:   DefaultOperationIdNamingConfig(),
			expected: "GetUserProfile_get_user_profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandOperationIdTemplate(tt.template, tt.ctx, tt.n, tt.config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Apply Modifier Tests
// =============================================================================

// TestApplyModifier tests the applyModifier function directly
func TestApplyModifier(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		modifier string
		expected string
	}{
		// Pascal case
		{
			name:     "pascal from snake_case",
			value:    "get_user",
			modifier: "pascal",
			expected: "GetUser",
		},
		{
			name:     "pascal from kebab-case",
			value:    "get-user-profile",
			modifier: "pascal",
			expected: "GetUserProfile",
		},

		// Camel case
		{
			name:     "camel from snake_case",
			value:    "get_user",
			modifier: "camel",
			expected: "getUser",
		},
		{
			name:     "camel from PascalCase",
			value:    "GetUser",
			modifier: "camel",
			expected: "getUser",
		},

		// Snake case
		{
			name:     "snake from camelCase",
			value:    "getUserProfile",
			modifier: "snake",
			expected: "get_user_profile",
		},
		{
			name:     "snake from PascalCase",
			value:    "GetUserProfile",
			modifier: "snake",
			expected: "get_user_profile",
		},

		// Kebab case
		{
			name:     "kebab from camelCase",
			value:    "getUserProfile",
			modifier: "kebab",
			expected: "get-user-profile",
		},
		{
			name:     "kebab from PascalCase",
			value:    "GetUserProfile",
			modifier: "kebab",
			expected: "get-user-profile",
		},

		// Upper case
		{
			name:     "upper from lowercase",
			value:    "get",
			modifier: "upper",
			expected: "GET",
		},
		{
			name:     "upper from mixed",
			value:    "getUserProfile",
			modifier: "upper",
			expected: "GETUSERPROFILE",
		},

		// Lower case
		{
			name:     "lower from UPPERCASE",
			value:    "GET",
			modifier: "lower",
			expected: "get",
		},
		{
			name:     "lower from mixed",
			value:    "GetUserProfile",
			modifier: "lower",
			expected: "getuserprofile",
		},

		// No modifier
		{
			name:     "empty modifier returns value unchanged",
			value:    "getValue",
			modifier: "",
			expected: "getValue",
		},

		// Unknown modifier
		{
			name:     "unknown modifier returns value unchanged",
			value:    "getValue",
			modifier: "unknown",
			expected: "getValue",
		},

		// Edge cases
		{
			name:     "empty value",
			value:    "",
			modifier: "pascal",
			expected: "",
		},
		{
			name:     "single character",
			value:    "a",
			modifier: "upper",
			expected: "A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyModifier(tt.value, tt.modifier)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Path Sanitization Tests
// =============================================================================

// TestSanitizePath tests converting path templates to safe identifier components
func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		separator string
		expected  string
	}{
		// Basic cases
		{
			name:      "simple path",
			path:      "/users",
			separator: "_",
			expected:  "users",
		},
		{
			name:      "path with single param",
			path:      "/users/{id}",
			separator: "_",
			expected:  "users_id",
		},
		{
			name:      "path with multiple params",
			path:      "/users/{id}/posts",
			separator: "_",
			expected:  "users_id_posts",
		},
		{
			name:      "path with nested params",
			path:      "/users/{userId}/posts/{postId}",
			separator: "_",
			expected:  "users_userId_posts_postId",
		},

		// Leading slash removed
		{
			name:      "leading slash removed",
			path:      "/api/v1/users",
			separator: "_",
			expected:  "api_v1_users",
		},

		// Path params braces removed
		{
			name:      "braces removed",
			path:      "/items/{itemId}",
			separator: "_",
			expected:  "items_itemId",
		},

		// Custom separators
		{
			name:      "custom separator hyphen",
			path:      "/users/{id}/posts",
			separator: "-",
			expected:  "users-id-posts",
		},
		{
			name:      "custom separator dot",
			path:      "/users/{id}",
			separator: ".",
			expected:  "users.id",
		},
		{
			name:      "empty separator defaults to underscore",
			path:      "/users/{id}",
			separator: "",
			expected:  "users_id",
		},

		// Edge cases
		{
			name:      "empty path",
			path:      "",
			separator: "_",
			expected:  "",
		},
		{
			name:      "root path only",
			path:      "/",
			separator: "_",
			expected:  "",
		},
		{
			name:      "path with numbers",
			path:      "/api/v2/items",
			separator: "_",
			expected:  "api_v2_items",
		},
		{
			name:      "path with hyphens",
			path:      "/user-profiles/{user-id}",
			separator: "_",
			expected:  "user-profiles_user-id",
		},
		{
			name:      "path with underscores",
			path:      "/user_profiles/{user_id}",
			separator: "_",
			expected:  "user_profiles_user_id",
		},
		{
			name:      "multiple consecutive slashes",
			path:      "/users//posts",
			separator: "_",
			expected:  "users_posts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePath(tt.path, tt.separator)
			assert.Equal(t, tt.expected, result)
		})
	}
}

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
// Helper Function Tests
// =============================================================================

// TestGetSortedMethods tests that methods are returned in consistent order
func TestGetSortedMethods(t *testing.T) {
	// Create operations map with various methods
	operations := map[string]*parser.Operation{
		"post":    {OperationID: "post"},
		"get":     {OperationID: "get"},
		"delete":  {OperationID: "delete"},
		"put":     {OperationID: "put"},
		"patch":   {OperationID: "patch"},
		"options": {OperationID: "options"},
		"head":    {OperationID: "head"},
		"trace":   {OperationID: "trace"},
	}

	methods := getSortedMethods(operations)

	// Should be in standard order: get, put, post, delete, options, head, patch, trace
	assert.Equal(t, []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}, methods)
}

// TestGetSortedMethods_CustomMethods tests sorting with custom/non-standard methods
func TestGetSortedMethods_CustomMethods(t *testing.T) {
	operations := map[string]*parser.Operation{
		"get":    {OperationID: "get"},
		"custom": {OperationID: "custom"},
		"zzz":    {OperationID: "zzz"},
		"aaa":    {OperationID: "aaa"},
	}

	methods := getSortedMethods(operations)

	// Standard methods first (get), then custom methods sorted alphabetically (aaa, custom, zzz)
	assert.Equal(t, []string{"get", "aaa", "custom", "zzz"}, methods)
}

// TestGetSortedMethods_NilOperations tests that nil operations are excluded
func TestGetSortedMethods_NilOperations(t *testing.T) {
	operations := map[string]*parser.Operation{
		"get":  {OperationID: "get"},
		"post": nil,
		"put":  {OperationID: "put"},
	}

	methods := getSortedMethods(operations)

	// Should only include non-nil operations
	assert.Equal(t, []string{"get", "put"}, methods)
}

// TestGetSortedMethods_Empty tests empty operations map
func TestGetSortedMethods_Empty(t *testing.T) {
	operations := map[string]*parser.Operation{}

	methods := getSortedMethods(operations)

	assert.Empty(t, methods)
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
