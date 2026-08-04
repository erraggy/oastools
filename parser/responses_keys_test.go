package parser

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "go.yaml.in/yaml/v4"
)

// responsesOf returns the Responses object of GET /a, which every fixture in
// this file declares.
func responsesOf(t *testing.T, res *ParseResult) *Responses {
	t.Helper()
	doc, ok := res.Document.(*OAS3Document)
	require.True(t, ok, "expected an OAS3Document, got %T", res.Document)
	require.Contains(t, doc.Paths, "/a")
	require.NotNil(t, doc.Paths["/a"].Get)
	require.NotNil(t, doc.Paths["/a"].Get.Responses)
	return doc.Paths["/a"].Get.Responses
}

// TestResponsesKeyClassification pins how each of the three decode paths sorts
// the keys of a Responses Object into Default, Codes and Extra.
//
// All three are exercised because they are separate implementations: YAML in
// paths.go, JSON in paths_json.go, and decodeFromMap in the generated decoder,
// which runs only under ResolveRefs. Covering one would leave the other two
// free to drift.
func TestResponsesKeyClassification(t *testing.T) {
	const specYAML = `openapi: 3.0.3
info:
  title: T
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
      responses:
        default:
          description: fallback
        "200":
          description: OK
        "4XX":
          description: client error
        x-object-ext:
          description: an extension whose value is an object
        x-scalar-ext: 100
`

	const specJSON = `{
  "openapi": "3.0.3",
  "info": {"title": "T", "version": "1.0.0"},
  "paths": {
    "/a": {
      "get": {
        "operationId": "a",
        "responses": {
          "default": {"description": "fallback"},
          "200": {"description": "OK"},
          "4XX": {"description": "client error"},
          "x-object-ext": {"description": "an extension whose value is an object"},
          "x-scalar-ext": 100
        }
      }
    }
  }
}`

	tests := []struct {
		name        string
		spec        string
		resolveRefs bool
	}{
		{name: "yaml", spec: specYAML},
		{name: "json", spec: specJSON},
		// ResolveRefs routes the document through decodeFromMap instead of the
		// format-specific decoders.
		{name: "yaml via decodeFromMap", spec: specYAML, resolveRefs: true},
		{name: "json via decodeFromMap", spec: specJSON, resolveRefs: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			p.ResolveRefs = tt.resolveRefs
			res, err := p.ParseBytes([]byte(tt.spec))
			require.NoError(t, err)
			require.Empty(t, res.Errors)

			r := responsesOf(t, res)

			// The default response has its own field.
			require.NotNil(t, r.Default)
			assert.Equal(t, "fallback", r.Default.Description)
			assert.NotContains(t, r.Codes, "default")

			// Status codes and wildcard ranges are the only Codes entries.
			assert.Len(t, r.Codes, 2)
			require.Contains(t, r.Codes, "200")
			require.Contains(t, r.Codes, "4XX")
			assert.Equal(t, "OK", r.Codes["200"].Description)
			assert.Equal(t, "client error", r.Codes["4XX"].Description)

			// Extensions are held apart from the status codes, whatever the
			// shape of their value. A scalar extension is as legal as an
			// object one, and neither is a response.
			assert.NotContains(t, r.Codes, "x-object-ext")
			assert.NotContains(t, r.Codes, "x-scalar-ext")
			assert.Contains(t, r.Extra, "x-object-ext")
			assert.Contains(t, r.Extra, "x-scalar-ext")
		})
	}
}

// TestResponsesInvalidStatusCodeIsReportedOnEveryDecodePath asserts that all
// three decoders agree about an invalid status code: each keeps the key, and
// validateStructure reports it into ParseResult.Errors.
//
// Agreement is the point (#449). While the YAML and JSON decoders failed the
// parse and decodeFromMap did not, one document had two verdicts depending on
// which decoder read it, which is a property of the caller rather than of the
// document. The table below runs the same two documents through every path,
// so a decoder that starts rejecting again fails here.
func TestResponsesInvalidStatusCodeIsReportedOnEveryDecodePath(t *testing.T) {
	const specYAML = `openapi: 3.0.3
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
        "999":
          description: not a status code
`

	const specJSON = `{
  "openapi": "3.0.3",
  "info": {"title": "T", "version": "1.0.0"},
  "paths": {
    "/a": {
      "get": {
        "operationId": "a",
        "responses": {
          "200": {"description": "OK"},
          "999": {"description": "not a status code"}
        }
      }
    }
  }
}`

	const want = "invalid status code '999'"

	for _, tt := range []struct {
		name        string
		spec        string
		resolveRefs bool
	}{
		{name: "yaml", spec: specYAML},
		{name: "json", spec: specJSON},
		// ResolveRefs selects decodeFromMap over the format decoders.
		{name: "yaml via decodeFromMap", spec: specYAML, resolveRefs: true},
		{name: "json via decodeFromMap", spec: specJSON, resolveRefs: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			p.ResolveRefs = tt.resolveRefs
			res, err := p.ParseBytes([]byte(tt.spec))

			// The document decodes: an unusable status code is a structural
			// fault, not a reason to refuse the bytes.
			require.NoError(t, err)
			assert.True(t, hasErrorContaining(res.Errors, want),
				"want a collected error containing %q; got %v", want, res.Errors)

			// The response itself survives. Reporting the key while discarding
			// its value would be a quieter form of the same data loss.
			r := responsesOf(t, res)
			require.Contains(t, r.Codes, "999")
			assert.Equal(t, "not a status code", r.Codes["999"].Description)
		})
	}
}

// TestParseBytesDoesNotFailOnAStructuralFault pins the contract change #449's
// last criterion required, at the level a caller sees it.
//
// ParseBytes returns a non-nil error when the bytes cannot be decoded, not when
// the decoded document breaks a rule. A caller that treated any invalid document
// as a parse failure must read ParseResult.Errors instead.
func TestParseBytesDoesNotFailOnAStructuralFault(t *testing.T) {
	const spec = `openapi: 3.0.3
info:
  title: T
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
      responses:
        "999":
          description: not a status code
`

	res, err := New().ParseBytes([]byte(spec))
	require.NoError(t, err, "a structural fault is reported, not returned")
	require.NotNil(t, res.Document, "and the document is still handed back")
	assert.NotEmpty(t, res.Errors)

	// The counterpart: bytes that are not YAML at all still fail outright, so
	// this is a narrowing of what the error channel carries rather than its
	// removal.
	_, err = New().ParseBytes([]byte("openapi: 3.0.3\n\tinfo: broken"))
	assert.Error(t, err, "malformed input must still fail the parse")
}

// TestStructureValidationOffReportsNothingAtParseTime documents the one place
// this reports nothing: a caller that turns off structure validation gets no
// parse-time diagnostic, because that is what the flag switches off.
//
// What it asserts here is that the key survives anyway, which is what leaves
// something for the validator to find. That the validator does find it is
// asserted by validator.TestValidatorReportsStatusCodeWhenStructureValidationIsOff,
// since validator imports parser and this package cannot import it back.
func TestStructureValidationOffReportsNothingAtParseTime(t *testing.T) {
	const spec = `openapi: 3.0.3
info:
  title: T
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
      responses:
        "999":
          description: not a status code
`

	p := New()
	p.ValidateStructure = false
	res, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)
	assert.Empty(t, res.Errors, "structure validation is off, so nothing reports here")

	// The key is still present, which is what lets the validator find it.
	assert.Contains(t, responsesOf(t, res).Codes, "999")
}

// TestResponsesInvalidKeyWithNonObjectValueIsStillReported covers an invalid
// status code whose value is not a Response Object either, which is two faults
// in one entry. decodeFromMap classifies the key before it looks at the value,
// so the reportable fault is not lost to the unreportable one.
//
// Reachable only under ResolveRefs: the YAML and JSON decoders reject the key
// outright, before its value matters.
func TestResponsesInvalidKeyWithNonObjectValueIsStillReported(t *testing.T) {
	const specYAML = `openapi: 3.0.3
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
        "999": 100
`

	const specJSON = `{
  "openapi": "3.0.3",
  "info": {"title": "T", "version": "1.0.0"},
  "paths": {
    "/a": {
      "get": {
        "operationId": "a",
        "responses": {
          "200": {"description": "OK"},
          "999": 100
        }
      }
    }
  }
}`

	for _, tt := range []struct {
		name string
		spec string
	}{
		{name: "yaml", spec: specYAML},
		{name: "json", spec: specJSON},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			p.ResolveRefs = true
			res, err := p.ParseBytes([]byte(tt.spec))
			require.NoError(t, err)

			assert.True(t, hasErrorContaining(res.Errors, "invalid status code '999'"),
				"a scalar value must not hide the invalid key; got %v", res.Errors)

			r := responsesOf(t, res)
			assert.Contains(t, r.Codes, "999")
			assert.Contains(t, r.Codes, "200")
		})
	}
}

// TestResponsesWellFormedCodeWithNonObjectValueIsNotInvented is the counterpart:
// a valid status code whose value is not an object is left out rather than
// turned into an empty Response. Nothing here validates the value, so keeping
// the key would report a response the document does not declare.
func TestResponsesWellFormedCodeWithNonObjectValueIsNotInvented(t *testing.T) {
	const spec = `openapi: 3.0.3
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
        "404": 100
`

	p := New()
	p.ResolveRefs = true
	res, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	r := responsesOf(t, res)
	assert.Contains(t, r.Codes, "200")
	assert.NotContains(t, r.Codes, "404")
}

// TestResponsesDecodeFromMapResetsDefault covers the third decode path for the
// same reuse question the format decoders answer: a value that already holds a
// default response must not keep it when the next map declares none.
func TestResponsesDecodeFromMapResetsDefault(t *testing.T) {
	const withDefault = `openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths:
  /a:
    get:
      operationId: a
      responses:
        default: {description: fallback}
        "200": {description: OK}
`
	const withoutDefault = `openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths:
  /a:
    get:
      operationId: a
      responses:
        "200": {description: OK}
`

	p := New()
	p.ResolveRefs = true

	first, err := p.ParseBytes([]byte(withDefault))
	require.NoError(t, err)
	require.NotNil(t, responsesOf(t, first).Default)

	// Reuse the same value, which is what a caller pooling documents does.
	target := responsesOf(t, first)
	second, err := p.ParseBytes([]byte(withoutDefault))
	require.NoError(t, err)
	source := responsesOf(t, second)
	assert.Nil(t, source.Default, "the second document declares no default")

	// And directly, since the pooled case above depends on the parser not
	// reusing the value at all. Every field is checked, not just Default: a
	// reset that cleared one and not the others would still merge documents.
	require.NotNil(t, target.Default, "precondition: the value holds a default")
	target.Extra = map[string]any{"x-note": "carried over?"}
	target.Codes["404"] = &Response{Description: "Not Found"}

	target.decodeFromMap(map[string]any{"200": map[string]any{"description": "OK"}})

	assert.Nil(t, target.Default, "decodeFromMap must clear a default it does not find")
	assert.NotContains(t, target.Extra, "x-note", "and an extension it does not find")
	assert.NotContains(t, target.Codes, "404", "and a status code it does not find")
	assert.Contains(t, target.Codes, "200")
}

// TestResponsesDeepCopyIsolatesExtensions asserts that a copied Responses does
// not share the inside of its extensions with the original. An extension value
// is arbitrary JSON, so it can nest maps and slices, and a copy that duplicated
// only the outer map would let a mutation reach across.
func TestResponsesDeepCopyIsolatesExtensions(t *testing.T) {
	original := &Responses{
		Default: &Response{Description: "fallback"},
		Codes:   map[string]*Response{"200": {Description: "OK"}},
		Extra: map[string]any{
			"x-nested": map[string]any{"inner": "before"},
			"x-list":   []any{"first"},
		},
	}

	copied := original.DeepCopy()
	require.NotNil(t, copied)

	nested, ok := copied.Extra["x-nested"].(map[string]any)
	require.True(t, ok, "expected a nested map, got %T", copied.Extra["x-nested"])
	nested["inner"] = "after"

	list, ok := copied.Extra["x-list"].([]any)
	require.True(t, ok, "expected a nested slice, got %T", copied.Extra["x-list"])
	list[0] = "changed"

	assert.Equal(t, map[string]any{"inner": "before"}, original.Extra["x-nested"],
		"mutating the copy must not reach the original")
	assert.Equal(t, []any{"first"}, original.Extra["x-list"],
		"mutating the copy must not reach the original")
}

// TestResponsesRoundTripPreservesExtensions asserts that splitting extensions
// out of Codes does not lose them on the way back out, and that both marshalers
// emit them in the one object the specification describes.
func TestResponsesRoundTripPreservesExtensions(t *testing.T) {
	const spec = `openapi: 3.0.3
info:
  title: T
  version: "1.0.0"
paths:
  /a:
    get:
      operationId: a
      responses:
        default:
          description: fallback
        "200":
          description: OK
        x-object-ext:
          description: an extension
        x-scalar-ext: 100
`

	res, err := New().ParseBytes([]byte(spec))
	require.NoError(t, err)
	doc, ok := res.Document.(*OAS3Document)
	require.True(t, ok)

	assertRoundTripped := func(t *testing.T, r *Responses) {
		t.Helper()
		require.NotNil(t, r.Default)
		assert.Equal(t, "fallback", r.Default.Description)
		require.Contains(t, r.Codes, "200")
		assert.Equal(t, "OK", r.Codes["200"].Description)

		// The values are asserted, not just the keys: an extension that
		// survives under the right name with the wrong content has still
		// been lost, and a key-only check cannot tell the difference.
		require.Contains(t, r.Extra, "x-object-ext")
		assert.Equal(t, map[string]any{"description": "an extension"}, r.Extra["x-object-ext"])
		require.Contains(t, r.Extra, "x-scalar-ext")
		assert.EqualValues(t, 100, r.Extra["x-scalar-ext"])

		assert.NotContains(t, r.Codes, "x-object-ext")
		assert.NotContains(t, r.Codes, "x-scalar-ext")
	}

	t.Run("yaml", func(t *testing.T) {
		out, err := yaml.Marshal(doc)
		require.NoError(t, err)

		reparsed, err := New().ParseBytes(out)
		require.NoError(t, err)
		assertRoundTripped(t, responsesOf(t, reparsed))
	})

	// The two marshalers are separate implementations, so a merge present in
	// one says nothing about the other.
	t.Run("json", func(t *testing.T) {
		out, err := json.Marshal(doc)
		require.NoError(t, err)

		reparsed, err := New().ParseBytes(out)
		require.NoError(t, err)
		assertRoundTripped(t, responsesOf(t, reparsed))
	})
}

// TestResponsesMarshalResolvesDefaultOnce covers a caller-assembled document
// holding `default` both in its own field and as a map entry. Emitting it twice
// produces a mapping that does not parse back, so the key is resolved to one
// value, and both marshalers resolve it the same way.
func TestResponsesMarshalResolvesDefaultOnce(t *testing.T) {
	tests := []struct {
		name string
		in   *Responses
		want string
	}{
		{
			name: "Codes wins over the Default field",
			in: &Responses{
				Default: &Response{Description: "from Default"},
				Codes:   map[string]*Response{"default": {Description: "from Codes"}},
			},
			want: "from Codes",
		},
		{
			name: "the Default field wins over Extra",
			in: &Responses{
				Default: &Response{Description: "from Default"},
				Extra:   map[string]any{"default": map[string]any{"description": "from Extra"}},
			},
			want: "from Default",
		},
		{
			name: "Extra supplies it when nothing else does",
			in: &Responses{
				Extra: map[string]any{"default": map[string]any{"description": "from Extra"}},
			},
			want: "from Extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := yaml.Marshal(tt.in)
			require.NoError(t, err)
			assert.Equal(t, 1, strings.Count(string(out), "default:"),
				"want exactly one default key; got:\n%s", out)
			assert.Contains(t, string(out), tt.want)

			// The duplicate this guards against is only visible on the way
			// back in: a repeated mapping key is a decode error.
			var back Responses
			require.NoError(t, yaml.Unmarshal(out, &back), "emitted YAML must parse back")

			encoded, err := tt.in.MarshalJSON()
			require.NoError(t, err)
			var decoded map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(encoded, &decoded))
			require.Contains(t, decoded, "default")
			assert.Contains(t, string(decoded["default"]), tt.want,
				"both marshalers must resolve `default` the same way")
		})
	}
}

// TestResponsesDecodeIntoReusedValueResets covers decoding a second document
// into a value that already holds one. Every field describes the document just
// decoded, so none may carry anything forward from the previous one: not an
// extension, not a status code, and not the default response.
func TestResponsesDecodeIntoReusedValueResets(t *testing.T) {
	const populatedYAML = `
"200": {description: OK}
"404": {description: Not Found}
default: {description: fallback}
x-note: carried over?
`
	const sparseYAML = `
"200": {description: OK}
`

	const populatedJSON = `{
  "200": {"description": "OK"},
  "404": {"description": "Not Found"},
  "default": {"description": "fallback"},
  "x-note": "carried over?"
}`
	const sparseJSON = `{"200": {"description": "OK"}}`

	assertOnlySparse := func(t *testing.T, r *Responses) {
		t.Helper()
		assert.NotContains(t, r.Extra, "x-note",
			"the second document declares no extension, so none may survive")
		assert.NotContains(t, r.Codes, "404",
			"the second document declares no 404, so none may survive")
		assert.Nil(t, r.Default,
			"the second document declares no default, so none may survive")
		assert.Contains(t, r.Codes, "200")
	}

	t.Run("yaml", func(t *testing.T) {
		var r Responses
		require.NoError(t, yaml.Unmarshal([]byte(populatedYAML), &r))
		require.Contains(t, r.Extra, "x-note")
		require.NotNil(t, r.Default)

		require.NoError(t, yaml.Unmarshal([]byte(sparseYAML), &r))
		assertOnlySparse(t, &r)
	})

	t.Run("json", func(t *testing.T) {
		var r Responses
		require.NoError(t, json.Unmarshal([]byte(populatedJSON), &r))
		require.Contains(t, r.Extra, "x-note")
		require.NotNil(t, r.Default)

		require.NoError(t, json.Unmarshal([]byte(sparseJSON), &r))
		assertOnlySparse(t, &r)
	})
}

// TestResponsesMarshalYAMLKeyOrder pins the emitted key order: `default` first,
// then the status codes and extensions together in sorted order.
//
// The order is asserted rather than left to map iteration because a result that
// varies between runs of the same binary makes any diff of validator or
// converter output useless (#425).
func TestResponsesMarshalYAMLKeyOrder(t *testing.T) {
	r := &Responses{
		Default: &Response{Description: "fallback"},
		Codes: map[string]*Response{
			"500": {Description: "server error"},
			"200": {Description: "OK"},
			"4XX": {Description: "client error"},
		},
		Extra: map[string]any{
			"x-note":  "an extension",
			"x-first": "another",
		},
	}

	// Quoting is the YAML encoder's own: a key that would otherwise read as an
	// integer is quoted, and 4XX cannot, so it is not.
	const want = `default:
    description: fallback
"200":
    description: OK
4XX:
    description: client error
"500":
    description: server error
x-first: another
x-note: an extension
`

	// Repeated to catch an ordering that depends on map iteration: a single
	// run can agree with the expectation by luck.
	for range 8 {
		out, err := yaml.Marshal(r)
		require.NoError(t, err)
		assert.Equal(t, want, string(out))
	}
}

// TestResponsesMarshalPrefersCodesOnAClash covers a document assembled in Go
// rather than parsed: no decode path can put one key in both maps, so the
// only way to reach this is to build it. The key must still be emitted once,
// or the output is not a valid mapping.
func TestResponsesMarshalPrefersCodesOnAClash(t *testing.T) {
	r := &Responses{
		Codes: map[string]*Response{"x-clash": {Description: "from codes"}},
		Extra: map[string]any{"x-clash": "from extra"},
	}

	out, err := yaml.Marshal(r)
	require.NoError(t, err)
	assert.Equal(t, "x-clash:\n    description: from codes\n", string(out))

	j, err := r.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"x-clash":{"description":"from codes"}}`, string(j))
}

// TestResponsesEqualityObservesExtensions pins Extra as part of a Responses
// value's identity. Two documents differing only in a response extension are
// different documents, and a comparison that ignored Extra would report them
// equal.
func TestResponsesEqualityObservesExtensions(t *testing.T) {
	base := func() *Responses {
		return &Responses{
			Default: &Response{Description: "fallback"},
			Codes:   map[string]*Response{"200": {Description: "OK"}},
			Extra:   map[string]any{"x-note": "one"},
		}
	}

	assert.True(t, equalResponses(base(), base()), "identical values must compare equal")

	differentValue := base()
	differentValue.Extra["x-note"] = "two"
	assert.False(t, equalResponses(base(), differentValue),
		"an extension with a different value makes the objects different")

	missingExtension := base()
	missingExtension.Extra = nil
	assert.False(t, equalResponses(base(), missingExtension),
		"an absent extension makes the objects different")
}
