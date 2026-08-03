package walker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// callbackRefDocument holds a callbacks entry in each of its two forms, at each
// of the two positions a callbacks object can occupy.
func callbackRefDocument() *parser.ParseResult {
	inline := parser.Callback{
		"{$request.query.url}": {Post: &parser.Operation{}},
	}
	shared := parser.Callback{
		"http://example.com": {Post: &parser.Operation{}},
	}

	return &parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
			Paths: parser.Paths{
				"/things": {
					Post: &parser.Operation{
						Callbacks:    map[string]*parser.Callback{"inline": &inline},
						CallbackRefs: map[string]*parser.Reference{"referenced": {Ref: "#/components/callbacks/shared"}},
					},
				},
			},
			Components: &parser.Components{
				Callbacks:    map[string]*parser.Callback{"shared": &shared},
				CallbackRefs: map[string]*parser.Reference{"alias": {Ref: "#/components/callbacks/shared"}},
			},
		},
		OASVersion: parser.OASVersion303,
	}
}

// TestCallbackRefsReachTheRefHandler covers the callbacks written as Reference
// Objects, which carry a $ref and nothing to walk into. Anything collecting
// references ($ref rewriting, unused-component pruning) reads them from
// [RefHandler], so a callback missing from it is a reference nothing accounts for.
func TestCallbackRefsReachTheRefHandler(t *testing.T) {
	var seen []*RefInfo
	err := Walk(callbackRefDocument(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			if ref.NodeType == RefNodeCallback {
				seen = append(seen, ref)
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	paths := make([]string, 0, len(seen))
	for _, ref := range seen {
		assert.Equal(t, "#/components/callbacks/shared", ref.Ref)
		paths = append(paths, ref.SourcePath)
	}
	assert.ElementsMatch(t,
		[]string{"$.paths['/things'].post.callbacks['referenced']", "$.components.callbacks['alias']"},
		paths,
		"a reference-form callback was not reported, or was reported at the wrong path")
}

// TestCallbackHandlerIgnoresReferences keeps the existing handler's contract:
// it receives Callback Objects, and a Reference Object is not one.
func TestCallbackHandlerIgnoresReferences(t *testing.T) {
	var expressions []string
	err := Walk(callbackRefDocument(),
		WithCallbackHandler(func(wc *WalkContext, callback parser.Callback) Action {
			for expression := range callback {
				expressions = append(expressions, expression)
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{"{$request.query.url}", "http://example.com"}, expressions,
		"CallbackHandler saw something other than the two Callback Objects")
}

// TestCallbackRefStopsTraversal covers the Stop action on the new ref position,
// which is the contract every other handler call site honors.
func TestCallbackRefStopsTraversal(t *testing.T) {
	var count int
	err := Walk(callbackRefDocument(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			count++
			return Stop
		}),
	)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "traversal continued past a Stop from the ref handler")
}

// TestCallbackRefStopsComponentTraversal covers the same Stop from the other
// position a callback reference can occupy. A document walk reaches paths before
// components, so a test that stops at an operation's reference returns before
// this branch is ever taken.
func TestCallbackRefStopsComponentTraversal(t *testing.T) {
	result := &parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
			Components: &parser.Components{
				CallbackRefs: map[string]*parser.Reference{"alias": {Ref: "#/components/callbacks/shared"}},
				// Examples are walked after callbacks, so this reference is
				// reached only if the Stop was dropped.
				Examples: map[string]*parser.Example{"ex": {Ref: "#/components/examples/other"}},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	var seen []RefNodeType
	err := Walk(result,
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			seen = append(seen, ref.NodeType)
			return Stop
		}),
	)
	require.NoError(t, err)

	assert.Equal(t, []RefNodeType{RefNodeCallback}, seen,
		"traversal continued past a Stop at a component callback reference")
}

// TestCallbackRefStopSkipsOperationPostHandler pins the reason the operation and
// component walks treat the returned Action differently: an operation has a
// post-visit handler after its callbacks, and that handler must not run once the
// walk has been stopped.
func TestCallbackRefStopSkipsOperationPostHandler(t *testing.T) {
	var completed []string
	err := Walk(callbackRefDocument(),
		WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			return Stop
		}),
		WithOperationPostHandler(func(wc *WalkContext, op *parser.Operation) {
			completed = append(completed, wc.JSONPath)
		}),
	)
	require.NoError(t, err)

	assert.NotContains(t, completed, "$.paths['/things'].post",
		"the operation's post-visit handler ran after the walk was stopped")
	// The operation inside the Callback Object did finish, before the reference
	// was reached, so this is not asserting that nothing ran at all.
	assert.Contains(t, completed,
		"$.paths['/things'].post.callbacks['inline']['{$request.query.url}'].post")
}

// TestCallbackRefsSkipNilEntries covers the nil guard in the reference walk. A
// map assembled in Go can hold one, and a walker that dereferenced it would take
// the whole traversal down.
func TestCallbackRefsSkipNilEntries(t *testing.T) {
	result := &parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
			Components: &parser.Components{
				CallbackRefs: map[string]*parser.Reference{
					"nilEntry": nil,
					"real":     {Ref: "#/components/callbacks/shared"},
				},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	var seen []string
	assert.NotPanics(t, func() {
		_ = Walk(result, WithRefHandler(func(wc *WalkContext, ref *RefInfo) Action {
			seen = append(seen, ref.Ref)
			return Continue
		}))
	})
	assert.Equal(t, []string{"#/components/callbacks/shared"}, seen,
		"a nil reference entry should be skipped, not reported")
}
