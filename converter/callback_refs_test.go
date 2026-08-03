package converter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// OAS 2.0 has no callbacks, so converting an operation that has them is a
// reported loss. A callbacks entry may be written as a Reference Object, which
// the parser carries on its own field (see parser.Callback), so a check reading
// only the Callback Object map would convert one form silently.
func TestConvertToOAS2ReportsCallbacksInBothForms(t *testing.T) {
	callback := parser.Callback{"{$request.query.url}": {Post: &parser.Operation{}}}

	tests := map[string]*parser.Operation{
		"callback objects": {
			Callbacks: map[string]*parser.Callback{"inline": &callback},
			Responses: &parser.Responses{Codes: map[string]*parser.Response{"200": {Description: "ok"}}},
		},
		"reference objects only": {
			CallbackRefs: map[string]*parser.Reference{"referenced": {Ref: "#/components/callbacks/shared"}},
			Responses:    &parser.Responses{Codes: map[string]*parser.Response{"200": {Description: "ok"}}},
		},
	}

	for name, op := range tests {
		t.Run(name, func(t *testing.T) {
			parseResult := parser.ParseResult{
				OASVersion: parser.OASVersion303,
				Version:    "3.0.3",
				Document: &parser.OAS3Document{
					OpenAPI: "3.0.3",
					Info:    &parser.Info{Title: "T", Version: "1.0.0"},
					Paths:   parser.Paths{"/things": {Post: op}},
				},
			}

			result, err := New().ConvertParsed(parseResult, "2.0")
			require.NoError(t, err)

			var reported bool
			for _, issue := range result.Issues {
				if strings.Contains(issue.Message, "callbacks which are not supported in OAS 2.0") {
					reported = true
				}
			}
			assert.True(t, reported,
				"converting an operation with callbacks reported no loss; issues: %v", result.Issues)
		})
	}
}
