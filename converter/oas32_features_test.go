package converter

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

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
				// summary, on a Tag and on a Response
				"tags[0]: 'summary'",
				"paths./pets.get.responses.200: 'summary'",
				// the Components section that is itself 3.2-only, and what is inside it:
				// reporting only the container would understate what the target loses
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
