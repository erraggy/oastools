// media_type_determinism_test.go pins that generation does not depend on Go's
// map ordering. A response or request body offering several media types has to
// yield the same Go types on every run.
package generator

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const multiContentSpec = `openapi: 3.1.0
info:
  title: t
  version: "1.0.0"
paths:
  /a:
    post:
      operationId: doThing
      requestBody:
        content:
          application/xml:
            schema: {type: boolean}
          application/ld+json:
            schema: {type: integer}
          application/json:
            schema: {type: string}
      responses:
        "200":
          description: OK
          content:
            application/atom+xml:
              schema: {type: boolean}
            application/geo+json:
              schema: {type: integer}
            application/ld+json:
              schema: {type: string}
`

func generateOnce(t *testing.T) *GenerateResult {
	t.Helper()
	parsed, err := parser.New().ParseBytes([]byte(multiContentSpec))
	require.NoError(t, err)

	res, err := GenerateWithOptions(
		WithParsed(*parsed),
		WithPackageName("api"),
		WithClient(true),
		WithServer(true),
		WithServerResponses(true),
		WithTypes(true),
	)
	require.NoError(t, err)
	require.NotNil(t, res)
	return res
}

// TestGenerationIsRepeatable runs the generator many times over one document
// and requires a single answer.
//
// Each media type carries a different primitive schema, so a choice made by map
// order changes the generated Go type rather than only the media type string,
// which is what makes a wrong choice visible in the output. Go randomizes map order per range, so a choice
// still driven by it appears here rather than in a single comparison.
func TestGenerationIsRepeatable(t *testing.T) {
	first := generateOnce(t)
	firstFiles := fileContents(t, first)

	for range 30 {
		got := fileContents(t, generateOnce(t))
		assert.Equal(t, firstFiles, got, "generated output changed between runs")
	}
}

// fileContents maps each generated file to its content, with the README's
// generation timestamp removed.
//
// That line carries the wall clock, so two runs either side of a second boundary
// differ there whatever the generator chose. Comparing it would make this test
// fail for a reason it is not about, and pass or fail by timing.
func fileContents(t *testing.T, res *GenerateResult) map[string]string {
	t.Helper()
	out := make(map[string]string, len(res.Files))
	for _, f := range res.Files {
		content := string(f.Content)
		if f.Name == "README.md" {
			content = stripGeneratedTimestamp(content)
		}
		out[f.Name] = content
	}
	require.NotEmpty(t, out, "the fixture should generate files; it did not, so this test proves nothing")
	require.Contains(t, out, "client.go", "the client is where a request body choice becomes a Go type")
	require.Contains(t, out, "server.go", "the server is where a response choice becomes a Go type")
	require.Contains(t, out, "server_responses.go", "the typed response helpers carry the per status code body type")
	return out
}

func stripGeneratedTimestamp(readme string) string {
	lines := strings.Split(readme, "\n")
	kept := make([]string, 0, len(lines))
	for _, l := range lines {
		if strings.HasPrefix(l, "| Generated |") {
			continue
		}
		kept = append(kept, l)
	}
	return strings.Join(kept, "\n")
}
