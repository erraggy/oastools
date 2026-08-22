// headers_test.go covers the Header Object conversions. parser.Header is a union
// of the OAS 2.0 and OAS 3.x spellings, so these tests assert both that the
// target version's fields arrive AND that the source version's fields do not
// survive, which is the half a "was it converted?" assertion cannot see.
package converter

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// headerSourceOAS31 carries a response header, a component header reached by
// $ref, and an array header, so one conversion exercises every position a
// Header Object occupies in an OAS 3.x document.
const headerSourceOAS31 = `openapi: 3.1.0
info:
  title: headers
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
      responses:
        "200":
          description: OK
          headers:
            X-Object:
              schema:
                type: object
                unevaluatedProperties:
                  - type: string
            X-Array:
              schema:
                type: array
                items:
                  type: string
                  maxLength: 4
            X-Ref:
              $ref: '#/components/headers/Shared'
components:
  headers:
    Shared:
      description: shared
      required: true
      deprecated: true
      style: simple
      schema:
        type: integer
        exclusiveMinimum: 3
`

func convertHeaderFixture(t *testing.T, source, target string) (*ConversionResult, map[string]*parser.Header) {
	t.Helper()

	parsed, err := parser.New().ParseBytes([]byte(source))
	require.NoError(t, err)

	result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion(target))
	require.NoError(t, err)

	var headers map[string]*parser.Header
	switch doc := result.Document.(type) {
	case *parser.OAS2Document:
		headers = doc.Paths["/a"].Get.Responses.Codes["200"].Headers
	case *parser.OAS3Document:
		headers = doc.Paths["/a"].Get.Responses.Codes["200"].Headers
	default:
		t.Fatalf("unexpected document type %T", result.Document)
	}
	require.NotEmpty(t, headers, "the fixture should carry response headers; it does not, so this test proves nothing")
	return result, headers
}

func TestOAS3HeaderDemotesToOAS2Spelling(t *testing.T) {
	_, headers := convertHeaderFixture(t, headerSourceOAS31, "2.0")

	obj := headers["X-Object"]
	require.NotNil(t, obj)
	assert.Nil(t, obj.Schema, "an OAS 2.0 Header Object has no 'schema' field")
	assert.Equal(t, "object", obj.Type, "the type should come across in the OAS 2.0 spelling")

	arr := headers["X-Array"]
	require.NotNil(t, arr)
	assert.Nil(t, arr.Schema)
	assert.Equal(t, "array", arr.Type)
	require.NotNil(t, arr.Items, "an array header should carry an OAS 2.0 Items Object")
	assert.Equal(t, "string", arr.Items.Type)
	require.NotNil(t, arr.Items.MaxLength, "the element keywords should come across, not just the type")
	assert.Equal(t, 4, *arr.Items.MaxLength)
}

// TestOAS3HeaderSchemaReachesTheSchemaPasses pins that a header's schema goes
// through the per-schema passes, so an array in a schema-or-bool field is
// cleared and reported rather than carried across.
func TestOAS3HeaderSchemaReachesTheSchemaPasses(t *testing.T) {
	result, headers := convertHeaderFixture(t, headerSourceOAS31, "2.0")

	obj := headers["X-Object"]
	require.NotNil(t, obj)
	assert.Nil(t, obj.Schema)

	// The counterpart to the assertion above: the loss has to be reported, not
	// merely absent from the output.
	assertIssueMentioning(t, result, "unevaluatedProperties")
}

func TestOAS3HeaderRefIsInlinedAndConverted(t *testing.T) {
	result, headers := convertHeaderFixture(t, headerSourceOAS31, "2.0")

	ref := headers["X-Ref"]
	require.NotNil(t, ref)
	assert.Nil(t, ref.Schema, "the inlined component header still needs demoting")
	assert.Equal(t, "integer", ref.Type)
	assert.True(t, ref.ExclusiveMinimum, "a numeric exclusiveMinimum becomes the draft 4 boolean plus a bound")
	require.NotNil(t, ref.Minimum)
	assert.InDelta(t, 3.0, *ref.Minimum, 0.0001)

	// OAS 2.0 Header Objects define none of these, so each is dropped loudly.
	assertIssueMentioning(t, result, "required")
	assertIssueMentioning(t, result, "deprecated")
	assertIssueMentioning(t, result, "style")
}

func TestOAS2HeaderPromotesToOAS3Spelling(t *testing.T) {
	const source = `swagger: "2.0"
info:
  title: headers
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
      responses:
        "200":
          description: OK
          headers:
            X-Array:
              type: array
              collectionFormat: pipes
              items:
                type: string
                maxLength: 4
            X-Count:
              type: integer
              minimum: 1
`

	result, headers := convertHeaderFixture(t, source, "3.1.0")

	arr := headers["X-Array"]
	require.NotNil(t, arr)
	assert.Empty(t, arr.Type, "the OAS 2.0 type declaration must not survive into an OAS 3.x header")
	assert.Empty(t, arr.CollectionFormat, "collectionFormat is an OAS 2.0 field")
	assert.Nil(t, arr.Items, "the OAS 2.0 Items Object must not survive")
	require.NotNil(t, arr.Schema, "the type declaration should become a Schema Object")
	assert.Equal(t, "array", arr.Schema.Type)

	elem, ok := arr.Schema.Items.(*parser.Schema)
	require.True(t, ok, "the element schema should be a single schema, got %T", arr.Schema.Items)
	assert.Equal(t, "string", elem.Type)

	count := headers["X-Count"]
	require.NotNil(t, count)
	require.NotNil(t, count.Schema)
	assert.Equal(t, "integer", count.Schema.Type)
	require.NotNil(t, count.Schema.Minimum)
	assert.InDelta(t, 1.0, *count.Schema.Minimum, 0.0001)

	// pipes has no OAS 3.x spelling, so it is reported rather than dropped quietly.
	assertIssueMentioning(t, result, "collectionFormat")
}

// TestOAS3HeaderTupleKeepsFirstPositionAndReports covers the one lossy case in
// the array demotion: an OAS 2.0 Items Object applies one schema to every
// element and cannot express positions.
func TestOAS3HeaderTupleKeepsFirstPositionAndReports(t *testing.T) {
	const source = `openapi: 3.1.0
info:
  title: headers
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
      responses:
        "200":
          description: OK
          headers:
            X-Tuple:
              schema:
                type: array
                prefixItems:
                  - type: string
                  - type: integer
`

	result, headers := convertHeaderFixture(t, source, "2.0")

	tuple := headers["X-Tuple"]
	require.NotNil(t, tuple)
	assert.Equal(t, "array", tuple.Type)
	require.NotNil(t, tuple.Items)
	assert.Equal(t, "string", tuple.Items.Type, "the first position is what survives")
	assertIssueMentioning(t, result, "tuple")
}

func assertIssueMentioning(t *testing.T, result *ConversionResult, substr string) {
	t.Helper()
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, substr) || strings.Contains(issue.Context, substr) {
			return
		}
	}
	t.Fatalf("no conversion issue mentions %q; got %d issues", substr, len(result.Issues))
}

// TestUnresolvableComponentHeaderRefIsDropped covers a header naming a component
// that cannot be inlined. OAS 2.0 has no components.headers, so carrying the
// reference through emits a document that cannot resolve it.
func TestUnresolvableComponentHeaderRefIsDropped(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
	}{
		{
			// No components.headers at all, so resolution is never attempted.
			name: "no components headers",
			source: `openapi: 3.1.0
info:
  title: t
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
      responses:
        "200":
          description: OK
          headers:
            X-H:
              $ref: '#/components/headers/Missing'
`,
		},
		{
			// The section exists but does not hold the name.
			name: "name not present",
			source: `openapi: 3.1.0
info:
  title: t
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
      responses:
        "200":
          description: OK
          headers:
            X-H:
              $ref: '#/components/headers/Missing'
components:
  headers:
    Other:
      schema: {type: string}
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, headers := convertHeaderFixture(t, tc.source, "2.0")

			h := headers["X-H"]
			require.NotNil(t, h)
			assert.Empty(t, h.Ref, "OAS 2.0 has no components.headers for this reference to name")
			assert.Equal(t, "string", h.Type, "an OAS 2.0 Header Object requires a type")
			assertIssueMentioning(t, result, "reference dropped")
		})
	}
}

// TestResolvableComponentHeaderRefIsStillInlined is the counterpart: dropping
// must not reach a reference that resolves.
func TestResolvableComponentHeaderRefIsStillInlined(t *testing.T) {
	const source = `openapi: 3.1.0
info:
  title: t
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
      responses:
        "200":
          description: OK
          headers:
            X-H:
              $ref: '#/components/headers/Shared'
components:
  headers:
    Shared:
      schema: {type: integer}
`

	_, headers := convertHeaderFixture(t, source, "2.0")

	h := headers["X-H"]
	require.NotNil(t, h)
	assert.Empty(t, h.Ref)
	assert.Equal(t, "integer", h.Type, "the component's own type should be inlined, not a default")
}
