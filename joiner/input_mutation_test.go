package joiner

import (
	"encoding/json"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// snapshot renders a parsed document so a difference reads as a diff.
func snapshot(t *testing.T, res parser.ParseResult) string {
	t.Helper()
	data, err := json.MarshalIndent(res.Document, "", "  ")
	require.NoError(t, err)
	return string(data)
}

// cloneDocument copies a document so it can be compared field by field. JSON
// alone would miss anything the marshaller drops, such as json:"-" fields.
func cloneDocument(t *testing.T, doc any) any {
	t.Helper()
	switch d := doc.(type) {
	case *parser.OAS2Document:
		return d.DeepCopy()
	case *parser.OAS3Document:
		return d.DeepCopy()
	default:
		t.Fatalf("unexpected document type %T", doc)
		return nil
	}
}

// discriminatedOAS3 builds a document whose Pet schema selects a variant through
// a discriminator, covering the mapping and defaultMapping rewrites as well as
// the plain $ref one.
func discriminatedOAS3(name string, extra bool) parser.ParseResult {
	cat := &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{"meow": {Type: "string"}}}
	if extra {
		cat.Properties["lives"] = &parser.Schema{Type: "integer"}
	}
	return parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI: "3.2.0",
			Info:    &parser.Info{Title: name, Version: "1.0.0"},
			Paths: parser.Paths{
				"/" + name: &parser.PathItem{Get: &parser.Operation{
					Responses: &parser.Responses{Codes: map[string]*parser.Response{
						"200": {Description: "ok", Content: map[string]*parser.MediaType{
							"application/json": {Schema: &parser.Schema{Ref: "#/components/schemas/Pet"}},
						}},
					}},
				}},
			},
			Components: &parser.Components{
				Schemas: map[string]*parser.Schema{
					"Pet": {
						Type:  "object",
						OneOf: []*parser.Schema{{Ref: "#/components/schemas/Cat"}},
						Discriminator: &parser.Discriminator{
							PropertyName:   "kind",
							Mapping:        map[string]string{"cat": "#/components/schemas/Cat"},
							DefaultMapping: "#/components/schemas/Cat",
						},
					},
					"Cat": cat,
				},
			},
			OASVersion: parser.OASVersion320,
		},
		Version: "3.2.0", OASVersion: parser.OASVersion320,
		SourcePath: name, SourceFormat: parser.SourceFormatJSON,
	}
}

// TestJoinLeavesInputsUnchanged covers #480. The joined document holds the very
// schemas and path items its inputs contributed, and the reference rewriting
// used to edit them in place, so joining rewrote the caller's documents.
func TestJoinLeavesInputsUnchanged(t *testing.T) {
	customHandler := func(c CollisionContext) (CollisionResolution, error) {
		if c.Type == CollisionTypeSchema && c.Name == "Pet" {
			return UseCustomValue(&parser.Schema{
				Type:       "object",
				Properties: map[string]*parser.Schema{"category": {Ref: "#/definitions/Category"}},
			}), nil
		}
		return ContinueWithStrategy(), nil
	}

	// sharingHandler returns a value built from the left document's own property
	// schemas, so the custom value shares pointers with an input. sharingRan
	// guards against the name drifting and the handler silently never firing,
	// which would leave the case passing for the wrong reason.
	sharingRan := false
	sharingHandler := func(c CollisionContext) (CollisionResolution, error) {
		if c.Type == CollisionTypeSchema && c.Name == "Api_Pet" {
			left, ok := c.LeftValue.(*parser.Schema)
			if !ok {
				return ContinueWithStrategy(), nil
			}
			sharingRan = true
			merged := &parser.Schema{Type: left.Type, Properties: map[string]*parser.Schema{}}
			for name, prop := range left.Properties {
				merged.Properties[name] = prop
			}
			return UseCustomValue(merged), nil
		}
		return ContinueWithStrategy(), nil
	}

	tests := []struct {
		name string
		docs func() []parser.ParseResult
		opts []Option
		// verify runs after the join, for a case that needs more than the
		// inputs being unchanged.
		verify func(t *testing.T)
	}{
		{
			name: "oas2 rename-right",
			docs: func() []parser.ParseResult {
				return []parser.ParseResult{petstoreFamily("store", false), petstoreFamily("clinic", true)}
			},
			opts: []Option{WithSchemaStrategy(StrategyRenameRight), WithRenameTemplate(`{{.Name}}.{{.Source}}`)},
		},
		{
			name: "oas2 rename-left",
			docs: func() []parser.ParseResult {
				return []parser.ParseResult{petstoreFamily("store", false), petstoreFamily("clinic", true)}
			},
			opts: []Option{WithSchemaStrategy(StrategyRenameLeft), WithRenameTemplate(`{{.Name}}.{{.Source}}`)},
		},
		{
			name: "oas2 namespace prefix",
			docs: func() []parser.ParseResult {
				return []parser.ParseResult{petstoreFamily("store", false), petstoreFamily("clinic", true)}
			},
			opts: []Option{
				WithSchemaStrategy(StrategyRenameRight),
				WithNamespacePrefix("store", "Api"), WithNamespacePrefix("clinic", "Api"),
				WithAlwaysApplyPrefix(true), WithRenameTemplate(`{{.Name}}.{{.Source}}`),
			},
		},
		{
			name: "oas2 semantic deduplication",
			docs: func() []parser.ParseResult {
				return []parser.ParseResult{petstoreFamily("store", false), petstoreFamily("clinic", false)}
			},
			opts: []Option{
				WithSchemaStrategy(StrategyRenameRight), WithSemanticDeduplication(true),
				WithEquivalenceMode("deep"), WithRenameTemplate(`{{.Name}}.{{.Source}}`),
			},
		},
		{
			name: "oas2 handler custom value",
			docs: func() []parser.ParseResult {
				return []parser.ParseResult{petstoreFamily("store", false), petstoreFamily("clinic", true)}
			},
			opts: []Option{
				WithSchemaStrategy(StrategyRenameRight), WithRenameTemplate(`{{.Name}}.{{.Source}}`),
				WithCollisionHandler(customHandler),
			},
		},
		{
			name: "oas2 handler custom value sharing input schemas",
			docs: func() []parser.ParseResult {
				return []parser.ParseResult{petstoreFamily("store", false), petstoreFamily("clinic", true)}
			},
			opts: []Option{
				WithSchemaStrategy(StrategyRenameRight),
				WithNamespacePrefix("store", "Api"), WithNamespacePrefix("clinic", "Api"),
				WithAlwaysApplyPrefix(true), WithRenameTemplate(`{{.Name}}.{{.Source}}`),
				WithCollisionHandler(sharingHandler),
			},
			verify: func(t *testing.T) {
				assert.True(t, sharingRan, "the handler never fired, so nothing shared an input")
			},
		},
		{
			name: "oas2 accept-left renames nothing",
			docs: func() []parser.ParseResult {
				return []parser.ParseResult{petstoreFamily("store", false), petstoreFamily("clinic", true)}
			},
			opts: []Option{WithSchemaStrategy(StrategyAcceptLeft)},
		},
		{
			name: "oas3 rename-left",
			docs: func() []parser.ParseResult {
				return []parser.ParseResult{petstoreFamilyOAS3("store", false), petstoreFamilyOAS3("clinic", true)}
			},
			opts: []Option{WithSchemaStrategy(StrategyRenameLeft), WithRenameTemplate(`{{.Name}}.{{.Source}}`)},
		},
		{
			name: "oas3 namespace prefix",
			docs: func() []parser.ParseResult {
				return []parser.ParseResult{petstoreFamilyOAS3("store", false), petstoreFamilyOAS3("clinic", true)}
			},
			opts: []Option{
				WithSchemaStrategy(StrategyRenameRight),
				WithNamespacePrefix("store", "Api"), WithNamespacePrefix("clinic", "Api"),
				WithAlwaysApplyPrefix(true), WithRenameTemplate(`{{.Name}}.{{.Source}}`),
			},
		},
		{
			name: "oas3 semantic deduplication",
			docs: func() []parser.ParseResult {
				return []parser.ParseResult{petstoreFamilyOAS3("store", false), petstoreFamilyOAS3("clinic", false)}
			},
			opts: []Option{
				WithSchemaStrategy(StrategyRenameRight), WithSemanticDeduplication(true),
				WithEquivalenceMode("deep"), WithRenameTemplate(`{{.Name}}.{{.Source}}`),
			},
		},
		{
			name: "oas3 every container",
			docs: func() []parser.ParseResult {
				return []parser.ParseResult{everyContainerOAS3("a", false), everyContainerOAS3("b", true)}
			},
			opts: []Option{WithSchemaStrategy(StrategyRenameRight), WithRenameTemplate(`{{.Name}}.{{.Source}}`)},
		},
		{
			name: "oas3 rename-right",
			docs: func() []parser.ParseResult {
				return []parser.ParseResult{petstoreFamilyOAS3("store", false), petstoreFamilyOAS3("clinic", true)}
			},
			opts: []Option{WithSchemaStrategy(StrategyRenameRight), WithRenameTemplate(`{{.Name}}.{{.Source}}`)},
		},
		{
			name: "oas3 discriminator mapping",
			docs: func() []parser.ParseResult {
				return []parser.ParseResult{discriminatedOAS3("a", false), discriminatedOAS3("b", true)}
			},
			opts: []Option{WithSchemaStrategy(StrategyRenameRight), WithRenameTemplate(`{{.Name}}.{{.Source}}`)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docs := tt.docs()
			beforeJSON := make([]string, len(docs))
			beforeDoc := make([]any, len(docs))
			for i := range docs {
				beforeJSON[i] = snapshot(t, docs[i])
				beforeDoc[i] = cloneDocument(t, docs[i].Document)
			}

			opts := append([]Option{WithParsed(docs...), WithPathStrategy(StrategyAcceptLeft)}, tt.opts...)
			_, err := JoinWithOptions(opts...)
			require.NoError(t, err)

			for i := range docs {
				assert.Equal(t, beforeJSON[i], snapshot(t, docs[i]),
					"joining modified input %d (%s)", i, docs[i].SourcePath)
				assert.Equal(t, beforeDoc[i], docs[i].Document,
					"joining modified input %d (%s) outside its JSON form", i, docs[i].SourcePath)
			}
			if tt.verify != nil {
				tt.verify(t)
			}
		})
	}
}

// TestJoinDiscriminatorRewriteStillApplies guards the other half: the copy is
// only worth anything if the joined document still gets the rewrite.
func TestJoinDiscriminatorRewriteStillApplies(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(discriminatedOAS3("a", false), discriminatedOAS3("b", true)),
		WithSchemaStrategy(StrategyRenameRight),
		WithPathStrategy(StrategyAcceptLeft),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS3Document)
	renamed := d.Components.Schemas["Pet.b"]
	require.NotNil(t, renamed)
	require.NotNil(t, renamed.Discriminator)

	assert.Equal(t, "#/components/schemas/Cat.b", renamed.OneOf[0].Ref)
	assert.Equal(t, "#/components/schemas/Cat.b", renamed.Discriminator.Mapping["cat"])
	assert.Equal(t, "#/components/schemas/Cat.b", renamed.Discriminator.DefaultMapping)

	// a's schema was written against the Cat that kept its name.
	original := d.Components.Schemas["Pet"]
	assert.Equal(t, "#/components/schemas/Cat", original.OneOf[0].Ref)
	assert.Equal(t, "#/components/schemas/Cat", original.Discriminator.Mapping["cat"])
	assert.Equal(t, "#/components/schemas/Cat", original.Discriminator.DefaultMapping)
}

// everyContainerOAS3 populates the OAS 3 containers rewriteEntries copies that
// the other fixtures leave empty: webhooks, components.pathItems,
// components.callbacks and components.requestBodies, plus the media type fields
// that reach a schema without going through MediaType.Schema.
func everyContainerOAS3(name string, extra bool) parser.ParseResult {
	target := &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{"id": {Type: "string"}}}
	if extra {
		target.Properties["note"] = &parser.Schema{Type: "string"}
	}
	ref := func() *parser.Schema { return &parser.Schema{Ref: "#/components/schemas/Target"} }
	header := func(name string) map[string]*parser.Header {
		return map[string]*parser.Header{name: {Schema: ref()}}
	}
	content := func() map[string]*parser.MediaType {
		return map[string]*parser.MediaType{
			"application/jsonl": {
				Schema:     ref(),
				ItemSchema: ref(),
				Encoding: map[string]*parser.Encoding{
					"part": {
						Headers:        header("X-Meta"),
						Encoding:       map[string]*parser.Encoding{"nested": {Headers: header("X-Nested")}},
						ItemEncoding:   &parser.Encoding{Headers: header("X-PartItem")},
						PrefixEncoding: []*parser.Encoding{{Headers: header("X-PartPrefix")}},
					},
				},
				ItemEncoding:   &parser.Encoding{Headers: header("X-Item")},
				PrefixEncoding: []*parser.Encoding{{Headers: header("X-Prefix")}},
			},
		}
	}
	op := func() *parser.Operation {
		return &parser.Operation{Responses: &parser.Responses{Codes: map[string]*parser.Response{
			"200": {Description: "ok", Content: content()},
		}}}
	}
	callback := parser.Callback{"{$request.body#/url}": &parser.PathItem{Post: op()}}

	return parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI:  "3.2.0",
			Info:     &parser.Info{Title: name, Version: "1.0.0"},
			Paths:    parser.Paths{"/" + name: &parser.PathItem{Get: op()}},
			Webhooks: map[string]*parser.PathItem{name + "Hook": {Post: op()}},
			Components: &parser.Components{
				Schemas:       map[string]*parser.Schema{"Target": target},
				RequestBodies: map[string]*parser.RequestBody{name + "Body": {Content: content()}},
				Callbacks:     map[string]*parser.Callback{name + "CB": &callback},
				PathItems:     map[string]*parser.PathItem{name + "PI": {Get: op()}},
				MediaTypes:    map[string]*parser.MediaType{name + "MT": content()["application/jsonl"]},
			},
			OASVersion: parser.OASVersion320,
		},
		Version: "3.2.0", OASVersion: parser.OASVersion320,
		SourcePath: name, SourceFormat: parser.SourceFormatJSON,
	}
}

// TestRewriteMediaTypeReachesEveryRef covers the reference locations inside a
// media type that are not MediaType.Schema: itemSchema (OAS 3.2+) and the
// headers an encoding describes. A rename used to leave both pointing at the
// name the other document now owns.
func TestRewriteMediaTypeReachesEveryRef(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(everyContainerOAS3("a", false), everyContainerOAS3("b", true)),
		WithSchemaStrategy(StrategyRenameRight),
		WithPathStrategy(StrategyAcceptLeft),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS3Document)
	require.Contains(t, d.Components.Schemas, "Target.b")

	// Every way a media type reaches a schema without going through Schema.
	refs := func(mt *parser.MediaType) map[string]string {
		require.NotNil(t, mt)
		part := mt.Encoding["part"]
		require.NotNil(t, part)
		return map[string]string{
			"schema":                          mt.Schema.Ref,
			"itemSchema":                      mt.ItemSchema.Ref,
			"encoding.headers":                part.Headers["X-Meta"].Schema.Ref,
			"encoding.encoding.headers":       part.Encoding["nested"].Headers["X-Nested"].Schema.Ref,
			"encoding.itemEncoding.headers":   part.ItemEncoding.Headers["X-PartItem"].Schema.Ref,
			"encoding.prefixEncoding.headers": part.PrefixEncoding[0].Headers["X-PartPrefix"].Schema.Ref,
			"itemEncoding.headers":            mt.ItemEncoding.Headers["X-Item"].Schema.Ref,
			"prefixEncoding.headers":          mt.PrefixEncoding[0].Headers["X-Prefix"].Schema.Ref,
		}
	}
	jsonl := func(op *parser.Operation) *parser.MediaType {
		return op.Responses.Codes["200"].Content["application/jsonl"]
	}

	for where, mt := range map[string]*parser.MediaType{
		"paths":                    jsonl(d.Paths["/b"].Get),
		"webhooks":                 jsonl(d.Webhooks["bHook"].Post),
		"components.pathItems":     jsonl(d.Components.PathItems["bPI"].Get),
		"components.callbacks":     jsonl((*d.Components.Callbacks["bCB"])["{$request.body#/url}"].Post),
		"components.requestBodies": d.Components.RequestBodies["bBody"].Content["application/jsonl"],
		"components.mediaTypes":    d.Components.MediaTypes["bMT"],
	} {
		for field, ref := range refs(mt) {
			assert.Equal(t, "#/components/schemas/Target.b", ref, "stale reference in %s %s", where, field)
		}
	}

	// a's references still name the schema that kept the original name.
	for where, mt := range map[string]*parser.MediaType{
		"paths":                 jsonl(d.Paths["/a"].Get),
		"components.mediaTypes": d.Components.MediaTypes["aMT"],
	} {
		for field, ref := range refs(mt) {
			assert.Equal(t, "#/components/schemas/Target", ref, "a's %s %s was repointed", where, field)
		}
	}
}
