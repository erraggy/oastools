package validator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// validationErrorsResolved is validationErrors through the ResolveRefs route,
// which decodes via the generated decodeFromMap rather than the YAML or JSON
// decoder. Presence of an empty container has to survive all three.
func validationErrorsResolved(t *testing.T, spec string) []string {
	t.Helper()

	p := parser.New()
	p.ValidateStructure = false
	p.ResolveRefs = true
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err, "test spec should parse")

	v := New()
	v.IncludeWarnings = true
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	messages := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		messages = append(messages, e.Path+": "+e.Message)
	}
	return messages
}

// TestMediaTypeEncodingExclusions covers the Media Type Object's encoding
// exclusions. `encoding` forbids both sequential forms; the two sequential
// forms do not forbid each other, and a rule pairing them would reject valid
// documents.
func TestMediaTypeEncodingExclusions(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		wantErrors []string
	}{
		{
			name: "encoding with itemEncoding",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  requestBodies:
    rb:
      content:
        multipart/mixed:
          encoding: {}
          itemEncoding: {}
`,
			wantErrors: []string{
				"Media Type must not have both encoding and itemEncoding",
			},
		},
		{
			name: "encoding with prefixEncoding",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  requestBodies:
    rb:
      content:
        multipart/mixed:
          encoding: {}
          prefixEncoding: []
`,
			wantErrors: []string{
				"Media Type must not have both encoding and prefixEncoding",
			},
		},
		{
			name: "encoding with both sequential forms reports both",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  requestBodies:
    rb:
      content:
        multipart/mixed:
          encoding: {}
          itemEncoding: {}
          prefixEncoding: []
`,
			wantErrors: []string{
				"Media Type must not have both encoding and prefixEncoding",
				"Media Type must not have both encoding and itemEncoding",
			},
		},
		{
			// The dependentSchemas clause names only `encoding` as the trigger,
			// so this combination is legal and no rule may pair the two.
			name: "itemEncoding with prefixEncoding is legal",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  requestBodies:
    rb:
      content:
        multipart/mixed:
          itemEncoding: {}
          prefixEncoding: []
`,
			wantErrors: nil,
		},
		{
			name: "encoding alone is legal",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  requestBodies:
    rb:
      content:
        multipart/mixed:
          encoding: {}
`,
			wantErrors: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrorsMatch(t, validationErrors(t, tt.spec), tt.wantErrors)
		})
	}
}

// TestEncodingObjectExclusions covers the same pair on the Encoding Object,
// which 3.2 lets nest inside itself. The rule has to reach the nested object,
// not only the Media Type that owns it.
func TestEncodingObjectExclusions(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		wantErrors []string
	}{
		{
			name: "nested under encoding",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  requestBodies:
    rb:
      content:
        multipart/mixed:
          encoding:
            meta:
              encoding: {}
              itemEncoding: {}
`,
			wantErrors: []string{
				"components.requestBodies.rb.content.multipart/mixed.encoding.meta: Encoding must not have both encoding and itemEncoding",
			},
		},
		{
			name: "nested under prefixEncoding",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  requestBodies:
    rb:
      content:
        multipart/mixed:
          prefixEncoding:
            - encoding: {}
              prefixEncoding: []
`,
			wantErrors: []string{
				"components.requestBodies.rb.content.multipart/mixed.prefixEncoding[0]: Encoding must not have both encoding and prefixEncoding",
			},
		},
		{
			name: "nested under itemEncoding",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  requestBodies:
    rb:
      content:
        multipart/mixed:
          itemEncoding:
            encoding: {}
            itemEncoding: {}
`,
			wantErrors: []string{
				"components.requestBodies.rb.content.multipart/mixed.itemEncoding: Encoding must not have both encoding and itemEncoding",
			},
		},
		{
			name: "two levels down",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  requestBodies:
    rb:
      content:
        multipart/mixed:
          encoding:
            outer:
              encoding:
                inner:
                  encoding: {}
                  prefixEncoding: []
`,
			wantErrors: []string{
				"components.requestBodies.rb.content.multipart/mixed.encoding.outer.encoding.inner: Encoding must not have both encoding and prefixEncoding",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrorsMatch(t, validationErrors(t, tt.spec), tt.wantErrors)
		})
	}
}

// TestExampleExamplesExclusion covers `example` with `examples`, which
// Parameter, Header and Media Type all carry because all three compose the
// specification's shared `examples` definition.
func TestExampleExamplesExclusion(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		wantErrors []string
	}{
		{
			name: "parameter",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  parameters:
    animal:
      name: animal
      in: header
      schema: {}
      example: bear
      examples:
        one:
          dataValue: bear
`,
			wantErrors: []string{
				"components.parameters.animal: Parameter must not have both example and examples",
			},
		},
		{
			name: "header",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  headers:
    X-Trace:
      schema: {}
      example: abc
      examples:
        one:
          dataValue: abc
`,
			wantErrors: []string{
				"components.headers.X-Trace: Header must not have both example and examples",
			},
		},
		{
			name: "media type",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  requestBodies:
    rb:
      content:
        application/json:
          example: {a: 1}
          examples:
            one:
              dataValue: {a: 1}
`,
			wantErrors: []string{
				"components.requestBodies.rb.content.application/json: Media Type must not have both example and examples",
			},
		},
		{
			name: "either one alone is legal",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  parameters:
    withExample:
      name: a
      in: header
      schema: {}
      example: bear
    withExamples:
      name: b
      in: header
      schema: {}
      examples:
        one:
          dataValue: bear
`,
			wantErrors: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrorsMatch(t, validationErrors(t, tt.spec), tt.wantErrors)
		})
	}
}

// TestExampleValueExternalValueExclusion covers the Example Object pair that
// predates the rest. 3.1 states it in its schema and 3.0 in its prose only, so
// unlike its three siblings it is not a 3.2 rule.
func TestExampleValueExternalValueExclusion(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		wantErrors []string
	}{
		{
			name: "3.2",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  examples:
    bad:
      value: foo
      externalValue: https://example.com/foo
`,
			wantErrors: []string{
				"components.examples.bad: Example must not have both value and externalValue",
			},
		},
		{
			name: "3.1",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
components:
  examples:
    bad:
      value: foo
      externalValue: https://example.com/foo
`,
			wantErrors: []string{
				"components.examples.bad: Example must not have both value and externalValue",
			},
		},
		{
			name: "3.0 states it in prose only, and the prose is what governs",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths: {}
components:
  examples:
    bad:
      value: foo
      externalValue: https://example.com/foo
`,
			wantErrors: []string{
				"components.examples.bad: Example must not have both value and externalValue",
			},
		},
		{
			name: "either one alone is legal",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths: {}
components:
  examples:
    withValue:
      value: foo
    withExternal:
      externalValue: https://example.com/foo
`,
			wantErrors: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrorsMatch(t, validationErrors(t, tt.spec), tt.wantErrors)
		})
	}
}

// TestExclusionVersionScoping pins each rule to the versions that state it.
// Enforcing one version's rule everywhere produces a false positive in one
// direction or a silent miss in the other, which is the defect #433 removed.
func TestExclusionVersionScoping(t *testing.T) {
	// example/examples is stated from 3.0, so every 3.x version reports it.
	for _, version := range []string{"3.0.3", "3.1.0", "3.2.0"} {
		t.Run("example/examples at "+version, func(t *testing.T) {
			spec := "openapi: " + version + `
info: {title: T, version: "1.0.0"}
paths: {}
components:
  parameters:
    animal:
      name: animal
      in: header
      schema: {}
      example: bear
      examples:
        one:
          value: bear
`
			assertErrorsMatch(t, validationErrors(t, spec), []string{
				"Parameter must not have both example and examples",
			})
		})
	}

	// The encoding exclusions read fields 3.2 introduced. Below 3.2 the fields
	// are reported as too new, by the field gate, and the exclusion must not
	// fire on top of that.
	t.Run("encoding exclusions do not fire below 3.2", func(t *testing.T) {
		spec := `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
components:
  requestBodies:
    rb:
      content:
        multipart/mixed:
          encoding: {}
          itemEncoding: {}
`
		errs := validationErrors(t, spec)
		// Pinned non-empty, and pinned to the report that should fire instead.
		// Without this the loop below runs zero times and the sub-test passes
		// even if the traversal stopped reporting anything at all.
		assertErrorsMatch(t, errs, []string{
			"itemEncoding was introduced in OpenAPI 3.2.0",
		})
		for _, got := range errs {
			assert.NotContains(t, got, "must not have both encoding",
				"the 3.2 encoding exclusion must not reach a 3.1 document")
		}
	})

	// dataValue and serializedValue do not exist before 3.2, so the three
	// exclusions naming them stay silent even though a 3.0 document can spell
	// the keys.
	t.Run("dataValue exclusions do not fire below 3.2", func(t *testing.T) {
		spec := `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths: {}
components:
  examples:
    bad:
      dataValue: foo
      value: foo
`
		errs := validationErrors(t, spec)
		assertErrorsMatch(t, errs, []string{
			"dataValue was introduced in OpenAPI 3.2.0",
		})
		for _, got := range errs {
			assert.NotContains(t, got, "must not have both dataValue and value",
				"the 3.2 dataValue exclusion must not reach a 3.0 document")
		}
	})
}

// TestExclusionPresenceSurvivesEveryDecodePath is the reason these rules can be
// stated at all: `encoding: {}` writes the key while leaving the map empty, so
// presence is the nil check rather than the length. parser keeps three decode
// paths and they have disagreed before, so each one is asserted rather than
// assumed.
func TestExclusionPresenceSurvivesEveryDecodePath(t *testing.T) {
	const yamlSpec = `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  requestBodies:
    rb:
      content:
        multipart/mixed:
          encoding: {}
          prefixEncoding: []
`

	const jsonSpec = `{"openapi":"3.2.0","info":{"title":"T","version":"1.0.0"},
"components":{"requestBodies":{"rb":{"content":{"multipart/mixed":{
"encoding":{},"prefixEncoding":[]}}}}}}`

	want := []string{"Media Type must not have both encoding and prefixEncoding"}

	t.Run("yaml", func(t *testing.T) {
		assertErrorsMatch(t, validationErrors(t, yamlSpec), want)
	})
	t.Run("json", func(t *testing.T) {
		assertErrorsMatch(t, validationErrors(t, jsonSpec), want)
	})
	t.Run("yaml through decodeFromMap", func(t *testing.T) {
		assertErrorsMatch(t, validationErrorsResolved(t, yamlSpec), want)
	})
	t.Run("json through decodeFromMap", func(t *testing.T) {
		assertErrorsMatch(t, validationErrorsResolved(t, jsonSpec), want)
	})
}

// TestExclusionsReachEveryMediaTypePosition asserts the rules fire wherever a
// Media Type Object can occur, not only the one position a fixture happens to
// use. A rule is only as good as the positions its walk reaches.
func TestExclusionsReachEveryMediaTypePosition(t *testing.T) {
	tests := []struct {
		name     string
		spec     string
		wantPath string
	}{
		{
			name: "components.requestBodies",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  requestBodies:
    rb:
      content:
        application/json:
          encoding: {}
          itemEncoding: {}
`,
			wantPath: "components.requestBodies.rb.content.application/json",
		},
		{
			name: "components.responses",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  responses:
    r:
      description: OK
      content:
        application/json:
          encoding: {}
          itemEncoding: {}
`,
			wantPath: "components.responses.r.content.application/json",
		},
		{
			name: "components.parameters content",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  parameters:
    p:
      name: p
      in: header
      content:
        application/json:
          encoding: {}
          itemEncoding: {}
`,
			wantPath: "components.parameters.p.content.application/json",
		},
		{
			name: "components.headers content",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  headers:
    X-Trace:
      content:
        application/json:
          encoding: {}
          itemEncoding: {}
`,
			wantPath: "components.headers.X-Trace.content.application/json",
		},
		{
			name: "operation requestBody",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    post:
      operationId: createPet
      requestBody:
        content:
          application/json:
            encoding: {}
            itemEncoding: {}
      responses:
        "204": {description: No Content}
`,
			wantPath: "paths./pets.post.requestBody.content.application/json",
		},
		{
			name: "operation response",
			spec: `
openapi: 3.2.0
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
              encoding: {}
              itemEncoding: {}
`,
			wantPath: "paths./pets.get.responses.200.content.application/json",
		},
		{
			name: "response header content",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          headers:
            X-Trace:
              content:
                application/json:
                  encoding: {}
                  itemEncoding: {}
`,
			wantPath: "paths./pets.get.responses.200.headers.X-Trace.content.application/json",
		},
		{
			name: "path item parameter content",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    parameters:
      - name: p
        in: header
        content:
          application/json:
            encoding: {}
            itemEncoding: {}
    get:
      operationId: listPets
      responses:
        "204": {description: No Content}
`,
			wantPath: "paths./pets.parameters[0].content.application/json",
		},
		{
			// The default response is a sibling of the coded ones rather than an
			// entry among them, so a walk over Responses.Codes alone never sees it.
			name: "default response",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        default:
          description: Anything else
          content:
            application/json:
              encoding: {}
              itemEncoding: {}
`,
			wantPath: "paths./pets.get.responses.default.content.application/json",
		},
		{
			// A Callback Object holds Path Item Objects, so every position inside a
			// path item exists again inside a callback.
			name: "operation callback",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    post:
      operationId: createPet
      responses:
        "204": {description: No Content}
      callbacks:
        onEvent:
          '{$request.body#/url}':
            post:
              operationId: notify
              requestBody:
                content:
                  application/json:
                    encoding: {}
                    itemEncoding: {}
              responses:
                "204": {description: No Content}
`,
			wantPath: "paths./pets.post.callbacks.onEvent.{$request.body#/url}.post." +
				"requestBody.content.application/json",
		},
		{
			name: "components.callbacks",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  callbacks:
    onEvent:
      '{$request.body#/url}':
        post:
          operationId: notify
          requestBody:
            content:
              application/json:
                encoding: {}
                itemEncoding: {}
          responses:
            "204": {description: No Content}
`,
			wantPath: "components.callbacks.onEvent.{$request.body#/url}.post." +
				"requestBody.content.application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrorsMatch(t, validationErrors(t, tt.spec), []string{
				tt.wantPath + ": Media Type must not have both encoding and itemEncoding",
			})
		})
	}
}

// TestCallbackTraversalTerminatesOnACycle pins the reason [Validator.visitCallbacks]
// tracks visited path items rather than relying on its depth bound alone. A
// parsed document cannot express this graph, but ValidateParsed takes the
// caller's, and a branching cycle goes exponential long before the bound.
func TestCallbackTraversalTerminatesOnACycle(t *testing.T) {
	item := &parser.PathItem{}
	cb := parser.Callback{"{$request.body#/url}": item}
	// Two operations, each holding the callback that leads back to this same path
	// item. Without the visited set the walk branches two ways per level.
	item.Post = &parser.Operation{Callbacks: map[string]*parser.Callback{"a": &cb}}
	item.Get = &parser.Operation{Callbacks: map[string]*parser.Callback{"b": &cb}}

	doc := &parser.OAS3Document{
		OpenAPI:    "3.2.0",
		OASVersion: parser.OASVersion320,
		Info:       &parser.Info{Title: "T", Version: "1.0.0"},
		Paths:      map[string]*parser.PathItem{"/pets": item},
	}

	done := make(chan error, 1)
	go func() {
		v := New()
		_, err := v.ValidateParsed(parser.ParseResult{
			Document:   doc,
			Version:    "3.2.0",
			OASVersion: parser.OASVersion320,
		})
		done <- err
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("callback traversal did not terminate on a cyclic graph")
	}
}

// TestExclusionsSkipReferenceForms asserts a $ref alias is left alone: the
// definition it names is checked in its own right.
//
// Both aliases deliberately carry the sibling fields of a rule they would
// otherwise trip. A `$ref` with no siblings proves nothing here, because no rule
// could fire on it whether or not the Ref guard exists. These spell the pair, so
// removing either guard makes this test fail.
func TestExclusionsSkipReferenceForms(t *testing.T) {
	spec := `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
components:
  parameters:
    alias:
      $ref: '#/components/parameters/real'
      example: bear
      examples:
        one:
          dataValue: bear
    real:
      name: p
      in: header
      schema: {}
  examples:
    exampleAlias:
      $ref: '#/components/examples/realExample'
      value: foo
      externalValue: https://example.com/example.json
    realExample:
      value: foo
`
	assertErrorsMatch(t, validationErrors(t, spec), nil)
}

// TestExclusionTablesAreWellFormed guards the tables themselves: a rule that
// names the same field twice, or carries no anchor, would report a message that
// reads as nonsense rather than failing loudly.
func TestExclusionTablesAreWellFormed(t *testing.T) {
	tables := map[string][]mutualExclusion{
		"example":   exampleExclusions,
		"mediaType": mediaTypeExclusions,
		"encoding":  encodingExclusions,
		"parameter": parameterExclusions,
		"header":    headerExclusions,
	}

	for name, table := range tables {
		t.Run(name, func(t *testing.T) {
			require.NotEmpty(t, table)
			seen := make(map[string]bool, len(table))
			for _, rule := range table {
				assert.NotEmpty(t, rule.object, "object label")
				assert.NotEmpty(t, rule.first, "first field")
				assert.NotEmpty(t, rule.second, "second field")
				assert.NotEqual(t, rule.first, rule.second, "a rule must name two distinct fields")
				assert.True(t, rule.since.IsValid(), "since must be a version this build knows")
				assert.Contains(t, rule.anchor, "#", "anchor must be a fragment")

				// Ordered, so that a rule stated in reverse counts as the same
				// pair: the two would report the same exclusion twice.
				a, b := rule.first, rule.second
				if a > b {
					a, b = b, a
				}
				pair := a + "/" + b
				assert.False(t, seen[pair], "duplicate rule for %s", pair)
				seen[pair] = true
			}
		})
	}
}

// TestFieldIsPresentTreatsUnknownNamesAsAbsent pins the lookup's behavior for a
// name the caller did not supply, which is what keeps a rule from firing on a
// field its call site cannot see.
func TestFieldIsPresentTreatsUnknownNamesAsAbsent(t *testing.T) {
	fields := []fieldPresence{
		{name: "encoding", present: true},
		{name: "itemEncoding", present: false},
	}

	assert.True(t, fieldIsPresent(fields, "encoding"))
	assert.False(t, fieldIsPresent(fields, "itemEncoding"))
	assert.False(t, fieldIsPresent(fields, "prefixEncoding"), "an unsupplied name is absent")
	assert.False(t, fieldIsPresent(nil, "encoding"))
}
