package joiner

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mediaTypeComponentDoc builds an OAS 3.2 document with a components.mediaTypes
// entry that references a schema, which is the shape #485 dropped.
func mediaTypeComponentDoc(name string, extra bool) parser.ParseResult {
	target := &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{"id": {Type: "string"}}}
	if extra {
		target.Properties["note"] = &parser.Schema{Type: "string"}
	}
	ref := func() *parser.Schema { return &parser.Schema{Ref: "#/components/schemas/Target"} }

	return parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI: "3.2.0",
			Info:    &parser.Info{Title: name, Version: "1.0.0"},
			Paths:   parser.Paths{"/" + name: &parser.PathItem{Get: &parser.Operation{}}},
			Components: &parser.Components{
				Schemas: map[string]*parser.Schema{"Target": target},
				MediaTypes: map[string]*parser.MediaType{
					// Named after its source so both survive the join.
					name + "Media": {
						Schema:     ref(),
						ItemSchema: ref(),
						Encoding: map[string]*parser.Encoding{
							"part": {Headers: map[string]*parser.Header{"X-Meta": {Schema: ref()}}},
						},
					},
					// Named the same in both, so it collides. Example carries the
					// source name so a collision resolution is distinguishable.
					"Shared": {Schema: ref(), Example: name},
				},
			},
			OASVersion: parser.OASVersion320,
		},
		Version: "3.2.0", OASVersion: parser.OASVersion320,
		SourcePath: name, SourceFormat: parser.SourceFormatJSON,
	}
}

// TestMediaTypeComponentsAreMerged covers #485. mergeOAS3Components handled every
// Components field except MediaTypes, so both documents' entries were discarded
// with nothing reported.
func TestMediaTypeComponentsAreMerged(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(mediaTypeComponentDoc("a", false), mediaTypeComponentDoc("b", true)),
		WithSchemaStrategy(StrategyRenameRight),
		WithPathStrategy(StrategyAcceptLeft),
		WithComponentStrategy(StrategyAcceptLeft),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS3Document)
	require.NotNil(t, d.Components)

	// Both documents' own entries survive.
	assert.Contains(t, d.Components.MediaTypes, "aMedia")
	assert.Contains(t, d.Components.MediaTypes, "bMedia")

	// The colliding one is resolved by the component strategy rather than
	// vanishing, and accept-left keeps the left value.
	require.Contains(t, d.Components.MediaTypes, "Shared")
	assert.Equal(t, "a", d.Components.MediaTypes["Shared"].Example)
}

// TestMediaTypeCollisionIsReported checks that a media type collision reaches a
// handler under its own type, the way every other component's does.
func TestMediaTypeCollisionIsReported(t *testing.T) {
	var seen []CollisionContext
	_, err := JoinWithOptions(
		WithParsed(mediaTypeComponentDoc("a", false), mediaTypeComponentDoc("b", true)),
		WithSchemaStrategy(StrategyRenameRight),
		WithPathStrategy(StrategyAcceptLeft),
		WithComponentStrategy(StrategyAcceptLeft),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
		WithCollisionHandler(func(c CollisionContext) (CollisionResolution, error) {
			if c.Type == CollisionTypeMediaType {
				seen = append(seen, c)
			}
			return ContinueWithStrategy(), nil
		}),
	)
	require.NoError(t, err)

	require.Len(t, seen, 1, "the colliding media type should be reported once")
	assert.Equal(t, "Shared", seen[0].Name)
	assert.Equal(t, "$.components.mediaTypes.Shared", seen[0].JSONPath)
	left, ok := seen[0].LeftValue.(*parser.MediaType)
	require.True(t, ok, "LeftValue is %T", seen[0].LeftValue)
	assert.Equal(t, "a", left.Example)
	right, ok := seen[0].RightValue.(*parser.MediaType)
	require.True(t, ok, "RightValue is %T", seen[0].RightValue)
	assert.Equal(t, "b", right.Example)
}

// TestMediaTypeComponentsAreRewritten checks the other half: a merged media type
// carries schema references, so a rename has to reach them and must reach only
// the document that wrote them.
func TestMediaTypeComponentsAreRewritten(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(mediaTypeComponentDoc("a", false), mediaTypeComponentDoc("b", true)),
		WithSchemaStrategy(StrategyRenameRight),
		WithPathStrategy(StrategyAcceptLeft),
		WithComponentStrategy(StrategyAcceptLeft),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS3Document)
	require.Contains(t, d.Components.Schemas, "Target.b", "b's Target should have been renamed")

	refs := func(mt *parser.MediaType) map[string]string {
		require.NotNil(t, mt)
		return map[string]string{
			"schema":           mt.Schema.Ref,
			"itemSchema":       mt.ItemSchema.Ref,
			"encoding.headers": mt.Encoding["part"].Headers["X-Meta"].Schema.Ref,
		}
	}

	for field, ref := range refs(d.Components.MediaTypes["bMedia"]) {
		assert.Equal(t, "#/components/schemas/Target.b", ref, "b's %s was not rewritten", field)
	}
	for field, ref := range refs(d.Components.MediaTypes["aMedia"]) {
		assert.Equal(t, "#/components/schemas/Target", ref, "a's %s was repointed", field)
	}
}

// TestMediaTypeComponentsLeaveInputsUnchanged extends the #480 guarantee to the
// container: a merged media type is copied before its references change.
func TestMediaTypeComponentsLeaveInputsUnchanged(t *testing.T) {
	docs := []parser.ParseResult{mediaTypeComponentDoc("a", false), mediaTypeComponentDoc("b", true)}
	before := make([]any, len(docs))
	for i := range docs {
		before[i] = cloneDocument(t, docs[i].Document)
	}

	_, err := JoinWithOptions(
		WithParsed(docs...),
		WithSchemaStrategy(StrategyRenameRight),
		WithPathStrategy(StrategyAcceptLeft),
		WithComponentStrategy(StrategyAcceptLeft),
		WithRenameTemplate(`{{.Name}}.{{.Source}}`),
	)
	require.NoError(t, err)

	for i := range docs {
		assert.Equal(t, before[i], docs[i].Document,
			"joining modified input %d (%s)", i, docs[i].SourcePath)
	}
}
