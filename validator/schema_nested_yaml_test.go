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

// validateNestedSchemas reaches into items, additionalProperties and friends by
// type-asserting the any-typed field to *parser.Schema. Those fields only hold
// a *parser.Schema once the decode path promotes them, so a YAML document whose
// nested schemas were left as map[string]any validated clean no matter what was
// inside them. See erraggy/oastools#396.

// validateSpecFile writes spec to a file with the given extension and validates
// it. The extension matters: the parser picks its decode path from it, so the
// same document takes the YAML struct-decode path as .yaml and the JSON
// fast-path as .json.
func validateSpecFile(t *testing.T, ext, spec string) *ValidationResult {
	t.Helper()

	path := filepath.Join(t.TempDir(), "spec"+ext)
	require.NoError(t, os.WriteFile(path, []byte(spec), 0o644))

	result, err := New().Validate(path)
	require.NoError(t, err)
	return result
}

// enumTypeErrorPaths returns the paths of the enum-type errors in result. Paths
// rather than messages, because the messages name the Go type of the offending
// value and YAML decodes an integer to int where JSON decodes it to float64 —
// a difference unrelated to which schemas were visited.
func enumTypeErrorPaths(t *testing.T, result *ValidationResult) []string {
	t.Helper()

	var paths []string
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "Enum value must be a string") {
			paths = append(paths, e.Path)
		}
	}
	return paths
}

// wantEnumErrorPaths is one path per schema in the fixtures below: the direct
// case that always worked, plus the three that are only reachable once the
// schema-or-bool fields hold a *parser.Schema.
var wantEnumErrorPaths = []string{
	"components.schemas.Direct.enum[1]",
	"components.schemas.UnderItems.items.enum[1]",
	"components.schemas.UnderAdditionalProperties.additionalProperties.enum[1]",
	"components.schemas.UnderNestedItems.items.items.enum[1]",
}

// A string enum holding an integer, buried one level deeper each time. The
// error is reported for the top-level form on every decode path, so reaching it
// under items is purely a question of whether the subtree was visited.
const nestedEnumSpecYAML = `openapi: "3.0.3"
info:
  title: Test
  version: "1.0.0"
paths: {}
components:
  schemas:
    Direct:
      type: string
      enum: [ok, 42]
    UnderItems:
      type: array
      items:
        type: string
        enum: [ok, 42]
    UnderAdditionalProperties:
      type: object
      additionalProperties:
        type: string
        enum: [ok, 42]
    UnderNestedItems:
      type: array
      items:
        type: array
        items:
          type: string
          enum: [ok, 42]
`

const nestedEnumSpecJSON = `{
  "openapi": "3.0.3",
  "info": {"title": "Test", "version": "1.0.0"},
  "paths": {},
  "components": {
    "schemas": {
      "Direct": {"type": "string", "enum": ["ok", 42]},
      "UnderItems": {
        "type": "array",
        "items": {"type": "string", "enum": ["ok", 42]}
      },
      "UnderAdditionalProperties": {
        "type": "object",
        "additionalProperties": {"type": "string", "enum": ["ok", 42]}
      },
      "UnderNestedItems": {
        "type": "array",
        "items": {
          "type": "array",
          "items": {"type": "string", "enum": ["ok", 42]}
        }
      }
    }
  }
}`

func TestValidateReportsEnumErrorsNestedInYAMLSpec(t *testing.T) {
	result := validateSpecFile(t, ".yaml", nestedEnumSpecYAML)

	assert.ElementsMatch(t, wantEnumErrorPaths, enumTypeErrorPaths(t, result))
	assert.False(t, result.Valid)
}

func TestValidateReportsSameNestedErrorsForYAMLAndJSON(t *testing.T) {
	// Identical documents must validate identically; before the YAML decode
	// path promoted its schema-or-bool fields, only the JSON form reported the
	// three nested errors.
	fromYAML := enumTypeErrorPaths(t, validateSpecFile(t, ".yaml", nestedEnumSpecYAML))
	fromJSON := enumTypeErrorPaths(t, validateSpecFile(t, ".json", nestedEnumSpecJSON))

	assert.ElementsMatch(t, fromJSON, fromYAML)
}

func TestValidateReportsNestedErrorsForYAMLPathsUnderResolveRefs(t *testing.T) {
	// The ResolveRefs path decodes through decodeFromMap rather than the YAML
	// struct decoder, so it is a third implementation that has to agree.
	path := filepath.Join(t.TempDir(), "spec.yaml")
	require.NoError(t, os.WriteFile(path, []byte(nestedEnumSpecYAML), 0o644))

	p := parser.New()
	p.ResolveRefs = true
	parseResult, err := p.Parse(path)
	require.NoError(t, err)

	result, err := New().ValidateParsed(*parseResult)
	require.NoError(t, err)

	assert.ElementsMatch(t, wantEnumErrorPaths, enumTypeErrorPaths(t, result))
}
