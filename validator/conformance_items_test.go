package validator

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
)

// validateSpec parses and validates a spec, returning the result.
func validateSpec(t *testing.T, spec string) *ValidationResult {
	t.Helper()
	parseResult, err := parser.New().ParseBytes([]byte(spec))
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	result, err := New().ValidateParsed(*parseResult)
	if err != nil {
		t.Fatalf("ValidateParsed: %v", err)
	}
	return result
}

// resultHasMessage reports whether any issue's message contains substr.
func resultHasMessage(result *ValidationResult, substr string) bool {
	for _, e := range result.Errors {
		if strings.Contains(e.Message, substr) {
			return true
		}
	}
	return false
}

// TestArraySchemaWithoutItemsIsValid covers the removal of the
// "Array schema must have 'items' defined" rule from the Schema Object path.
//
// No OAS version states that constraint for a Schema Object: it is absent from
// schema.yaml, meta.yaml and dialect.yaml, and no version's prose asserts it.
// OAS 2.0 requires `items` when `type` is "array", but only on the Items,
// Parameter and Header Objects — see TestOAS2PrimitiveItemsRequired.
func TestArraySchemaWithoutItemsIsValid(t *testing.T) {
	const wantAbsent = "Array schema must have 'items' defined"

	tests := []struct {
		name string
		spec string
	}{
		{
			name: "2.0 array definition without items",
			spec: `
swagger: "2.0"
info:
  title: API
  version: 1.0.0
paths: {}
definitions:
  Tags:
    type: array
`,
		},
		{
			name: "3.0 array schema without items",
			spec: `
openapi: 3.0.3
info:
  title: API
  version: 1.0.0
paths: {}
components:
  schemas:
    Tags:
      type: array
`,
		},
		{
			name: "3.1 array schema without items",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
components:
  schemas:
    Tags:
      type: array
`,
		},
		{
			name: "3.2 array schema without items",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  schemas:
    Tags:
      type: array
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateSpec(t, tt.spec)
			if resultHasMessage(result, wantAbsent) {
				t.Errorf("no OAS version requires 'items' on an array Schema Object, got: %v", result.Errors)
			}
		})
	}
}

// TestOAS2PrimitiveItemsRequired covers the narrowing half: Swagger 2.0 does
// require `items` when `type` is "array", on the Parameter, Header and Items
// Objects — the "primitives" form, which is an Items Object rather than a
// Schema Object.
func TestOAS2PrimitiveItemsRequired(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		want    string
		wantErr bool
	}{
		{
			name: "non-body parameter of type array without items",
			spec: `
swagger: "2.0"
info:
  title: API
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - name: tags
          in: query
          type: array
      responses:
        "200":
          description: ok
`,
			want:    `Non-body parameter with type "array" must have 'items' defined`,
			wantErr: true,
		},
		{
			name: "non-body parameter of type array with items is valid",
			spec: `
swagger: "2.0"
info:
  title: API
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - name: tags
          in: query
          type: array
          items:
            type: string
      responses:
        "200":
          description: ok
`,
			want: `must have 'items' defined`,
		},
		{
			name: "body parameter of type array is a Schema Object and is exempt",
			spec: `
swagger: "2.0"
info:
  title: API
  version: 1.0.0
paths:
  /pets:
    post:
      parameters:
        - name: body
          in: body
          schema:
            type: array
      responses:
        "200":
          description: ok
`,
			want: `must have 'items' defined`,
		},
		{
			name: "path-item-level parameter is covered",
			spec: `
swagger: "2.0"
info:
  title: API
  version: 1.0.0
paths:
  /pets:
    parameters:
      - name: tags
        in: query
        type: array
    get:
      responses:
        "200":
          description: ok
`,
			want:    `Non-body parameter with type "array" must have 'items' defined`,
			wantErr: true,
		},
		{
			name: "root-level parameter definition is covered",
			spec: `
swagger: "2.0"
info:
  title: API
  version: 1.0.0
paths: {}
parameters:
  tagsParam:
    name: tags
    in: query
    type: array
`,
			want:    `Non-body parameter with type "array" must have 'items' defined`,
			wantErr: true,
		},
		{
			name: "nested Items Object of type array without items",
			spec: `
swagger: "2.0"
info:
  title: API
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - name: matrix
          in: query
          type: array
          items:
            type: array
      responses:
        "200":
          description: ok
`,
			want:    `Items with type "array" must have 'items' defined`,
			wantErr: true,
		},
		{
			name: "operation response header of type array without items",
			spec: `
swagger: "2.0"
info:
  title: API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          headers:
            X-Rate-Limits:
              type: array
`,
			want:    `Header with type "array" must have 'items' defined`,
			wantErr: true,
		},
		{
			name: "root-level response header of type array without items",
			spec: `
swagger: "2.0"
info:
  title: API
  version: 1.0.0
paths: {}
responses:
  Standard:
    description: ok
    headers:
      X-Rate-Limits:
        type: array
`,
			want:    `Header with type "array" must have 'items' defined`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateSpec(t, tt.spec)
			if got := resultHasMessage(result, tt.want); got != tt.wantErr {
				t.Errorf("message %q present = %v, want %v; errors: %v", tt.want, got, tt.wantErr, result.Errors)
			}
		})
	}
}

// TestOAS3ArrayParameterIsNotSubjectToOAS2ItemsRule guards the version gate on
// the narrowed rule: OAS 3.x parameters carry a Schema Object, not an Items
// Object, and no 3.x version requires `items`.
func TestOAS3ArrayParameterIsNotSubjectToOAS2ItemsRule(t *testing.T) {
	spec := `
openapi: 3.0.3
info:
  title: API
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - name: tags
          in: query
          schema:
            type: array
      responses:
        "200":
          description: ok
`
	result := validateSpec(t, spec)
	if resultHasMessage(result, "must have 'items' defined") {
		t.Errorf("OAS 3.x has no items-required rule, got: %v", result.Errors)
	}
}

// TestOAS2ItemsNestingDepthBounded guards the depth bound on the Items chain.
func TestOAS2ItemsNestingDepthBounded(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`swagger: "2.0"
info:
  title: API
  version: 1.0.0
paths: {}
parameters:
  deep:
    name: deep
    in: query
    type: array
`)
	// Build an items chain deeper than maxSchemaNestingDepth. Each level is
	// indented one step further than the last.
	indent := "    "
	for i := 0; i < maxSchemaNestingDepth+5; i++ {
		sb.WriteString(indent + "items:\n")
		indent += "  "
		sb.WriteString(indent + "type: array\n")
	}

	result := validateSpec(t, sb.String())
	if !resultHasMessage(result, "Items nesting depth") {
		t.Errorf("want a depth-bound error, got: %v", result.Errors)
	}
}
