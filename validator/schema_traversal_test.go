package validator

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// TestSchemaValidationReachesEveryPosition covers issue #423.
//
// validateSchema was called from two places — the Request Body media type loop
// and `components.schemas` — so a schema in a response, a parameter or a header
// was never validated by any of the six rule families it runs. The OAS 3.2 field
// check is only how the gap surfaced; `xml.nodeType` is used here because it is
// unambiguous and needs nothing else in the document to fire.
//
// One position per case, so a failure names the position that lost its
// traversal rather than a count.
func TestSchemaValidationReachesEveryPosition(t *testing.T) {
	// nodeType is OAS 3.2+, so a 3.0.3 document carrying it is an error wherever
	// the traversal reaches.
	const xml = `{type: string, xml: {name: x, nodeType: attribute}}`

	tests := []struct {
		name     string
		spec     string
		wantPath string
	}{
		{
			name: "request body content schema",
			spec: `
paths:
  /pets:
    post:
      operationId: addPet
      requestBody:
        content: {application/json: {schema: ` + xml + `}}
      responses: {"200": {description: OK}}
`,
			wantPath: "paths./pets.post.requestBody.content.application/json.schema.xml",
		},
		{
			name: "response content schema",
			spec: `
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content: {application/json: {schema: ` + xml + `}}
`,
			wantPath: "paths./pets.get.responses.200.content.application/json.schema.xml",
		},
		{
			name: "default response content schema",
			spec: `
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        default:
          description: Fallback
          content: {application/json: {schema: ` + xml + `}}
`,
			wantPath: "paths./pets.get.responses.default.content.application/json.schema.xml",
		},
		{
			name: "operation parameter schema",
			spec: `
paths:
  /pets:
    get:
      operationId: listPets
      parameters: [{name: q, in: query, schema: ` + xml + `}]
      responses: {"200": {description: OK}}
`,
			wantPath: "paths./pets.get.parameters[0].schema.xml",
		},
		{
			name: "path item parameter schema",
			spec: `
paths:
  /pets:
    parameters: [{name: q, in: query, schema: ` + xml + `}]
    get:
      operationId: listPets
      responses: {"200": {description: OK}}
`,
			wantPath: "paths./pets.parameters[0].schema.xml",
		},
		{
			name: "response header schema",
			spec: `
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          headers: {X-Rate: {schema: ` + xml + `}}
`,
			wantPath: "paths./pets.get.responses.200.headers.X-Rate.schema.xml",
		},
		{
			name: "component response content schema",
			spec: `
paths: {}
components:
  responses:
    Shared:
      description: OK
      content: {application/json: {schema: ` + xml + `}}
`,
			wantPath: "components.responses.Shared.content.application/json.schema.xml",
		},
		{
			name: "component parameter schema",
			spec: `
paths: {}
components:
  parameters:
    Shared: {name: q, in: query, schema: ` + xml + `}
`,
			wantPath: "components.parameters.Shared.schema.xml",
		},
		{
			name: "component header schema",
			spec: `
paths: {}
components:
  headers:
    Shared: {schema: ` + xml + `}
`,
			wantPath: "components.headers.Shared.schema.xml",
		},
		{
			name: "component request body content schema",
			spec: `
paths: {}
components:
  requestBodies:
    Shared:
      content: {application/json: {schema: ` + xml + `}}
`,
			wantPath: "components.requestBodies.Shared.content.application/json.schema.xml",
		},
		{
			name: "request body itemSchema",
			spec: `
paths:
  /pets:
    post:
      operationId: addPet
      requestBody:
        content: {application/json: {itemSchema: ` + xml + `}}
      responses: {"200": {description: OK}}
`,
			wantPath: "paths./pets.post.requestBody.content.application/json.itemSchema.xml",
		},
		{
			name: "component request body itemSchema",
			spec: `
paths: {}
components:
  requestBodies:
    Shared:
      content: {application/json: {itemSchema: ` + xml + `}}
`,
			wantPath: "components.requestBodies.Shared.content.application/json.itemSchema.xml",
		},
		{
			name: "webhook operation parameter schema",
			spec: `
paths: {}
webhooks:
  petEvent:
    post:
      operationId: onPet
      parameters: [{name: q, in: query, schema: ` + xml + `}]
      responses: {"200": {description: OK}}
`,
			wantPath: "webhooks.petEvent.post.parameters[0].schema.xml",
		},
		{
			name: "webhook operation response schema",
			spec: `
paths: {}
webhooks:
  petEvent:
    post:
      operationId: onPet
      responses:
        "200":
          description: OK
          content: {application/json: {schema: ` + xml + `}}
`,
			wantPath: "webhooks.petEvent.post.responses.200.content.application/json.schema.xml",
		},
		{
			name: "component path item response schema",
			spec: `
paths: {}
components:
  pathItems:
    shared:
      post:
        operationId: shared
        responses:
          "200":
            description: OK
            content: {application/json: {schema: ` + xml + `}}
`,
			wantPath: "components.pathItems.shared.post.responses.200.content.application/json.schema.xml",
		},
		{
			name: "component path item parameter schema",
			spec: `
paths: {}
components:
  pathItems:
    shared:
      parameters: [{name: q, in: query, schema: ` + xml + `}]
      post:
        operationId: shared
        responses: {"200": {description: OK}}
`,
			wantPath: "components.pathItems.shared.parameters[0].schema.xml",
		},
		{
			name: "encoding header schema",
			spec: `
paths:
  /pets:
    post:
      operationId: addPet
      requestBody:
        content:
          multipart/form-data:
            schema: {type: object}
            encoding:
              part:
                contentType: text/plain
                headers: {X-Enc: {schema: ` + xml + `}}
      responses: {"200": {description: OK}}
`,
			wantPath: "paths./pets.post.requestBody.content.multipart/form-data.encoding.part.headers.X-Enc.schema.xml",
		},
		{
			name: "operation callback response schema",
			spec: `
paths:
  /pets:
    post:
      operationId: addPet
      callbacks:
        onEvent:
          '{$request.body#/url}':
            post:
              operationId: cb
              responses:
                "200":
                  description: OK
                  content: {application/json: {schema: ` + xml + `}}
      responses: {"200": {description: OK}}
`,
			wantPath: "paths./pets.post.callbacks.onEvent.{$request.body#/url}.post.responses.200.content.application/json.schema.xml",
		},
		{
			name: "component callback response schema",
			spec: `
paths: {}
components:
  callbacks:
    Shared:
      '{$request.body#/u}':
        post:
          operationId: cbShared
          responses:
            "200":
              description: OK
              content: {application/json: {schema: ` + xml + `}}
`,
			wantPath: "components.callbacks.Shared.{$request.body#/u}.post.responses.200.content.application/json.schema.xml",
		},
		{
			name: "response media type itemSchema",
			spec: `
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content: {application/json: {itemSchema: ` + xml + `}}
`,
			wantPath: "paths./pets.get.responses.200.content.application/json.itemSchema.xml",
		},
		{
			name: "response header content schema",
			spec: `
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          headers:
            X-Hdr:
              content: {application/json: {schema: ` + xml + `}}
`,
			wantPath: "paths./pets.get.responses.200.headers.X-Hdr.content.application/json.schema.xml",
		},
		{
			name: "components.mediaTypes schema",
			spec: `
paths: {}
components:
  mediaTypes:
    PetStream: {schema: ` + xml + `}
`,
			wantPath: "components.mediaTypes.PetStream.schema.xml",
		},
		{
			name: "media type itemEncoding header schema",
			spec: `
paths:
  /pets:
    post:
      operationId: addPet
      requestBody:
        content:
          multipart/mixed:
            itemEncoding:
              contentType: text/plain
              headers: {X-Item: {schema: ` + xml + `}}
      responses: {"200": {description: OK}}
`,
			wantPath: "paths./pets.post.requestBody.content.multipart/mixed.itemEncoding.headers.X-Item.schema.xml",
		},
		{
			name: "media type prefixEncoding header schema",
			spec: `
paths:
  /pets:
    post:
      operationId: addPet
      requestBody:
        content:
          multipart/mixed:
            prefixEncoding:
              - contentType: text/plain
                headers: {X-Prefix: {schema: ` + xml + `}}
      responses: {"200": {description: OK}}
`,
			wantPath: "paths./pets.post.requestBody.content.multipart/mixed.prefixEncoding[0].headers.X-Prefix.schema.xml",
		},
		{
			name: "nested encoding header schema",
			spec: `
paths:
  /pets:
    post:
      operationId: addPet
      requestBody:
        content:
          multipart/form-data:
            schema: {type: object}
            encoding:
              part:
                contentType: text/plain
                encoding:
                  nested:
                    contentType: text/plain
                    headers: {X-Nested: {schema: ` + xml + `}}
      responses: {"200": {description: OK}}
`,
			wantPath: "paths./pets.post.requestBody.content.multipart/form-data.encoding.part.encoding.nested.headers.X-Nested.schema.xml",
		},
		{
			name: "additionalOperations inside a callback path item",
			spec: `
paths:
  /pets:
    post:
      operationId: addPet
      callbacks:
        onEvent:
          '{$request.body#/url}':
            additionalOperations:
              PURGE:
                operationId: cbPurge
                responses:
                  "200":
                    description: OK
                    content: {application/json: {schema: ` + xml + `}}
      responses: {"200": {description: OK}}
`,
			wantPath: "paths./pets.post.callbacks.onEvent.{$request.body#/url}.additionalOperations.PURGE.responses.200.content.application/json.schema.xml",
		},
		{
			name: "parameter content schema",
			spec: `
paths:
  /pets:
    post:
      operationId: addPet
      parameters:
        - name: q
          in: query
          content: {application/json: {schema: ` + xml + `}}
      responses: {"200": {description: OK}}
`,
			wantPath: "paths./pets.post.parameters[0].content.application/json.schema.xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := "openapi: 3.0.3\ninfo: {title: T, version: \"1.0.0\"}\n" + tt.spec
			errs := validationErrors(t, spec)

			var matched []string
			for _, e := range errs {
				if strings.Contains(e, "nodeType") {
					matched = append(matched, e)
				}
			}
			require.Len(t, matched, 1,
				"the schema at %s should be validated exactly once; got %v", tt.wantPath, errs)
			assert.Contains(t, matched[0], tt.wantPath,
				"the error should be reported at the schema's own position")
		})
	}
}

// TestSchemaValidationDoesNotDoubleReport pins that a schema reachable from two
// places is still reported once.
//
// validateSchema builds a fresh visited set per call, so widening the traversal
// raised the question. It does not double-report, because a `$ref` media type
// schema is a wrapper carrying only the reference: the definition it names is
// validated once, under components.
func TestSchemaValidationDoesNotDoubleReport(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    post:
      operationId: addPet
      requestBody:
        content: {application/json: {schema: {$ref: '#/components/schemas/Bad'}}}
      responses:
        "200":
          description: OK
          content: {application/json: {schema: {$ref: '#/components/schemas/Bad'}}}
components:
  schemas:
    Bad: {type: string, enum: [1]}
`
	errs := validationErrors(t, spec)

	var enumErrors []string
	for _, e := range errs {
		if strings.Contains(e, "Enum value") {
			enumErrors = append(enumErrors, e)
		}
	}
	require.Len(t, enumErrors, 1,
		"the shared schema should be reported once, at its definition; got %v", enumErrors)
	assert.Contains(t, enumErrors[0], "components.schemas.Bad.enum[0]")
}

// TestSchemaValidationBoundsCyclicCallbacks pins termination for the one graph the
// walk cannot treat as a tree.
//
// A Callback holds path items whose operations hold callbacks, so the two can
// point at each other. A depth bound alone does not contain that: a path item
// whose operations each lead back to it branches, and the walk is exponential in
// depth long before the bound is reached. Two such operations ran past forty
// seconds without returning; the visited set is what makes it terminate.
//
// Built by hand because a parsed document cannot close the loop, and
// ValidateParsed takes the caller's.
func TestSchemaValidationBoundsCyclicCallbacks(t *testing.T) {
	item := &parser.PathItem{
		// An enum that disagrees with its type, so each pass round the loop has
		// something to report. Not a 3.2 field: this calls the traversal directly,
		// with no document to carry a version for the 3.2 rules to read.
		Parameters: []*parser.Parameter{
			{Name: "q", In: "query", Schema: &parser.Schema{
				Type: "string", Enum: []any{1}}},
		},
	}
	callback := parser.Callback{"loop": item}
	// Two operations, both cycling back: branching, not just depth.
	item.Get = &parser.Operation{Callbacks: map[string]*parser.Callback{"c": &callback}}
	item.Post = &parser.Operation{Callbacks: map[string]*parser.Callback{"c": &callback}}

	result := &ValidationResult{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		New().validateOAS3OperationSchemas(item.Get, "paths./a.get", result)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the callback walk did not terminate; the cycle guard is gone")
	}

	assert.Len(t, result.Errors, 1,
		"the cycle should be walked once, reporting the parameter schema a single time")
}
