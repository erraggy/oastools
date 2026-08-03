package fixer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// A callbacks entry written as a Reference Object is carried on its own parser
// field (see parser.Callback), so RefCollector has to read that field as well as
// the Callback Object map. Reading only the latter would report a referenced
// callback as unreferenced, which is what IsCallbackReferenced is asked.
const callbackRefCollectorSpec = `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /things:
    post:
      responses:
        "200":
          description: ok
      callbacks:
        fromOperation:
          $ref: "#/components/callbacks/usedByOperation"
components:
  callbacks:
    usedByOperation:
      "{$request.query.url}":
        post:
          responses:
            "200":
              description: ok
    usedByComponent:
      "{$request.query.url}":
        post:
          responses:
            "200":
              description: ok
    fromComponent:
      $ref: "#/components/callbacks/usedByComponent"
    orphan:
      "{$request.query.url}":
        post:
          responses:
            "200":
              description: ok
`

func TestCallbackRefsAreCollectedAsReferences(t *testing.T) {
	parseResult, err := parser.New().ParseBytes([]byte(callbackRefCollectorSpec))
	require.NoError(t, err)
	doc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	collector := NewRefCollector()
	collector.CollectOAS3(doc)

	assert.True(t, collector.IsCallbackReferenced("usedByOperation"),
		"a callback referenced from an operation was not collected")
	assert.True(t, collector.IsCallbackReferenced("usedByComponent"),
		"a callback referenced from another component was not collected")
	assert.False(t, collector.IsCallbackReferenced("orphan"),
		"a callback nothing points at should not be reported as referenced")
}

// TestComponentsHoldingOnlyCallbackRefsIsNotEmpty covers the emptiness check that
// decides whether the whole Components Object is dropped. A reference-form
// callback lives on its own field, so a check reading only the Callback Object
// map would call this document's components empty and delete a component it has.
func TestComponentsHoldingOnlyCallbackRefsIsNotEmpty(t *testing.T) {
	assert.False(t, isComponentsEmpty(&parser.Components{
		CallbackRefs: map[string]*parser.Reference{"alias": {Ref: "#/components/callbacks/shared"}},
	}), "components holding a reference-form callback were reported empty")

	assert.True(t, isComponentsEmpty(&parser.Components{}),
		"components with nothing in them should be empty")
}

// TestCallbackRefCollectionSkipsUnusableEntries covers the guards on the
// collector's loop: a nil entry and one with no $ref name nothing, so neither
// should reach the reference set.
func TestCallbackRefCollectionSkipsUnusableEntries(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "T", Version: "1.0.0"},
		Components: &parser.Components{
			CallbackRefs: map[string]*parser.Reference{
				"nilEntry":   nil,
				"emptyRef":   {Ref: ""},
				"realTarget": {Ref: "#/components/callbacks/wanted"},
			},
		},
	}

	collector := NewRefCollector()
	assert.NotPanics(t, func() { collector.CollectOAS3(doc) },
		"a nil reference entry crashed collection")
	assert.True(t, collector.IsCallbackReferenced("wanted"))
}
