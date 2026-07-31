package parser

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// The two fixtures are the same document in the two formats. Both are asserted
// against the same expectations, which is the point: issue #397 was that the
// formats disagreed, and no per-format test could have caught it.
const (
	oas32FixtureYAML = "../testdata/oas32-all-fields.yaml"
	oas32FixtureJSON = "../testdata/oas32-all-fields.json"
)

// assertOAS32FieldsPresent asserts every fixed field OAS 3.2.0 added over 3.1.1
// is populated, per the audit in issue #397.
//
// Deliberately one function applied to every parsed form of the document —
// fixture, re-parsed output, cross-format conversion — rather than assertions
// spread across the tests. The original defect was that a field survived one path
// and vanished on another, so any check that does not run identically on both
// paths could have passed while the bug was live.
func assertOAS32FieldsPresent(t *testing.T, doc *OAS3Document) {
	t.Helper()

	// Server Object
	require.Len(t, doc.Servers, 1)
	assert.Equal(t, "production", doc.Servers[0].Name, "servers[0].name")

	// Tag Object
	require.NotEmpty(t, doc.Tags)
	pets := doc.Tags[0]
	assert.Equal(t, "Pet operations", pets.Summary, "tags[0].summary")
	assert.Equal(t, "root", pets.Parent, "tags[0].parent")
	assert.Equal(t, "nav", pets.Kind, "tags[0].kind")

	// Response Object
	get := doc.Paths["/pets"].Get
	require.NotNil(t, get)
	resp := get.Responses.Codes["200"]
	require.NotNil(t, resp)
	assert.Equal(t, "Pets, or the default shape", resp.Summary, "response.summary")

	// Media Type Object: itemSchema describes each item of a sequential media type.
	jsonl := resp.Content["application/jsonl"]
	require.NotNil(t, jsonl, "application/jsonl media type")
	require.NotNil(t, jsonl.ItemSchema, "mediaType.itemSchema")
	assert.Equal(t, "#/components/schemas/Pet", jsonl.ItemSchema.Ref, "mediaType.itemSchema.$ref")

	// itemEncoding and prefixEncoding only apply to a multipart media type.
	mixed := resp.Content["multipart/mixed"]
	require.NotNil(t, mixed, "multipart/mixed media type")
	require.NotNil(t, mixed.ItemEncoding, "mediaType.itemEncoding")
	assert.Equal(t, "application/json", mixed.ItemEncoding.ContentType)
	require.Len(t, mixed.PrefixEncoding, 1, "mediaType.prefixEncoding")
	assert.Equal(t, "application/json", mixed.PrefixEncoding[0].ContentType)

	// Path Item Object: the two methods 3.2 added.
	require.NotNil(t, doc.Paths["/pets"].Query, "pathItem.query")
	assert.Equal(t, "queryPets", doc.Paths["/pets"].Query.OperationID)
	purge := doc.Paths["/pets"].AdditionalOperations["PURGE"]
	require.NotNil(t, purge, "pathItem.additionalOperations.PURGE")
	assert.Equal(t, "purgePets", purge.OperationID)

	// Encoding Object: recursive as of 3.2
	multipart := resp.Content["multipart/form-data"]
	require.NotNil(t, multipart)
	meta := multipart.Encoding["meta"]
	require.NotNil(t, meta, "encoding.meta")
	require.NotNil(t, meta.Encoding["nested"], "encoding.meta.encoding.nested")
	assert.Equal(t, "text/plain", meta.Encoding["nested"].ContentType)
	require.NotNil(t, meta.ItemEncoding, "encoding.meta.itemEncoding")
	assert.Equal(t, "text/plain", meta.ItemEncoding.ContentType)
	require.Len(t, meta.PrefixEncoding, 1, "encoding.meta.prefixEncoding")
	assert.Equal(t, "text/plain", meta.PrefixEncoding[0].ContentType)

	// Parameter Object: in: "querystring"
	post := doc.Paths["/pets"].Post
	require.NotNil(t, post)
	require.Len(t, post.Parameters, 1)
	assert.Equal(t, ParamInQueryString, post.Parameters[0].In, "parameter.in")

	require.NotNil(t, doc.Components)

	// Discriminator Object
	pet := doc.Components.Schemas["Pet"]
	require.NotNil(t, pet)
	require.NotNil(t, pet.Discriminator)
	assert.Equal(t, "OtherPet", pet.Discriminator.DefaultMapping, "discriminator.defaultMapping")

	// XML Object
	tag := pet.Properties["tag"]
	require.NotNil(t, tag)
	require.NotNil(t, tag.XML)
	assert.Equal(t, "attribute", tag.XML.NodeType, "xml.nodeType")

	// Example Object
	ex := doc.Components.Examples["withData"]
	require.NotNil(t, ex)
	assert.NotNil(t, ex.DataValue, "example.dataValue")
	assert.Equal(t, `{"petType":"dog"}`, ex.SerializedValue, "example.serializedValue")

	// Security Scheme Object and the OAuth flow objects
	oauth := doc.Components.SecuritySchemes["oauth"]
	require.NotNil(t, oauth)
	assert.True(t, oauth.Deprecated, "securityScheme.deprecated")
	assert.Equal(t, "https://auth.example.com/.well-known/oauth-authorization-server",
		oauth.OAuth2MetadataURL, "securityScheme.oauth2MetadataUrl")
	require.NotNil(t, oauth.Flows)
	device := oauth.Flows.DeviceAuthorization
	require.NotNil(t, device, "oauthFlows.deviceAuthorization")
	assert.Equal(t, "https://auth.example.com/device",
		device.DeviceAuthorizationURL, "oauthFlow.deviceAuthorizationUrl")
}

// parseOAS32Fixture parses one of the fixtures and returns its document.
func parseOAS32Fixture(t *testing.T, path string) *OAS3Document {
	t.Helper()

	src, err := os.ReadFile(path)
	require.NoError(t, err)

	result, err := ParseWithOptions(WithBytes(src), WithValidateStructure(true))
	require.NoError(t, err, "fixture should parse")

	doc, ok := result.OAS3Document()
	require.True(t, ok)
	return doc
}

// TestOAS32AllFieldsParse is the decode half of issue #397: the fields have to
// reach the struct at all, from either format.
func TestOAS32AllFieldsParse(t *testing.T) {
	for name, path := range map[string]string{"yaml": oas32FixtureYAML, "json": oas32FixtureJSON} {
		t.Run(name, func(t *testing.T) {
			assertOAS32FieldsPresent(t, parseOAS32Fixture(t, path))
		})
	}
}

// TestOAS32AllFieldsSurviveRoundTrip is the encode half, and the regression for
// the issue's headline symptom: "oastools fix on a valid OAS 3.2 JSON document
// deletes spec-conformant content and reports success."
//
// Marshaling with encoding/json and go.yaml.in/yaml directly is what the CLI does
// when it writes output, so this exercises the same serialization the reported
// command did.
func TestOAS32AllFieldsSurviveRoundTrip(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		doc := parseOAS32Fixture(t, oas32FixtureJSON)

		out, err := json.MarshalIndent(doc, "", "  ")
		require.NoError(t, err)

		result, err := ParseWithOptions(WithBytes(out), WithValidateStructure(true))
		require.NoError(t, err, "re-emitted JSON should parse")
		reparsed, ok := result.OAS3Document()
		require.True(t, ok)

		assertOAS32FieldsPresent(t, reparsed)
	})

	t.Run("yaml", func(t *testing.T) {
		doc := parseOAS32Fixture(t, oas32FixtureYAML)

		out, err := yaml.Marshal(doc)
		require.NoError(t, err)

		result, err := ParseWithOptions(WithBytes(out), WithValidateStructure(true))
		require.NoError(t, err, "re-emitted YAML should parse")
		reparsed, ok := result.OAS3Document()
		require.True(t, ok)

		assertOAS32FieldsPresent(t, reparsed)
	})
}

// TestOAS32AllFieldsRoundTripWithoutExtensions covers the marshaling path the
// fixture cannot reach on its own.
//
// Every object in the fixture carries an "x-" extension, which sends each type
// down its slow MarshalJSON path — the one that hand-builds a map[string]any.
// Clearing Extra sends the same document down the fast path, which delegates to
// the struct tags instead. Both paths must emit every field; a field added to the
// struct but not to the map builder passes here and fails above, and one added to
// neither fails in both.
func TestOAS32AllFieldsRoundTripWithoutExtensions(t *testing.T) {
	doc := parseOAS32Fixture(t, oas32FixtureJSON)
	stripOAS32Extensions(doc)

	out, err := json.MarshalIndent(doc, "", "  ")
	require.NoError(t, err)
	assert.NotContains(t, string(out), `"x-`,
		"extensions should have been cleared, so this exercises the fast marshal path")

	result, err := ParseWithOptions(WithBytes(out), WithValidateStructure(true))
	require.NoError(t, err)
	reparsed, ok := result.OAS3Document()
	require.True(t, ok)

	assertOAS32FieldsPresent(t, reparsed)
}

// stripOAS32Extensions clears Extra on every object the fixture puts an extension
// on, so the document marshals through the fast path.
func stripOAS32Extensions(doc *OAS3Document) {
	for _, s := range doc.Servers {
		s.Extra = nil
	}
	for _, tag := range doc.Tags {
		tag.Extra = nil
	}

	resp := doc.Paths["/pets"].Get.Responses.Codes["200"]
	resp.Extra = nil
	for _, mt := range resp.Content {
		mt.Extra = nil
		if mt.ItemEncoding != nil {
			mt.ItemEncoding.Extra = nil
		}
		for _, enc := range mt.PrefixEncoding {
			enc.Extra = nil
		}
		for _, enc := range mt.Encoding {
			enc.Extra = nil
			for _, nested := range enc.Encoding {
				nested.Extra = nil
			}
		}
	}

	pet := doc.Components.Schemas["Pet"]
	pet.Discriminator.Extra = nil
	pet.Properties["tag"].XML.Extra = nil
	doc.Components.Examples["withData"].Extra = nil

	oauth := doc.Components.SecuritySchemes["oauth"]
	oauth.Extra = nil
	oauth.Flows.Extra = nil
	oauth.Flows.DeviceAuthorization.Extra = nil
}

// TestOAS32FormatsAgree pins that neither format is privileged: the same document
// written as YAML and as JSON must parse to equal documents.
//
// This is the assertion the issue's root-cause analysis calls for. Before the fix
// the YAML fixture kept the 3.2 fields in its inline Extra map while the JSON one
// discarded them, so the two parsed to different documents while both "worked".
func TestOAS32FormatsAgree(t *testing.T) {
	fromYAML := parseOAS32Fixture(t, oas32FixtureYAML)
	fromJSON := parseOAS32Fixture(t, oas32FixtureJSON)

	assert.True(t, fromYAML.Equals(fromJSON),
		"the same document in YAML and JSON must parse to equal documents")
}

// TestOAS32DeepCopyPreservesNewFields covers the generated DeepCopy methods,
// which are a third path a new field has to be wired into — the deepcopy
// generator is driven by a hand-maintained field table, not by the structs, so a
// pointer, slice, or map field added to a struct alone is silently shallow-copied.
func TestOAS32DeepCopyPreservesNewFields(t *testing.T) {
	original := parseOAS32Fixture(t, oas32FixtureYAML)
	clone := original.DeepCopy()

	assertOAS32FieldsPresent(t, clone)
	assert.True(t, original.Equals(clone), "a deep copy must equal its original")

	// Mutating the clone's nested 3.2 structures must not reach the original, or
	// the copy is shallow and the two share state.
	cloneContent := clone.Paths["/pets"].Get.Responses.Codes["200"].Content
	cloneContent["application/jsonl"].ItemSchema.Ref = "#/components/schemas/Mutated"
	cloneContent["multipart/mixed"].ItemEncoding.ContentType = "text/mutated"
	cloneContent["multipart/mixed"].PrefixEncoding[0].ContentType = "text/mutated"
	clone.Paths["/pets"].Query.OperationID = "mutated"
	clone.Paths["/pets"].AdditionalOperations["PURGE"].OperationID = "mutated"
	clone.Components.SecuritySchemes["oauth"].Flows.DeviceAuthorization.DeviceAuthorizationURL = "https://mutated"

	originalContent := original.Paths["/pets"].Get.Responses.Codes["200"].Content
	assert.Equal(t, "#/components/schemas/Pet", originalContent["application/jsonl"].ItemSchema.Ref)
	assert.Equal(t, "application/json", originalContent["multipart/mixed"].ItemEncoding.ContentType)
	assert.Equal(t, "application/json", originalContent["multipart/mixed"].PrefixEncoding[0].ContentType)
	assert.Equal(t, "queryPets", original.Paths["/pets"].Query.OperationID)
	assert.Equal(t, "purgePets", original.Paths["/pets"].AdditionalOperations["PURGE"].OperationID)
	assert.Equal(t, "https://auth.example.com/device",
		original.Components.SecuritySchemes["oauth"].Flows.DeviceAuthorization.DeviceAuthorizationURL)
}

// TestQueryStringLocationVersionGate covers the parser's own enforcement of the
// version that introduced in: "querystring". It is the parser's job rather than
// the validator's because an unrecognized `in` value is a structural problem, and
// the parser already owns the list of legal locations.
func TestQueryStringLocationVersionGate(t *testing.T) {
	spec := func(version string) string {
		return `
openapi: "` + version + `"
info: {title: T, version: "1.0.0"}
paths:
  /search:
    get:
      operationId: search
      parameters:
        - name: q
          in: querystring
          content:
            application/x-www-form-urlencoded:
              schema: {type: object}
      responses:
        "200": {description: OK}
`
	}

	// Structure validation reports through ParseResult.Errors rather than the
	// returned error, which is reserved for a document that could not be decoded
	// at all.
	structureErrors := func(t *testing.T, version string) []string {
		t.Helper()

		result, err := ParseWithOptions(WithBytes([]byte(spec(version))), WithValidateStructure(true))
		require.NoError(t, err, "the document should still decode")

		messages := make([]string, 0, len(result.Errors))
		for _, e := range result.Errors {
			messages = append(messages, e.Error())
		}
		return messages
	}

	t.Run("accepted at 3.2", func(t *testing.T) {
		for _, msg := range structureErrors(t, "3.2.0") {
			assert.NotContains(t, msg, "querystring",
				"querystring is a legal location at 3.2 and must not be reported")
		}

		result, err := ParseWithOptions(WithBytes([]byte(spec("3.2.0"))), WithValidateStructure(true))
		require.NoError(t, err)
		doc, ok := result.OAS3Document()
		require.True(t, ok)
		assert.Equal(t, ParamInQueryString, doc.Paths["/search"].Get.Parameters[0].In)
	})

	for _, version := range []string{"3.0.3", "3.1.0"} {
		t.Run("rejected at "+version, func(t *testing.T) {
			errs := structureErrors(t, version)

			var found bool
			for _, msg := range errs {
				if strings.Contains(msg, "querystring") && strings.Contains(msg, "not a valid parameter location") {
					found = true
					break
				}
			}
			assert.True(t, found,
				"querystring predates %s and must be reported as an invalid location; got: %v", version, errs)
		})
	}
}
