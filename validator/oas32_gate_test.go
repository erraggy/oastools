package validator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// gateMarker identifies an error raised by the version gate rather than by any
// other rule. Downgrading a 3.2 document produces plenty of unrelated errors, so
// the tests below select on this rather than on the total count.
const gateMarker = "was introduced in OpenAPI 3.2.0"

// gateErrors validates the document and returns the gate's errors, in the order
// reported. A nil parse error is not a clean parse — collected errors ride on
// Errors — so both are checked before the document reaches the validator.
func gateErrors(t *testing.T, spec string) []ValidationError {
	t.Helper()

	p := parser.New()
	parsed, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)
	require.Empty(t, parsed.Errors)

	result, err := New().ValidateParsed(*parsed)
	require.NoError(t, err)

	var gate []ValidationError
	for _, e := range result.Errors {
		if strings.Contains(e.Message, gateMarker) {
			gate = append(gate, e)
		}
	}
	return gate
}

// gateFields keys the gate's error paths by field.
func gateFields(t *testing.T, spec string) map[string][]string {
	t.Helper()

	found := make(map[string][]string)
	for _, e := range gateErrors(t, spec) {
		found[e.Field] = append(found[e.Field], e.Path)
	}
	return found
}

// gatePaths lists the gate's error paths, in the order reported.
func gatePaths(t *testing.T, spec string) []string {
	t.Helper()

	errs := gateErrors(t, spec)
	if len(errs) == 0 {
		return nil
	}
	paths := make([]string, 0, len(errs))
	for _, e := range errs {
		paths = append(paths, e.Path)
	}
	return paths
}

// TestOAS32FieldGate_ReportsEachField covers issue #411's inventory one field at a
// time, so a failure names the field that lost its gate rather than a count.
//
// Each case is a whole document because the gate reads the declared version, and
// the version is the thing under test.
func TestOAS32FieldGate_ReportsEachField(t *testing.T) {
	tests := []struct {
		name      string
		spec      string
		wantField string
		wantPath  string
	}{
		{
			name: "$self on the OpenAPI Object",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
$self: https://example.com/spec
paths: {}
`,
			wantField: "$self",
			wantPath:  "$self",
		},
		{
			name: "mediaTypes on the Components Object",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths: {}
components:
  mediaTypes:
    application/json:
      schema: {type: object}
`,
			wantField: "mediaTypes",
			wantPath:  "components.mediaTypes",
		},
		{
			name: "query on a Path Item",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    query:
      operationId: queryPets
      responses:
        "200": {description: OK}
`,
			wantField: "query",
			wantPath:  "paths./pets.query",
		},
		{
			name: "additionalOperations on a Path Item",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    additionalOperations:
      PURGE:
        operationId: purgePets
        responses:
          "200": {description: OK}
`,
			wantField: "additionalOperations",
			wantPath:  "paths./pets.additionalOperations",
		},
		{
			name: "summary on a Tag",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
tags:
  - name: pets
    summary: Pet operations
paths: {}
`,
			wantField: "summary",
			wantPath:  "tags[0].summary",
		},
		{
			name: "parent on a Tag",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
tags:
  - name: pets
    parent: root
paths: {}
`,
			wantField: "parent",
			wantPath:  "tags[0].parent",
		},
		{
			name: "kind on a Tag",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
tags:
  - name: pets
    kind: nav
paths: {}
`,
			wantField: "kind",
			wantPath:  "tags[0].kind",
		},
		{
			name: "name on a Server",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
servers:
  - url: https://api.example.com
    name: production
paths: {}
`,
			wantField: "name",
			wantPath:  "servers[0].name",
		},
		{
			// The one Server Object that is not part of a servers list. Missed by the
			// converter's detector too, which walks only the document's own servers.
			name: "name on a Server inside a Link",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          links:
            self:
              operationId: listPets
              server:
                url: https://api.example.com
                name: production
`,
			wantField: "name",
			wantPath:  "paths./pets.get.responses.200.links.self.server.name",
		},
		{
			name: "name on a Server inside a component Link",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths: {}
components:
  links:
    shared:
      operationId: listPets
      server:
        url: https://api.example.com
        name: staging
`,
			wantField: "name",
			wantPath:  "components.links.shared.server.name",
		},
		{
			name: "summary on a Response",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          summary: A pet
`,
			wantField: "summary",
			wantPath:  "paths./pets.get.responses.200.summary",
		},
		{
			name: "itemSchema on a Media Type",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              itemSchema: {type: object}
`,
			wantField: "itemSchema",
			wantPath:  "paths./pets.get.responses.200.content.application/json.itemSchema",
		},
		{
			name: "itemEncoding on a Media Type",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    post:
      operationId: addPet
      requestBody:
        content:
          multipart/mixed:
            itemEncoding: {contentType: application/json}
      responses:
        "200": {description: OK}
`,
			wantField: "itemEncoding",
			wantPath:  "paths./pets.post.requestBody.content.multipart/mixed.itemEncoding",
		},
		{
			name: "prefixEncoding on a Media Type",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    post:
      operationId: addPet
      requestBody:
        content:
          multipart/mixed:
            prefixEncoding:
              - contentType: application/json
      responses:
        "200": {description: OK}
`,
			wantField: "prefixEncoding",
			wantPath:  "paths./pets.post.requestBody.content.multipart/mixed.prefixEncoding",
		},
		{
			name: "encoding nested in an Encoding",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    post:
      operationId: addPet
      requestBody:
        content:
          multipart/form-data:
            encoding:
              profile:
                encoding:
                  avatar: {contentType: image/png}
      responses:
        "200": {description: OK}
`,
			wantField: "encoding",
			wantPath:  "paths./pets.post.requestBody.content.multipart/form-data.encoding.profile.encoding",
		},
		{
			name: "encoding nested inside a Media Type itemEncoding",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths: {}
components:
  requestBodies:
    body:
      content:
        multipart/mixed:
          itemEncoding:
            contentType: application/json
            encoding:
              nested: {contentType: image/png}
`,
			wantField: "encoding",
			wantPath:  "components.requestBodies.body.content.multipart/mixed.itemEncoding.encoding",
		},
		{
			name: "itemEncoding nested inside a Media Type prefixEncoding entry",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths: {}
components:
  requestBodies:
    body:
      content:
        multipart/mixed:
          prefixEncoding:
            - contentType: application/json
              itemEncoding: {contentType: image/png}
`,
			wantField: "itemEncoding",
			wantPath:  "components.requestBodies.body.content.multipart/mixed.prefixEncoding[0].itemEncoding",
		},
		{
			name: "dataValue on an Example",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths: {}
components:
  examples:
    pet:
      dataValue: {name: Fido}
`,
			wantField: "dataValue",
			wantPath:  "components.examples.pet.dataValue",
		},
		{
			name: "serializedValue on an Example",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths: {}
components:
  examples:
    pet:
      serializedValue: '{"name":"Fido"}'
`,
			wantField: "serializedValue",
			wantPath:  "components.examples.pet.serializedValue",
		},
		{
			name: "deprecated on a Security Scheme",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths: {}
components:
  securitySchemes:
    key:
      type: apiKey
      name: X-Key
      in: header
      deprecated: true
`,
			wantField: "deprecated",
			wantPath:  "components.securitySchemes.key.deprecated",
		},
		{
			name: "oauth2MetadataUrl on a Security Scheme",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths: {}
components:
  securitySchemes:
    key:
      type: apiKey
      name: X-Key
      in: header
      oauth2MetadataUrl: https://auth.example.com/meta
`,
			wantField: "oauth2MetadataUrl",
			wantPath:  "components.securitySchemes.key.oauth2MetadataUrl",
		},
		{
			name: "deviceAuthorization on OAuth Flows",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths: {}
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
        deviceAuthorization:
          tokenUrl: https://auth.example.com/token
          scopes: {read: Read}
`,
			wantField: "deviceAuthorization",
			wantPath:  "components.securitySchemes.oauth.flows.deviceAuthorization",
		},
		{
			name: "deviceAuthorizationUrl on an OAuth Flow",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths: {}
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
        deviceAuthorization:
          deviceAuthorizationUrl: https://auth.example.com/device
          tokenUrl: https://auth.example.com/token
          scopes: {read: Read}
`,
			wantField: "deviceAuthorizationUrl",
			wantPath:  "components.securitySchemes.oauth.flows.deviceAuthorization.deviceAuthorizationUrl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := gateFields(t, tt.spec)
			require.Contains(t, found, tt.wantField,
				"%s is an OAS 3.2 field but a 3.0.3 document using it reported nothing; reported: %v",
				tt.wantField, found)
			assert.Contains(t, found[tt.wantField], tt.wantPath,
				"the error should point at the field's own location")
			// Exactly once. Several of these field names also spell a field that is
			// legal before 3.2 — a Media Type's own `encoding`, a Tag's `name` — so a
			// membership check alone would pass a gate that reported the legal one too.
			assert.Len(t, found[tt.wantField], 1,
				"%s should be reported once, at %s; got %v", tt.wantField, tt.wantPath, found[tt.wantField])
		})
	}
}

// TestOAS32FieldGate_SilentAtAndAbove32 is the other half: at 3.2 these fields are
// legal, so the gate must not fire. Without this the suite would pass with a gate
// that reported the fields unconditionally.
func TestOAS32FieldGate_SilentAtAndAbove32(t *testing.T) {
	spec := `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
$self: https://example.com/spec
servers:
  - url: https://api.example.com
    name: production
tags:
  - name: pets
    summary: Pet operations
    parent: root
    kind: nav
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          summary: A pet
          content:
            application/json:
              itemSchema: {type: object}
`
	assert.Empty(t, gateFields(t, spec),
		"every field here is legal at 3.2.0, so none should be gated")
}

// TestOAS32FieldGate_AppliesAt31 pins the boundary. 3.1 is the version most likely
// to be mistaken for new enough, since it postdates most of the OAS 3.x surface.
func TestOAS32FieldGate_AppliesAt31(t *testing.T) {
	spec := `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
$self: https://example.com/spec
paths: {}
`
	assert.Contains(t, gateFields(t, spec), "$self",
		"$self arrived in 3.2, so a 3.1 document using it is still too early")
}

// TestOAS32FieldGate_PathItemFieldsWhereverPathItemsAppear covers the exception
// issue #411 calls out: `query` was reported only under `paths`, and
// `additionalOperations` nowhere. A path item appears in four places.
func TestOAS32FieldGate_PathItemFieldsWhereverPathItemsAppear(t *testing.T) {
	tests := []struct {
		name string
		// container is the YAML above the path item, ending with the key that owns
		// it. indent is how deep that path item's own fields sit.
		container string
		indent    int
		wantPath  string
	}{
		{
			name: "under paths",
			container: `
paths:
  /pets:`,
			indent:   4,
			wantPath: "paths./pets",
		},
		{
			name: "under webhooks",
			container: `
paths: {}
webhooks:
  petEvent:`,
			indent:   4,
			wantPath: "webhooks.petEvent",
		},
		{
			name: "under components.pathItems",
			container: `
paths: {}
components:
  pathItems:
    shared:`,
			indent:   6,
			wantPath: "components.pathItems.shared",
		},
		{
			name: "inside a component callback",
			container: `
paths: {}
components:
  callbacks:
    onEvent:
      '{$request.body#/url}':`,
			indent:   8,
			wantPath: "components.callbacks.onEvent.{$request.body#/url}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := "openapi: 3.1.0\ninfo: {title: T, version: \"1.0.0\"}" +
				tt.container + pathItemFields(tt.indent)

			found := gateFields(t, spec)
			assert.Contains(t, found["query"], tt.wantPath+".query",
				"query should be reported wherever a path item appears")
			assert.Contains(t, found["additionalOperations"], tt.wantPath+".additionalOperations",
				"additionalOperations should be reported wherever a path item appears")
		})
	}
}

// pathItemFields renders a path item's two OAS 3.2 operation fields at the given
// indent. Built from an indent rather than reindented from a fixed template: a
// template shifted by exact-whitespace replacement silently produced sibling keys
// instead of nested ones, and the test passed for the wrong reason until the
// paths were checked.
func pathItemFields(indent int) string {
	pad := strings.Repeat(" ", indent)
	return "\n" + pad + "query:" +
		"\n" + pad + "  operationId: itemQuery" +
		"\n" + pad + "  responses:" +
		"\n" + pad + `    "200": {description: OK}` +
		"\n" + pad + "additionalOperations:" +
		"\n" + pad + "  PURGE:" +
		"\n" + pad + "    operationId: itemPurge" +
		"\n" + pad + "    responses:" +
		"\n" + pad + `      "200": {description: OK}` + "\n"
}

// TestOAS32FieldGate_ReportsFieldsNestedInsideA32Operation covers the operations
// the gate reports as containers. Stopping at the container would hide a 3.2 field
// nested inside one.
func TestOAS32FieldGate_ReportsFieldsNestedInsideA32Operation(t *testing.T) {
	spec := `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    query:
      operationId: queryPets
      responses:
        "200":
          description: OK
          summary: Nested inside a query operation
`
	found := gateFields(t, spec)
	assert.Contains(t, found["query"], "paths./pets.query")
	assert.Contains(t, found["summary"], "paths./pets.query.responses.200.summary",
		"a 3.2 field inside a 3.2-only operation should be reported in its own right")
}

// TestOAS32GatePartitionsVersionsWithTraversal pins the two predicates against each
// other. They divide the version space: oas32TraversalApplies runs the rules that
// exist at 3.2, oas32FieldGateApplies reports the fields that do not exist before
// it. A version answering true or false to both would be checked twice or not at
// all.
func TestOAS32GatePartitionsVersionsWithTraversal(t *testing.T) {
	versions := []parser.OASVersion{
		parser.OASVersion20,
		parser.OASVersion300,
		parser.OASVersion303,
		parser.OASVersion304,
		parser.OASVersion310,
		parser.OASVersion311,
		parser.OASVersion312,
		parser.OASVersion320,
	}

	for _, version := range versions {
		t.Run(version.String(), func(t *testing.T) {
			assert.NotEqual(t, oas32TraversalApplies(version), oas32FieldGateApplies(version),
				"exactly one of the two 3.2 passes must claim %s", version)
		})
	}

	// An unclassifiable version is the deliberate exception: it is treated as new
	// enough for the rules, and never reported for using a field, since there is no
	// version to call the field newer than.
	var unknown parser.OASVersion
	require.False(t, unknown.IsValid(), "the zero version should not be valid")
	assert.True(t, oas32TraversalApplies(unknown))
	assert.False(t, oas32FieldGateApplies(unknown))
}

// TestOAS32FieldGate_ErrorOrderIsDeterministic pins the ordering.
//
// The walk ranges over maps, and Go randomizes that order, so the same document
// reported its errors in a different order on each run — six distinct orderings
// in eight runs of the eight-path document below. Anything diffing validator
// output across runs saw phantom changes.
//
// The gate sorts the errors it added rather than sorting each map's keys during
// the walk, so a clean document sorts an empty range and pays nothing.
func TestOAS32FieldGate_ErrorOrderIsDeterministic(t *testing.T) {
	var spec strings.Builder
	spec.WriteString("openapi: 3.0.3\ninfo: {title: T, version: \"1.0.0\"}\npaths:\n")
	for _, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		spec.WriteString("  /" + name + ":\n    query:\n      operationId: q" + name +
			"\n      responses:\n        \"200\": {description: OK}\n")
	}

	first := gatePaths(t, spec.String())
	require.Len(t, first, 8, "each path should report its query")

	// Repeated rather than compared against a hardcoded list: the assertion is that
	// the order does not move, not that it is any particular order.
	for range 12 {
		assert.Equal(t, first, gatePaths(t, spec.String()),
			"the same document must report its errors in the same order every run")
	}

	assert.IsIncreasing(t, first, "sorted by path, so the order is predictable and not merely stable")
}

// TestOAS32FieldGate_BoundsCyclicPathItems covers the one graph the walk cannot
// treat as a tree. A Callback Object holds path items, whose operations hold
// callbacks, so the two can point at each other.
//
// A parsed document cannot build that cycle, but ValidateParsed takes a document
// from the caller, and before the bound this recursed until the stack gave out.
// Constructed by hand because that is precisely the case that reaches it.
func TestOAS32FieldGate_BoundsCyclicPathItems(t *testing.T) {
	item := &parser.PathItem{}
	callback := parser.Callback{"loop": item}
	item.Get = &parser.Operation{
		Callbacks: map[string]*parser.Callback{"cycle": &callback},
	}
	// A 3.2 field, so each pass round the loop has something to report and the
	// bound is observable rather than merely survived.
	item.Query = &parser.Operation{}

	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		Info:       &parser.Info{Title: "T", Version: "1.0.0"},
		Paths:      parser.Paths{"/a": item},
		OASVersion: parser.OASVersion303,
	}

	result := &ValidationResult{}
	require.NotPanics(t, func() {
		New().validateOAS32FieldsNotYetIntroduced(doc, result)
	})
	assert.Len(t, result.Errors, maxPathItemNestingDepth,
		"the walk should stop at the nesting bound, having reported once per level")
}
