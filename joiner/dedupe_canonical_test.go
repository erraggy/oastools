package joiner

import (
	"maps"
	"slices"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The rename template from outranks's own doc comment, which puts every alias
// ahead of the name it was generated from alphabetically. That ordering is what
// made semantic deduplication consolidate into the alias (#498).
const aliasFirstTemplate = `Api_{{.Name}}`

// storefrontOAS2 is one of several documents that all declare the same Pet and
// Category, so every join after the first collides on both.
func storefrontOAS2(name string) parser.ParseResult {
	return parser.ParseResult{
		Document: &parser.OAS2Document{
			Swagger: "2.0",
			Info:    &parser.Info{Title: name, Version: "1.0.0"},
			Paths: parser.Paths{
				"/" + name + "/pet": &parser.PathItem{Get: &parser.Operation{
					OperationID: "getPet" + name,
					Responses: &parser.Responses{Codes: map[string]*parser.Response{
						"200": {
							Description: "ok",
							Schema:      &parser.Schema{Ref: "#/definitions/Pet"},
						},
					}},
				}},
			},
			Definitions: map[string]*parser.Schema{
				"Pet": {
					Type:     "object",
					Required: []string{"name"},
					Properties: map[string]*parser.Schema{
						"category": {Ref: "#/definitions/Category"},
						"name":     {Type: "string"},
					},
				},
				"Category": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"id":   {Type: "integer", Format: "int64"},
						"name": {Type: "string"},
					},
				},
			},
			OASVersion: parser.OASVersion20,
		},
		Version:      "2.0",
		OASVersion:   parser.OASVersion20,
		SourcePath:   name,
		SourceFormat: "json",
	}
}

// storefrontOAS3 is storefrontOAS2's OAS 3 counterpart. The semantic pass runs
// through the same schemautil deduplicator in both, so both reach the bug.
func storefrontOAS3(name string) parser.ParseResult {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: name, Version: "1.0.0"},
		Paths: parser.Paths{
			"/" + name + "/pet": &parser.PathItem{Get: &parser.Operation{
				OperationID: "getPet" + name,
				Responses: &parser.Responses{Codes: map[string]*parser.Response{
					"200": {
						Description: "ok",
						Content: map[string]*parser.MediaType{
							"application/json": {Schema: &parser.Schema{Ref: "#/components/schemas/Pet"}},
						},
					},
				}},
			}},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type:     "object",
					Required: []string{"name"},
					Properties: map[string]*parser.Schema{
						"category": {Ref: "#/components/schemas/Category"},
						"name":     {Type: "string"},
					},
				},
				"Category": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"id":   {Type: "integer", Format: "int64"},
						"name": {Type: "string"},
					},
				},
			},
		},
		OASVersion: parser.OASVersion303,
	}
	return parser.ParseResult{
		Document:     doc,
		Version:      "3.0.3",
		OASVersion:   parser.OASVersion303,
		SourcePath:   name,
		SourceFormat: "json",
	}
}

// emptyCombined is the accumulator a caller folding documents one at a time
// starts from.
func emptyCombined(version parser.OASVersion) parser.ParseResult {
	if version == parser.OASVersion20 {
		return parser.ParseResult{
			Document: &parser.OAS2Document{
				Swagger:     "2.0",
				Info:        &parser.Info{Title: "combined", Version: "1.0.0"},
				Paths:       make(parser.Paths),
				Definitions: map[string]*parser.Schema{},
				OASVersion:  parser.OASVersion20,
			},
			Version:      "2.0",
			OASVersion:   parser.OASVersion20,
			SourcePath:   "combined",
			SourceFormat: "json",
		}
	}
	return parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI:    "3.0.3",
			Info:       &parser.Info{Title: "combined", Version: "1.0.0"},
			Paths:      make(parser.Paths),
			Components: &parser.Components{Schemas: map[string]*parser.Schema{}},
			OASVersion: parser.OASVersion303,
		},
		Version:      "3.0.3",
		OASVersion:   parser.OASVersion303,
		SourcePath:   "combined",
		SourceFormat: "json",
	}
}

// foldStorefronts joins the documents one at a time the way a caller that
// accumulates does, and returns the last result.
func foldStorefronts(t *testing.T, combined parser.ParseResult, storefront func(string) parser.ParseResult, names ...string) parser.ParseResult {
	t.Helper()
	for _, name := range names {
		res, err := JoinWithOptions(
			WithParsed(combined, storefront(name)),
			WithSchemaStrategy(StrategyDeduplicateOrRename),
			WithEquivalenceMode("deep"),
			WithSemanticDeduplication(true),
			WithRenameTemplate(aliasFirstTemplate),
		)
		require.NoError(t, err)
		parsed := res.ToParseResult()
		parsed.SourcePath = "combined"
		combined = *parsed
		t.Logf("after %s -> %v", name, sortedSchemaNames(t, combined.Document))
	}
	return combined
}

// sortedSchemaNames returns the top-level schema names, sorted.
func sortedSchemaNames(t *testing.T, document any) []string {
	t.Helper()
	names := schemaNamesOf(t, document)
	slices.Sort(names)
	return names
}

// refsOf returns every $ref the storefront documents place: one per path
// response, plus Pet's category property.
func refsOf(t *testing.T, document any) (responses, category []string) {
	t.Helper()
	switch doc := document.(type) {
	case *parser.OAS2Document:
		for _, item := range doc.Paths {
			responses = append(responses, item.Get.Responses.Codes["200"].Schema.Ref)
		}
		for _, schema := range doc.Definitions {
			if property, ok := schema.Properties["category"]; ok {
				category = append(category, property.Ref)
			}
		}
	case *parser.OAS3Document:
		for _, item := range doc.Paths {
			responses = append(responses,
				item.Get.Responses.Codes["200"].Content["application/json"].Schema.Ref)
		}
		for _, schema := range doc.Components.Schemas {
			if property, ok := schema.Properties["category"]; ok {
				category = append(category, property.Ref)
			}
		}
	default:
		t.Fatalf("unexpected document type %T", document)
	}
	return responses, category
}

// assertRefsPointAt checks that the surviving names are what references spell,
// so nothing still points at a name the deduplication removed.
func assertRefsPointAt(t *testing.T, document any, pet, category string) {
	t.Helper()
	responses, categories := refsOf(t, document)
	require.NotEmpty(t, responses)
	require.NotEmpty(t, categories)
	for _, ref := range responses {
		assert.Equal(t, pet, ref, "a response still references a removed name")
	}
	for _, ref := range categories {
		assert.Equal(t, category, ref, "Pet still references a removed name")
	}
}

// schemaNamesOf returns the top-level schema names of either document version.
func schemaNamesOf(t *testing.T, document any) []string {
	t.Helper()
	switch doc := document.(type) {
	case *parser.OAS2Document:
		return slices.Collect(maps.Keys(doc.Definitions))
	case *parser.OAS3Document:
		return slices.Collect(maps.Keys(doc.Components.Schemas))
	default:
		t.Fatalf("unexpected document type %T", document)
		return nil
	}
}

func TestJoiner_SemanticDeduplication_KeepsDeclaredNameOverAlias_OAS2(t *testing.T) {
	combined := foldStorefronts(t, emptyCombined(parser.OASVersion20), storefrontOAS2,
		"downtown", "airport", "harbor")

	// Every storefront declared Pet and Category and they are all equivalent, so
	// there is nothing to consolidate into a name the joiner synthesized.
	assert.Equal(t, []string{"Category", "Pet"}, sortedSchemaNames(t, combined.Document))

	doc := combined.Document.(*parser.OAS2Document)
	assertRefsPointAt(t, combined.Document, "#/definitions/Pet", "#/definitions/Category")
	assert.Len(t, doc.Paths, 3, "one path per storefront, so every response ref is checked")
}

func TestJoiner_SemanticDeduplication_KeepsDeclaredNameOverAlias_OAS3(t *testing.T) {
	combined := foldStorefronts(t, emptyCombined(parser.OASVersion303), storefrontOAS3,
		"downtown", "airport", "harbor")

	assert.Equal(t, []string{"Category", "Pet"}, sortedSchemaNames(t, combined.Document))

	doc := combined.Document.(*parser.OAS3Document)
	assertRefsPointAt(t, combined.Document,
		"#/components/schemas/Pet", "#/components/schemas/Category")
	assert.Len(t, doc.Paths, 3, "one path per storefront, so every response ref is checked")
}

// The collapse in StrategyDeduplicateOrRename is not what makes the declared
// name win: StrategyRenameRight generates the same aliases and never collapses,
// so only the semantic pass is left to choose between a name and its alias.
func TestJoiner_SemanticDeduplication_KeepsDeclaredNameUnderRename(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(storefrontOAS3("downtown"), storefrontOAS3("airport")),
		WithSchemaStrategy(StrategyRenameRight),
		WithEquivalenceMode("deep"),
		WithSemanticDeduplication(true),
		WithRenameTemplate(aliasFirstTemplate),
	)
	require.NoError(t, err)

	schemas := schemaNamesOf(t, res.Document)
	slices.Sort(schemas)
	// Api_Category is gone and Category survived. Api_Pet remains because
	// deduplication is a single pass: it compares the two Pets while their
	// $refs still read Category and Api_Category, so it does not yet see them as
	// equivalent. That is separate from which name a settled group keeps.
	assert.Equal(t, []string{"Api_Pet", "Category", "Pet"}, schemas)
}

// A name no rename generated is preferred, but among names that documents wrote
// the tiebreak is still alphabetical, matching what the deduplicator does on its
// own. Neither of these two names is an alias.
func TestJoiner_SemanticDeduplication_TiesStillGoAlphabetically(t *testing.T) {
	left := storefrontOAS3("downtown")
	right := storefrontOAS3("airport")
	// Rename the right document's Pet before joining, so the two names are
	// equivalent, differ, and neither is generated.
	rightDoc := right.Document.(*parser.OAS3Document)
	rightDoc.Components.Schemas["Zebra"] = rightDoc.Components.Schemas["Pet"]
	delete(rightDoc.Components.Schemas, "Pet")
	rightDoc.Paths["/airport/pet"].Get.Responses.Codes["200"].
		Content["application/json"].Schema.Ref = "#/components/schemas/Zebra"

	res, err := JoinWithOptions(
		WithParsed(left, right),
		WithSchemaStrategy(StrategyDeduplicateOrRename),
		WithEquivalenceMode("deep"),
		WithSemanticDeduplication(true),
		WithRenameTemplate(aliasFirstTemplate),
	)
	require.NoError(t, err)

	schemas := schemaNamesOf(t, res.Document)
	slices.Sort(schemas)
	assert.Equal(t, []string{"Category", "Pet"}, schemas, "Pet sorts before Zebra")
}
