package joiner

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchemaRewriter(t *testing.T) {
	rewriter := NewSchemaRewriter()

	assert.NotNil(t, rewriter)
	assert.NotNil(t, rewriter.refMap)
	assert.NotNil(t, rewriter.bareNameMap)
	assert.NotNil(t, rewriter.visited)
	assert.Equal(t, 0, len(rewriter.refMap))
	assert.Equal(t, 0, len(rewriter.bareNameMap))
}

func TestSchemaRewriter_RegisterRename_OAS3(t *testing.T) {
	rewriter := NewSchemaRewriter()

	rewriter.RegisterRename("User", "User_left", parser.OASVersion300)

	assert.Equal(t, "#/components/schemas/User_left", rewriter.refMap["#/components/schemas/User"])
	assert.Equal(t, "User_left", rewriter.bareNameMap["User"])
}

func TestSchemaRewriter_RegisterRename_OAS2(t *testing.T) {
	rewriter := NewSchemaRewriter()

	rewriter.RegisterRename("User", "User_left", parser.OASVersion20)

	assert.Equal(t, "#/definitions/User_left", rewriter.refMap["#/definitions/User"])
	assert.Equal(t, "User_left", rewriter.bareNameMap["User"])
}

func TestSchemaRewriter_RegisterMultipleRenames(t *testing.T) {
	rewriter := NewSchemaRewriter()

	rewriter.RegisterRename("User", "User_left", parser.OASVersion300)
	rewriter.RegisterRename("Product", "Product_right", parser.OASVersion300)
	rewriter.RegisterRename("Order", "Order_v2", parser.OASVersion300)

	assert.Equal(t, 3, len(rewriter.refMap))
	assert.Equal(t, 3, len(rewriter.bareNameMap))
	assert.Equal(t, "#/components/schemas/User_left", rewriter.refMap["#/components/schemas/User"])
	assert.Equal(t, "#/components/schemas/Product_right", rewriter.refMap["#/components/schemas/Product"])
	assert.Equal(t, "#/components/schemas/Order_v2", rewriter.refMap["#/components/schemas/Order"])
}

func TestSchemaRewriter_RewriteSchema_SimpleRef(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("User", "User_left", parser.OASVersion300)

	schema := &parser.Schema{
		Ref: "#/components/schemas/User",
	}

	rewriter.rewriteSchema(schema)

	assert.Equal(t, "#/components/schemas/User_left", schema.Ref)
}

func TestSchemaRewriter_RewriteSchema_Properties(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("Address", "Address_v2", parser.OASVersion300)

	schema := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
			"address": {
				Ref: "#/components/schemas/Address",
			},
		},
	}

	rewriter.rewriteSchema(schema)

	assert.Equal(t, "#/components/schemas/Address_v2", schema.Properties["address"].Ref)
}

func TestSchemaRewriter_RewriteSchema_NestedProperties(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("City", "City_updated", parser.OASVersion300)

	schema := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"user": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"address": {
						Type: "object",
						Properties: map[string]*parser.Schema{
							"city": {
								Ref: "#/components/schemas/City",
							},
						},
					},
				},
			},
		},
	}

	rewriter.rewriteSchema(schema)

	assert.Equal(t, "#/components/schemas/City_updated",
		schema.Properties["user"].Properties["address"].Properties["city"].Ref)
}

func TestSchemaRewriter_RewriteSchema_Items(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("User", "User_v2", parser.OASVersion300)

	schema := &parser.Schema{
		Type: "array",
		Items: &parser.Schema{
			Ref: "#/components/schemas/User",
		},
	}

	rewriter.rewriteSchema(schema)

	itemsSchema := schema.Items.(*parser.Schema)
	assert.Equal(t, "#/components/schemas/User_v2", itemsSchema.Ref)
}

func TestSchemaRewriter_RewriteSchema_AdditionalProperties(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("Value", "Value_new", parser.OASVersion300)

	schema := &parser.Schema{
		Type: "object",
		AdditionalProperties: &parser.Schema{
			Ref: "#/components/schemas/Value",
		},
	}

	rewriter.rewriteSchema(schema)

	addProp := schema.AdditionalProperties.(*parser.Schema)
	assert.Equal(t, "#/components/schemas/Value_new", addProp.Ref)
}

func TestSchemaRewriter_RewriteSchema_AllOf(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("Base", "Base_v2", parser.OASVersion300)
	rewriter.RegisterRename("Extended", "Extended_v2", parser.OASVersion300)

	schema := &parser.Schema{
		AllOf: []*parser.Schema{
			{Ref: "#/components/schemas/Base"},
			{Ref: "#/components/schemas/Extended"},
		},
	}

	rewriter.rewriteSchema(schema)

	assert.Equal(t, "#/components/schemas/Base_v2", schema.AllOf[0].Ref)
	assert.Equal(t, "#/components/schemas/Extended_v2", schema.AllOf[1].Ref)
}

func TestSchemaRewriter_RewriteSchema_AnyOf(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("TypeA", "TypeA_new", parser.OASVersion300)
	rewriter.RegisterRename("TypeB", "TypeB_new", parser.OASVersion300)

	schema := &parser.Schema{
		AnyOf: []*parser.Schema{
			{Ref: "#/components/schemas/TypeA"},
			{Ref: "#/components/schemas/TypeB"},
		},
	}

	rewriter.rewriteSchema(schema)

	assert.Equal(t, "#/components/schemas/TypeA_new", schema.AnyOf[0].Ref)
	assert.Equal(t, "#/components/schemas/TypeB_new", schema.AnyOf[1].Ref)
}

func TestSchemaRewriter_RewriteSchema_OneOf(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("Dog", "Dog_v2", parser.OASVersion300)
	rewriter.RegisterRename("Cat", "Cat_v2", parser.OASVersion300)

	schema := &parser.Schema{
		OneOf: []*parser.Schema{
			{Ref: "#/components/schemas/Dog"},
			{Ref: "#/components/schemas/Cat"},
		},
	}

	rewriter.rewriteSchema(schema)

	assert.Equal(t, "#/components/schemas/Dog_v2", schema.OneOf[0].Ref)
	assert.Equal(t, "#/components/schemas/Cat_v2", schema.OneOf[1].Ref)
}

func TestSchemaRewriter_RewriteSchema_Not(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("Invalid", "Invalid_v2", parser.OASVersion300)

	schema := &parser.Schema{
		Not: &parser.Schema{
			Ref: "#/components/schemas/Invalid",
		},
	}

	rewriter.rewriteSchema(schema)

	assert.Equal(t, "#/components/schemas/Invalid_v2", schema.Not.Ref)
}

func TestSchemaRewriter_RewriteSchema_Discriminator_FullPath(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("Dog", "Dog_v2", parser.OASVersion300)
	rewriter.RegisterRename("Cat", "Cat_v2", parser.OASVersion300)

	schema := &parser.Schema{
		Discriminator: &parser.Discriminator{
			PropertyName: "petType",
			Mapping: map[string]string{
				"dog": "#/components/schemas/Dog",
				"cat": "#/components/schemas/Cat",
			},
		},
	}

	rewriter.rewriteSchema(schema)

	assert.Equal(t, "#/components/schemas/Dog_v2", schema.Discriminator.Mapping["dog"])
	assert.Equal(t, "#/components/schemas/Cat_v2", schema.Discriminator.Mapping["cat"])
}

func TestSchemaRewriter_RewriteSchema_Discriminator_BareName(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("Dog", "Dog_v2", parser.OASVersion300)
	rewriter.RegisterRename("Cat", "Cat_v2", parser.OASVersion300)

	schema := &parser.Schema{
		Discriminator: &parser.Discriminator{
			PropertyName: "petType",
			Mapping: map[string]string{
				"dog": "Dog",
				"cat": "Cat",
			},
		},
	}

	rewriter.rewriteSchema(schema)

	assert.Equal(t, "Dog_v2", schema.Discriminator.Mapping["dog"])
	assert.Equal(t, "Cat_v2", schema.Discriminator.Mapping["cat"])
}

func TestSchemaRewriter_RewriteSchema_Discriminator_Mixed(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("Dog", "Dog_v2", parser.OASVersion300)
	rewriter.RegisterRename("Cat", "Cat_v2", parser.OASVersion300)

	schema := &parser.Schema{
		Discriminator: &parser.Discriminator{
			PropertyName: "petType",
			Mapping: map[string]string{
				"dog": "#/components/schemas/Dog",
				"cat": "Cat",
			},
		},
	}

	rewriter.rewriteSchema(schema)

	assert.Equal(t, "#/components/schemas/Dog_v2", schema.Discriminator.Mapping["dog"])
	assert.Equal(t, "Cat_v2", schema.Discriminator.Mapping["cat"])
}

func TestSchemaRewriter_RewriteSchema_CircularReference(t *testing.T) {
	rewriter := NewSchemaRewriter()

	// Create circular reference: Node -> children -> Node
	node := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
		},
	}
	node.Properties["children"] = &parser.Schema{
		Type:  "array",
		Items: node,
	}

	// Should not panic or infinite loop
	require.NotPanics(t, func() {
		rewriter.rewriteSchema(node)
	})
}

func TestSchemaRewriter_RewriteOAS3Document_Schemas(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("User", "User_v2", parser.OASVersion300)

	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"User_v2": {Type: "object"},
				"Profile": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"user": {Ref: "#/components/schemas/User"},
					},
				},
			},
		},
	}

	err := rewriter.RewriteDocument(doc)

	assert.NoError(t, err)
	assert.Equal(t, "#/components/schemas/User_v2", doc.Components.Schemas["Profile"].Properties["user"].Ref)
}

func TestSchemaRewriter_RewriteOAS3Document_Paths(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("User", "User_left", parser.OASVersion300)

	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "Success",
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{
											Ref: "#/components/schemas/User",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	err := rewriter.RewriteDocument(doc)

	assert.NoError(t, err)
	assert.Equal(t, "#/components/schemas/User_left",
		doc.Paths["/users"].Get.Responses.Codes["200"].Content["application/json"].Schema.Ref)
}

func TestSchemaRewriter_RewriteOAS3Document_RequestBody(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("CreateUserRequest", "CreateUserRequest_v2", parser.OASVersion300)

	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Post: &parser.Operation{
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"application/json": {
								Schema: &parser.Schema{
									Ref: "#/components/schemas/CreateUserRequest",
								},
							},
						},
					},
				},
			},
		},
	}

	err := rewriter.RewriteDocument(doc)

	assert.NoError(t, err)
	assert.Equal(t, "#/components/schemas/CreateUserRequest_v2",
		doc.Paths["/users"].Post.RequestBody.Content["application/json"].Schema.Ref)
}

func TestSchemaRewriter_RewriteOAS2Document_Definitions(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("User", "User_v2", parser.OASVersion20)

	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Definitions: map[string]*parser.Schema{
			"User_v2": {Type: "object"},
			"Profile": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"user": {Ref: "#/definitions/User"},
				},
			},
		},
	}

	err := rewriter.RewriteDocument(doc)

	assert.NoError(t, err)
	assert.Equal(t, "#/definitions/User_v2", doc.Definitions["Profile"].Properties["user"].Ref)
}

func TestSchemaRewriter_RewriteOAS2Document_Paths(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("User", "User_left", parser.OASVersion20)

	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "Success",
								Schema: &parser.Schema{
									Ref: "#/definitions/User",
								},
							},
						},
					},
				},
			},
		},
	}

	err := rewriter.RewriteDocument(doc)

	assert.NoError(t, err)
	assert.Equal(t, "#/definitions/User_left",
		doc.Paths["/users"].Get.Responses.Codes["200"].Schema.Ref)
}

func TestSchemaRewriter_RewriteOAS2Document_Parameters(t *testing.T) {
	rewriter := NewSchemaRewriter()
	rewriter.RegisterRename("UserQuery", "UserQuery_v2", parser.OASVersion20)

	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Paths: parser.Paths{
			"/users": &parser.PathItem{
				Get: &parser.Operation{
					Parameters: []*parser.Parameter{
						{
							Name:   "filter",
							In:     "body",
							Schema: &parser.Schema{Ref: "#/definitions/UserQuery"},
						},
					},
				},
			},
		},
	}

	err := rewriter.RewriteDocument(doc)

	assert.NoError(t, err)
	assert.Equal(t, "#/definitions/UserQuery_v2",
		doc.Paths["/users"].Get.Parameters[0].Schema.Ref)
}

func TestSchemaRewriter_RewriteDocument_UnsupportedType(t *testing.T) {
	rewriter := NewSchemaRewriter()

	err := rewriter.RewriteDocument("invalid")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported document type")
}

func TestSchemaRefPath(t *testing.T) {
	tests := []struct {
		name     string
		schema   string
		version  parser.OASVersion
		expected string
	}{
		{
			name:     "OAS 3.0",
			schema:   "User",
			version:  parser.OASVersion300,
			expected: "#/components/schemas/User",
		},
		{
			name:     "OAS 3.1",
			schema:   "Product",
			version:  parser.OASVersion310,
			expected: "#/components/schemas/Product",
		},
		{
			name:     "OAS 2.0",
			schema:   "Order",
			version:  parser.OASVersion20,
			expected: "#/definitions/Order",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := schemaRefPath(tt.schema, tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractSchemaName(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{
			name:     "OAS 3.0 ref",
			ref:      "#/components/schemas/User",
			expected: "User",
		},
		{
			name:     "OAS 2.0 ref",
			ref:      "#/definitions/Product",
			expected: "Product",
		},
		{
			name:     "invalid ref",
			ref:      "#/invalid/path",
			expected: "",
		},
		{
			name:     "empty ref",
			ref:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSchemaName(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}
