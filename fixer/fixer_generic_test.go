package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Generic Schema Name Fixing Tests
// =============================================================================

// TestFixInvalidSchemaNamesOAS3 tests renaming schemas with invalid characters in OAS 3.x
func TestFixInvalidSchemaNamesOAS3(t *testing.T) {
	tests := []struct {
		name           string
		spec           string
		strategy       GenericNamingStrategy
		expectedSchema string
		expectedRef    string
		expectFix      bool
	}{
		{
			name: "brackets renamed with underscore strategy",
			spec: `
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
                $ref: "#/components/schemas/Response[User]"
components:
  schemas:
    Response[User]:
      type: object
      properties:
        data:
          $ref: "#/components/schemas/User"
    User:
      type: object
      properties:
        id:
          type: integer
`,
			strategy:       GenericNamingUnderscore,
			expectedSchema: "Response_User_",
			expectedRef:    "#/components/schemas/Response_User_",
			expectFix:      true,
		},
		{
			name: "brackets renamed with of strategy",
			spec: `
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
                $ref: "#/components/schemas/Response[User]"
components:
  schemas:
    Response[User]:
      type: object
      properties:
        data:
          $ref: "#/components/schemas/User"
    User:
      type: object
      properties:
        id:
          type: integer
`,
			strategy:       GenericNamingOf,
			expectedSchema: "ResponseOfUser",
			expectedRef:    "#/components/schemas/ResponseOfUser",
			expectFix:      true,
		},
		{
			name: "brackets renamed with for strategy",
			spec: `
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
                $ref: "#/components/schemas/List[Item]"
components:
  schemas:
    List[Item]:
      type: array
      items:
        $ref: "#/components/schemas/Item"
    Item:
      type: object
`,
			strategy:       GenericNamingFor,
			expectedSchema: "ListForItem",
			expectedRef:    "#/components/schemas/ListForItem",
			expectFix:      true,
		},
		{
			name: "brackets renamed with flattened strategy",
			spec: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /data:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Container[Value]"
components:
  schemas:
    Container[Value]:
      type: object
    Value:
      type: string
`,
			strategy:       GenericNamingFlattened,
			expectedSchema: "ContainerValue",
			expectedRef:    "#/components/schemas/ContainerValue",
			expectFix:      true,
		},
		{
			name: "brackets renamed with dot strategy",
			spec: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /data:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Wrapper[Data]"
components:
  schemas:
    Wrapper[Data]:
      type: object
    Data:
      type: string
`,
			strategy:       GenericNamingDot,
			expectedSchema: "Wrapper.Data",
			expectedRef:    "#/components/schemas/Wrapper.Data",
			expectFix:      true,
		},
		{
			name: "valid schema names not modified",
			spec: `
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
        data:
          $ref: "#/components/schemas/User"
    User:
      type: object
`,
			strategy:       GenericNamingOf,
			expectedSchema: "UserResponse",
			expectedRef:    "#/components/schemas/UserResponse",
			expectFix:      false,
		},
		{
			name: "angle brackets renamed",
			spec: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /data:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/List<Item>"
components:
  schemas:
    List<Item>:
      type: array
    Item:
      type: object
`,
			strategy:       GenericNamingOf,
			expectedSchema: "ListOfItem",
			expectedRef:    "#/components/schemas/ListOfItem",
			expectFix:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(tt.spec)))
			require.NoError(t, err)

			// Fix with specific strategy
			f := New()
			f.EnabledFixes = []FixType{FixTypeRenamedGenericSchema}
			f.GenericNamingConfig.Strategy = tt.strategy
			result, err := f.FixParsed(*parseResult)
			require.NoError(t, err)

			// Assert
			doc := result.Document.(*parser.OAS3Document)

			if tt.expectFix {
				assert.True(t, result.HasFixes(), "expected fixes to be applied")
				assert.Contains(t, doc.Components.Schemas, tt.expectedSchema,
					"expected schema %s to exist", tt.expectedSchema)

				// Verify the ref was rewritten in paths
				pathItem := doc.Paths["/users"]
				if pathItem == nil {
					pathItem = doc.Paths["/data"]
				}
				require.NotNil(t, pathItem)
				require.NotNil(t, pathItem.Get)
				respContent := pathItem.Get.Responses.Codes["200"].Content["application/json"]
				assert.Equal(t, tt.expectedRef, respContent.Schema.Ref)
			} else {
				assert.False(t, result.HasFixes(), "expected no fixes to be applied")
				assert.Contains(t, doc.Components.Schemas, tt.expectedSchema)
			}
		})
	}
}

// TestFixInvalidSchemaNamesOAS2 tests renaming schemas with brackets in OAS 2.0
func TestFixInvalidSchemaNamesOAS2(t *testing.T) {
	tests := []struct {
		name           string
		spec           string
		strategy       GenericNamingStrategy
		expectedSchema string
		expectedRef    string
		expectFix      bool
	}{
		{
			name: "brackets renamed with underscore strategy",
			spec: `
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
            $ref: "#/definitions/Response[User]"
definitions:
  Response[User]:
    type: object
    properties:
      data:
        $ref: "#/definitions/User"
  User:
    type: object
    properties:
      id:
        type: integer
`,
			strategy:       GenericNamingUnderscore,
			expectedSchema: "Response_User_",
			expectedRef:    "#/definitions/Response_User_",
			expectFix:      true,
		},
		{
			name: "brackets renamed with of strategy",
			spec: `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /items:
    get:
      operationId: getItems
      produces:
        - application/json
      responses:
        "200":
          description: Success
          schema:
            $ref: "#/definitions/List[Item]"
definitions:
  List[Item]:
    type: array
    items:
      $ref: "#/definitions/Item"
  Item:
    type: object
`,
			strategy:       GenericNamingOf,
			expectedSchema: "ListOfItem",
			expectedRef:    "#/definitions/ListOfItem",
			expectFix:      true,
		},
		{
			name: "valid schema names not modified",
			spec: `
swagger: "2.0"
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
          schema:
            $ref: "#/definitions/UserList"
definitions:
  UserList:
    type: array
    items:
      $ref: "#/definitions/User"
  User:
    type: object
`,
			strategy:       GenericNamingOf,
			expectedSchema: "UserList",
			expectedRef:    "#/definitions/UserList",
			expectFix:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(tt.spec)))
			require.NoError(t, err)

			// Fix with specific strategy
			f := New()
			f.EnabledFixes = []FixType{FixTypeRenamedGenericSchema}
			f.GenericNamingConfig.Strategy = tt.strategy
			result, err := f.FixParsed(*parseResult)
			require.NoError(t, err)

			// Assert
			doc := result.Document.(*parser.OAS2Document)

			if tt.expectFix {
				assert.True(t, result.HasFixes(), "expected fixes to be applied")
				assert.Contains(t, doc.Definitions, tt.expectedSchema,
					"expected definition %s to exist", tt.expectedSchema)

				// Verify the ref was rewritten in responses
				pathItem := doc.Paths["/users"]
				if pathItem == nil {
					pathItem = doc.Paths["/items"]
				}
				require.NotNil(t, pathItem)
				require.NotNil(t, pathItem.Get)
				assert.Equal(t, tt.expectedRef, pathItem.Get.Responses.Codes["200"].Schema.Ref)
			} else {
				assert.False(t, result.HasFixes(), "expected no fixes to be applied")
				assert.Contains(t, doc.Definitions, tt.expectedSchema)
			}
		})
	}
}

// TestFixNestedGenericTypesOAS3 tests renaming nested generic types
func TestFixNestedGenericTypesOAS3(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /data:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Response[List[User]]"
components:
  schemas:
    Response[List[User]]:
      type: object
      properties:
        data:
          $ref: "#/components/schemas/List[User]"
    List[User]:
      type: array
      items:
        $ref: "#/components/schemas/User"
    User:
      type: object
      properties:
        id:
          type: integer
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix with "of" strategy
	f := New()
	f.EnabledFixes = []FixType{FixTypeRenamedGenericSchema}
	f.GenericNamingConfig.Strategy = GenericNamingOf
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert - nested generics should be transformed recursively
	doc := result.Document.(*parser.OAS3Document)

	// Should have 2 fixes (Response[List[User]] and List[User])
	assert.Equal(t, 2, result.FixCount)

	// Check the transformed names exist
	assert.Contains(t, doc.Components.Schemas, "ResponseOfListOfUser")
	assert.Contains(t, doc.Components.Schemas, "ListOfUser")
	assert.Contains(t, doc.Components.Schemas, "User") // unchanged

	// Verify refs were rewritten
	responseSchema := doc.Components.Schemas["ResponseOfListOfUser"]
	assert.Equal(t, "#/components/schemas/ListOfUser", responseSchema.Properties["data"].Ref)
}

// TestFixGenericSchemaNameCollision tests that name collisions are handled
func TestFixGenericSchemaNameCollision(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /data:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Response[User]"
components:
  schemas:
    Response[User]:
      type: object
      properties:
        data:
          type: string
    ResponseOfUser:
      type: object
      properties:
        existing:
          type: boolean
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix - should avoid collision with existing ResponseOfUser
	f := New()
	f.EnabledFixes = []FixType{FixTypeRenamedGenericSchema}
	f.GenericNamingConfig.Strategy = GenericNamingOf
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS3Document)
	assert.True(t, result.HasFixes())

	// Original ResponseOfUser should still exist
	assert.Contains(t, doc.Components.Schemas, "ResponseOfUser")

	// Renamed schema should have numeric suffix to avoid collision
	assert.Contains(t, doc.Components.Schemas, "ResponseOfUser2")

	// Response[User] should be gone
	assert.NotContains(t, doc.Components.Schemas, "Response[User]")
}
