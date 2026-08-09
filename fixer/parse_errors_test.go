package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// missingResponsesSpec has one problem no fix covers: an OAS 3.0 operation must
// have a responses object. The parser reports it, converter refuses the document
// over it, and validator rejects it.
const missingResponsesSpec = `openapi: 3.0.3
info:
  title: T
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
`

// mixedSpec pairs a fixable problem with an unfixable one: /a/{id} declares no
// parameter for its path template, which the fixer adds, while /b is missing the
// responses object nothing fixes.
const mixedSpec = `openapi: 3.0.3
info:
  title: T
  version: "1.0.0"
paths:
  /a/{id}:
    get:
      operationId: a
      responses:
        "200":
          description: OK
  /b:
    get:
      operationId: b
`

// cleanSpec has nothing wrong with it.
const cleanSpec = `openapi: 3.0.3
info:
  title: T
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
      responses:
        "200":
          description: OK
`

// TestFixRecordsParseErrors covers #470. The fixer used to ignore
// ParseResult.Errors entirely, so a caller had no way to tell a document fixing
// left in good shape from one converter and validator both refuse.
func TestFixRecordsParseErrors(t *testing.T) {
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(missingResponsesSpec)))
	require.NoError(t, err)
	require.NotEmpty(t, parseResult.Errors, "the fixture must produce a parse error to be worth testing")

	result, err := New().FixParsed(*parseResult)
	require.NoError(t, err)

	assert.True(t, result.HasParseErrors(), "the source document's errors should be reported")
	assert.Len(t, result.ParseErrors, len(parseResult.Errors))
	assert.False(t, result.HasFixes(), "no fix covers a missing responses object")
}

// TestFixRecordsParseErrorsAlongsideFixes keeps the two independent: applying a
// fix is not evidence the document came out clean, which is the reading the old
// exit code invited.
func TestFixRecordsParseErrorsAlongsideFixes(t *testing.T) {
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(mixedSpec)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypeMissingPathParameter}

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	assert.True(t, result.HasFixes(), "the missing path parameter is fixable")
	assert.True(t, result.HasParseErrors(), "the missing responses object is not")
}

// TestFixReportsNoParseErrorsForCleanDocument is the half that must not regress:
// a good document still exits 0.
func TestFixReportsNoParseErrorsForCleanDocument(t *testing.T) {
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(cleanSpec)))
	require.NoError(t, err)
	require.Empty(t, parseResult.Errors)

	result, err := New().FixParsed(*parseResult)
	require.NoError(t, err)

	assert.False(t, result.HasParseErrors())
	assert.Empty(t, result.ParseErrors)
}

// TestFixParseErrorsMatchConverterRefusal ties the two packages together. The
// converter refuses exactly the documents the fixer now reports on, which is the
// agreement #470 asked for.
func TestFixParseErrorsMatchConverterRefusal(t *testing.T) {
	for _, tc := range []struct {
		name        string
		spec        string
		wantRefusal bool
	}{
		{"document converter refuses", missingResponsesSpec, true},
		{"document converter accepts", cleanSpec, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(tc.spec)))
			require.NoError(t, err)

			result, err := New().FixParsed(*parseResult)
			require.NoError(t, err)

			assert.Equal(t, tc.wantRefusal, result.HasParseErrors(),
				"fixer should report on exactly the documents converter refuses")
			assert.Equal(t, tc.wantRefusal, len(parseResult.Errors) > 0,
				"and converter's refusal is driven by the same field")
		})
	}
}
