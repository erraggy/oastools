package converter

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// The 3.2 fixed fields now round trip (issue #397), which means they now reach
// conversion targets that have no meaning for them. This package's convention for
// such a field is to report it and leave it in place — detectOAS3SchemaFeatures
// does exactly that for nullable, if, and prefixItems — so these tests assert the
// reporting, not removal.

// oas32FeatureIssues converts the full-field 3.2 fixture to target and returns the
// issue messages mentioning 3.2.
func oas32FeatureIssues(t *testing.T, target string) []string {
	t.Helper()

	result, err := New().Convert("../testdata/oas32-all-fields.yaml", target)
	require.NoError(t, err)

	var messages []string
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, "OAS 3.2+ only") {
			messages = append(messages, issue.Path+": "+issue.Message)
		}
	}
	return messages
}

// TestDownconvertReportsOAS32Fields covers every fixed field OAS 3.2 added. Before
// this ran, a 3.2 to 3.0 conversion emitted all of them with no warning at all:
// convertOAS3ToOAS3 only bumped the version string.
func TestDownconvertReportsOAS32Fields(t *testing.T) {
	for _, target := range []string{"3.0.3", "3.1.0"} {
		t.Run(target, func(t *testing.T) {
			joined := strings.Join(oas32FeatureIssues(t, target), "\n")

			for _, field := range []string{
				"'$self'",
				"'mediaTypes'",        // Components Object
				"'name'",              // Server Object, including a Link's own
				"'summary'",           // Tag and Response Objects
				"'parent'",            // Tag Object
				"'kind'",              // Tag Object
				"'itemSchema'",        // Media Type Object
				"'itemEncoding'",      // Media Type and Encoding Objects
				"'prefixEncoding'",    // Media Type and Encoding Objects
				"'encoding'",          // nested Encoding Object
				"'dataValue'",         // Example Object
				"'serializedValue'",   // Example Object
				"'deprecated'",        // Security Scheme Object
				"'oauth2MetadataUrl'", // Security Scheme Object
				"'deviceAuthorization'",
				"'deviceAuthorizationUrl'",
				"'defaultMapping'", // Discriminator Object
				"'nodeType'",       // XML Object
				`'in: "querystring"'`,
				"'query'",                // Path Item Object
				"'additionalOperations'", // Path Item Object
			} {
				assert.Contains(t, joined, field,
					"converting to %s should report the 3.2 field %s", target, field)
			}

			assert.Contains(t, joined, "has no equivalent in OAS "+target,
				"the message should name the target version")
		})
	}
}

// TestDownconvertPreservesOAS32Fields pins that the fields are reported rather than
// stripped, matching how detectOAS3SchemaFeatures treats nullable and prefixItems.
// Removing them would make conversion lossy in a way no other field here is.
func TestDownconvertPreservesOAS32Fields(t *testing.T) {
	result, err := New().Convert("../testdata/oas32-all-fields.yaml", "3.1.0")
	require.NoError(t, err)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok)

	assert.Equal(t, "OtherPet", doc.Components.Schemas["Pet"].Discriminator.DefaultMapping,
		"defaultMapping must survive a 3.x to 3.y conversion, which strips nothing")
	assert.Equal(t, "production", doc.Servers[0].Name,
		"a server name must survive too")
}

// TestDownconvertReportsBare32Methods pins that the 3.2 HTTP methods are reported
// for themselves, not only when something 3.2-only is nested inside them. The walk
// over their contents cannot see the method, so a bare `query` operation converted
// to 3.0 previously produced no warning at all.
func TestDownconvertReportsBare32Methods(t *testing.T) {
	spec := `openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    query:
      operationId: queryPets
      responses:
        "200": {description: OK}
    additionalOperations:
      PURGE:
        operationId: purgePets
        responses:
          "204": {description: No Content}
`
	path := filepath.Join(t.TempDir(), "bare32.yaml")
	require.NoError(t, os.WriteFile(path, []byte(spec), 0o600))

	result, err := New().Convert(path, "3.0.3")
	require.NoError(t, err)

	var joined string
	for _, issue := range result.Issues {
		joined += issue.Path + ": " + issue.Message + "\n"
	}

	assert.Contains(t, joined, "'query' is OAS 3.2+ only",
		"a query operation with no 3.2-only content must still be reported")
	assert.Contains(t, joined, "'additionalOperations' is OAS 3.2+ only",
		"additionalOperations must be reported for itself")
}

// TestConvertToOAS32TargetReportsNothing is the control: the fields are legal at
// 3.2, so converting between 3.2 versions must stay silent about them.
func TestConvertToOAS32TargetReportsNothing(t *testing.T) {
	assert.Empty(t, oas32FeatureIssues(t, "3.2.0"),
		"nothing is lost converting a 3.2 document to 3.2")
}

// TestDownconvertReportsAmbiguousFieldsAtTheirLocation pins where a field is
// reported, not only that its name appears somewhere.
//
// TestDownconvertReportsOAS32Fields checks field names, which is enough for a name
// used once. It is not enough for `name` and `summary`, each of which several
// objects carry: `name` was already reported for the document's own servers, so
// that test passed for the whole time a Link Object's server went unreported.
func TestDownconvertReportsAmbiguousFieldsAtTheirLocation(t *testing.T) {
	for _, target := range []string{"3.0.3", "3.1.0"} {
		t.Run(target, func(t *testing.T) {
			issues := oas32FeatureIssues(t, target)

			for _, want := range []string{
				// name, on each of the two kinds of Server Object
				"servers[0]: 'name'",
				"components.links.petById.server: 'name'",
				"paths./pets.get.responses.200.links.firstPet.server: 'name'",
				// summary, on a Tag and on a Response
				"tags[0]: 'summary'",
				"paths./pets.get.responses.200: 'summary'",
				// the Components section that is itself 3.2-only, and what is inside it:
				// reporting only the container would understate what the target loses
				// nodeType, on a components schema and on an inline one: the schema
				// rules reached components.schemas and the request body alone
				"components.schemas.Pet.properties.tag.xml: 'nodeType'",
				"paths./pets.get.responses.200.content.multipart/form-data.schema.properties.meta.xml: 'nodeType'",
				"components.mediaTypes: 'mediaTypes'",
				"components.mediaTypes.PetStream: 'itemSchema'",
			} {
				// Matched per issue rather than against the joined text: every path here
				// is a prefix of some other path in the document, so a substring search
				// over one blob would let a nested report satisfy its parent's assertion.
				assert.True(t, slices.ContainsFunc(issues, func(issue string) bool {
					return strings.HasPrefix(issue, want+" ")
				}), "converting to %s should report %s at that location; got %v", target, want, issues)
			}
		})
	}
}

// TestDownconvertIssueOrderIsDeterministic pins the ordering.
//
// The section walks range over maps, so the full-field fixture reported these
// issues in four distinct orderings across eight runs. Anything diffing conversion
// output between runs saw changes that were not there.
func TestDownconvertIssueOrderIsDeterministic(t *testing.T) {
	first := oas32FeatureIssues(t, "3.0.3")
	require.NotEmpty(t, first)

	// Repeated rather than compared against a fixed list: the assertion is that the
	// order holds, not that it is any particular order.
	for range 12 {
		assert.Equal(t, first, oas32FeatureIssues(t, "3.0.3"),
			"the same document must report its issues in the same order every run")
	}

	// Asserted on the sort key itself, not the rendered "path: message" line: '.'
	// sorts before ':', so a parent path and its child render out of order while
	// their paths are sorted correctly.
	paths := make([]string, 0, len(first))
	for _, issue := range first {
		path, _, _ := strings.Cut(issue, ": ")
		paths = append(paths, path)
	}
	assert.True(t, slices.IsSorted(paths),
		"sorted by path, so the order is predictable and not merely stable: %v", paths)
}

// TestDownconvertWalksInsideTheThreeTwoOnlyOperations pins that a 3.2 field
// nested inside `query` or `additionalOperations` is reported.
//
// detectOAS32PathItemFeatures used to take its operations from
// parser.GetOperations, which is version-aware and omits those two below 3.2.
// This pass runs only when the target is below 3.2, and on a 3.x to 3.x
// conversion the document already carries the target version, so the accessor
// dropped exactly the operations whose contents needed walking: converting to
// 3.1 reported `query` itself but nothing inside it.
//
// The 3.0/3.1 targets are the cases that were broken. 2.0 worked by accident,
// because that path passes the source document, whose version is still 3.2.
func TestDownconvertWalksInsideTheThreeTwoOnlyOperations(t *testing.T) {
	spec := `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths:
  /p:
    get:
      operationId: g
      responses:
        "200": {description: OK, summary: from get}
    query:
      operationId: q
      responses:
        "200": {description: OK, summary: from query}
    additionalOperations:
      PURGE:
        operationId: p
        responses:
          "200": {description: OK, summary: from purge}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "spec.yaml")
	require.NoError(t, os.WriteFile(path, []byte(spec), 0o600))

	for _, target := range []string{"3.0.3", "3.1.0", "2.0"} {
		t.Run(target, func(t *testing.T) {
			result, err := New().Convert(path, target)
			require.NoError(t, err)

			var reported []string
			for _, issue := range result.Issues {
				if strings.Contains(issue.Message, "OAS 3.2+ only") {
					reported = append(reported, issue.Path+": "+issue.Message)
				}
			}
			joined := strings.Join(reported, "\n")

			for _, want := range []string{
				"paths./p.get.responses.200: 'summary'",
				"paths./p.query.responses.200: 'summary'",
				"paths./p.additionalOperations.PURGE.responses.200: 'summary'",
			} {
				assert.Contains(t, joined, want,
					"converting to %s should report the summary nested in that operation", target)
			}
		})
	}
}

// cyclicEncoding returns an Encoding Object whose nested encodings all lead back
// to it, so the graph is a cycle that branches `cycling` ways at every step.
//
// Hand-built because no parsed document can reach the same Encoding Object twice.
// Encoding is not a referenceable component, and the two routes that could share
// a pointer do not: `$ref` resolution and YAML aliases each produce a copy.
// Convert takes the caller's document.
func cyclicEncoding(cycling int) *parser.Encoding {
	enc := &parser.Encoding{}
	nested := make(map[string]*parser.Encoding, cycling)
	for i := range cycling {
		nested[string(rune('a'+i))] = enc
	}
	enc.Encoding = nested
	return enc
}

// TestDetectOAS32EncodingFeaturesTerminatesOnACycle pins the visited set. The
// depth bound alone does not contain a cycle whose encoding nests more than once,
// because the walk branches and goes exponential in depth long before the bound
// is reached; removing the set hangs this rather than failing it.
//
// The validator carries three more Encoding walks with the same shape, pinned by
// TestEncodingWalksTerminateOnACycle. Nothing in the type system connects the
// four.
func TestDetectOAS32EncodingFeaturesTerminatesOnACycle(t *testing.T) {
	for _, cycling := range []int{1, 2, 3} {
		var reported []string
		report := func(path, field string) {
			reported = append(reported, path+": "+field)
		}

		done := make(chan struct{})
		go func() {
			defer close(done)
			detectOAS32EncodingFeatures(cyclicEncoding(cycling), "content.x", report, nil, 0)
		}()
		select {
		case <-done:
		case <-time.After(15 * time.Second):
			t.Fatalf("fan-out %d: the walk did not terminate; the visited set is gone", cycling)
		}

		assert.Equal(t, []string{"content.x: encoding"}, reported,
			"fan-out %d: the encoding should be reported once however many nested keys lead back to it", cycling)
	}
}

// TestDetectOAS32EncodingFeaturesStopsAtTheDepthBound pins the bound, which the
// visited set does not subsume: a chain of distinct encodings repeats nothing, so
// only the counter stops it.
func TestDetectOAS32EncodingFeaturesStopsAtTheDepthBound(t *testing.T) {
	const links = maxEncodingNestingDepth + 150

	head := &parser.Encoding{}
	current := head
	for range links - 1 {
		next := &parser.Encoding{}
		current.Encoding = map[string]*parser.Encoding{"next": next}
		current = next
	}

	var reported int
	report := func(string, string) { reported++ }
	detectOAS32EncodingFeatures(head, "content.x", report, nil, 0)

	// Every link but the last holds a nested encoding to report, and the head
	// sits at depth 0, so the bound admits one more link than it names.
	assert.Equal(t, maxEncodingNestingDepth+1, reported,
		"the walk should stop at the nesting bound rather than following all %d links", links)
}
