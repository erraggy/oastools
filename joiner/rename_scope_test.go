package joiner

import (
	"sort"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// petstoreFamily builds an OAS 2.0 document with a Pet whose category property
// refs Category, reached through one path. withDescription makes Category
// genuinely differ between two documents.
func petstoreFamily(name string, withDescription bool) parser.ParseResult {
	category := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"id":   {Type: "integer", Format: "int64"},
			"name": {Type: "string"},
		},
	}
	if withDescription {
		category.Properties["description"] = &parser.Schema{Type: "string"}
	}

	return parser.ParseResult{
		Document: &parser.OAS2Document{
			Swagger: "2.0",
			Info:    &parser.Info{Title: name, Version: "1.0.0"},
			Paths: parser.Paths{
				"/" + name + "/pet/{petId}": &parser.PathItem{Get: &parser.Operation{
					OperationID: "getPetById" + name,
					Responses: &parser.Responses{Codes: map[string]*parser.Response{
						"200": {
							Description: "successful operation",
							Schema:      &parser.Schema{Ref: "#/definitions/Pet"},
						},
					}},
				}},
			},
			Definitions: map[string]*parser.Schema{
				"Pet": {
					Type:     "object",
					Required: []string{"name", "photoUrls"},
					Properties: map[string]*parser.Schema{
						"id":        {Type: "integer", Format: "int64"},
						"category":  {Ref: "#/definitions/Category"},
						"name":      {Type: "string"},
						"photoUrls": {Type: "array", Items: &parser.Schema{Type: "string"}},
						"status": {
							Type: "string",
							Enum: []any{"available", "pending", "sold"},
						},
					},
				},
				"Category": category,
			},
			OASVersion: parser.OASVersion20,
		},
		Version:      "2.0",
		OASVersion:   parser.OASVersion20,
		SourcePath:   name,
		SourceFormat: parser.SourceFormatJSON,
	}
}

// petResponseRef returns the $ref of the 200 response schema for a path.
func petResponseRef(t *testing.T, doc *parser.OAS2Document, path string) string {
	t.Helper()
	item, ok := doc.Paths[path]
	require.True(t, ok, "path %s missing", path)
	require.NotNil(t, item.Get)
	require.NotNil(t, item.Get.Responses)
	resp, ok := item.Get.Responses.Codes["200"]
	require.True(t, ok, "200 response missing on %s", path)
	require.NotNil(t, resp.Schema)
	return resp.Schema.Ref
}

// definitionRefs collects every $ref in the fixtures: definition properties and
// path response schemas.
func definitionRefs(doc *parser.OAS2Document) []string {
	var refs []string
	for _, schema := range doc.Definitions {
		for _, prop := range schema.Properties {
			if prop.Ref != "" {
				refs = append(refs, prop.Ref)
			}
		}
	}
	for _, item := range doc.Paths {
		if item.Get == nil || item.Get.Responses == nil {
			continue
		}
		for _, resp := range item.Get.Responses.Codes {
			if resp.Schema != nil && resp.Schema.Ref != "" {
				refs = append(refs, resp.Schema.Ref)
			}
		}
	}
	sort.Strings(refs)
	return refs
}

// TestRenameScopeRenameRight covers #478: renaming an incoming schema must not
// repoint references that arrived with an earlier document.
func TestRenameScopeRenameRight(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(petstoreFamily("store", false), petstoreFamily("clinic", true)),
		WithSchemaStrategy(StrategyRenameRight),
		WithEquivalenceMode("deep"),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS2Document)

	// store's Pet was not renamed, and store's Category kept its name, so this ref
	// should still name it.
	assert.Equal(t, "#/definitions/Category", d.Definitions["Pet"].Properties["category"].Ref)
	assert.Equal(t, "#/definitions/Pet", petResponseRef(t, d, "/store/pet/{petId}"))

	// clinic's Pet was renamed, and so was the Category it was written against.
	assert.Equal(t, "#/definitions/Category.clinic", d.Definitions["Pet.clinic"].Properties["category"].Ref)
	assert.Equal(t, "#/definitions/Pet.clinic", petResponseRef(t, d, "/clinic/pet/{petId}"))

	// The two Categories stay distinct, which is why the rename happened at all.
	assert.NotContains(t, d.Definitions["Category"].Properties, "description")
	assert.Contains(t, d.Definitions["Category.clinic"].Properties, "description")
}

// TestRenameScopeRenameLeft is the mirror case: renaming a schema already in the
// joined document moves the earlier documents' references, and only those.
func TestRenameScopeRenameLeft(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(petstoreFamily("store", false), petstoreFamily("clinic", true)),
		WithSchemaStrategy(StrategyRenameLeft),
		WithEquivalenceMode("deep"),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS2Document)

	// store's schemas were moved aside, so store's references follow them.
	assert.Equal(t, "#/definitions/Category.store", d.Definitions["Pet.store"].Properties["category"].Ref)
	assert.Equal(t, "#/definitions/Pet.store", petResponseRef(t, d, "/store/pet/{petId}"))

	// clinic's schemas took the original names, so clinic's references stand.
	assert.Equal(t, "#/definitions/Category", d.Definitions["Pet"].Properties["category"].Ref)
	assert.Equal(t, "#/definitions/Pet", petResponseRef(t, d, "/clinic/pet/{petId}"))

	assert.NotContains(t, d.Definitions["Category.store"].Properties, "description")
	assert.Contains(t, d.Definitions["Category"].Properties, "description")
}

// TestRenameScopeIncrementalJoin covers the first follow-on observation in #478:
// a caller accumulating combined = join(combined, next) feeds the output back in,
// where a reference left pointing at an alias made aliases pile up.
func TestRenameScopeIncrementalJoin(t *testing.T) {
	first, err := JoinWithOptions(
		WithParsed(petstoreFamily("store", false), petstoreFamily("clinic", true)),
		WithSchemaStrategy(StrategyRenameRight),
		WithEquivalenceMode("deep"),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	// store2 is byte-identical to store, so deduplicating it against the
	// accumulator must succeed rather than report a $ref target mismatch.
	second, err := JoinWithOptions(
		WithParsed(*first.ToParseResult(), petstoreFamily("store2", false)),
		WithSchemaStrategy(StrategyDeduplicateEquivalent),
		WithEquivalenceMode("deep"),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	d := second.Document.(*parser.OAS2Document)

	// No new alias: the four definitions from the first join are all there is.
	assert.ElementsMatch(t,
		[]string{"Pet", "Category", "Pet.clinic", "Category.clinic"},
		definitionNames(d))
	assert.Equal(t, "#/definitions/Category", d.Definitions["Pet"].Properties["category"].Ref)
	assert.Equal(t, "#/definitions/Pet", petResponseRef(t, d, "/store2/pet/{petId}"))
}

// TestRenameScopeSemanticDeduplication covers the second follow-on observation in
// #478: the collision and deduplication passes shared one rewriter and could
// register opposing directions for the same name.
func TestRenameScopeSemanticDeduplication(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(petstoreFamily("store", false), petstoreFamily("clinic", false)),
		WithSchemaStrategy(StrategyRenameRight),
		WithEquivalenceMode("deep"),
		WithSemanticDeduplication(true),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS2Document)

	// The two Categories were identical, so deduplication kept one.
	assert.NotContains(t, d.Definitions, "Category.clinic")

	// Every reference names a definition that is still in the document.
	for _, ref := range definitionRefs(d) {
		name := extractSchemaName(ref)
		assert.Contains(t, d.Definitions, name, "dangling reference %s", ref)
	}
}

// TestRenameScopeNamespacePrefixThenCollision checks that a schema renamed twice,
// by a prefix then a collision, resolves in one step: its references spell the
// original name, not the prefixed one.
func TestRenameScopeNamespacePrefixThenCollision(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(petstoreFamily("store", false), petstoreFamily("clinic", true)),
		WithSchemaStrategy(StrategyRenameRight),
		WithEquivalenceMode("deep"),
		WithNamespacePrefix("store", "Api"),
		WithNamespacePrefix("clinic", "Api"),
		WithAlwaysApplyPrefix(true),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS2Document)

	assert.Equal(t, "#/definitions/Api_Category", d.Definitions["Api_Pet"].Properties["category"].Ref)
	assert.Equal(t, "#/definitions/Api_Category.clinic", d.Definitions["Api_Pet.clinic"].Properties["category"].Ref)

	for _, ref := range definitionRefs(d) {
		name := extractSchemaName(ref)
		assert.Contains(t, d.Definitions, name, "dangling reference %s", ref)
	}
}

// petstoreFamilyOAS3 is the OAS 3.0 counterpart of petstoreFamily.
func petstoreFamilyOAS3(name string, withDescription bool) parser.ParseResult {
	category := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"id":   {Type: "integer", Format: "int64"},
			"name": {Type: "string"},
		},
	}
	if withDescription {
		category.Properties["description"] = &parser.Schema{Type: "string"}
	}

	return parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: name, Version: "1.0.0"},
			Paths: parser.Paths{
				"/" + name + "/pet/{petId}": &parser.PathItem{Get: &parser.Operation{
					OperationID: "getPetById" + name,
					Responses: &parser.Responses{Codes: map[string]*parser.Response{
						"200": {
							Description: "successful operation",
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
						Type: "object",
						Properties: map[string]*parser.Schema{
							"id":       {Type: "integer", Format: "int64"},
							"category": {Ref: "#/components/schemas/Category"},
							"name":     {Type: "string"},
						},
					},
					"Category": category,
				},
			},
			OASVersion: parser.OASVersion303,
		},
		Version:      "3.0.3",
		OASVersion:   parser.OASVersion303,
		SourcePath:   name,
		SourceFormat: parser.SourceFormatJSON,
	}
}

// TestRenameScopeOAS3 confirms the OAS 3 path scopes renames the same way. #478
// reproduced on OAS 2.0 but noted the same rewrite-then-traverse structure here.
func TestRenameScopeOAS3(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(petstoreFamilyOAS3("store", false), petstoreFamilyOAS3("clinic", true)),
		WithSchemaStrategy(StrategyRenameRight),
		WithEquivalenceMode("deep"),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS3Document)
	schemas := d.Components.Schemas

	assert.Equal(t, "#/components/schemas/Category", schemas["Pet"].Properties["category"].Ref)
	assert.Equal(t, "#/components/schemas/Category.clinic", schemas["Pet.clinic"].Properties["category"].Ref)

	assert.Equal(t, "#/components/schemas/Pet",
		oas3ResponseRef(t, d, "/store/pet/{petId}"))
	assert.Equal(t, "#/components/schemas/Pet.clinic",
		oas3ResponseRef(t, d, "/clinic/pet/{petId}"))
}

func oas3ResponseRef(t *testing.T, doc *parser.OAS3Document, path string) string {
	t.Helper()
	item, ok := doc.Paths[path]
	require.True(t, ok, "path %s missing", path)
	require.NotNil(t, item.Get)
	require.NotNil(t, item.Get.Responses)
	resp, ok := item.Get.Responses.Codes["200"]
	require.True(t, ok, "200 response missing on %s", path)
	media, ok := resp.Content["application/json"]
	require.True(t, ok, "json content missing on %s", path)
	require.NotNil(t, media.Schema)
	return media.Schema.Ref
}

// TestRenameScopeCoversEveryContainer exercises every top-level container the
// rewriter traverses, so a container renameScope forgets to attribute fails here
// rather than silently taking every document's renames.
func TestRenameScopeCoversEveryContainer(t *testing.T) {
	// containerRef builds an operation whose 200 response refs Target.
	containerRef := func() *parser.Operation {
		return &parser.Operation{
			Responses: &parser.Responses{Codes: map[string]*parser.Response{
				"200": {
					Description: "ok",
					Content: map[string]*parser.MediaType{
						"application/json": {Schema: &parser.Schema{Ref: "#/components/schemas/Target"}},
					},
				},
			}},
		}
	}

	// Components are named after their source so only Target collides.
	doc := func(name string, extra bool) parser.ParseResult {
		target := &parser.Schema{
			Type:       "object",
			Properties: map[string]*parser.Schema{"id": {Type: "string"}},
		}
		if extra {
			target.Properties["note"] = &parser.Schema{Type: "string"}
		}
		targetRef := func() *parser.Schema { return &parser.Schema{Ref: "#/components/schemas/Target"} }
		jsonContent := func() map[string]*parser.MediaType {
			return map[string]*parser.MediaType{"application/json": {Schema: targetRef()}}
		}

		callback := parser.Callback{"{$request.body#/url}": &parser.PathItem{Post: containerRef()}}

		return parser.ParseResult{
			Document: &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: name, Version: "1.0.0"},
				Paths: parser.Paths{
					"/" + name: &parser.PathItem{Get: containerRef()},
				},
				Webhooks: map[string]*parser.PathItem{
					name + "Hook": {Post: containerRef()},
				},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"Target": target,
						name + "Holder": {
							Type:       "object",
							Properties: map[string]*parser.Schema{"target": targetRef()},
						},
					},
					Parameters: map[string]*parser.Parameter{
						name + "Param": {Name: "q", In: "query", Schema: targetRef()},
					},
					Responses: map[string]*parser.Response{
						name + "Resp": {Description: "ok", Content: jsonContent()},
					},
					RequestBodies: map[string]*parser.RequestBody{
						name + "Body": {Content: jsonContent()},
					},
					Headers: map[string]*parser.Header{
						name + "Header": {Schema: targetRef()},
					},
					Callbacks: map[string]*parser.Callback{
						name + "CB": &callback,
					},
					PathItems: map[string]*parser.PathItem{
						name + "PI": {Get: containerRef()},
					},
				},
				OASVersion: parser.OASVersion310,
			},
			Version:      "3.1.0",
			OASVersion:   parser.OASVersion310,
			SourcePath:   name,
			SourceFormat: parser.SourceFormatJSON,
		}
	}

	res, err := JoinWithOptions(
		WithParsed(doc("a", false), doc("b", true)),
		WithSchemaStrategy(StrategyRenameRight),
		WithEquivalenceMode("deep"),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS3Document)
	c := d.Components
	require.Contains(t, c.Schemas, "Target.b", "b's Target should have been renamed")

	// Each document's references, keyed by the container they live in.
	refs := func(source string) map[string]string {
		callback := *c.Callbacks[source+"CB"]
		return map[string]string{
			"components.schemas":       c.Schemas[source+"Holder"].Properties["target"].Ref,
			"components.parameters":    c.Parameters[source+"Param"].Schema.Ref,
			"components.responses":     c.Responses[source+"Resp"].Content["application/json"].Schema.Ref,
			"components.requestBodies": c.RequestBodies[source+"Body"].Content["application/json"].Schema.Ref,
			"components.headers":       c.Headers[source+"Header"].Schema.Ref,
			"components.callbacks":     oas3ResponseRefIn(t, callback["{$request.body#/url}"].Post),
			"components.pathItems":     oas3ResponseRefIn(t, c.PathItems[source+"PI"].Get),
			"paths":                    oas3ResponseRefIn(t, d.Paths["/"+source].Get),
			"webhooks":                 oas3ResponseRefIn(t, d.Webhooks[source+"Hook"].Post),
		}
	}

	for container, ref := range refs("b") {
		assert.Equal(t, "#/components/schemas/Target.b", ref,
			"%s: b's reference was not rewritten, so renameScope does not attribute this container", container)
	}
	for container, ref := range refs("a") {
		assert.Equal(t, "#/components/schemas/Target", ref,
			"%s: a's reference was repointed at b's schema", container)
	}
}

// TestRenameScopeCoversEveryContainerOAS2 is the OAS 2 counterpart, covering the
// four containers applyOAS2 attributes and rewriteOAS2Document traverses.
func TestRenameScopeCoversEveryContainerOAS2(t *testing.T) {
	targetRef := func() *parser.Schema { return &parser.Schema{Ref: "#/definitions/Target"} }

	doc := func(name string, extra bool) parser.ParseResult {
		target := &parser.Schema{
			Type:       "object",
			Properties: map[string]*parser.Schema{"id": {Type: "string"}},
		}
		if extra {
			target.Properties["note"] = &parser.Schema{Type: "string"}
		}

		return parser.ParseResult{
			Document: &parser.OAS2Document{
				Swagger: "2.0",
				Info:    &parser.Info{Title: name, Version: "1.0.0"},
				Paths: parser.Paths{
					"/" + name: &parser.PathItem{Get: &parser.Operation{
						Responses: &parser.Responses{Codes: map[string]*parser.Response{
							"200": {Description: "ok", Schema: targetRef()},
						}},
					}},
				},
				Definitions: map[string]*parser.Schema{
					"Target": target,
					name + "Holder": {
						Type:       "object",
						Properties: map[string]*parser.Schema{"target": targetRef()},
					},
				},
				Parameters: map[string]*parser.Parameter{
					name + "Param": {Name: "body", In: "body", Schema: targetRef()},
				},
				Responses: map[string]*parser.Response{
					name + "Resp": {Description: "ok", Schema: targetRef()},
				},
				OASVersion: parser.OASVersion20,
			},
			Version:      "2.0",
			OASVersion:   parser.OASVersion20,
			SourcePath:   name,
			SourceFormat: parser.SourceFormatJSON,
		}
	}

	res, err := JoinWithOptions(
		WithParsed(doc("a", false), doc("b", true)),
		WithSchemaStrategy(StrategyRenameRight),
		WithEquivalenceMode("deep"),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS2Document)
	require.Contains(t, d.Definitions, "Target.b", "b's Target should have been renamed")

	refs := func(source string) map[string]string {
		return map[string]string{
			"definitions": d.Definitions[source+"Holder"].Properties["target"].Ref,
			"parameters":  d.Parameters[source+"Param"].Schema.Ref,
			"responses":   d.Responses[source+"Resp"].Schema.Ref,
			"paths":       petResponseRef(t, d, "/"+source),
		}
	}

	for container, ref := range refs("b") {
		assert.Equal(t, "#/definitions/Target.b", ref,
			"%s: b's reference was not rewritten, so renameScope does not attribute this container", container)
	}
	for container, ref := range refs("a") {
		assert.Equal(t, "#/definitions/Target", ref,
			"%s: a's reference was repointed at b's definition", container)
	}
}

func oas3ResponseRefIn(t *testing.T, op *parser.Operation) string {
	t.Helper()
	require.NotNil(t, op)
	require.NotNil(t, op.Responses)
	resp, ok := op.Responses.Codes["200"]
	require.True(t, ok, "200 response missing")
	media, ok := resp.Content["application/json"]
	require.True(t, ok, "json content missing")
	require.NotNil(t, media.Schema)
	return media.Schema.Ref
}

// TestRenameScopeRewritesHandlerCustomValues covers the one entry belonging to no
// source document: a value from ResolutionCustom. Scoping must not skip it, or
// its references keep naming schemas the join no longer has.
func TestRenameScopeRewritesHandlerCustomValues(t *testing.T) {
	// Prefixing both documents means every reference needs rewriting, so a value
	// that is skipped dangles.
	handler := func(c CollisionContext) (CollisionResolution, error) {
		if c.Type == CollisionTypeSchema && c.Name == "Api_Pet" {
			return UseCustomValue(&parser.Schema{
				Type: "object",
				Properties: map[string]*parser.Schema{
					"category": {Ref: "#/definitions/Category"},
				},
			}), nil
		}
		return ContinueWithStrategy(), nil
	}

	res, err := JoinWithOptions(
		WithParsed(petstoreFamily("a", false), petstoreFamily("b", true)),
		WithSchemaStrategy(StrategyRenameRight),
		WithEquivalenceMode("deep"),
		WithNamespacePrefix("a", "Api"),
		WithNamespacePrefix("b", "Api"),
		WithAlwaysApplyPrefix(true),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
		WithCollisionHandler(handler),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS2Document)
	require.Contains(t, d.Definitions, "Api_Pet")

	// Both documents renamed Category, so merge order breaks the tie.
	assert.Equal(t, "#/definitions/Api_Category.b",
		d.Definitions["Api_Pet"].Properties["category"].Ref,
		"the handler's value was skipped by every pass, so its reference was never rewritten")
	for _, ref := range definitionRefs(d) {
		assert.Contains(t, d.Definitions, extractSchemaName(ref), "dangling reference %s", ref)
	}
}

// TestRenameScopeRewritesHandlerCustomPathItem is the same case for paths, the
// other resolution that accepts a custom value.
func TestRenameScopeRewritesHandlerCustomPathItem(t *testing.T) {
	// Both documents publish the same path, so the handler can replace it.
	const shared = "/shared/pet"
	withSharedPath := func(name string, withDescription bool) parser.ParseResult {
		res := petstoreFamily(name, withDescription)
		doc := res.Document.(*parser.OAS2Document)
		doc.Paths[shared] = &parser.PathItem{Get: &parser.Operation{
			OperationID: "shared" + name,
			Responses: &parser.Responses{Codes: map[string]*parser.Response{
				"200": {Description: "ok", Schema: &parser.Schema{Ref: "#/definitions/Pet"}},
			}},
		}}
		return res
	}

	handler := func(c CollisionContext) (CollisionResolution, error) {
		if c.Type == CollisionTypePath && c.Name == shared {
			return UseCustomValue(&parser.PathItem{Get: &parser.Operation{
				OperationID: "sharedMerged",
				Responses: &parser.Responses{Codes: map[string]*parser.Response{
					"200": {Description: "ok", Schema: &parser.Schema{Ref: "#/definitions/Pet"}},
				}},
			}}), nil
		}
		return ContinueWithStrategy(), nil
	}

	res, err := JoinWithOptions(
		WithParsed(withSharedPath("a", false), withSharedPath("b", true)),
		WithSchemaStrategy(StrategyRenameRight),
		WithEquivalenceMode("deep"),
		WithNamespacePrefix("a", "Api"),
		WithNamespacePrefix("b", "Api"),
		WithAlwaysApplyPrefix(true),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
		WithCollisionHandler(handler),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS2Document)
	ref := petResponseRef(t, d, shared)
	assert.Equal(t, "#/definitions/Api_Pet.b", ref,
		"the handler's path item was skipped by every pass, so its reference was never rewritten")
	assert.Contains(t, d.Definitions, extractSchemaName(ref), "dangling reference %s", ref)
}

// petVariant is a minimal document whose one definition carries a property named
// after the document, so a schema can be traced to its source.
func petVariant(name string) parser.ParseResult {
	return parser.ParseResult{
		Document: &parser.OAS2Document{
			Swagger: "2.0",
			Info:    &parser.Info{Title: name, Version: "1.0.0"},
			Paths: parser.Paths{
				"/" + name: &parser.PathItem{Get: &parser.Operation{
					Responses: &parser.Responses{Codes: map[string]*parser.Response{
						"200": {Description: "ok", Schema: &parser.Schema{Ref: "#/definitions/Pet"}},
					}},
				}},
			},
			Definitions: map[string]*parser.Schema{
				"Pet": {Type: "object", Properties: map[string]*parser.Schema{
					"from" + name: {Type: "string"},
				}},
			},
			OASVersion: parser.OASVersion20,
		},
		Version:      "2.0",
		OASVersion:   parser.OASVersion20,
		SourcePath:   name,
		SourceFormat: parser.SourceFormatJSON,
	}
}

// TestRenameLeftNamesEachContributor covers #479: rename-left named the moved
// schema after the first document every time, so a three document join
// generated the same name twice and lost a schema.
func TestRenameLeftNamesEachContributor(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(petVariant("a"), petVariant("b"), petVariant("c")),
		WithSchemaStrategy(StrategyRenameLeft),
		WithPathStrategy(StrategyAcceptLeft),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS2Document)

	// Every document keeps a schema, each named after the document it came from
	// except the last, which holds the original name.
	assert.ElementsMatch(t, []string{"Pet.a", "Pet.b", "Pet"}, definitionNames(d))
	assert.Contains(t, d.Definitions["Pet.a"].Properties, "froma")
	assert.Contains(t, d.Definitions["Pet.b"].Properties, "fromb")
	assert.Contains(t, d.Definitions["Pet"].Properties, "fromc")

	// And each document's path names its own schema.
	assert.Equal(t, "#/definitions/Pet.a", petResponseRef(t, d, "/a"))
	assert.Equal(t, "#/definitions/Pet.b", petResponseRef(t, d, "/b"))
	assert.Equal(t, "#/definitions/Pet", petResponseRef(t, d, "/c"))
}

// TestRenameLeftReportsContributingSource checks that the collision report and
// the rename name agree on which document the left schema came from.
func TestRenameLeftReportsContributingSource(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(petVariant("a"), petVariant("b"), petVariant("c")),
		WithSchemaStrategy(StrategyRenameLeft),
		WithPathStrategy(StrategyAcceptLeft),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
		WithCollisionReport(true),
	)
	require.NoError(t, err)
	require.NotNil(t, res.CollisionDetails)

	byNewName := make(map[string]string)
	for _, event := range res.CollisionDetails.Events {
		byNewName[event.NewName] = event.LeftSource
	}
	assert.Equal(t, "a", byNewName["Pet.a"])
	assert.Equal(t, "b", byNewName["Pet.b"], "the second rename moved b's schema, not a's")

	assert.Contains(t, res.Warnings, "definition 'Pet' from a renamed to 'Pet.a' (incoming document takes the original name)")
	assert.Contains(t, res.Warnings, "definition 'Pet' from b renamed to 'Pet.b' (incoming document takes the original name)")
}

// TestJoinSameDocumentTwice covers #481: the two positions shared every pointer,
// so keeping both sides stored one schema under two names.
func TestJoinSameDocumentTwice(t *testing.T) {
	doc := petVariant("a")

	res, err := JoinWithOptions(
		WithParsed(doc, doc),
		WithSchemaStrategy(StrategyRenameRight),
		WithPathStrategy(StrategyAcceptLeft),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS2Document)
	require.Contains(t, d.Definitions, "Pet")
	require.Contains(t, d.Definitions, "Pet.a")
	assert.NotSame(t, d.Definitions["Pet"], d.Definitions["Pet.a"],
		"the two positions must not share one schema")

	// Editing one must not be visible through the other.
	d.Definitions["Pet.a"].Description = "the second position"
	assert.Empty(t, d.Definitions["Pet"].Description)
}

// TestRenameTargetAlreadyTaken covers #483: a generated name that the documents
// already use must not overwrite the schema stored under it.
func TestRenameTargetAlreadyTaken(t *testing.T) {
	docA := petVariant("a")
	// a already has a schema under the name the template will generate for b's Pet.
	docA.Document.(*parser.OAS2Document).Definitions["Pet.b"] = &parser.Schema{
		Type:       "object",
		Properties: map[string]*parser.Schema{"preexisting": {Type: "string"}},
	}

	res, err := JoinWithOptions(
		WithParsed(docA, petVariant("b")),
		WithSchemaStrategy(StrategyRenameRight),
		WithPathStrategy(StrategyAcceptLeft),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS2Document)

	// Three definitions in, three out: a's Pet, a's own Pet.b, and b's renamed Pet.
	assert.ElementsMatch(t, []string{"Pet", "Pet.b", "Pet.b_2"}, definitionNames(d))
	assert.Contains(t, d.Definitions["Pet.b"].Properties, "preexisting",
		"a's own Pet.b was overwritten by b's renamed Pet")
	assert.Contains(t, d.Definitions["Pet.b_2"].Properties, "fromb")

	// b's path names the schema under the name it actually ended up with.
	assert.Equal(t, "#/definitions/Pet.b_2", petResponseRef(t, d, "/b"))
	assert.Contains(t, res.Warnings, "definition 'Pet' from b renamed to 'Pet.b_2'")
}

// TestRenameTemplateWithoutName covers the other way into #483: a template that
// discards the schema name generates one name for every schema.
func TestRenameTemplateWithoutName(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(petstoreFamily("store", false), petstoreFamily("clinic", true)),
		WithSchemaStrategy(StrategyRenameRight),
		WithEquivalenceMode("deep"),
		WithRenameTemplate(`{{.Source}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS2Document)

	// clinic's two schemas both generated the name "clinic", so one is suffixed
	// rather than dropped.
	assert.ElementsMatch(t, []string{"Pet", "Category", "clinic", "clinic_2"}, definitionNames(d))
	for _, ref := range definitionRefs(d) {
		assert.Contains(t, d.Definitions, extractSchemaName(ref), "dangling reference %s", ref)
	}
}

func TestUniqueSchemaName(t *testing.T) {
	taken := map[string]*parser.Schema{
		"Pet":     {},
		"Pet_2":   {},
		"Pet_3":   {},
		"Unrelat": {},
	}

	assert.Equal(t, "Cat", uniqueSchemaName(taken, "Cat"), "a free name is returned unchanged")
	assert.Equal(t, "Pet_4", uniqueSchemaName(taken, "Pet"), "the first free suffix is used")
	assert.Equal(t, "Pet_2_2", uniqueSchemaName(taken, "Pet_2"), "suffixing is applied to the candidate as given")
	assert.Equal(t, "Cat", uniqueSchemaName(map[string]*parser.Schema{}, "Cat"))
}

func definitionNames(doc *parser.OAS2Document) []string {
	names := make([]string, 0, len(doc.Definitions))
	for name := range doc.Definitions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func TestRenameScopeRegisterRight(t *testing.T) {
	s := newRenameScope(2, parser.OASVersion20)

	s.registerRight(1, "Pet", "Api_Pet")
	assert.Equal(t, map[string]string{"Pet": "Api_Pet"}, s.byDoc[1])

	// A second rename of the same schema (a prefix, then a collision) replaces the
	// mapping rather than stranding references at the intermediate name.
	s.registerRight(1, "Pet", "Api_Pet.clinic")
	assert.Equal(t, map[string]string{"Pet": "Api_Pet.clinic"}, s.byDoc[1])

	// Documents that did not contribute the schema are untouched.
	assert.Empty(t, s.byDoc[0])

	// Out-of-range indices are ignored rather than panicking.
	s.registerRight(-1, "Pet", "Pet_x")
	s.registerRight(9, "Pet", "Pet_x")
}

func TestRenameScopeRegisterLeft(t *testing.T) {
	t.Run("applies to earlier documents only", func(t *testing.T) {
		s := newRenameScope(3, parser.OASVersion20)

		s.registerLeft(2, "Pet", "Pet.store")

		assert.Equal(t, map[string]string{"Pet": "Pet.store"}, s.byDoc[0])
		assert.Equal(t, map[string]string{"Pet": "Pet.store"}, s.byDoc[1])
		assert.Empty(t, s.byDoc[2])
	})

	t.Run("redirects an existing mapping instead of adding a second", func(t *testing.T) {
		s := newRenameScope(2, parser.OASVersion20)
		s.registerRight(0, "Pet", "Api_Pet")

		s.registerLeft(1, "Api_Pet", "Api_Pet.store")

		// Pet follows the schema. The Api_Pet entry is inert for this document,
		// which spells Pet, but a document referencing Api_Pet would need it.
		assert.Equal(t, map[string]string{
			"Pet":     "Api_Pet.store",
			"Api_Pet": "Api_Pet.store",
		}, s.byDoc[0])
	})

	t.Run("leaves a document whose name already resolves elsewhere", func(t *testing.T) {
		s := newRenameScope(3, parser.OASVersion20)
		s.registerRight(1, "Pet", "Pet.clinic")

		// Document 1's Pet is already Pet.clinic, so moving the schema currently
		// under "Pet" is not about document 1.
		s.registerLeft(2, "Pet", "Pet.store")

		assert.Equal(t, map[string]string{"Pet": "Pet.store"}, s.byDoc[0])
		assert.Equal(t, map[string]string{"Pet": "Pet.clinic"}, s.byDoc[1])
	})
}

func TestRenameScopeApplyWithoutRenames(t *testing.T) {
	var nilScope *renameScope
	assert.NoError(t, nilScope.applyOAS2(&parser.OAS2Document{}, nil))
	assert.NoError(t, nilScope.applyOAS3(&parser.OAS3Document{}, nil))

	// A scope that recorded nothing skips the rewrite entirely.
	empty := newRenameScope(2, parser.OASVersion20)
	assert.True(t, empty.empty())
	assert.NoError(t, empty.applyOAS2(&parser.OAS2Document{}, nil))
}
