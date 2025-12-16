package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// RefCollector Tests
// =============================================================================

// TestRefCollectorOAS3 tests collecting refs from OAS 3.x documents
func TestRefCollectorOAS3(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      parameters:
        - $ref: "#/components/parameters/PageParam"
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/UserList"
        default:
          $ref: "#/components/responses/ErrorResponse"
    post:
      requestBody:
        $ref: "#/components/requestBodies/CreateUser"
      responses:
        "201":
          description: Created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
components:
  schemas:
    UserList:
      type: array
      items:
        $ref: "#/components/schemas/User"
    User:
      type: object
      properties:
        id:
          type: integer
        address:
          $ref: "#/components/schemas/Address"
    Address:
      type: object
      properties:
        city:
          type: string
    Orphan:
      type: object
  parameters:
    PageParam:
      name: page
      in: query
      schema:
        type: integer
  responses:
    ErrorResponse:
      description: Error
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"
  requestBodies:
    CreateUser:
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/User"
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	doc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	// Collect refs
	collector := NewRefCollector()
	collector.CollectOAS3(doc)

	// Verify schema refs were collected
	assert.True(t, collector.IsSchemaReferenced("UserList", parser.OASVersion303))
	assert.True(t, collector.IsSchemaReferenced("User", parser.OASVersion303))
	assert.True(t, collector.IsSchemaReferenced("Address", parser.OASVersion303))
	assert.False(t, collector.IsSchemaReferenced("Orphan", parser.OASVersion303))

	// Verify parameter refs were collected
	assert.True(t, collector.IsParameterReferenced("PageParam", parser.OASVersion303))

	// Verify response refs were collected
	assert.True(t, collector.IsResponseReferenced("ErrorResponse", parser.OASVersion303))

	// Verify request body refs were collected
	assert.True(t, collector.IsRequestBodyReferenced("CreateUser"))

	// Verify GetSchemaRefs returns all schema refs
	schemaRefs := collector.GetSchemaRefs()
	assert.Contains(t, schemaRefs, "#/components/schemas/UserList")
	assert.Contains(t, schemaRefs, "#/components/schemas/User")
	assert.Contains(t, schemaRefs, "#/components/schemas/Address")
}

// TestRefCollectorOAS2 tests collecting refs from OAS 2.0 documents
func TestRefCollectorOAS2(t *testing.T) {
	spec := `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      parameters:
        - $ref: "#/parameters/PageParam"
      responses:
        "200":
          description: Success
          schema:
            $ref: "#/definitions/UserList"
        default:
          $ref: "#/responses/ErrorResponse"
    post:
      operationId: createUser
      parameters:
        - name: body
          in: body
          schema:
            $ref: "#/definitions/User"
      responses:
        "201":
          description: Created
definitions:
  UserList:
    type: array
    items:
      $ref: "#/definitions/User"
  User:
    type: object
    properties:
      id:
        type: integer
      profile:
        $ref: "#/definitions/Profile"
  Profile:
    type: object
  UnusedDef:
    type: object
parameters:
  PageParam:
    name: page
    in: query
    type: integer
responses:
  ErrorResponse:
    description: Error
    schema:
      $ref: "#/definitions/Error"
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	doc, ok := parseResult.OAS2Document()
	require.True(t, ok)

	// Collect refs
	collector := NewRefCollector()
	collector.CollectOAS2(doc)

	// Verify schema refs were collected
	assert.True(t, collector.IsSchemaReferenced("UserList", parser.OASVersion20))
	assert.True(t, collector.IsSchemaReferenced("User", parser.OASVersion20))
	assert.True(t, collector.IsSchemaReferenced("Profile", parser.OASVersion20))
	assert.False(t, collector.IsSchemaReferenced("UnusedDef", parser.OASVersion20))

	// Verify parameter refs were collected
	assert.True(t, collector.IsParameterReferenced("PageParam", parser.OASVersion20))

	// Verify response refs were collected
	assert.True(t, collector.IsResponseReferenced("ErrorResponse", parser.OASVersion20))
}

// TestRefCollector_GetUnreferencedSchemas tests the GetUnreferencedSchemas helper
func TestRefCollector_GetUnreferencedSchemas(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /items:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Item"
components:
  schemas:
    Item:
      type: object
    Orphan1:
      type: string
    Orphan2:
      type: integer
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	doc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	// Collect refs
	collector := NewRefCollector()
	collector.CollectOAS3(doc)

	// Get unreferenced schemas
	unreferenced := collector.GetUnreferencedSchemas(doc)
	assert.Len(t, unreferenced, 2)
	assert.Contains(t, unreferenced, "Orphan1")
	assert.Contains(t, unreferenced, "Orphan2")
	assert.NotContains(t, unreferenced, "Item")
}

// TestRefCollector_CircularRefs tests that circular refs don't cause infinite loops
func TestRefCollector_CircularRefs(t *testing.T) {
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
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	doc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	// Collect refs - should not hang
	collector := NewRefCollector()
	collector.CollectOAS3(doc)

	// Node should be referenced
	assert.True(t, collector.IsSchemaReferenced("Node", parser.OASVersion303))
}

// TestRefCollector_AllOfAnyOfOneOf tests ref collection from composition keywords
func TestRefCollector_AllOfAnyOfOneOf(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /items:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ComposedType"
components:
  schemas:
    ComposedType:
      allOf:
        - $ref: "#/components/schemas/BaseType"
        - $ref: "#/components/schemas/ExtType"
      anyOf:
        - $ref: "#/components/schemas/OptionA"
      oneOf:
        - $ref: "#/components/schemas/OptionB"
    BaseType:
      type: object
    ExtType:
      type: object
    OptionA:
      type: object
    OptionB:
      type: object
    Unused:
      type: string
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	doc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	// Collect refs
	collector := NewRefCollector()
	collector.CollectOAS3(doc)

	// All composed types should be referenced
	assert.True(t, collector.IsSchemaReferenced("ComposedType", parser.OASVersion303))
	assert.True(t, collector.IsSchemaReferenced("BaseType", parser.OASVersion303))
	assert.True(t, collector.IsSchemaReferenced("ExtType", parser.OASVersion303))
	assert.True(t, collector.IsSchemaReferenced("OptionA", parser.OASVersion303))
	assert.True(t, collector.IsSchemaReferenced("OptionB", parser.OASVersion303))
	assert.False(t, collector.IsSchemaReferenced("Unused", parser.OASVersion303))
}

// TestRefCollector_Webhooks tests ref collection from webhooks (OAS 3.1+)
func TestRefCollector_Webhooks(t *testing.T) {
	spec := `
openapi: "3.1.0"
info:
  title: Test API
  version: "1.0"
paths: {}
webhooks:
  userCreated:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/WebhookPayload"
      responses:
        "200":
          description: OK
components:
  schemas:
    WebhookPayload:
      type: object
    Unused:
      type: string
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	doc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	// Collect refs
	collector := NewRefCollector()
	collector.CollectOAS3(doc)

	// WebhookPayload should be referenced via webhooks
	assert.True(t, collector.IsSchemaReferenced("WebhookPayload", parser.OASVersion310))
	assert.False(t, collector.IsSchemaReferenced("Unused", parser.OASVersion310))
}

// TestRefCollector_Callbacks tests ref collection from callbacks
func TestRefCollector_Callbacks(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /subscribe:
    post:
      callbacks:
        onEvent:
          "{$request.body#/callbackUrl}":
            post:
              requestBody:
                content:
                  application/json:
                    schema:
                      $ref: "#/components/schemas/EventPayload"
              responses:
                "200":
                  description: OK
      responses:
        "201":
          description: Subscribed
components:
  schemas:
    EventPayload:
      type: object
    Unused:
      type: string
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	doc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	// Collect refs
	collector := NewRefCollector()
	collector.CollectOAS3(doc)

	// EventPayload should be referenced via callbacks
	assert.True(t, collector.IsSchemaReferenced("EventPayload", parser.OASVersion303))
	assert.False(t, collector.IsSchemaReferenced("Unused", parser.OASVersion303))
}

// TestRefCollector_Headers tests ref collection from response headers
func TestRefCollector_Headers(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /items:
    get:
      responses:
        "200":
          description: Success
          headers:
            X-Rate-Limit:
              $ref: "#/components/headers/RateLimitHeader"
          content:
            application/json:
              schema:
                type: array
components:
  headers:
    RateLimitHeader:
      description: Rate limit info
      schema:
        $ref: "#/components/schemas/RateLimit"
  schemas:
    RateLimit:
      type: object
      properties:
        limit:
          type: integer
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	doc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	// Collect refs
	collector := NewRefCollector()
	collector.CollectOAS3(doc)

	// Header and schema should be referenced
	assert.True(t, collector.IsHeaderReferenced("RateLimitHeader", parser.OASVersion303))
	assert.True(t, collector.IsSchemaReferenced("RateLimit", parser.OASVersion303))
}

// TestRefCollector_NilDocument tests that nil documents don't panic
func TestRefCollector_NilDocument(t *testing.T) {
	collector := NewRefCollector()

	// Should not panic
	collector.CollectOAS2(nil)
	collector.CollectOAS3(nil)

	assert.Empty(t, collector.Refs)
}

// TestRefCollector_DiscriminatorMapping tests ref collection from discriminator mappings
func TestRefCollector_DiscriminatorMapping(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /pets:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: petType
        mapping:
          dog: "#/components/schemas/Dog"
          cat: "#/components/schemas/Cat"
      oneOf:
        - $ref: "#/components/schemas/Dog"
        - $ref: "#/components/schemas/Cat"
    Dog:
      type: object
      properties:
        petType:
          type: string
        bark:
          type: boolean
    Cat:
      type: object
      properties:
        petType:
          type: string
        meow:
          type: boolean
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	doc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	// Collect refs
	collector := NewRefCollector()
	collector.CollectOAS3(doc)

	// Pet, Dog, and Cat should all be referenced
	assert.True(t, collector.IsSchemaReferenced("Pet", parser.OASVersion303))
	assert.True(t, collector.IsSchemaReferenced("Dog", parser.OASVersion303))
	assert.True(t, collector.IsSchemaReferenced("Cat", parser.OASVersion303))
}

// =============================================================================
// Reference Helper Tests
// =============================================================================

// TestExtractSchemaNameFromRef tests the ExtractSchemaNameFromRef function
func TestExtractSchemaNameFromRef(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		version  parser.OASVersion
		expected string
	}{
		{
			name:     "OAS 3.x schema ref",
			ref:      "#/components/schemas/User",
			version:  parser.OASVersion303,
			expected: "User",
		},
		{
			name:     "OAS 2.0 definition ref",
			ref:      "#/definitions/User",
			version:  parser.OASVersion20,
			expected: "User",
		},
		{
			name:     "non-schema ref OAS 3.x",
			ref:      "#/components/parameters/PageParam",
			version:  parser.OASVersion303,
			expected: "",
		},
		{
			name:     "non-schema ref OAS 2.0",
			ref:      "#/parameters/PageParam",
			version:  parser.OASVersion20,
			expected: "",
		},
		{
			name:     "external ref",
			ref:      "external.yaml#/components/schemas/User",
			version:  parser.OASVersion303,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractSchemaNameFromRef(tt.ref, tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractComponentNameFromRef tests the ExtractComponentNameFromRef function
func TestExtractComponentNameFromRef(t *testing.T) {
	tests := []struct {
		name         string
		ref          string
		expectedType string
		expectedName string
	}{
		{
			name:         "OAS 3.x schema",
			ref:          "#/components/schemas/User",
			expectedType: "schema",
			expectedName: "User",
		},
		{
			name:         "OAS 3.x parameter",
			ref:          "#/components/parameters/PageParam",
			expectedType: "parameter",
			expectedName: "PageParam",
		},
		{
			name:         "OAS 3.x response",
			ref:          "#/components/responses/NotFound",
			expectedType: "response",
			expectedName: "NotFound",
		},
		{
			name:         "OAS 3.x requestBody",
			ref:          "#/components/requestBodies/CreateUser",
			expectedType: "requestBody",
			expectedName: "CreateUser",
		},
		{
			name:         "OAS 3.x header",
			ref:          "#/components/headers/RateLimit",
			expectedType: "header",
			expectedName: "RateLimit",
		},
		{
			name:         "OAS 3.x securityScheme",
			ref:          "#/components/securitySchemes/BearerAuth",
			expectedType: "securityScheme",
			expectedName: "BearerAuth",
		},
		{
			name:         "OAS 3.x link",
			ref:          "#/components/links/GetUserById",
			expectedType: "link",
			expectedName: "GetUserById",
		},
		{
			name:         "OAS 3.x callback",
			ref:          "#/components/callbacks/OnEvent",
			expectedType: "callback",
			expectedName: "OnEvent",
		},
		{
			name:         "OAS 3.x example",
			ref:          "#/components/examples/UserExample",
			expectedType: "example",
			expectedName: "UserExample",
		},
		{
			name:         "OAS 3.1 pathItem",
			ref:          "#/components/pathItems/SharedPath",
			expectedType: "pathItem",
			expectedName: "SharedPath",
		},
		{
			name:         "OAS 2.0 definition",
			ref:          "#/definitions/User",
			expectedType: "schema",
			expectedName: "User",
		},
		{
			name:         "OAS 2.0 parameter",
			ref:          "#/parameters/PageParam",
			expectedType: "parameter",
			expectedName: "PageParam",
		},
		{
			name:         "OAS 2.0 response",
			ref:          "#/responses/NotFound",
			expectedType: "response",
			expectedName: "NotFound",
		},
		{
			name:         "OAS 2.0 securityDefinition",
			ref:          "#/securityDefinitions/ApiKey",
			expectedType: "securityScheme",
			expectedName: "ApiKey",
		},
		{
			name:         "invalid ref",
			ref:          "not-a-ref",
			expectedType: "",
			expectedName: "",
		},
		{
			name:         "external ref",
			ref:          "external.yaml#/components/schemas/User",
			expectedType: "",
			expectedName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compType, name := ExtractComponentNameFromRef(tt.ref)
			assert.Equal(t, tt.expectedType, compType)
			assert.Equal(t, tt.expectedName, name)
		})
	}
}

// TestRefType_String tests the RefType String method
func TestRefType_String(t *testing.T) {
	tests := []struct {
		refType  RefType
		expected string
	}{
		{RefTypeSchema, "schema"},
		{RefTypeParameter, "parameter"},
		{RefTypeResponse, "response"},
		{RefTypeRequestBody, "requestBody"},
		{RefTypeHeader, "header"},
		{RefTypeSecurityScheme, "securityScheme"},
		{RefTypeLink, "link"},
		{RefTypeCallback, "callback"},
		{RefTypeExample, "example"},
		{RefTypePathItem, "pathItem"},
		{RefType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.refType.String())
		})
	}
}

// TestIsHeaderReferenced_OAS2 tests that OAS 2.0 doesn't support global headers
func TestIsHeaderReferenced_OAS2(t *testing.T) {
	collector := NewRefCollector()

	// OAS 2.0 doesn't have global header definitions
	assert.False(t, collector.IsHeaderReferenced("SomeHeader", parser.OASVersion20))
}

// TestIsLinkReferenced tests the IsLinkReferenced method
func TestIsLinkReferenced(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users/{id}:
    get:
      responses:
        "200":
          description: Success
          links:
            GetUserPosts:
              $ref: "#/components/links/GetUserPosts"
components:
  links:
    GetUserPosts:
      operationId: getUserPosts
      parameters:
        userId: "$response.body#/id"
    UnusedLink:
      operationId: unused
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	doc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	// Collect refs
	collector := NewRefCollector()
	collector.CollectOAS3(doc)

	assert.True(t, collector.IsLinkReferenced("GetUserPosts"))
	assert.False(t, collector.IsLinkReferenced("UnusedLink"))
}

// TestIsCallbackReferenced tests the IsCallbackReferenced method
func TestIsCallbackReferenced(t *testing.T) {
	// Note: Callbacks are maps of expressions to path items, not $ref targets.
	// The callback reference is typically within components where another
	// component references it. In practice, callbacks are referenced by name
	// in the operation's callbacks map - we test the method directly.
	collector := NewRefCollector()

	// Manually add a callback ref to test the IsCallbackReferenced method
	collector.addRef("#/components/callbacks/OnEvent", "paths./subscribe.post.callbacks.onEvent", RefTypeCallback)

	assert.True(t, collector.IsCallbackReferenced("OnEvent"))
	assert.False(t, collector.IsCallbackReferenced("UnusedCallback"))
}

// TestIsExampleReferenced tests the IsExampleReferenced method
func TestIsExampleReferenced(t *testing.T) {
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
                type: object
              examples:
                sample:
                  $ref: "#/components/examples/UserExample"
components:
  examples:
    UserExample:
      value:
        id: 1
        name: John
    UnusedExample:
      value:
        unused: true
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	doc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	// Collect refs
	collector := NewRefCollector()
	collector.CollectOAS3(doc)

	assert.True(t, collector.IsExampleReferenced("UserExample"))
	assert.False(t, collector.IsExampleReferenced("UnusedExample"))
}

// TestIsPathItemReferenced tests the IsPathItemReferenced method
func TestIsPathItemReferenced(t *testing.T) {
	spec := `
openapi: "3.1.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    $ref: "#/components/pathItems/UsersPath"
components:
  pathItems:
    UsersPath:
      get:
        responses:
          "200":
            description: Success
    UnusedPath:
      get:
        responses:
          "200":
            description: Unused
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	doc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	// Collect refs
	collector := NewRefCollector()
	collector.CollectOAS3(doc)

	assert.True(t, collector.IsPathItemReferenced("UsersPath"))
	assert.False(t, collector.IsPathItemReferenced("UnusedPath"))
}
