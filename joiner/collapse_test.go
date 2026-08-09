package joiner

import (
	"fmt"
	"io"
	"log/slog"
	"maps"
	"slices"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// flatDoc builds a document whose schemas reference nothing, so a comparison
// of any two of them is settled by their own fields.
func flatDoc(path string, schemas map[string]*parser.Schema) *parser.OAS3Document {
	return &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		Info:       &parser.Info{Title: "API", Version: "1.0.0"},
		Paths:      parser.Paths{path: {Get: &parser.Operation{}}},
		Components: &parser.Components{Schemas: schemas},
		OASVersion: parser.OASVersion303,
	}
}

// petStoreDoc builds a document whose Pet references a Category, with the
// Category's own shape left to the caller. It is the shape issue #487 is
// written around: the two Pets are byte identical and the divergence is one
// level down, in what their $ref resolves to.
func petStoreDoc(path string, category *parser.Schema) *parser.OAS3Document {
	doc := flatDoc(path, map[string]*parser.Schema{
		"Pet": {
			Type: "object",
			Properties: map[string]*parser.Schema{
				"category": {Ref: "#/components/schemas/Category"},
			},
		},
		"Category": category,
	})
	doc.Paths[path].Get.Responses = &parser.Responses{
		Codes: map[string]*parser.Response{
			"200": {
				Description: "ok",
				Content: map[string]*parser.MediaType{
					"application/json": {Schema: &parser.Schema{Ref: "#/components/schemas/Pet"}},
				},
			},
		},
	}
	return doc
}

// object is a schema with the given property names, all strings.
func object(properties ...string) *parser.Schema {
	schema := &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{}}
	for _, name := range properties {
		schema.Properties[name] = &parser.Schema{Type: "string"}
	}
	return schema
}

// parsedAs wraps a document the way JoinParsed expects it.
func parsedAs(doc any, sourcePath string) parser.ParseResult {
	return parser.ParseResult{
		Document:     doc,
		Version:      "3.0.3",
		OASVersion:   parser.OASVersion303,
		SourcePath:   sourcePath,
		SourceFormat: parser.SourceFormatYAML,
	}
}

// dedupeOrRenameConfig configures the strategy with a rename template that
// names schemas after their document's position, so the expected names in a
// test do not depend on how a source path is turned into a template variable.
func dedupeOrRenameConfig() JoinerConfig {
	config := DefaultConfig()
	config.SchemaStrategy = StrategyDeduplicateOrRename
	config.RenameTemplate = "{{.Name}}_v{{.Index}}"
	return config
}

func joinedSchemas(t *testing.T, result *JoinResult) map[string]*parser.Schema {
	t.Helper()
	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected an OAS 3 document")
	return doc.Components.Schemas
}

// schemaNames returns the joined schema names, sorted.
func schemaNames(t *testing.T, config JoinerConfig, docs []parser.ParseResult) []string {
	t.Helper()
	result, err := New(config).JoinParsed(docs)
	require.NoError(t, err)
	return slices.Sorted(maps.Keys(joinedSchemas(t, result)))
}

// TestDeduplicateOrRename_TransitiveDivergenceKeepsBoth is the case that makes
// a merge-time decision unsound: the two Pet schemas are identical, and only
// what their $ref resolves to tells them apart.
func TestDeduplicateOrRename_TransitiveDivergenceKeepsBoth(t *testing.T) {
	store := petStoreDoc("/store", object("id", "name"))
	clinic := petStoreDoc("/clinic", object("id", "name", "description"))

	result, err := New(dedupeOrRenameConfig()).JoinParsed([]parser.ParseResult{
		parsedAs(store, "store.yaml"),
		parsedAs(clinic, "clinic.yaml"),
	})
	require.NoError(t, err)

	schemas := joinedSchemas(t, result)
	assert.ElementsMatch(t, []string{"Pet", "Category", "Pet_v1", "Category_v1"}, slices.Collect(maps.Keys(schemas)),
		"the Categories differ, so neither they nor the Pets that reach them may collapse")

	// Clinic's callers must still be told that Pet.category has a description.
	assert.Equal(t, "#/components/schemas/Category_v1", schemas["Pet_v1"].Properties["category"].Ref)
	assert.Contains(t, schemas["Category_v1"].Properties, "description")
	assert.Equal(t, "#/components/schemas/Category", schemas["Pet"].Properties["category"].Ref)
	assert.NotContains(t, schemas["Category"].Properties, "description")

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok)
	assert.Equal(t, "#/components/schemas/Pet_v1",
		doc.Paths["/clinic"].Get.Responses.Codes["200"].Content["application/json"].Schema.Ref)
	assert.Equal(t, "#/components/schemas/Pet",
		doc.Paths["/store"].Get.Responses.Codes["200"].Content["application/json"].Schema.Ref)
}

// TestDeduplicateOrRename_AgreeingDocumentsCollapse covers the case the
// strategy exists for: documents that agree keep the one name they all wrote,
// and nothing is renamed.
func TestDeduplicateOrRename_AgreeingDocumentsCollapse(t *testing.T) {
	docs := make([]parser.ParseResult, 0, 4)
	for i := range 4 {
		docs = append(docs, parsedAs(
			flatDoc(fmt.Sprintf("/svc%d", i), map[string]*parser.Schema{"Common": object("id", "name")}),
			fmt.Sprintf("svc%d.yaml", i)))
	}

	result, err := New(dedupeOrRenameConfig()).JoinParsed(docs)
	require.NoError(t, err)

	assert.Equal(t, []string{"Common"}, slices.Sorted(maps.Keys(joinedSchemas(t, result))),
		"four agreeing documents should leave one schema under the name they all wrote")
	assert.Empty(t, result.StructuredWarnings.ByCategory(WarnSchemaRenamed),
		"no rename survived, so none should be reported")
	assert.Len(t, result.StructuredWarnings.ByCategory(WarnSchemaDeduplicated), 3)
}

// TestDeduplicateOrRename_SelfReference pins the behaviour of a schema that
// reaches itself, which is the documented limit of deciding once in its
// smallest form: the comparison asks what Node resolves to while Node is
// exactly what is being renamed.
//
// It also proves the recursion terminates. Both sides walk a cycle, and the
// comparison relies on its visited set to stop.
func TestDeduplicateOrRename_SelfReference(t *testing.T) {
	node := func(extra ...string) *parser.Schema {
		schema := object(append([]string{"id"}, extra...)...)
		schema.Properties["next"] = &parser.Schema{Ref: "#/components/schemas/Node"}
		return schema
	}
	docs := func(second *parser.Schema) []parser.ParseResult {
		return []parser.ParseResult{
			parsedAs(flatDoc("/one", map[string]*parser.Schema{"Node": node()}), "one.yaml"),
			parsedAs(flatDoc("/two", map[string]*parser.Schema{"Node": second}), "two.yaml"),
		}
	}

	assert.Equal(t, []string{"Node", "Node_v1"}, schemaNames(t, dedupeOrRenameConfig(), docs(node())),
		"agreeing self-referential schemas do not collapse: each was compared while its own next still mapped apart")

	assert.Equal(t, []string{"Node", "Node_v1"}, schemaNames(t, dedupeOrRenameConfig(), docs(node("extra"))),
		"and a genuine difference renames, as it would without the cycle")

	// A cycle is the one shape the second pass cannot finish either: after the
	// rewrite the two are still Node.next -> Node and Node_v1.next -> Node_v1.
	both := dedupeOrRenameConfig()
	both.SemanticDeduplication = true
	assert.Equal(t, []string{"Node", "Node_v1"}, schemaNames(t, both, docs(node())))
}

// TestDeduplicateOrRename_NeverFails contrasts the strategy with
// StrategyDeduplicateEquivalent, which fails the whole join on the first
// colliding pair that is not equivalent.
func TestDeduplicateOrRename_NeverFails(t *testing.T) {
	docs := []parser.ParseResult{
		parsedAs(flatDoc("/left", map[string]*parser.Schema{
			"Shared": object("id"), "Divergent": object("id"),
		}), "left.yaml"),
		parsedAs(flatDoc("/right", map[string]*parser.Schema{
			"Shared": object("id"), "Divergent": object("id", "extra"),
		}), "right.yaml"),
	}

	strict := DefaultConfig()
	strict.SchemaStrategy = StrategyDeduplicateEquivalent
	strict.EquivalenceMode = string(EquivalenceModeDeep)
	_, err := New(strict).JoinParsed(docs)
	require.Error(t, err, "deduplicate fails on a pair that is not equivalent")

	assert.Equal(t, []string{"Divergent", "Divergent_v1", "Shared"}, schemaNames(t, dedupeOrRenameConfig(), docs),
		"only the schema that differs is renamed; the identical pair collapses")
}

// TestDeduplicateOrRename_KeepsTheNameDocumentsWrote covers the naming rule:
// the surviving name is the one no rename generated, not the one that happens
// to sort first.
func TestDeduplicateOrRename_KeepsTheNameDocumentsWrote(t *testing.T) {
	docs := []parser.ParseResult{
		parsedAs(flatDoc("/one", map[string]*parser.Schema{"Common": object("id")}), "one.yaml"),
		parsedAs(flatDoc("/two", map[string]*parser.Schema{"Common": object("id")}), "two.yaml"),
	}

	config := dedupeOrRenameConfig()
	// Api_Common sorts before Common, so an alphabetical rule would discard the
	// name every document actually wrote.
	config.RenameTemplate = "Api_{{.Name}}"

	assert.Equal(t, []string{"Common"}, schemaNames(t, config, docs))
}

// TestDeduplicateOrRename_NamespacePrefix covers the other name the strategy
// can rename to: a prefix configured for the incoming document is used instead
// of the template, and is withdrawn like any other rename when the schemas
// agree.
func TestDeduplicateOrRename_NamespacePrefix(t *testing.T) {
	docs := []parser.ParseResult{
		parsedAs(flatDoc("/one", map[string]*parser.Schema{
			"Shared": object("id"), "Divergent": object("id"),
		}), "one.yaml"),
		parsedAs(flatDoc("/two", map[string]*parser.Schema{
			"Shared": object("id"), "Divergent": object("id", "extra"),
		}), "two.yaml"),
	}

	config := dedupeOrRenameConfig()
	config.NamespacePrefix = map[string]string{"two.yaml": "Billing"}

	assert.Equal(t, []string{"Billing_Divergent", "Divergent", "Shared"}, schemaNames(t, config, docs),
		"the prefix names the schema that differs; the identical pair still collapses")
}

// TestDeduplicateOrRename_InvalidTemplateFallsBack covers the fallback path
// with the template parsed once and reused: every rename in the join has to
// take it, not just the first.
func TestDeduplicateOrRename_InvalidTemplateFallsBack(t *testing.T) {
	// joinerLogger is package level, so this test must stay sequential: with
	// t.Parallel() the cleanup runs after the parent returns and races whatever
	// else is logging.
	restore := joinerLogger
	joinerLogger = slog.New(slog.NewTextHandler(io.Discard, nil))
	t.Cleanup(func() { joinerLogger = restore })

	docs := []parser.ParseResult{
		parsedAs(flatDoc("/one", map[string]*parser.Schema{
			"First": object("id"), "Second": object("id"),
		}), "one.yaml"),
		parsedAs(flatDoc("/two", map[string]*parser.Schema{
			"First": object("id", "extra"), "Second": object("id", "extra"),
		}), "two.yaml"),
	}

	config := dedupeOrRenameConfig()
	config.RenameTemplate = "{{.Name}"

	assert.Equal(t, []string{"First", "First_two", "Second", "Second_two"},
		schemaNames(t, config, docs),
		"both renames fall back to Name_Source, not just the one that parsed it")
}

// TestDeduplicateOrRename_WarningLocationUnderPrefix covers the source map
// lookup for the incoming document.
//
// Under AlwaysApplyPrefix the joined name is not the name that document wrote,
// and its source map only knows the name it wrote. Looking the joined name up
// found nothing and the warning lost its line and column.
func TestDeduplicateOrRename_WarningLocationUnderPrefix(t *testing.T) {
	spec := func(extra string) []byte {
		return []byte(`openapi: 3.0.3
info:
  title: API
  version: 1.0.0
paths: {}
components:
  schemas:
    Divergent:
      type: object
      properties:
        id:
          type: string
` + extra)
	}

	parse := func(name string, data []byte) (parser.ParseResult, *parser.SourceMap) {
		result, err := parser.ParseWithOptions(
			parser.WithBytes(data),
			parser.WithSourceName(name),
			parser.WithSourceMap(true),
		)
		require.NoError(t, err)
		return *result, result.SourceMap
	}

	one, mapOne := parse("one.yaml", spec(""))
	two, mapTwo := parse("two.yaml", spec("        extra:\n          type: string\n"))

	config := dedupeOrRenameConfig()
	// The same prefix for both, so the prefixed names still collide.
	config.NamespacePrefix = map[string]string{"one.yaml": "Api", "two.yaml": "Api"}
	config.AlwaysApplyPrefix = true

	j := New(config)
	j.SourceMaps = map[string]*parser.SourceMap{"one.yaml": mapOne, "two.yaml": mapTwo}

	result, err := j.JoinParsed([]parser.ParseResult{one, two})
	require.NoError(t, err)

	renamed := result.StructuredWarnings.ByCategory(WarnSchemaRenamed)
	require.Len(t, renamed, 1)
	assert.Positive(t, renamed[0].Line,
		"the incoming document is looked up under the name it wrote, not the prefixed one")
}

// TestDeduplicateOrRename_ReportsTheOutcomeNotTheRename checks that a rename
// the collapse withdraws is never reported as a rename.
func TestDeduplicateOrRename_ReportsTheOutcomeNotTheRename(t *testing.T) {
	config := dedupeOrRenameConfig()
	config.CollisionReport = true

	result, err := New(config).JoinParsed([]parser.ParseResult{
		parsedAs(flatDoc("/one", map[string]*parser.Schema{
			"Shared": object("id"), "Divergent": object("id"),
		}), "one.yaml"),
		parsedAs(flatDoc("/two", map[string]*parser.Schema{
			"Shared": object("id"), "Divergent": object("id", "extra"),
		}), "two.yaml"),
	})
	require.NoError(t, err)

	renamed := result.StructuredWarnings.ByCategory(WarnSchemaRenamed)
	require.Len(t, renamed, 1, "only the schema that differs was really renamed")
	assert.Equal(t, "Divergent", renamed[0].Context["original_name"])
	assert.Equal(t, "Divergent_v1", renamed[0].Context["new_name"])

	deduplicated := result.StructuredWarnings.ByCategory(WarnSchemaDeduplicated)
	require.Len(t, deduplicated, 1, "the identical pair collapsed, so its rename is not reported")
	assert.Contains(t, deduplicated[0].Message, "Shared")

	require.NotNil(t, result.CollisionDetails)
	assert.Equal(t, 1, result.CollisionDetails.ResolvedByRename)
	assert.Equal(t, 1, result.CollisionDetails.ResolvedByDedup)
}

// TestDeduplicateOrRename_DiscriminatorMappingIsRenameAware covers the second
// place a schema names another schema. Both discriminators spell the same
// mapping, and only the pending renames tell them apart.
func TestDeduplicateOrRename_DiscriminatorMappingIsRenameAware(t *testing.T) {
	build := func(path, dogProperty string) *parser.OAS3Document {
		return flatDoc(path, map[string]*parser.Schema{
			"Animal": {
				Type:  "object",
				OneOf: []*parser.Schema{{Ref: "#/components/schemas/Dog"}},
				Discriminator: &parser.Discriminator{
					PropertyName: "kind",
					Mapping:      map[string]string{"dog": "#/components/schemas/Dog"},
					// defaultMapping names a schema the same way a mapping value
					// does, and the rewriter rewrites both, so both have to be
					// read through the view.
					DefaultMapping: "#/components/schemas/Dog",
				},
			},
			"Dog": object(dogProperty),
		})
	}

	result, err := New(dedupeOrRenameConfig()).JoinParsed([]parser.ParseResult{
		parsedAs(build("/one", "bark"), "one.yaml"),
		parsedAs(build("/two", "woof"), "two.yaml"),
	})
	require.NoError(t, err)

	schemas := joinedSchemas(t, result)
	assert.ElementsMatch(t, []string{"Animal", "Dog", "Animal_v1", "Dog_v1"}, slices.Collect(maps.Keys(schemas)),
		"the Dogs differ, so the Animals that discriminate to them differ too")
	assert.Equal(t, "#/components/schemas/Dog_v1", schemas["Animal_v1"].Discriminator.Mapping["dog"])
	assert.Equal(t, "#/components/schemas/Dog_v1", schemas["Animal_v1"].Discriminator.DefaultMapping)
}

// TestDeduplicateOrRename_DefaultMappingAloneDecides isolates defaultMapping in
// the comparison.
//
// Above it rides along with a mapping and a oneOf that name Dog too, so the
// Animals would be told apart even if defaultMapping were compared as written.
// Here it is the only thing that names Dog, so reading it through the view is
// the whole verdict: without that, two Animals selecting different subschemas
// would collapse into one.
func TestDeduplicateOrRename_DefaultMappingAloneDecides(t *testing.T) {
	build := func(path, dogProperty string) *parser.OAS3Document {
		return flatDoc(path, map[string]*parser.Schema{
			"Animal": {
				Type:       "object",
				Properties: map[string]*parser.Schema{"kind": {Type: "string"}},
				Discriminator: &parser.Discriminator{
					PropertyName:   "kind",
					DefaultMapping: "#/components/schemas/Dog",
				},
			},
			"Dog": object(dogProperty),
		})
	}

	assert.Equal(t, []string{"Animal", "Animal_v1", "Dog", "Dog_v1"},
		schemaNames(t, dedupeOrRenameConfig(), []parser.ParseResult{
			parsedAs(build("/one", "bark"), "one.yaml"),
			parsedAs(build("/two", "woof"), "two.yaml"),
		}),
		"the Dogs differ, so the Animals that fall back to them differ too")
}

// TestDeduplicateOrRename_OAS2 covers the definitions section, which runs the
// same collapse over its own merge loop.
func TestDeduplicateOrRename_OAS2(t *testing.T) {
	build := func(path string, properties ...string) parser.ParseResult {
		return parser.ParseResult{
			Version:      "2.0",
			OASVersion:   parser.OASVersion20,
			SourcePath:   path[1:] + ".yaml",
			SourceFormat: parser.SourceFormatYAML,
			Document: &parser.OAS2Document{
				Swagger:    "2.0",
				Info:       &parser.Info{Title: "API", Version: "1.0.0"},
				OASVersion: parser.OASVersion20,
				Paths: parser.Paths{
					path: {Get: &parser.Operation{Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "ok", Schema: &parser.Schema{Ref: "#/definitions/Pet"}},
						},
					}}},
				},
				Definitions: map[string]*parser.Schema{
					"Pet": {
						Type:       "object",
						Properties: map[string]*parser.Schema{"category": {Ref: "#/definitions/Category"}},
					},
					"Category": object(properties...),
				},
			},
		}
	}

	result, err := New(dedupeOrRenameConfig()).JoinParsed([]parser.ParseResult{
		build("/one", "id"),
		build("/two", "id", "note"),
	})
	require.NoError(t, err)

	doc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok)
	assert.ElementsMatch(t, []string{"Pet", "Category", "Pet_v1", "Category_v1"}, slices.Collect(maps.Keys(doc.Definitions)))
	assert.Equal(t, "#/definitions/Category_v1", doc.Definitions["Pet_v1"].Properties["category"].Ref)
	assert.Equal(t, "#/definitions/Pet_v1", doc.Paths["/two"].Get.Responses.Codes["200"].Schema.Ref)
}

// TestDeduplicateOrRename_LeavesInputsAlone checks the strategy keeps the
// promise JoinParsed makes: the documents the caller handed in are not
// modified, including the ones whose renames were withdrawn.
func TestDeduplicateOrRename_LeavesInputsAlone(t *testing.T) {
	first := petStoreDoc("/one", object("id"))
	second := petStoreDoc("/two", object("id", "extra"))

	_, err := New(dedupeOrRenameConfig()).JoinParsed([]parser.ParseResult{
		parsedAs(first, "one.yaml"),
		parsedAs(second, "two.yaml"),
	})
	require.NoError(t, err)

	for _, doc := range []*parser.OAS3Document{first, second} {
		assert.ElementsMatch(t, []string{"Pet", "Category"}, slices.Collect(maps.Keys(doc.Components.Schemas)))
		assert.Equal(t, "#/components/schemas/Category",
			doc.Components.Schemas["Pet"].Properties["category"].Ref)
	}
}

// TestDeduplicateOrRename_MatchesRenameRightThenDedup pins the two claims the
// strategy is worth having for.
//
// One pass reaches what rename-right followed by semantic deduplication
// reaches, so nothing is given up. Neither collapses the Pets, because each
// was compared while its own Category still mapped somewhere else: that is the
// documented limit of deciding once. Turning semantic deduplication on as well
// clears it, because the withdrawn Category renames leave the Pets identical
// in the rewritten document, and no configuration available before could do
// that.
func TestDeduplicateOrRename_MatchesRenameRightThenDedup(t *testing.T) {
	docs := make([]parser.ParseResult, 0, 4)
	for i := range 4 {
		docs = append(docs, parsedAs(petStoreDoc(fmt.Sprintf("/svc%d", i), object("id", "name")),
			fmt.Sprintf("svc%d.yaml", i)))
	}

	renameRight := DefaultConfig()
	renameRight.SchemaStrategy = StrategyRenameRight
	renameRight.RenameTemplate = "{{.Name}}_v{{.Index}}"
	renameRight.SemanticDeduplication = true

	survivors := []string{"Category", "Pet", "Pet_v1", "Pet_v2", "Pet_v3"}
	assert.Equal(t, survivors, schemaNames(t, renameRight, docs))
	assert.Equal(t, survivors, schemaNames(t, dedupeOrRenameConfig(), docs))

	both := dedupeOrRenameConfig()
	both.SemanticDeduplication = true
	assert.Equal(t, []string{"Category", "Pet"}, schemaNames(t, both, docs))
}
