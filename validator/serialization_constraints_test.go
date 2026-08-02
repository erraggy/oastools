package validator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// TestHeaderNameMustBeToken covers the RFC 9110 token constraint on header
// names. OAS 3.2 introduced it (the `token` definition does not exist in the
// 3.1 schema), so a 3.1 document with the same name must still validate clean.
func TestHeaderNameMustBeToken(t *testing.T) {
	const wantMsg = "is not a valid HTTP field name"

	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{
			name: "3.2 rejects a component header name with an illegal character",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  headers:
    'Bad=Header':
      schema: {}
`,
			wantErr: true,
		},
		{
			name: "3.2 rejects a header parameter name with an illegal character",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  parameters:
    BadHeader:
      name: 'Bad[Header]'
      in: header
      schema: {}
`,
			wantErr: true,
		},
		{
			name: "3.2 accepts the token punctuation RFC 9110 allows",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  headers:
    "X-Rate-Limit_v2.1":
      schema: {}
`,
		},
		{
			// The rule is 3.2+. Enforcing it at 3.1 would reject documents that
			// version's own schema considers valid.
			name: "3.1 accepts a name 3.2 would reject",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
components:
  headers:
    'Bad=Header':
      schema: {}
`,
		},
		{
			// A query parameter's name is not a field name, so the token rule
			// does not reach it.
			name: "3.2 does not constrain a query parameter name",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  parameters:
    Weird:
      name: 'not[a]token'
      in: query
      schema: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateSpec(t, tt.spec)
			assert.Equal(t, tt.wantErr, resultHasMessage(result, wantMsg),
				"header-name error presence; errors: %v", result.Errors)
		})
	}
}

// TestAllowReservedPlacement covers where `allowReserved` may appear. The
// permitted set widened in 3.2, so the rule is version-scoped in both
// directions: enforcing 3.1's rule everywhere would reject valid 3.2 documents,
// and enforcing 3.2's would accept invalid 3.1 ones.
//
// Each case carries its own document rather than assembling one from fragments,
// so the `in` and `style` under test sit in the document at the indentation
// they are actually read from.
func TestAllowReservedPlacement(t *testing.T) {
	const wantMsg = "allowReserved is not permitted"

	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		// 3.2 widened the permitted set to the `in` and `style` combinations
		// that percent-encode: in: path, in: query, and in: cookie with
		// style: form.
		{
			name: "3.2 permits allowReserved on a query parameter",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  parameters:
    p:
      name: p
      in: query
      allowReserved: true
      schema: {}
`,
		},
		{
			name: "3.2 permits allowReserved on a path parameter",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  parameters:
    p:
      name: p
      in: path
      required: true
      allowReserved: true
      schema: {}
`,
		},
		{
			// form is the default style for a cookie parameter, so an unset
			// style is the permitted case rather than the forbidden one.
			name: "3.2 permits allowReserved on a cookie parameter with the default style",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  parameters:
    p:
      name: p
      in: cookie
      allowReserved: true
      schema: {}
`,
		},
		{
			name: "3.2 permits allowReserved on a cookie parameter with style form",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  parameters:
    p:
      name: p
      in: cookie
      style: form
      allowReserved: true
      schema: {}
`,
		},
		{
			name: "3.2 rejects allowReserved on a cookie parameter with style cookie",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  parameters:
    p:
      name: p
      in: cookie
      style: cookie
      allowReserved: true
      schema: {}
`,
			wantErr: true,
		},
		{
			name: "3.2 rejects allowReserved on a header parameter",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  parameters:
    p:
      name: p
      in: header
      allowReserved: true
      schema: {}
`,
			wantErr: true,
		},

		// 3.1 permits it on query parameters only.
		{
			name: "3.1 permits allowReserved on a query parameter",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
components:
  parameters:
    p:
      name: p
      in: query
      allowReserved: true
      schema: {}
`,
		},
		{
			name: "3.1 rejects allowReserved on a path parameter",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
components:
  parameters:
    p:
      name: p
      in: path
      required: true
      allowReserved: true
      schema: {}
`,
			wantErr: true,
		},
		{
			name: "3.1 rejects allowReserved on a cookie parameter",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
components:
  parameters:
    p:
      name: p
      in: cookie
      allowReserved: true
      schema: {}
`,
			wantErr: true,
		},
		{
			name: "3.1 rejects allowReserved on a header parameter",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
components:
  parameters:
    p:
      name: p
      in: header
      allowReserved: true
      schema: {}
`,
			wantErr: true,
		},

		{
			// 3.0's schema lists allowReserved as a plain Parameter property
			// with no conditional, and its prose describes effect rather than
			// validity, so placement is not enforced there.
			name: "3.0 does not enforce placement",
			spec: `
openapi: 3.0.3
info:
  title: API
  version: 1.0.0
paths: {}
components:
  parameters:
    p:
      name: p
      in: header
      allowReserved: true
      schema: {}
`,
		},

		{
			name: "3.2 rejects allowReserved on a Header Object",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  headers:
    Style:
      schema:
        type: array
      style: simple
      explode: true
      allowReserved: true
`,
			wantErr: true,
		},
		{
			name: "3.1 also rejects allowReserved on a Header Object",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
components:
  headers:
    Style:
      schema:
        type: array
      allowReserved: true
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateSpec(t, tt.spec)
			assert.Equal(t, tt.wantErr, resultHasMessage(result, wantMsg),
				"allowReserved error presence; errors: %v", result.Errors)
		})
	}
}

// TestServerVariableEnumMustNotBeEmpty covers the `minItems: 1` OAS 3.1 added to
// the Server Variable Object's enum. An empty enum permits no value at all, so
// not even the required default could satisfy it.
//
// Presence is the test, not length: an absent enum means "any value". The parser
// keeps the two apart, and the difference between `enum: []`, an absent enum
// line, and a populated block is visible in each document below rather than
// hidden in an escaped fragment.
func TestServerVariableEnumMustNotBeEmpty(t *testing.T) {
	const wantMsg = "Server variable enum must not be empty"

	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{
			name: "3.2 rejects an empty enum",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
servers:
  - url: https://example.com/{var}
    variables:
      var:
        enum: []
        default: a
paths: {}
`,
			wantErr: true,
		},
		{
			name: "3.1 rejects an empty enum",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
servers:
  - url: https://example.com/{var}
    variables:
      var:
        enum: []
        default: a
paths: {}
`,
			wantErr: true,
		},
		{
			// 3.0's schema has no minItems on the enum.
			name: "3.0 does not enforce it",
			spec: `
openapi: 3.0.3
info:
  title: API
  version: 1.0.0
servers:
  - url: https://example.com/{var}
    variables:
      var:
        enum: []
        default: a
paths: {}
`,
		},
		{
			name: "an absent enum is fine",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
servers:
  - url: https://example.com/{var}
    variables:
      var:
        default: a
paths: {}
`,
		},
		{
			name: "a populated enum is fine",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
servers:
  - url: https://example.com/{var}
    variables:
      var:
        enum:
          - a
        default: a
paths: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateSpec(t, tt.spec)
			assert.Equal(t, tt.wantErr, resultHasMessage(result, wantMsg),
				"empty-enum error presence; errors: %v", result.Errors)
		})
	}
}

// TestSpecRefTracksDocumentVersion pins the citation standard: a rule whose
// applicability varies by version must point at the version the document is
// actually being held to. Citing 3.2 at a 3.1 document describes a rule that
// document is not subject to.
func TestSpecRefTracksDocumentVersion(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantRef string
	}{
		{
			name: "3.1",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
components:
  parameters:
    p:
      name: p
      in: header
      allowReserved: true
      schema: {}
`,
			wantRef: "https://spec.openapis.org/oas/v3.1.0.html#parameter-object",
		},
		{
			name: "3.2",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  parameters:
    p:
      name: p
      in: header
      allowReserved: true
      schema: {}
`,
			wantRef: "https://spec.openapis.org/oas/v3.2.0.html#parameter-object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateSpec(t, tt.spec)

			var refs []string
			for _, e := range result.Errors {
				if strings.Contains(e.Message, "allowReserved is not permitted") {
					refs = append(refs, e.SpecRef)
				}
			}
			require.Len(t, refs, 1, "expected exactly one allowReserved error")
			assert.Equal(t, tt.wantRef, refs[0])
		})
	}
}

// TestServerVariableEnumEmptyInJSON covers the JSON decode path. The rule turns
// on nil versus empty, which is decode behaviour, and parser keeps separate YAML
// and JSON implementations: a YAML-only test covers half the surface.
func TestServerVariableEnumEmptyInJSON(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{
			name: "empty enum",
			spec: `{
  "openapi": "3.2.0",
  "info": {"title": "API", "version": "1.0.0"},
  "servers": [
    {"url": "https://example.com/{var}", "variables": {"var": {"enum": [], "default": "a"}}}
  ],
  "paths": {}
}`,
			wantErr: true,
		},
		{
			name: "absent enum",
			spec: `{
  "openapi": "3.2.0",
  "info": {"title": "API", "version": "1.0.0"},
  "servers": [
    {"url": "https://example.com/{var}", "variables": {"var": {"default": "a"}}}
  ],
  "paths": {}
}`,
		},
		{
			name: "populated enum",
			spec: `{
  "openapi": "3.2.0",
  "info": {"title": "API", "version": "1.0.0"},
  "servers": [
    {"url": "https://example.com/{var}", "variables": {"var": {"enum": ["a"], "default": "a"}}}
  ],
  "paths": {}
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateSpec(t, tt.spec)
			assert.Equal(t, tt.wantErr, resultHasMessage(result, "Server variable enum must not be empty"),
				"empty-enum error presence; errors: %v", result.Errors)
		})
	}
}

// TestServerVariableEmptyEnumErrorDetail checks the error a caller actually
// receives, not just that one was raised: the path names the variable, the field
// names the offending key, and the citation points at the document's version.
func TestServerVariableEmptyEnumErrorDetail(t *testing.T) {
	spec := `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
servers:
  - url: https://example.com/{var}
    variables:
      var:
        enum: []
        default: a
paths: {}
`
	result := validateSpec(t, spec)

	var found *ValidationError
	for i, e := range result.Errors {
		if strings.Contains(e.Message, "Server variable enum must not be empty") {
			found = &result.Errors[i]
			break
		}
	}
	require.NotNil(t, found, "expected an empty-enum error; errors: %v", result.Errors)

	assert.Equal(t, "servers[0].variables.var", found.Path)
	assert.Equal(t, "enum", found.Field)
	assert.Equal(t, "https://spec.openapis.org/oas/v3.1.0.html#server-variable-object", found.SpecRef)
}

// TestHeaderRulesReachEveryPosition is the reachability guard for the rules this
// file adds. They hook into the structural traversal rather than into individual
// call sites, so a Header Object is checked wherever it occurs. See #423 for the
// case where a rule was correct but never ran in most of those positions.
func TestHeaderRulesReachEveryPosition(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantMsg string
	}{
		{
			name:    "components.headers",
			wantMsg: "is not a valid HTTP field name",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  headers:
    'Bad=Header':
      schema: {}
`,
		},
		{
			name:    "response headers on an inline path",
			wantMsg: "is not a valid HTTP field name",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
paths:
  /x:
    get:
      responses:
        "200":
          description: ok
          headers:
            'Bad=Header':
              schema: {}
`,
		},
		{
			name:    "components.responses headers",
			wantMsg: "is not a valid HTTP field name",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  responses:
    R:
      description: ok
      headers:
        'Bad=Header':
          schema: {}
`,
		},
		{
			name:    "encoding headers",
			wantMsg: "is not a valid HTTP field name",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
paths:
  /x:
    post:
      requestBody:
        content:
          multipart/form-data:
            schema:
              type: object
            encoding:
              part:
                headers:
                  'Bad=Header':
                    schema: {}
      responses:
        "200":
          description: ok
`,
		},
		{
			name:    "operation parameter",
			wantMsg: "allowReserved is not permitted",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
paths:
  /x:
    get:
      parameters:
        - name: h
          in: header
          allowReserved: true
          schema: {}
      responses:
        "200":
          description: ok
`,
		},
		{
			name:    "path-item parameter",
			wantMsg: "is not a valid HTTP field name",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
paths:
  /x:
    parameters:
      - name: 'Bad[Header]'
        in: header
        schema: {}
    get:
      responses:
        "200":
          description: ok
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateSpec(t, tt.spec)
			assert.True(t, resultHasMessage(result, tt.wantMsg),
				"the rule did not reach this position; errors: %v", result.Errors)
		})
	}
}

// TestVersionGatesTreatUnrecognizedVersionsAsInScope pins the convention shared
// by every version gate here and by oas3TraversalApplies: a constraint
// introduced at a threshold is assumed to hold in later versions this build does
// not yet recognize.
//
// The alternative, skipping the rule, means a document oastools cannot classify
// is held to fewer rules than one it can. The gates are written separately, so
// this pins them to one answer.
func TestVersionGatesTreatUnrecognizedVersionsAsInScope(t *testing.T) {
	unrecognized := parser.OASVersion(0)
	require.False(t, unrecognized.IsValid(), "the test needs a version this build does not know")

	t.Run("header name rule", func(t *testing.T) {
		assert.True(t, headerNameRulesApply(unrecognized))
	})

	t.Run("empty server enum rule", func(t *testing.T) {
		assert.True(t, emptyServerEnumApplies(unrecognized))
	})

	t.Run("allowReserved on a parameter uses the newest table", func(t *testing.T) {
		// The permitted set widened in 3.2, so "in scope" means the 3.2 table
		// rather than 3.1's narrower one.
		assert.True(t, allowReservedPermitted(unrecognized, parser.ParamInPath, ""),
			"3.2 permits allowReserved on a path parameter")
		assert.False(t, allowReservedPermitted(unrecognized, parser.ParamInHeader, ""),
			"no version permits allowReserved on a header parameter")
	})

	t.Run("allowReserved on a Header Object", func(t *testing.T) {
		v := &Validator{oasVersion: unrecognized}
		result := &ValidationResult{}
		header := &parser.Header{Extra: map[string]any{"allowReserved": true}}
		v.validateHeaderAllowReserved(header, "components.headers.X", result)
		assert.True(t, resultHasMessage(result, "allowReserved is not permitted"))
	})
}
