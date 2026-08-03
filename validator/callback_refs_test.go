package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// callbackRefSpec builds a document whose operation and components each hold a
// callbacks entry written as a Reference Object, pointing wherever the caller
// says.
func callbackRefSpec(operationTarget, componentTarget string) string {
	return `
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
          $ref: '` + operationTarget + `'
components:
  callbacks:
    shared:
      'http://example.com':
        post:
          responses:
            '200':
              description: ok
    alias:
      $ref: '` + componentTarget + `'
`
}

// requireCleanParse fails unless the document parsed with no collected errors,
// so a validation result of zero errors cannot be the parser having stopped
// short of the callbacks.
func requireCleanParse(t *testing.T, spec string) {
	t.Helper()
	parsed, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)
	require.Empty(t, parsed.Errors)
}

// TestCallbackRefsResolveToComponents covers the ref-target check on the
// callbacks written as Reference Objects. They are held apart from the Callback
// Objects (see parser.Callback), so the traversal that checks every other $ref
// does not reach them on its own.
func TestCallbackRefsResolveToComponents(t *testing.T) {
	t.Run("a resolvable reference reports nothing", func(t *testing.T) {
		spec := callbackRefSpec("#/components/callbacks/shared", "#/components/callbacks/shared")
		requireCleanParse(t, spec)
		assert.Empty(t, validateSpec(t, spec).Errors)
	})

	t.Run("a dangling reference is reported at each position", func(t *testing.T) {
		spec := callbackRefSpec("#/components/callbacks/missingOp", "#/components/callbacks/missingComponent")
		requireCleanParse(t, spec)
		result := validateSpec(t, spec)

		assert.True(t, resultHasMessage(result, "missingOp"),
			"the operation's callback reference was not checked")
		assert.True(t, resultHasMessage(result, "missingComponent"),
			"the component's callback reference was not checked")
	})

	t.Run("a component written as a reference is itself referenceable", func(t *testing.T) {
		// `alias` is a Reference Object, so it lives in CallbackRefs rather than
		// Callbacks. It is still a component, and pointing at it must resolve.
		spec := callbackRefSpec("#/components/callbacks/alias", "#/components/callbacks/shared")
		requireCleanParse(t, spec)
		assert.Empty(t, validateSpec(t, spec).Errors)
	})
}

// TestCallbackRefComponentNameCharset covers the Components name charset rule on
// the reference form of a callbacks entry.
//
// The two forms are separate Go maps (see parser.Callback), so the rule has to
// be applied to each of them. Checking only the Callback Object map would leave
// a hole the exact shape of a `$ref`.
func TestCallbackRefComponentNameCharset(t *testing.T) {
	// `shared` gives the reference a real target, so a name error is the only
	// error the document can produce.
	spec := func(name string) string {
		return `openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  callbacks:
    shared: {}
    ` + name + `:
      $ref: '#/components/callbacks/shared'
`
	}

	t.Run("rejects an illegal name", func(t *testing.T) {
		assertErrorsMatch(t, validationErrors(t, spec(`"pet/summary"`)), []string{
			`components.callbacks.pet/summary: Component name "pet/summary" must match`,
		})
	})

	t.Run("accepts a legal name", func(t *testing.T) {
		assertErrorsMatch(t, validationErrors(t, spec("Pet.Summary-v2_1")), nil)
	})
}
