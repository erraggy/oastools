package validator

import (
	"testing"
)

// TestOAS32ExampleValueExclusivity covers the Example Object rules from issue
// #397 item 4. These are the constraints the issue flags as "a validation rule,
// not just a field": dataValue and serializedValue each forbid siblings.
func TestOAS32ExampleValueExclusivity(t *testing.T) {
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

// TestOAS32XMLNodeType covers the XML Object rules. The coexistence rules are
// errors rather than warnings because the 3.2 fixed fields table says of both
// attribute and wrapped: "If nodeType is present, this field MUST NOT be
// present."
func TestOAS32XMLNodeType(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		wantErrors []string
	}{
		{
			name: "nodeType with attribute",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  schemas:
    Pet:
      type: string
      xml: {nodeType: attribute, attribute: true}
`,
			wantErrors: []string{
				"components.schemas.Pet.xml: XML attribute must not be present when nodeType is present",
			},
		},
		{
			name: "nodeType with wrapped",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  schemas:
    Pets:
      type: array
      items: {type: string}
      xml: {nodeType: element, wrapped: true}
`,
			wantErrors: []string{
				"components.schemas.Pets.xml: XML wrapped must not be present when nodeType is present",
			},
		},
		{
			name: "invalid nodeType value",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  schemas:
    Pet:
      type: string
      xml: {nodeType: elemental}
`,
			wantErrors: []string{
				`components.schemas.Pet.xml: Invalid XML nodeType "elemental"`,
			},
		},
		{
			name: "nodeType is rejected before 3.2",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  schemas:
    Pet:
      type: string
      xml: {nodeType: attribute}
`,
			wantErrors: []string{
				"components.schemas.Pet.xml: XML nodeType is only supported in OAS 3.2+",
			},
		},
		{
			name: "each legal nodeType value is accepted",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  schemas:
    A: {type: string, xml: {nodeType: element}}
    B: {type: string, xml: {nodeType: attribute}}
    C: {type: string, xml: {nodeType: text}}
    D: {type: string, xml: {nodeType: cdata}}
    E: {type: string, xml: {nodeType: none}}
`,
			wantErrors: nil,
		},
		{
			// The deprecated bools remain legal on their own — nothing here makes
			// a pre-3.2 document invalid.
			name: "attribute and wrapped without nodeType stay legal",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  schemas:
    Pets:
      type: array
      items: {type: string}
      xml: {wrapped: true}
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

// TestOAS32DiscriminatorDefaultMapping covers the conditional requirement:
// "when defined as an optional property the Discriminator Object MUST include a
// defaultMapping field".
//
// The controls matter more than the positive case here. The rule is only reported
// where optionality is locally provable, so the cases that must stay silent are
// what keep it from firing on correct documents.
func TestOAS32DiscriminatorDefaultMapping(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		wantErrors []string
	}{
		{
			name: "optional discriminating property without defaultMapping",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        petType: {type: string}
      discriminator:
        propertyName: petType
`,
			wantErrors: []string{
				"components.schemas.Pet.discriminator: Discriminator must include defaultMapping " +
					"because the discriminating property 'petType' is optional",
			},
		},
		{
			name: "required discriminating property needs no defaultMapping",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  schemas:
    Pet:
      type: object
      required: [petType]
      properties:
        petType: {type: string}
      discriminator:
        propertyName: petType
`,
			wantErrors: nil,
		},
		{
			name: "optional property with defaultMapping is satisfied",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        petType: {type: string}
      discriminator:
        propertyName: petType
        defaultMapping: OtherPet
    OtherPet: {type: object}
`,
			wantErrors: nil,
		},
		{
			// The common oneOf shape: the discriminating property is declared in
			// the subschemas, so its optionality is not knowable here. Staying
			// silent is deliberate — guessing would flag correct documents.
			name: "property declared only in subschemas is not reported",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  schemas:
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Dog'
      discriminator:
        propertyName: petType
    Dog:
      type: object
      required: [petType]
      properties:
        petType: {type: string}
`,
			wantErrors: nil,
		},
		{
			name: "defaultMapping is rejected before 3.2",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  schemas:
    Pet:
      type: object
      required: [petType]
      properties:
        petType: {type: string}
      discriminator:
        propertyName: petType
        defaultMapping: OtherPet
    OtherPet: {type: object}
`,
			wantErrors: []string{
				"components.schemas.Pet.discriminator: discriminator defaultMapping is only supported in OAS 3.2+",
			},
		},
		{
			// A 3.0/3.1 document with an optional discriminating property must not
			// be told to add a field its version does not have.
			name: "the requirement does not apply before 3.2",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        petType: {type: string}
      discriminator:
        propertyName: petType
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

// TestOAS32QueryStringParam covers the three constraints on in: "querystring",
// including the path-item interaction the spec spells out as "in the same
// operation (or in the operation's path-item)".
func TestOAS32QueryStringParam(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		wantErrors []string
	}{
		{
			name: "querystring with content is legal",
			spec: `
openapi: 3.2.0
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
`,
			wantErrors: nil,
		},
		{
			name: "querystring using schema instead of content",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths:
  /search:
    get:
      operationId: search
      parameters:
        - name: q
          in: querystring
          schema: {type: string}
      responses:
        "200": {description: OK}
`,
			wantErrors: []string{
				"paths./search.get.parameters[0]: A querystring parameter must be specified using the content field",
				"paths./search.get.parameters[0]: A querystring parameter must not use schema",
			},
		},
		{
			name: "querystring with style is rejected",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths:
  /search:
    get:
      operationId: search
      parameters:
        - name: q
          in: querystring
          style: form
          content:
            application/x-www-form-urlencoded:
              schema: {type: object}
      responses:
        "200": {description: OK}
`,
			wantErrors: []string{
				"paths./search.get.parameters[0]: A querystring parameter must not use style",
			},
		},
		{
			name: "two querystring parameters in one operation",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths:
  /search:
    get:
      operationId: search
      parameters:
        - name: a
          in: querystring
          content:
            application/x-www-form-urlencoded: {schema: {type: object}}
        - name: b
          in: querystring
          content:
            application/x-www-form-urlencoded: {schema: {type: object}}
      responses:
        "200": {description: OK}
`,
			wantErrors: []string{
				"paths./search.get: A querystring parameter must not appear more than once, but 2 were found",
			},
		},
		{
			name: "querystring alongside query in the same operation",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths:
  /search:
    get:
      operationId: search
      parameters:
        - name: a
          in: querystring
          content:
            application/x-www-form-urlencoded: {schema: {type: object}}
        - name: limit
          in: query
          schema: {type: integer}
      responses:
        "200": {description: OK}
`,
			wantErrors: []string{
				"paths./search.get: A querystring parameter must not appear alongside any 'in: query' parameter",
			},
		},
		{
			// The interaction the spec's "or in the operation's path-item" clause
			// exists for: neither list contains both, but the effective list does.
			name: "path-item query with operation querystring conflicts",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths:
  /search:
    parameters:
      - name: limit
        in: query
        schema: {type: integer}
    get:
      operationId: search
      parameters:
        - name: a
          in: querystring
          content:
            application/x-www-form-urlencoded: {schema: {type: object}}
      responses:
        "200": {description: OK}
`,
			wantErrors: []string{
				"paths./search.get: A querystring parameter must not appear alongside any 'in: query' parameter",
			},
		},
		{
			// A conflict wholly inside the path item's own list is reported once
			// against the path item, not repeated for each of its operations.
			name: "a path-item-only conflict is reported once",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths:
  /search:
    parameters:
      - name: limit
        in: query
        schema: {type: integer}
      - name: a
        in: querystring
        content:
          application/x-www-form-urlencoded: {schema: {type: object}}
    get:
      operationId: search
      responses:
        "200": {description: OK}
    post:
      operationId: searchPost
      responses:
        "200": {description: OK}
`,
			wantErrors: []string{
				"paths./search: A querystring parameter must not appear alongside any 'in: query' parameter",
			},
		},
		{
			// These tests parse with ValidateStructure disabled, so the parser's own
			// rejection of in: "querystring" below 3.2 does not fire here and the
			// validator's 3.2 rules are gated off. TestQueryStringLocationVersionGate
			// covers the version enforcement on the normal parse path.
			name: "the querystring rules do not run below 3.2",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths:
  /search:
    get:
      operationId: search
      parameters:
        - name: q
          in: querystring
          schema: {type: string}
      responses:
        "200": {description: OK}
`,
			wantErrors: nil,
		},
		{
			// Plain query parameters must be unaffected — the rule only triggers
			// when a querystring parameter is present.
			name: "several query parameters without querystring stay legal",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths:
  /search:
    get:
      operationId: search
      parameters:
        - name: limit
          in: query
          schema: {type: integer}
        - name: offset
          in: query
          schema: {type: integer}
      responses:
        "200": {description: OK}
`,
			wantErrors: nil,
		},
		{
			name: "querystring in a reusable path item is checked too",
			spec: `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  pathItems:
    Search:
      get:
        operationId: search
        parameters:
          - name: q
            in: querystring
            schema: {type: string}
        responses:
          "200": {description: OK}
`,
			wantErrors: []string{
				"components.pathItems.Search.get.parameters[0]: A querystring parameter must be specified using the content field",
				"components.pathItems.Search.get.parameters[0]: A querystring parameter must not use schema",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrorsMatch(t, validationErrors(t, tt.spec), tt.wantErrors)
		})
	}
}
