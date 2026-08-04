package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			require.NoError(t, err)
			if tt.wantErr {
				assert.True(t, hasErrorContaining(result.Errors, tt.errSubst),
					"want error containing %q, got errors: %v", tt.errSubst, result.Errors)
				return
			}
			// Neither form of the root requirement may fire.
			assert.False(t, hasErrorContaining(result.Errors, wantMsg),
				"want no root-requirement error, got: %v", result.Errors)
			assert.False(t, hasErrorContaining(result.Errors, want30Msg),
				"want no root-requirement error, got: %v", result.Errors)
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
			require.NoError(t, err)
			assert.Equal(t, tt.wantErr, hasErrorContaining(result.Errors, wantMsg),
				"responses-required error presence; errors: %v", result.Errors)
		})
	}
}

// TestOperationResponsesStatusCodesStillChecked guards the relaxation from the
// obvious way it could go wrong: dropping the `responses` requirement for 3.1+
// must not stop an invalid status code being reported when `responses` *is*
// present.
//
// Note that the check which fires here is the decoder's, not the structure
// validator's. paths.go (YAML) and paths_json.go (JSON) both reject an invalid
// status code outright, so a Responses object is never built and the structure
// validator's loop over Responses.Codes never sees one by this route.
//
// The third decode path, decodeFromMap, is not reachable from here: it runs
// only under ResolveRefs, cannot return an error, and keeps the key so the
// structure validator reports it instead.
// TestResponsesInvalidStatusCodeIsReportedOnEveryDecodePath covers that route
// and how it reports.
func TestOperationResponsesStatusCodesStillChecked(t *testing.T) {
	const want = "invalid status code '999'"

	// paths.go and paths_json.go implement this check separately, so both are
	// exercised — the same document in each source format. Covering only YAML
	// would leave the JSON decoder's copy free to drift.
	tests := []struct {
		name string
		spec string
	}{
		{
			name: "yaml",
			spec: `
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
`,
		},
		{
			name: "json",
			spec: `{
  "openapi": "3.2.0",
  "info": {"title": "API", "version": "1.0.0"},
  "paths": {
    "/pets": {
      "get": {
        "responses": {
          "999": {"description": "not a status code"}
        }
      }
    }
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pinned to how this is actually reported today: the
			// decoder rejects an invalid status code, so ParseBytes returns a
			// hard error and never reaches structure validation.
			//
			// Asserting where the diagnostic arrives, and not just its message,
			// is deliberate. If
			// a change moves this diagnostic into the collected result.Errors
			// instead, that is a change to ParseBytes's external contract —
			// callers today can rely on a non-nil error for this input — and
			// the test should fail so the move is a decision rather than a
			// side effect.
			_, err := New().ParseBytes([]byte(tt.spec))
			require.Error(t, err, "want a hard ParseBytes error containing %q", want)
			assert.Contains(t, err.Error(), want)
		})
	}
}
