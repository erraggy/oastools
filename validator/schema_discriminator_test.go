package validator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The parser accepts both discriminator spellings regardless of version — it
// decodes a Schema Object before it knows the document version — so the
// validator is the only thing enforcing that OAS 2.0 documents use the bare
// string and OAS 3.0+ documents use the object. See erraggy/oastools#394.

// discriminatorErrors returns the validation errors mentioning discriminator.
func discriminatorErrors(t *testing.T, result *ValidationResult) []string {
	t.Helper()

	var msgs []string
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "discriminator") {
			msgs = append(msgs, e.Message)
		}
	}
	return msgs
}

// validateInlineSpec validates a spec written inline in the test, since the
// discriminator-form cases are too small to justify their own fixture files.
func validateInlineSpec(t *testing.T, spec string) *ValidationResult {
	t.Helper()

	path := filepath.Join(t.TempDir(), "spec.yaml")
	require.NoError(t, os.WriteFile(path, []byte(spec), 0o644))

	result, err := New().Validate(path)
	require.NoError(t, err)
	return result
}

func TestValidateDiscriminatorStringFormValidInOAS2(t *testing.T) {
	v := New()
	result, err := v.Validate("../testdata/discriminator-2.0.yaml")
	require.NoError(t, err)

	assert.Empty(t, discriminatorErrors(t, result))
	assert.True(t, result.Valid)
}

func TestValidateDiscriminatorObjectFormRejectedInOAS2(t *testing.T) {
	v := New()
	result, err := v.Validate("../testdata/discriminator-object-form-2.0.yaml")
	require.NoError(t, err)

	msgs := discriminatorErrors(t, result)
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0], "must be a string naming the property in OpenAPI 2.0")
	assert.False(t, result.Valid)
}

func TestValidateDiscriminatorObjectFormValidInOAS3(t *testing.T) {
	v := New()
	result, err := v.Validate("../testdata/join-discriminator-base-3.0.yaml")
	require.NoError(t, err)

	assert.Empty(t, discriminatorErrors(t, result))
}

func TestValidateDiscriminatorStringFormRejectedInOAS3(t *testing.T) {
	spec := `openapi: 3.0.3
info:
  title: String discriminator in a 3.0 document
  version: 1.0.0
paths: {}
components:
  schemas:
    Pet:
      type: object
      discriminator: petType
      required:
        - petType
      properties:
        petType:
          type: string
`
	result := validateInlineSpec(t, spec)

	msgs := discriminatorErrors(t, result)
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0], "must be an object with 'propertyName' in OpenAPI 3.0+")
	assert.False(t, result.Valid)
}

func TestValidateDiscriminatorFormSkippedForUnknownVersion(t *testing.T) {
	// ValidateParsed always sets oasVersion, but a hand-assembled ParseResult
	// may not. An unrecognized version says nothing about which form is
	// correct, so neither form may be flagged.
	for _, stringForm := range []bool{true, false} {
		schema := &parser.Schema{
			Discriminator: &parser.Discriminator{PropertyName: "petType", StringForm: stringForm},
		}
		result := &ValidationResult{}

		// Zero Validator: oasVersion is the invalid zero value.
		New().validateDiscriminatorForm(schema, "components.schemas.Pet", result)

		assert.Empty(t, result.Errors, "StringForm=%v must not be flagged without a known version", stringForm)
	}
}

func TestValidateDiscriminatorFormCheckedOnNestedSchemas(t *testing.T) {
	// The check hangs off the recursive schema walk, so it has to fire on
	// schemas nested well below a definition's root.
	//
	// This nests through allOf rather than items because the YAML decode path
	// leaves items as a raw map, so the validator never descends into it —
	// see erraggy/oastools#396.
	spec := `openapi: 3.0.3
info:
  title: Nested string discriminator
  version: 1.0.0
paths: {}
components:
  schemas:
    Wrapper:
      type: object
      properties:
        pet:
          allOf:
            - type: object
              discriminator: petType
              properties:
                petType:
                  type: string
`
	result := validateInlineSpec(t, spec)

	msgs := discriminatorErrors(t, result)
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0], "must be an object with 'propertyName' in OpenAPI 3.0+")
}
