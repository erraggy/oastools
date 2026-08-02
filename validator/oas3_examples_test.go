package validator

import "testing"

// TestOAS3ExampleValueExclusivity covers the Example Object rules from issue
// #397 item 4. These are the constraints the issue flags as "a validation rule,
// not just a field": dataValue and serializedValue each forbid siblings.
func TestOAS3ExampleValueExclusivity(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		wantErrors []string
	}{
		{
			name: "dataValue with value",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  examples:
    bad:
      dataValue: {a: 1}
      value: {a: 1}
`,
			wantErrors: []string{
				"components.examples.bad: Example must not have both dataValue and value",
			},
		},
		{
			name: "serializedValue with value",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  examples:
    bad:
      serializedValue: "a=1"
      value: {a: 1}
`,
			wantErrors: []string{
				"components.examples.bad: Example must not have both serializedValue and value",
			},
		},
		{
			name: "serializedValue with externalValue",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  examples:
    bad:
      serializedValue: "a=1"
      externalValue: https://example.com/example.txt
`,
			wantErrors: []string{
				"components.examples.bad: Example must not have both serializedValue and externalValue",
			},
		},
		{
			// The spec's own Example Object example sets both, so no rule may pair
			// these two. A check that rejected the combination would flag correct
			// documents.
			name: "dataValue with serializedValue is legal",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  examples:
    good:
      dataValue: {a: 1}
      serializedValue: '{"a":1}'
`,
			wantErrors: nil,
		},
		{
			// The traversal that finds Example Objects is gated on 3.2 — see
			// oas32TraversalApplies for why paying it on every 3.0/3.1 document is
			// not worth catching a field those versions do not define. So the
			// exclusivity rule stays silent here even though dataValue and value are
			// set together.
			//
			// What reports instead is the version gate: below 3.2 the defect is not
			// that dataValue conflicts with value, it is that dataValue does not
			// exist yet. Issue #411 covers that pass.
			name: "the exclusivity rules do not run below 3.2, but the field is still too early",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  examples:
    tooEarly:
      dataValue: {a: 1}
      value: {a: 1}
`,
			wantErrors: []string{
				"components.examples.tooEarly.dataValue: dataValue was introduced in OpenAPI 3.2.0",
			},
		},
		{
			// Reached only through the media-type traversal, not through
			// components.examples, so it proves the traversal covers nested sites.
			name: "an example inside a response media type is reached",
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
              schema: {type: object}
              examples:
                bad:
                  dataValue: {a: 1}
                  value: {a: 1}
`,
			wantErrors: []string{
				"paths./pets.get.responses.200.content.application/json.examples.bad: " +
					"Example must not have both dataValue and value",
			},
		},
		{
			// Deepest reachable site: an example on a header of a nested encoding,
			// which only exists because OAS 3.2 made Encoding recursive.
			name: "an example under a nested encoding header is reached",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths:
  /upload:
    post:
      operationId: upload
      requestBody:
        content:
          multipart/form-data:
            schema: {type: object}
            encoding:
              meta:
                contentType: application/json
                encoding:
                  nested:
                    contentType: text/plain
                    headers:
                      X-Trace:
                        schema: {type: string}
                        examples:
                          bad:
                            dataValue: abc
                            value: abc
      responses:
        "204": {description: No Content}
`,
			wantErrors: []string{
				"paths./upload.post.requestBody.content.multipart/form-data.encoding.meta." +
					"encoding.nested.headers.X-Trace.examples.bad: " +
					"Example must not have both dataValue and value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrorsMatch(t, validationErrors(t, tt.spec), tt.wantErrors)
		})
	}
}
