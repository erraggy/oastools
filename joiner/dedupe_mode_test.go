package joiner

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// labelShape is the shape the three services each declare under their own
// name. Nothing but the names tells the three apart.
func labelShape() *parser.Schema {
	return &parser.Schema{
		Properties: map[string]*parser.Schema{
			"name":  {Type: "string"},
			"title": {Type: "string"},
		},
	}
}

// serviceOAS2 is one document declaring schemaName for the label shape and
// referencing it from a single operation's response (#553).
func serviceOAS2(title, path, operationID, schemaName string) parser.ParseResult {
	return parser.ParseResult{
		Document: &parser.OAS2Document{
			Swagger: "2.0",
			Info:    &parser.Info{Title: title, Version: "1.0.0"},
			Paths: parser.Paths{
				path: &parser.PathItem{Get: &parser.Operation{
					OperationID: operationID,
					Responses: &parser.Responses{Codes: map[string]*parser.Response{
						"200": {
							Description: "OK",
							Schema:      &parser.Schema{Ref: "#/definitions/" + schemaName},
						},
					}},
				}},
			},
			Definitions: map[string]*parser.Schema{schemaName: labelShape()},
			OASVersion:  parser.OASVersion20,
		},
		Version:      "2.0",
		OASVersion:   parser.OASVersion20,
		SourcePath:   title,
		SourceFormat: "json",
	}
}

// serviceOAS3 is serviceOAS2's OAS 3 counterpart. Both versions apply the
// semantic pass through the same code, so both reach the pointer form.
func serviceOAS3(title, path, operationID, schemaName string) parser.ParseResult {
	return parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: title, Version: "1.0.0"},
			Paths: parser.Paths{
				path: &parser.PathItem{Get: &parser.Operation{
					OperationID: operationID,
					Responses: &parser.Responses{Codes: map[string]*parser.Response{
						"200": {
							Description: "OK",
							Content: map[string]*parser.MediaType{
								"application/json": {Schema: &parser.Schema{
									Ref: "#/components/schemas/" + schemaName,
								}},
							},
						},
					}},
				}},
			},
			Components: &parser.Components{Schemas: map[string]*parser.Schema{
				schemaName: labelShape(),
			}},
			OASVersion: parser.OASVersion303,
		},
		Version:      "3.0.3",
		OASVersion:   parser.OASVersion303,
		SourcePath:   title,
		SourceFormat: "json",
	}
}

// threeServicesOAS2 is the issue's repro: three documents, each declaring its
// own name for one shape, none of the names colliding. Pets is first, so it is
// the document a first-declared ranking picks the survivor from.
func threeServicesOAS2() []parser.ParseResult {
	return []parser.ParseResult{
		serviceOAS2("pets", "/pets", "listPets", "pets.Label"),
		serviceOAS2("store", "/store/inventory", "getInventory", "store.Tag"),
		serviceOAS2("orders", "/orders", "listOrders", "orders.Marker"),
	}
}

// joinPointer joins documents with semantic deduplication in pointer mode.
func joinPointer(t *testing.T, version parser.OASVersion, docs ...parser.ParseResult) *JoinResult {
	t.Helper()
	return joinMode(t, DeduplicationModePointer, version, docs...)
}

func joinMode(t *testing.T, mode DeduplicationMode, version parser.OASVersion, docs ...parser.ParseResult) *JoinResult {
	t.Helper()
	res, err := JoinWithOptions(
		WithParsed(append([]parser.ParseResult{emptyCombined(version)}, docs...)...),
		WithSchemaStrategy(StrategyDeduplicateOrRename),
		WithEquivalenceMode("deep"),
		WithSemanticDeduplication(true),
		WithDeduplicationMode(mode),
		WithDeduplicationReport(true),
	)
	require.NoError(t, err)
	return res
}

// responseRef returns the $ref an OAS 2.0 operation's 200 response names.
func responseRef(t *testing.T, doc *parser.OAS2Document, path string) string {
	t.Helper()
	schema := doc.Paths[path].Get.Responses.Codes["200"].Schema
	require.NotNil(t, schema)
	return schema.Ref
}

// Every name in the group stays resolvable and no reference is rewritten, so
// each operation still answers with the name its own document declared (#553).
func TestJoiner_PointerMode_KeepsEveryNameResolvable_OAS2(t *testing.T) {
	res := joinPointer(t, parser.OASVersion20, threeServicesOAS2()...)

	assert.Equal(t, []string{"orders.Marker", "pets.Label", "store.Tag"},
		sortedSchemaNames(t, res.Document))

	doc := res.Document.(*parser.OAS2Document)
	assert.Equal(t, "#/definitions/pets.Label", responseRef(t, doc, "/pets"))
	assert.Equal(t, "#/definitions/store.Tag", responseRef(t, doc, "/store/inventory"))
	assert.Equal(t, "#/definitions/orders.Marker", responseRef(t, doc, "/orders"))

	// The shape is stored once, under the name the first document declared,
	// and the other two names are bare pointers to it.
	assert.NotEmpty(t, doc.Definitions["pets.Label"].Properties)
	assert.Empty(t, doc.Definitions["pets.Label"].Ref)
	assert.Equal(t, "#/definitions/pets.Label", doc.Definitions["store.Tag"].Ref)
	assert.Empty(t, doc.Definitions["store.Tag"].Properties)
	assert.Equal(t, "#/definitions/pets.Label", doc.Definitions["orders.Marker"].Ref)
	assert.Empty(t, doc.Definitions["orders.Marker"].Properties)
}

func TestJoiner_PointerMode_KeepsEveryNameResolvable_OAS3(t *testing.T) {
	res := joinPointer(t, parser.OASVersion303,
		serviceOAS3("pets", "/pets", "listPets", "pets.Label"),
		serviceOAS3("store", "/store/inventory", "getInventory", "store.Tag"),
	)

	assert.Equal(t, []string{"pets.Label", "store.Tag"},
		sortedSchemaNames(t, res.Document))

	doc := res.Document.(*parser.OAS3Document)
	schemas := doc.Components.Schemas
	assert.NotEmpty(t, schemas["pets.Label"].Properties)
	assert.Equal(t, "#/components/schemas/pets.Label", schemas["store.Tag"].Ref)

	// A pointer spells its $ref the way its own document does.
	assert.Equal(t, "#/components/schemas/store.Tag",
		doc.Paths["/store/inventory"].Get.Responses.Codes["200"].
			Content["application/json"].Schema.Ref)
}

// The default is unchanged: the folded names are removed and every reference to
// them is repointed at the survivor.
func TestJoiner_RemoveMode_IsUnchangedAndStillTheDefault(t *testing.T) {
	explicit := joinMode(t, DeduplicationModeRemove, parser.OASVersion20, threeServicesOAS2()...)

	defaulted, err := JoinWithOptions(
		WithParsed(append([]parser.ParseResult{emptyCombined(parser.OASVersion20)},
			threeServicesOAS2()...)...),
		WithSchemaStrategy(StrategyDeduplicateOrRename),
		WithEquivalenceMode("deep"),
		WithSemanticDeduplication(true),
	)
	require.NoError(t, err)

	for name, res := range map[string]*JoinResult{"explicit": explicit, "defaulted": defaulted} {
		doc := res.Document.(*parser.OAS2Document)
		assert.Equal(t, []string{"orders.Marker"}, sortedSchemaNames(t, res.Document), name)
		assert.Equal(t, "#/definitions/orders.Marker", responseRef(t, doc, "/pets"), name)
		assert.Equal(t, "#/definitions/orders.Marker", responseRef(t, doc, "/store/inventory"), name)
	}
}

// Pointer mode does not weaken #501: two names one schema tree references
// distinctly are held apart, so nothing is consolidated and no pointer appears.
func TestJoiner_PointerMode_KeepsNamesAParentDistinguishes(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(emptyCombined(parser.OASVersion20), shipmentOAS2(true)),
		WithSchemaStrategy(StrategyDeduplicateOrRename),
		WithEquivalenceMode("deep"),
		WithSemanticDeduplication(true),
		WithDeduplicationMode(DeduplicationModePointer),
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"DestinationAddress", "OriginAddress", "Shipment"},
		sortedSchemaNames(t, res.Document))

	doc := res.Document.(*parser.OAS2Document)
	assert.Empty(t, doc.Definitions["DestinationAddress"].Ref, "no pointer was written")
	assert.Equal(t, map[string]string{
		"shippedFrom": "#/definitions/OriginAddress",
		"shippedTo":   "#/definitions/DestinationAddress",
	}, propertyRefs(t, doc.Definitions["Shipment"]))
}

// The report distinguishes a name kept as a pointer from one folded away.
func TestJoiner_PointerMode_ReportMarksKeptNames(t *testing.T) {
	pointer := joinPointer(t, parser.OASVersion20, threeServicesOAS2()...)
	require.Len(t, pointer.Consolidations, 1)
	assert.Equal(t, Consolidation{
		Survivor: "pets.Label",
		Folded: []FoldedName{
			{Name: "orders.Marker", Pointer: true},
			{Name: "store.Tag", Pointer: true},
		},
	}, pointer.Consolidations[0])

	removed := joinMode(t, DeduplicationModeRemove, parser.OASVersion20, threeServicesOAS2()...)
	require.Len(t, removed.Consolidations, 1)
	for _, folded := range removed.Consolidations[0].Folded {
		assert.False(t, folded.Pointer, "%s was removed, not kept", folded.Name)
	}
}

// Pointer mode composes with the scope: generated-only still refuses to fold a
// declared name, so there is nothing left for a pointer to stand in for.
func TestJoiner_PointerMode_ComposesWithGeneratedOnlyScope(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(append([]parser.ParseResult{emptyCombined(parser.OASVersion20)},
			threeServicesOAS2()...)...),
		WithSchemaStrategy(StrategyDeduplicateOrRename),
		WithEquivalenceMode("deep"),
		WithSemanticDeduplication(true),
		WithDeduplicationMode(DeduplicationModePointer),
		WithDeduplicationScope(DeduplicationScopeGeneratedOnly),
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"orders.Marker", "pets.Label", "store.Tag"},
		sortedSchemaNames(t, res.Document))

	doc := res.Document.(*parser.OAS2Document)
	for _, name := range []string{"orders.Marker", "pets.Label", "store.Tag"} {
		assert.Empty(t, doc.Definitions[name].Ref, "%s kept its own schema", name)
		assert.NotEmpty(t, doc.Definitions[name].Properties, "%s kept its own schema", name)
	}
}

// A pointer is a new schema written into the joined document, so the documents
// the joiner was handed are left as they were.
func TestJoiner_PointerMode_LeavesSourceDocumentsAlone(t *testing.T) {
	docs := threeServicesOAS2()
	joinPointer(t, parser.OASVersion20, docs...)

	for _, doc := range docs {
		source := doc.Document.(*parser.OAS2Document)
		for name, schema := range source.Definitions {
			assert.Empty(t, schema.Ref, "%s in %s", name, doc.SourcePath)
			assert.NotEmpty(t, schema.Properties, "%s in %s", name, doc.SourcePath)
		}
	}
}

// A schema referencing a name the group consolidates. The two modes part ways
// on it, and the map the document ends up with has to be the one the pass
// worked in: installing it after a rewrite would carry the originals and lose
// every rewrite the default mode makes here.
//
// Order lives in the Pets document and references pets.Label, which pointer
// mode keeps as the survivor and the default folds away.
func TestJoiner_PointerMode_LeavesAReferencingSchemaAlone(t *testing.T) {
	documents := func() []parser.ParseResult {
		pets := serviceOAS3("pets", "/pets", "listPets", "pets.Label")
		pets.Document.(*parser.OAS3Document).Components.Schemas["Order"] = &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"marker": {Ref: "#/components/schemas/pets.Label"},
			},
		}
		return []parser.ParseResult{pets, serviceOAS3("orders", "/orders", "listOrders", "orders.Marker")}
	}

	pointer := joinPointer(t, parser.OASVersion303, documents()...)
	pointerDoc := pointer.Document.(*parser.OAS3Document)
	assert.Equal(t, map[string]string{"marker": "#/components/schemas/pets.Label"},
		propertyRefs(t, pointerDoc.Components.Schemas["Order"]),
		"the reference is untouched and still resolves")
	assert.Equal(t, "#/components/schemas/pets.Label",
		pointerDoc.Components.Schemas["orders.Marker"].Ref)

	// The default folds pets.Label away, so the same reference is repointed at
	// the survivor. Losing that rewrite is what the ordering guards against.
	removed := joinMode(t, DeduplicationModeRemove, parser.OASVersion303, documents()...)
	removedDoc := removed.Document.(*parser.OAS3Document)
	assert.Equal(t, []string{"Order", "orders.Marker"}, sortedSchemaNames(t, removedDoc))
	assert.Equal(t, map[string]string{"marker": "#/components/schemas/orders.Marker"},
		propertyRefs(t, removedDoc.Components.Schemas["Order"]))
}

func TestIsValidDeduplicationMode(t *testing.T) {
	for _, mode := range append(ValidDeduplicationModes(), "") {
		assert.True(t, IsValidDeduplicationMode(mode), mode)
	}
	assert.False(t, IsValidDeduplicationMode("reference"))
}

// A JoinerConfig is a struct a caller can fill in directly, so an unrecognized
// mode reaches the pass and has to read as the default rather than as pointer.
func TestResolveDeduplicationMode(t *testing.T) {
	tests := map[string]struct {
		mode DeduplicationMode
		want DeduplicationMode
	}{
		"empty is remove":        {mode: "", want: DeduplicationModeRemove},
		"remove":                 {mode: DeduplicationModeRemove, want: DeduplicationModeRemove},
		"pointer":                {mode: DeduplicationModePointer, want: DeduplicationModePointer},
		"unrecognized is remove": {mode: "reference", want: DeduplicationModeRemove},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, resolveDeduplicationMode(tc.mode))
		})
	}
}

func TestWithDeduplicationMode_RejectsUnknownMode(t *testing.T) {
	_, err := JoinWithOptions(
		WithParsed(emptyCombined(parser.OASVersion20)),
		WithDeduplicationMode("reference"),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid deduplication mode")
}

func TestOutranksDeclaration(t *testing.T) {
	generated := map[string]bool{"Api_Common": true}
	declaredIn := map[string]int{"Zebra": 0, "Apple": 1, "Api_Common": 1}
	outranks := outranksDeclaration(generated, declaredIn)
	require.NotNil(t, outranks)

	assert.True(t, outranks("Zebra", "Apple"),
		"the earlier document wins over sort order")
	assert.False(t, outranks("Apple", "Zebra"))
	assert.True(t, outranks("Apple", "Api_Common"),
		"a declared name wins over a generated one from the same document")
	assert.False(t, outranks("Api_Common", "Apple"))
	assert.True(t, outranks("Alpha", "Beta"),
		"neither is attributed, so sort order breaks the tie")

	// An unattributed name never outranks an attributed one, whichever way the
	// comparison is asked.
	assert.False(t, outranks("Alpha", "Zebra"))
	assert.True(t, outranks("Zebra", "Alpha"))

	assert.Nil(t, outranksDeclaration(nil, nil),
		"nothing to go on, so the deduplicator's own tiebreak stands")
}

func TestDeclarationOrder(t *testing.T) {
	first, second, unclaimed := labelShape(), labelShape(), labelShape()
	owner := map[any]int{first: 0, second: 2}
	schemas := map[string]*parser.Schema{
		"First":     first,
		"Second":    second,
		"Unclaimed": unclaimed,
		"Nil":       nil,
	}

	assert.Equal(t, map[string]int{"First": 0, "Second": 2},
		declarationOrder(schemas, owner))
	assert.Nil(t, declarationOrder(schemas, nil),
		"no ownership recorded, so no order to report")
}
