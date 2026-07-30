package converter

import (
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
				"'name'",              // Server Object
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

// TestConvertToOAS32TargetReportsNothing is the control: the fields are legal at
// 3.2, so converting between 3.2 versions must stay silent about them.
func TestConvertToOAS32TargetReportsNothing(t *testing.T) {
	assert.Empty(t, oas32FeatureIssues(t, "3.2.0"),
		"nothing is lost converting a 3.2 document to 3.2")
}
