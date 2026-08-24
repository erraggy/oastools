package joiner

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// address is the shape OriginAddress and DestinationAddress share. The two
// compare equal, so nothing but how they are referenced tells them apart.
func address() *parser.Schema {
	return &parser.Schema{
		Type:     "object",
		Required: []string{"street", "city"},
		Properties: map[string]*parser.Schema{
			"street": {Type: "string"},
			"city":   {Type: "string"},
		},
	}
}

// shipmentOAS2 records where a pet was collected from and where it is going.
// Both are required and both are an address, so consolidating the two leaves
// the document saying a shipment's origin is its destination (#501).
//
// named declares the parent in definitions. Otherwise it is the response's
// inline schema, which deduplication rewrites references in just the same, so
// reading only named parents would miss it.
func shipmentOAS2(named bool) parser.ParseResult {
	parent := &parser.Schema{
		Type:     "object",
		Required: []string{"shippedFrom", "shippedTo"},
		Properties: map[string]*parser.Schema{
			"shippedFrom": {Ref: "#/definitions/OriginAddress"},
			"shippedTo":   {Ref: "#/definitions/DestinationAddress"},
		},
	}
	definitions := map[string]*parser.Schema{
		"OriginAddress":      address(),
		"DestinationAddress": address(),
	}
	responseSchema := parent
	if named {
		definitions["Shipment"] = parent
		responseSchema = &parser.Schema{Ref: "#/definitions/Shipment"}
	}

	return parser.ParseResult{
		Document: &parser.OAS2Document{
			Swagger: "2.0",
			Info:    &parser.Info{Title: "store", Version: "1.0.0"},
			Paths: parser.Paths{
				"/store/shipment": &parser.PathItem{Get: &parser.Operation{
					OperationID: "getStoreShipment",
					Responses: &parser.Responses{Codes: map[string]*parser.Response{
						"200": {Description: "ok", Schema: responseSchema},
					}},
				}},
			},
			Definitions: definitions,
			OASVersion:  parser.OASVersion20,
		},
		Version:      "2.0",
		OASVersion:   parser.OASVersion20,
		SourcePath:   "store",
		SourceFormat: "json",
	}
}

// shipmentOAS3 is shipmentOAS2's OAS 3 counterpart. Both versions run the
// semantic pass through the same deduplicator, so both reach the bug.
func shipmentOAS3() parser.ParseResult {
	return parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "store", Version: "1.0.0"},
			Paths: parser.Paths{
				"/store/shipment": &parser.PathItem{Get: &parser.Operation{
					OperationID: "getStoreShipment",
					Responses: &parser.Responses{Codes: map[string]*parser.Response{
						"200": {
							Description: "ok",
							Content: map[string]*parser.MediaType{
								"application/json": {Schema: &parser.Schema{
									Ref: "#/components/schemas/Shipment",
								}},
							},
						},
					}},
				}},
			},
			Components: &parser.Components{Schemas: map[string]*parser.Schema{
				"Shipment": {
					Type:     "object",
					Required: []string{"shippedFrom", "shippedTo"},
					Properties: map[string]*parser.Schema{
						"shippedFrom": {Ref: "#/components/schemas/OriginAddress"},
						"shippedTo":   {Ref: "#/components/schemas/DestinationAddress"},
					},
				},
				"OriginAddress":      address(),
				"DestinationAddress": address(),
			}},
			OASVersion: parser.OASVersion303,
		},
		Version:      "3.0.3",
		OASVersion:   parser.OASVersion303,
		SourcePath:   "store",
		SourceFormat: "json",
	}
}

// joinDeduplicated folds a document into an empty one of the same version with
// every deduplication pass enabled, which is the configuration that reaches the
// semantic pass.
func joinDeduplicated(t *testing.T, version parser.OASVersion, doc parser.ParseResult) *JoinResult {
	t.Helper()
	res, err := JoinWithOptions(
		WithParsed(emptyCombined(version), doc),
		WithSchemaStrategy(StrategyDeduplicateOrRename),
		WithEquivalenceMode("deep"),
		WithSemanticDeduplication(true),
	)
	require.NoError(t, err)
	return res
}

// propertyRefs returns the $refs a schema's properties point at, keyed by
// property name.
func propertyRefs(t *testing.T, schema *parser.Schema) map[string]string {
	t.Helper()
	require.NotNil(t, schema)
	refs := make(map[string]string, len(schema.Properties))
	for name, property := range schema.Properties {
		require.NotNil(t, property, "property %s", name)
		refs[name] = property.Ref
	}
	return refs
}

func TestJoiner_SemanticDeduplication_KeepsNamesANamedParentDistinguishes_OAS2(t *testing.T) {
	res := joinDeduplicated(t, parser.OASVersion20, shipmentOAS2(true))

	assert.Equal(t, []string{"DestinationAddress", "OriginAddress", "Shipment"},
		sortedSchemaNames(t, res.Document))

	doc := res.Document.(*parser.OAS2Document)
	assert.Equal(t, map[string]string{
		"shippedFrom": "#/definitions/OriginAddress",
		"shippedTo":   "#/definitions/DestinationAddress",
	}, propertyRefs(t, doc.Definitions["Shipment"]))
	assert.Empty(t, res.Warnings, "nothing was consolidated, so nothing to report")
}

func TestJoiner_SemanticDeduplication_KeepsNamesANamedParentDistinguishes_OAS3(t *testing.T) {
	res := joinDeduplicated(t, parser.OASVersion303, shipmentOAS3())

	assert.Equal(t, []string{"DestinationAddress", "OriginAddress", "Shipment"},
		sortedSchemaNames(t, res.Document))

	doc := res.Document.(*parser.OAS3Document)
	assert.Equal(t, map[string]string{
		"shippedFrom": "#/components/schemas/OriginAddress",
		"shippedTo":   "#/components/schemas/DestinationAddress",
	}, propertyRefs(t, doc.Components.Schemas["Shipment"]))
}

// The parent does not have to be a component. An operation's inline schema
// distinguishes the two names just as a declared one does.
func TestJoiner_SemanticDeduplication_KeepsNamesAnInlineParentDistinguishes(t *testing.T) {
	res := joinDeduplicated(t, parser.OASVersion20, shipmentOAS2(false))

	assert.Equal(t, []string{"DestinationAddress", "OriginAddress"},
		sortedSchemaNames(t, res.Document))

	doc := res.Document.(*parser.OAS2Document)
	inline := doc.Paths["/store/shipment"].Get.Responses.Codes["200"].Schema
	assert.Equal(t, map[string]string{
		"shippedFrom": "#/definitions/OriginAddress",
		"shippedTo":   "#/definitions/DestinationAddress",
	}, propertyRefs(t, inline))
}

// Two same-shaped schemas that no single schema references together are what
// deduplication is for, and sharing an operation is not sharing a schema: a
// request body and a response are separate trees, so the pair still merges.
func TestJoiner_SemanticDeduplication_MergesAcrossSeparateSchemaTrees(t *testing.T) {
	label := func() *parser.Schema {
		return &parser.Schema{
			Type:       "object",
			Properties: map[string]*parser.Schema{"label": {Type: "string"}},
		}
	}
	doc := parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "store", Version: "1.0.0"},
			Paths: parser.Paths{
				"/store/shipment": &parser.PathItem{Post: &parser.Operation{
					OperationID: "createShipment",
					RequestBody: &parser.RequestBody{Content: map[string]*parser.MediaType{
						"application/json": {Schema: &parser.Schema{
							Ref: "#/components/schemas/CreateShipmentRequest",
						}},
					}},
					Responses: &parser.Responses{Codes: map[string]*parser.Response{
						"200": {
							Description: "ok",
							Content: map[string]*parser.MediaType{
								"application/json": {Schema: &parser.Schema{
									Ref: "#/components/schemas/CreateShipmentResponse",
								}},
							},
						},
					}},
				}},
			},
			Components: &parser.Components{Schemas: map[string]*parser.Schema{
				"CreateShipmentRequest":  label(),
				"CreateShipmentResponse": label(),
			}},
			OASVersion: parser.OASVersion303,
		},
		Version:      "3.0.3",
		OASVersion:   parser.OASVersion303,
		SourcePath:   "store",
		SourceFormat: "json",
	}

	res := joinDeduplicated(t, parser.OASVersion303, doc)

	assert.Equal(t, []string{"CreateShipmentRequest"}, sortedSchemaNames(t, res.Document))
	assert.NotEmpty(t, res.Warnings, "a consolidation is reported")
}

// shipmentTreeOAS2 puts two references to the same address shape in one schema
// tree, in the arrangement the caller names.
func shipmentTreeOAS2(parent *parser.Schema) parser.ParseResult {
	return parser.ParseResult{
		Document: &parser.OAS2Document{
			Swagger: "2.0",
			Info:    &parser.Info{Title: "store", Version: "1.0.0"},
			Paths: parser.Paths{
				"/store/shipment": &parser.PathItem{Get: &parser.Operation{
					OperationID: "getStoreShipment",
					Responses: &parser.Responses{Codes: map[string]*parser.Response{
						"200": {Description: "ok", Schema: &parser.Schema{
							Ref: "#/definitions/Shipment",
						}},
					}},
				}},
			},
			Definitions: map[string]*parser.Schema{
				"Shipment":           parent,
				"OriginAddress":      address(),
				"DestinationAddress": address(),
			},
			OASVersion: parser.OASVersion20,
		},
		Version:      "2.0",
		OASVersion:   parser.OASVersion20,
		SourcePath:   "store",
		SourceFormat: "json",
	}
}

// The two references do not have to be siblings. What matters is that one tree
// names both shapes, so nesting one of them deeper changes nothing.
func TestJoiner_SemanticDeduplication_HoldsApartReferencesAtAnyDepth(t *testing.T) {
	res := joinDeduplicated(t, parser.OASVersion20, shipmentTreeOAS2(&parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"shippedFrom": {Ref: "#/definitions/OriginAddress"},
			"detail": {Type: "object", Properties: map[string]*parser.Schema{
				"shippedTo": {Ref: "#/definitions/DestinationAddress"},
			}},
		},
	}))

	assert.Equal(t, []string{"DestinationAddress", "OriginAddress", "Shipment"},
		sortedSchemaNames(t, res.Document))
}

// Alternatives are held apart too, though a value only ever satisfies one of
// them: a tree offering a choice between two names is still treating them as
// two things.
func TestJoiner_SemanticDeduplication_HoldsApartAlternatives(t *testing.T) {
	res := joinDeduplicated(t, parser.OASVersion20, shipmentTreeOAS2(&parser.Schema{
		OneOf: []*parser.Schema{
			{Ref: "#/definitions/OriginAddress"},
			{Ref: "#/definitions/DestinationAddress"},
		},
	}))

	assert.Equal(t, []string{"DestinationAddress", "OriginAddress", "Shipment"},
		sortedSchemaNames(t, res.Document))
}
