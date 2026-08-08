package joiner

import (
	"encoding/json"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// snapshot renders a parsed document so it can be compared before and after a join.
func snapshot(t *testing.T, res parser.ParseResult) string {
	t.Helper()
	data, err := json.MarshalIndent(res.Document, "", "  ")
	require.NoError(t, err)
	return string(data)
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

	tests := []struct {
		name string
		docs func() []parser.ParseResult
		opts []Option
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
			name: "oas2 accept-left renames nothing",
			docs: func() []parser.ParseResult {
				return []parser.ParseResult{petstoreFamily("store", false), petstoreFamily("clinic", true)}
			},
			opts: []Option{WithSchemaStrategy(StrategyAcceptLeft)},
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
			before := make([]string, len(docs))
			for i := range docs {
				before[i] = snapshot(t, docs[i])
			}

			opts := append([]Option{WithParsed(docs...), WithPathStrategy(StrategyAcceptLeft)}, tt.opts...)
			_, err := JoinWithOptions(opts...)
			require.NoError(t, err)

			for i := range docs {
				assert.Equal(t, before[i], snapshot(t, docs[i]),
					"joining modified input %d (%s)", i, docs[i].SourcePath)
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
