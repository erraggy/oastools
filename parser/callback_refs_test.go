package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// callbackRefSpecs is the same document in both source formats. parser keeps
// separate YAML and JSON decode paths, so anything asserted about decoding has
// to be asserted twice or it covers half the surface.
//
// Both positions a `callbacks` object can occupy carry both forms an entry can
// take, and one reference has a sibling field so the classification is shown to
// depend on the `$ref` key rather than on the entry being a lone `$ref`.
var callbackRefSpecs = map[string]string{
	"yaml": `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
paths:
  /things:
    post:
      responses:
        '200':
          description: ok
      callbacks:
        inline:
          '{$request.query.url}':
            post:
              responses:
                '200':
                  description: ok
        referenced:
          $ref: '#/components/callbacks/shared'
          summary: the shared callback
components:
  callbacks:
    shared:
      'http://example.com?id={$request.body#/id}':
        post:
          responses:
            '200':
              description: ok
    alias:
      $ref: '#/components/callbacks/shared'
`,
	"json": `{
  "openapi": "3.2.0",
  "info": {"title": "API", "version": "1.0.0"},
  "paths": {
    "/things": {
      "post": {
        "responses": {"200": {"description": "ok"}},
        "callbacks": {
          "inline": {
            "{$request.query.url}": {
              "post": {"responses": {"200": {"description": "ok"}}}
            }
          },
          "referenced": {
            "$ref": "#/components/callbacks/shared",
            "summary": "the shared callback"
          }
        }
      }
    }
  },
  "components": {
    "callbacks": {
      "shared": {
        "http://example.com?id={$request.body#/id}": {
          "post": {"responses": {"200": {"description": "ok"}}}
        }
      },
      "alias": {"$ref": "#/components/callbacks/shared"}
    }
  }
}`,
}

// assertCallbackRefSplit checks the split every decode path has to produce: the
// Callback Object entries in Callbacks, the Reference Object entries in
// CallbackRefs, and no name in both.
func assertCallbackRefSplit(t *testing.T, doc *OAS3Document) {
	t.Helper()

	require.NotNil(t, doc.Paths["/things"], "path was dropped")
	op := doc.Paths["/things"].Post
	require.NotNil(t, op, "operation was dropped")

	require.Contains(t, op.Callbacks, "inline")
	assert.NotContains(t, op.Callbacks, "referenced",
		"a Reference Object was decoded as a Callback Object")
	require.Contains(t, op.CallbackRefs, "referenced")
	assert.Equal(t, "#/components/callbacks/shared", op.CallbackRefs["referenced"].Ref)
	assert.Equal(t, "the shared callback", op.CallbackRefs["referenced"].Summary,
		"a sibling of $ref was dropped")

	require.NotNil(t, doc.Components)
	require.Contains(t, doc.Components.Callbacks, "shared")
	assert.NotContains(t, doc.Components.Callbacks, "alias",
		"a Reference Object was decoded as a Callback Object")
	require.Contains(t, doc.Components.CallbackRefs, "alias")
	assert.Equal(t, "#/components/callbacks/shared", doc.Components.CallbackRefs["alias"].Ref)
}

// TestCallbackRefsDecode covers the Reference Object form of a callbacks entry,
// which the specification tells apart from the Callback Object form by the
// presence of a `$ref` key.
func TestCallbackRefsDecode(t *testing.T) {
	for _, format := range []string{"yaml", "json"} {
		t.Run(format, func(t *testing.T) {
			result, err := New().ParseBytes([]byte(callbackRefSpecs[format]))
			require.NoError(t, err)
			doc, ok := result.OAS3Document()
			require.True(t, ok, "expected an OAS3 document")

			assertCallbackRefSplit(t, doc)
		})
	}
}

// TestCallbackRefsUnderResolveRefs covers the third decode path, which sees a
// document the resolver has already rewritten.
//
// A reference it could resolve is gone by then, replaced by what it pointed at,
// which is what ResolveRefs is for and what every other reference form does. One
// it could not resolve arrives at decodeFromMap intact, and that is the case
// worth pinning: the map path drops a value whose shape it does not expect, with
// no error to say so.
func TestCallbackRefsUnderResolveRefs(t *testing.T) {
	t.Run("a resolvable reference is inlined", func(t *testing.T) {
		for _, format := range []string{"yaml", "json"} {
			t.Run(format, func(t *testing.T) {
				p := New()
				p.ResolveRefs = true
				result, err := p.ParseBytes([]byte(callbackRefSpecs[format]))
				require.NoError(t, err)
				doc, ok := result.OAS3Document()
				require.True(t, ok)

				op := doc.Paths["/things"].Post
				require.NotNil(t, op)
				assert.Empty(t, op.CallbackRefs, "a resolved reference should not remain a reference")

				inlined := op.Callbacks["referenced"]
				require.NotNil(t, inlined, "the resolved callback was dropped")
				assert.Contains(t, *inlined, "http://example.com?id={$request.body#/id}",
					"the reference was not replaced by what it pointed at")

				// The other carrier: components.callbacks.alias is a reference to
				// a sibling component, and resolves the same way.
				require.NotNil(t, doc.Components)
				assert.Empty(t, doc.Components.CallbackRefs,
					"a resolved component reference should not remain a reference")
				alias := doc.Components.Callbacks["alias"]
				require.NotNil(t, alias, "the resolved component callback was dropped")
				assert.Contains(t, *alias, "http://example.com?id={$request.body#/id}",
					"the component reference was not replaced by what it pointed at")
			})
		}
	})

	t.Run("an unresolved reference reaches CallbackRefs", func(t *testing.T) {
		for _, format := range []string{"yaml", "json"} {
			t.Run(format, func(t *testing.T) {
				p := New()
				p.ResolveRefs = true
				result, err := p.ParseBytes([]byte(danglingCallbackRefSpecs[format]))
				require.NoError(t, err)
				doc, ok := result.OAS3Document()
				require.True(t, ok)

				op := doc.Paths["/things"].Post
				require.NotNil(t, op)
				require.Contains(t, op.CallbackRefs, "referenced",
					"the map decode path dropped a reference-form callback")
				assert.Equal(t, "#/components/callbacks/missing", op.CallbackRefs["referenced"].Ref)
			})
		}
	})
}

// TestCallbackRefsRoundTrip checks that the two Go fields serialize back into the
// one `callbacks` object the specification defines, with the reference verbatim.
func TestCallbackRefsRoundTrip(t *testing.T) {
	for _, format := range []string{"yaml", "json"} {
		t.Run(format, func(t *testing.T) {
			result, err := New().ParseBytes([]byte(callbackRefSpecs[format]))
			require.NoError(t, err)
			doc, ok := result.OAS3Document()
			require.True(t, ok, "expected an OAS3 document")

			var encoded []byte
			if format == "yaml" {
				encoded, err = yaml.Marshal(doc)
			} else {
				encoded, err = json.Marshal(doc)
			}
			require.NoError(t, err)

			// Read the output back generically: the reference has to be a member
			// of `callbacks` beside the Callback Object, not a field of its own.
			var raw map[string]any
			if format == "yaml" {
				require.NoError(t, yaml.Unmarshal(encoded, &raw))
			} else {
				require.NoError(t, json.Unmarshal(encoded, &raw))
			}
			opCallbacks := objectAt(t, raw, "paths", "/things", "post", "callbacks")
			assert.Contains(t, opCallbacks, "inline")
			referenced := objectAt(t, opCallbacks, "referenced")
			assert.Equal(t, "#/components/callbacks/shared", referenced["$ref"])
			assert.Equal(t, "the shared callback", referenced["summary"])

			componentCallbacks := objectAt(t, raw, "components", "callbacks")
			assert.Contains(t, componentCallbacks, "shared")
			assert.Equal(t, "#/components/callbacks/shared",
				objectAt(t, componentCallbacks, "alias")["$ref"])

			// Reparsing has to produce the same document, which is the property a
			// consumer round-tripping a specification depends on.
			reparsed, err := New().ParseBytes(encoded)
			require.NoError(t, err)
			again, ok := reparsed.OAS3Document()
			require.True(t, ok)
			assertCallbackRefSplit(t, again)
			assert.True(t, doc.Equals(again), "document changed across a round trip")
		})
	}
}

// TestCallbackRefsRoundTripAlongsideExtensions covers the hand-built marshal path,
// which a specification extension anywhere in the object switches on. It builds
// the `callbacks` member from a separate list of fields, so the merge has to be
// present there too.
func TestCallbackRefsRoundTripAlongsideExtensions(t *testing.T) {
	op := &Operation{
		Responses:    &Responses{Codes: map[string]*Response{"200": {Description: "ok"}}},
		CallbackRefs: map[string]*Reference{"referenced": {Ref: "#/components/callbacks/shared"}},
		Extra:        map[string]any{"x-vendor": "value"},
	}

	encoded, err := json.Marshal(op)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(encoded, &raw))
	assert.Equal(t, "value", raw["x-vendor"])
	assert.Equal(t, "#/components/callbacks/shared",
		objectAt(t, raw, "callbacks", "referenced")["$ref"])
}

// objectAt walks a decoded document to the object at a path of keys.
//
// A chain of `.(map[string]any)` assertions would do the same, but it panics
// part way down when the shape is not what the test expected, and the stack
// trace does not say which key was missing. This reports the level that failed.
func objectAt(t *testing.T, node map[string]any, keys ...string) map[string]any {
	t.Helper()

	for i, key := range keys {
		value, ok := node[key]
		require.True(t, ok, "no %q under %v", key, keys[:i])
		node, ok = value.(map[string]any)
		require.True(t, ok, "%v is not an object", keys[:i+1])
	}
	return node
}

// danglingCallbackRefSpecs holds a single reference-form callback whose target
// does not exist, so the resolver leaves it alone and it reaches the decode paths
// as written.
var danglingCallbackRefSpecs = map[string]string{
	"yaml": `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
paths:
  /things:
    post:
      responses:
        '200':
          description: ok
      callbacks:
        referenced:
          $ref: '#/components/callbacks/missing'
components:
  callbacks:
    alias:
      $ref: '#/components/callbacks/missing'
`,
	"json": `{
  "openapi": "3.2.0",
  "info": {"title": "API", "version": "1.0.0"},
  "paths": {
    "/things": {
      "post": {
        "responses": {"200": {"description": "ok"}},
        "callbacks": {"referenced": {"$ref": "#/components/callbacks/missing"}}
      }
    }
  },
  "components": {"callbacks": {"alias": {"$ref": "#/components/callbacks/missing"}}}
}`,
}

// TestCallbackObjectPresenceSurvivesDecoding pins the rule the three paths agree
// on: Callbacks is non-nil exactly when the document carried a `callbacks` key,
// whether or not every entry in it turned out to be a reference. A caller cannot
// otherwise tell an absent `callbacks` from one holding only references.
func TestCallbackObjectPresenceSurvivesDecoding(t *testing.T) {
	cases := map[string]struct {
		specs    map[string]string
		wantRefs int
	}{
		// An empty callbacks object holds no entry of either form. It is still a
		// key the document carried, and it decoded to a non-nil empty map before
		// the reference form existed.
		"an empty callbacks object": {specs: emptyCallbacksSpecs, wantRefs: 0},
		// Every entry is a reference, so nothing populates Callbacks. Without the
		// allocation a caller could not tell this from an absent callbacks key.
		"only references": {specs: danglingCallbackRefSpecs, wantRefs: 1},
	}

	for name, tc := range cases {
		for _, format := range []string{"yaml", "json"} {
			for _, resolveRefs := range []bool{false, true} {
				subtest := name + "/" + format
				if resolveRefs {
					subtest += "/resolveRefs"
				}
				t.Run(subtest, func(t *testing.T) {
					p := New()
					p.ResolveRefs = resolveRefs
					result, err := p.ParseBytes([]byte(tc.specs[format]))
					require.NoError(t, err)
					doc, ok := result.OAS3Document()
					require.True(t, ok)

					op := doc.Paths["/things"].Post
					require.NotNil(t, op)
					assert.NotNil(t, op.Callbacks,
						"the document carried a callbacks key, so Callbacks must not be nil")
					assert.Empty(t, op.Callbacks, "no entry was a Callback Object")
					assert.Len(t, op.CallbackRefs, tc.wantRefs)

					// The same rule on the other carrier, which has its own decode
					// path entries and so proves nothing by the operation passing.
					require.NotNil(t, doc.Components)
					assert.NotNil(t, doc.Components.Callbacks,
						"components carried a callbacks key, so Callbacks must not be nil")
					assert.Empty(t, doc.Components.Callbacks, "no entry was a Callback Object")
					assert.Len(t, doc.Components.CallbackRefs, tc.wantRefs)
				})
			}
		}
	}
}

// emptyCallbacksSpecs carries a callbacks object with no entries at all.
var emptyCallbacksSpecs = map[string]string{
	"yaml": `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
paths:
  /things:
    post:
      responses:
        '200':
          description: ok
      callbacks: {}
components:
  callbacks: {}
`,
	"json": `{
  "openapi": "3.2.0",
  "info": {"title": "API", "version": "1.0.0"},
  "paths": {
    "/things": {
      "post": {
        "responses": {"200": {"description": "ok"}},
        "callbacks": {}
      }
    }
  },
  "components": {"callbacks": {}}
}`,
}

// TestCallbackNameInBothFormsIsRejected covers a value no decode path can
// produce but Go code can assemble. Serializing it would have to pick one of the
// two forms, which is a choice no caller asked for.
func TestCallbackNameInBothFormsIsRejected(t *testing.T) {
	callback := Callback{"{$request.query.url}": {}}

	subjects := map[string]any{
		"Operation": &Operation{
			Responses:    &Responses{Codes: map[string]*Response{"200": {Description: "ok"}}},
			Callbacks:    map[string]*Callback{"clash": &callback},
			CallbackRefs: map[string]*Reference{"clash": {Ref: "#/components/callbacks/shared"}},
		},
		"Components": &Components{
			Callbacks:    map[string]*Callback{"clash": &callback},
			CallbackRefs: map[string]*Reference{"clash": {Ref: "#/components/callbacks/shared"}},
		},
	}

	for name, subject := range subjects {
		t.Run(name, func(t *testing.T) {
			_, err := json.Marshal(subject)
			assert.ErrorContains(t, err, `callback "clash" is present as both`)

			_, err = yaml.Marshal(subject)
			assert.ErrorContains(t, err, `callback "clash" is present as both`)
		})
	}
}

// TestCallbackRefsDeepCopyIsIndependent covers the generated deep copy, whose
// field list is maintained by hand and so can omit a new field silently.
func TestCallbackRefsDeepCopyIsIndependent(t *testing.T) {
	// Both carriers, because the generator's field list has a separate entry per
	// type: covering one proves nothing about the other.
	op := &Operation{CallbackRefs: map[string]*Reference{"referenced": {Ref: "#/components/callbacks/shared"}}}
	components := &Components{CallbackRefs: map[string]*Reference{"alias": {Ref: "#/components/callbacks/shared"}}}

	opCopy := op.DeepCopy()
	componentsCopy := components.DeepCopy()

	require.Contains(t, opCopy.CallbackRefs, "referenced", "CallbackRefs was dropped by Operation.DeepCopy")
	require.Contains(t, componentsCopy.CallbackRefs, "alias", "CallbackRefs was dropped by Components.DeepCopy")

	opCopy.CallbackRefs["referenced"].Ref = "#/components/callbacks/other"
	componentsCopy.CallbackRefs["alias"].Ref = "#/components/callbacks/other"

	assert.Equal(t, "#/components/callbacks/shared", op.CallbackRefs["referenced"].Ref,
		"the copy aliases the original's Reference")
	assert.Equal(t, "#/components/callbacks/shared", components.CallbackRefs["alias"].Ref,
		"the copy aliases the original's Reference")
}

// TestCallbackRefsAffectEquality keeps equality from reporting two documents the
// same when they reference different callbacks.
func TestCallbackRefsAffectEquality(t *testing.T) {
	t.Run("Operation", func(t *testing.T) {
		left := &Operation{CallbackRefs: map[string]*Reference{"a": {Ref: "#/components/callbacks/one"}}}
		right := &Operation{CallbackRefs: map[string]*Reference{"a": {Ref: "#/components/callbacks/two"}}}
		assert.False(t, equalOperation(left, right))
		assert.False(t, equalOperation(left, &Operation{}))
		assert.True(t, equalOperation(left, left.DeepCopy()))
	})

	t.Run("Components", func(t *testing.T) {
		left := &Components{CallbackRefs: map[string]*Reference{"a": {Ref: "#/components/callbacks/one"}}}
		right := &Components{CallbackRefs: map[string]*Reference{"a": {Ref: "#/components/callbacks/two"}}}
		assert.False(t, equalComponents(left, right))
		assert.False(t, equalComponents(left, &Components{}))
		assert.True(t, equalComponents(left, left.DeepCopy()))
	})
}
