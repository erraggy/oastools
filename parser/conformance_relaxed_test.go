package parser

import (
	"strings"
	"testing"
)

// hasErrorContaining reports whether any parse error mentions substr.
func hasErrorContaining(errs []error, substr string) bool {
	for _, err := range errs {
		if strings.Contains(err.Error(), substr) {
			return true
		}
	}
	return false
}

// TestRootRequirementRelaxedIn31 covers the OAS 3.1 relaxation of the root
// requirement. OAS 3.0 requires `paths`; 3.1+ replaced that with a three-way
// `anyOf` over `paths`, `components` and `webhooks`.
func TestRootRequirementRelaxedIn31(t *testing.T) {
	const wantMsg = "at least one of 'paths', 'components' or 'webhooks'"
	const want30Msg = "missing required root field 'paths'"

	tests := []struct {
		name     string
		spec     string
		wantErr  bool
		errSubst string
	}{
		{
			name: "3.0 with paths is valid",
			spec: `
openapi: 3.0.3
info:
  title: API
  version: 1.0.0
paths: {}
`,
		},
		{
			name: "3.0 with only components is invalid: paths is required in 3.0",
			spec: `
openapi: 3.0.3
info:
  title: API
  version: 1.0.0
components: {}
`,
			wantErr:  true,
			errSubst: want30Msg,
		},
		{
			name: "3.1 with only components is valid",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
components: {}
`,
		},
		{
			name: "3.2 with only components is valid",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components: {}
`,
		},
		{
			name: "3.1 with only paths is valid",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
paths: {}
`,
		},
		{
			// `anyOf: [required: webhooks]` asks only that the key be present,
			// so an empty map satisfies it. This is why the check compares
			// against nil rather than using len().
			name: "3.1 with an empty but present webhooks map is valid",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
webhooks: {}
`,
		},
		{
			name: "3.2 with an empty but present components map is valid",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components: {}
`,
		},
		{
			name: "3.1 with none of the three is invalid",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
`,
			wantErr:  true,
			errSubst: wantMsg,
		},
		{
			name: "3.2 with none of the three is invalid",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
`,
			wantErr:  true,
			errSubst: wantMsg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := New().ParseBytes([]byte(tt.spec))
			if err != nil {
				t.Fatalf("ParseBytes: %v", err)
			}
			got := hasErrorContaining(result.Errors, tt.errSubst)
			if tt.wantErr && !got {
				t.Errorf("want error containing %q, got errors: %v", tt.errSubst, result.Errors)
			}
			if !tt.wantErr {
				// Neither form of the root requirement may fire.
				if hasErrorContaining(result.Errors, wantMsg) || hasErrorContaining(result.Errors, want30Msg) {
					t.Errorf("want no root-requirement error, got: %v", result.Errors)
				}
			}
		})
	}
}

// TestOperationResponsesRelaxedIn31 covers the OAS 3.1 relaxation of the
// Operation Object's `responses` field. It is REQUIRED in 2.0 and 3.0 and
// optional from 3.1 onward: 3.1+ `$defs.operation` carries no `required:`
// clause, and the prose drops the **REQUIRED** marker, which per its own
// preamble means OPTIONAL.
func TestOperationResponsesRelaxedIn31(t *testing.T) {
	const wantMsg = "Operation must have a responses object"

	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{
			name: "2.0 operation without responses is invalid",
			spec: `
swagger: "2.0"
info:
  title: API
  version: 1.0.0
paths:
  /pets:
    get:
      summary: list pets
`,
			wantErr: true,
		},
		{
			name: "3.0 operation without responses is invalid",
			spec: `
openapi: 3.0.3
info:
  title: API
  version: 1.0.0
paths:
  /pets:
    get:
      summary: list pets
`,
			wantErr: true,
		},
		{
			name: "3.1 operation without responses is valid",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
paths:
  /pets:
    get:
      summary: list pets
`,
		},
		{
			name: "3.2 operation without responses is valid",
			spec: `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
paths:
  /pets:
    get:
      summary: list pets
`,
		},
		{
			name: "3.1 operation with responses is valid",
			spec: `
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := New().ParseBytes([]byte(tt.spec))
			if err != nil {
				t.Fatalf("ParseBytes: %v", err)
			}
			got := hasErrorContaining(result.Errors, wantMsg)
			if got != tt.wantErr {
				t.Errorf("responses-required error = %v, want %v; errors: %v", got, tt.wantErr, result.Errors)
			}
		})
	}
}

// TestOperationResponsesStatusCodesStillChecked guards the else-branch of the
// relaxation: dropping the requirement must not stop status codes from being
// validated when `responses` *is* present.
func TestOperationResponsesStatusCodesStillChecked(t *testing.T) {
	spec := `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
paths:
  /pets:
    get:
      responses:
        "999":
          description: not a status code
`
	result, err := New().ParseBytes([]byte(spec))

	// The status-code check reaches this document through the decoder, which
	// reports it as a hard parse failure rather than a collected error, so
	// accept either channel — the point is that relaxing the `responses`
	// requirement did not stop the code being checked.
	const want = "invalid status code '999'"
	if err != nil {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("want error containing %q, got: %v", want, err)
		}
		return
	}
	if !hasErrorContaining(result.Errors, want) {
		t.Errorf("want error containing %q, got: %v", want, result.Errors)
	}
}
