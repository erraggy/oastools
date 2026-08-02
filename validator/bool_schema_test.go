package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBoolSchemaVersionGate covers the version applicability of the bare-boolean
// schema form. JSON Schema 2020-12 allows `true` and `false` wherever a schema is
// expected, and OAS 3.1 adopted that dialect wholesale. OAS 3.0 is based on an
// earlier draft where a schema is always an object.
//
// The parser accepts the form regardless of version — a Schema Object is decoded
// before the document version is known to it — so this check is the only thing
// standing between a 3.0 document and a silently accepted boolean schema. Same
// division of labour as the discriminator dialects.
func TestBoolSchemaVersionGate(t *testing.T) {
	const wantMsg = "Boolean schemas"

	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{
			name: "3.1 accepts a boolean component schema",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
components:
  schemas:
    anything: true
`,
		},
		{
			name: "3.2 accepts a boolean component schema",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  schemas:
    nothing: false
`,
		},
		{
			name: "3.0 rejects a boolean component schema",
			spec: `
openapi: 3.0.3
info:
  title: API
  version: 1.0.0
paths: {}
components:
  schemas:
    anything: true
`,
			wantErr: true,
		},
		{
			name: "2.0 rejects a boolean definition",
			spec: `
swagger: "2.0"
info:
  title: API
  version: 1.0.0
paths: {}
definitions:
  anything: true
`,
			wantErr: true,
		},
		{
			name: "3.0 rejects a boolean nested in properties",
			spec: `
openapi: 3.0.3
info:
  title: API
  version: 1.0.0
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        anything: true
`,
			wantErr: true,
		},
		{
			name: "3.1 accepts a boolean nested in properties",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
components:
  schemas:
    Pet:
      type: object
      properties:
        anything: true
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateSpec(t, tt.spec)
			assert.Equal(t, tt.wantErr, resultHasMessage(result, wantMsg),
				"boolean-schema error presence; errors: %v", result.Errors)

			// The accepting cases assert full validity, not merely the absence
			// of this one message — otherwise they would pass on a document
			// that is invalid for some unrelated reason.
			if !tt.wantErr {
				assert.True(t, result.Valid, "document should validate clean; errors: %v", result.Errors)
			}
		})
	}
}
