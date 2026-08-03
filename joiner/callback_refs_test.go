package joiner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// A callbacks entry is either a Callback Object or a Reference Object, carried on
// two parser fields that share one name space (see parser.Callback). Merging a
// name held as one form into a document holding it as the other would put the
// key in both maps, and that document cannot be written at all.
func TestJoinRejectsCallbackFormCollision(t *testing.T) {
	callback := parser.Callback{"{$request.query.url}": {Post: &parser.Operation{}}}

	tests := map[string]struct {
		target, source *parser.Components
	}{
		"object in the target, reference in the source": {
			target: &parser.Components{Callbacks: map[string]*parser.Callback{"clash": &callback}},
			source: &parser.Components{CallbackRefs: map[string]*parser.Reference{"clash": {Ref: "#/components/callbacks/other"}}},
		},
		"reference in the target, object in the source": {
			target: &parser.Components{CallbackRefs: map[string]*parser.Reference{"clash": {Ref: "#/components/callbacks/other"}}},
			source: &parser.Components{Callbacks: map[string]*parser.Callback{"clash": &callback}},
		},
		"both forms inside the incoming document": {
			target: &parser.Components{},
			source: &parser.Components{
				Callbacks:    map[string]*parser.Callback{"clash": &callback},
				CallbackRefs: map[string]*parser.Reference{"clash": {Ref: "#/components/callbacks/other"}},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.target.Callbacks == nil {
				tc.target.Callbacks = map[string]*parser.Callback{}
			}
			if tc.target.CallbackRefs == nil {
				tc.target.CallbackRefs = map[string]*parser.Reference{}
			}

			err := New(JoinerConfig{}).mergeAllCallbacks(tc.target, tc.source,
				StrategyFailOnCollision, documentContext{}, &JoinResult{})

			require.Error(t, err)
			assert.ErrorContains(t, err, "clash")
			assert.ErrorContains(t, err, "the two forms cannot be merged")
		})
	}
}

// TestJoinCarriesCallbackRefs is the ordinary case: both forms survive a merge.
func TestJoinCarriesCallbackRefs(t *testing.T) {
	callback := parser.Callback{"{$request.query.url}": {Post: &parser.Operation{}}}
	target := &parser.Components{
		Callbacks:    map[string]*parser.Callback{},
		CallbackRefs: map[string]*parser.Reference{},
	}
	source := &parser.Components{
		Callbacks:    map[string]*parser.Callback{"inline": &callback},
		CallbackRefs: map[string]*parser.Reference{"referenced": {Ref: "#/components/callbacks/inline"}},
	}

	require.NoError(t, New(JoinerConfig{}).mergeAllCallbacks(target, source,
		StrategyFailOnCollision, documentContext{}, &JoinResult{}))

	assert.Contains(t, target.Callbacks, "inline")
	assert.Contains(t, target.CallbackRefs, "referenced",
		"the reference form was dropped by the join")
}
