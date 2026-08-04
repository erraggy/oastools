package validator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// TestDefaultOnlyResponsesSatisfiesTheSuccessCheck covers an operation whose
// only response is `default`. The strict-mode warning names default as
// sufficient, and it is: the Responses Object's default covers every code not
// listed individually.
//
// The check reads Responses.Default because every decode path routes that key
// to its own field, so it is never observable from a scan of Responses.Codes.
func TestDefaultOnlyResponsesSatisfiesTheSuccessCheck(t *testing.T) {
	spec := `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths:
  /a:
    get:
      operationId: a
      summary: only a default response
      responses:
        default: {description: covers every code}
`
	warnings := strings.Join(warningsWithStrict(t, spec, true), "\n")

	assert.NotContains(t, warnings, "successful response",
		"a default response covers every code, so the success check is satisfied")
}

// TestNonSuccessResponsesStillWarn is the counterpart: without a 2XX and
// without a default, the warning must still fire. Without this, a change that
// simply deleted the check would pass the test above.
func TestNonSuccessResponsesStillWarn(t *testing.T) {
	spec := `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths:
  /a:
    get:
      operationId: a
      summary: no success response
      responses:
        "404": {description: Not Found}
`
	warnings := strings.Join(warningsWithStrict(t, spec, true), "\n")

	assert.Contains(t, warnings, "Operation should define at least one successful response")
}

// TestResponseExtensionsAreNotTreatedAsStatusCodes covers a specification
// extension on the Responses Object, which OAS admits like any other object.
// It is not a status code, so neither the format check nor strict mode's
// non-standard-code check may say anything about it.
//
// This asserts the end-to-end result for a parsed document, which the decoders
// deliver by keeping extensions out of Responses.Codes entirely. The validator's
// own predicate is pinned separately, by
// TestExtensionKeyInCodesIsReportedAsInvalid, because no parsed document can
// present it with the input that exercises it.
func TestResponseExtensionsAreNotTreatedAsStatusCodes(t *testing.T) {
	spec := `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths:
  /a:
    get:
      operationId: a
      summary: carries an extension
      responses:
        "200": {description: OK}
        x-rate-limit: {description: not a response}
        x-scalar: 100
`
	warnings := strings.Join(warningsWithStrict(t, spec, true), "\n")

	assert.NotContains(t, warnings, "x-rate-limit")
	assert.NotContains(t, warnings, "x-scalar")
	assert.NotContains(t, warnings, "Non-standard HTTP status code")
}

// TestInvalidStatusCodeInCodesIsReported pins the format check itself. Reaching
// it takes a Responses object holding a key no decoder would have accepted,
// which is why the value is built here rather than parsed: the YAML and JSON
// decoders reject such a key outright.
func TestInvalidStatusCodeInCodesIsReported(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Info:       &parser.Info{Title: "T", Version: "1.0.0"},
		Paths: parser.Paths{
			"/a": {
				Get: &parser.Operation{
					OperationID: "a",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "OK"},
							"999": {Description: "not a status code"},
						},
					},
				},
			},
		},
	}

	v := New()
	v.IncludeWarnings = true
	result, err := v.ValidateParsed(parser.ParseResult{
		Document:   doc,
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
	})
	require.NoError(t, err)

	messages := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		messages = append(messages, e.Path+": "+e.Message)
	}
	joined := strings.Join(messages, "\n")

	assert.Contains(t, joined, "Invalid HTTP status code: 999")
	assert.NotContains(t, joined, "Invalid HTTP status code: 200")
}

// TestMalformedCodeDoesNotSatisfyTheSuccessCheck covers a key that is not a
// status code but begins with a 2, which is how the success check recognises
// one. A key already reported as malformed must not also be accepted as the
// operation's success response, or reporting it would suppress a second,
// unrelated diagnostic.
//
// The value is assembled in Go so the check can be exercised on its own.
// TestMalformedCodeFromAParsedDocument covers the same input arriving through
// the parser, which is what makes the case reachable in practice.
func TestMalformedCodeDoesNotSatisfyTheSuccessCheck(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Info:       &parser.Info{Title: "T", Version: "1.0.0"},
		Paths: parser.Paths{
			"/a": {
				Get: &parser.Operation{
					OperationID: "a",
					Summary:     "only a malformed code",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"2foo": {Description: "not a status code"},
						},
					},
				},
			},
		},
	}

	v := New()
	v.IncludeWarnings = true
	v.StrictMode = true
	result, err := v.ValidateParsed(parser.ParseResult{
		Document:   doc,
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
	})
	require.NoError(t, err)

	errs := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		errs = append(errs, e.Message)
	}
	warnings := make([]string, 0, len(result.Warnings))
	for _, w := range result.Warnings {
		warnings = append(warnings, w.Message)
	}

	assert.Contains(t, strings.Join(errs, "\n"), "Invalid HTTP status code: 2foo")
	assert.Contains(t, strings.Join(warnings, "\n"),
		"Operation should define at least one successful response",
		"a malformed key cannot stand in for a 2XX response")

	// The non-standard-code warning is for keys that are status codes but not
	// registered ones. A malformed key is already reported as an error, so
	// saying it twice in two different vocabularies helps nobody.
	assert.NotContains(t, strings.Join(warnings, "\n"), "Non-standard HTTP status code: 2foo")
}

// TestExtensionKeyInCodesIsReportedAsInvalid pins which question the validator
// asks of a Responses.Codes key. Codes holds status codes, so an extension
// sitting there is a defect in whatever assembled it and must be reported,
// even though the key would be perfectly legal one level up in the Responses
// Object itself.
//
// Only a caller-assembled document can reach this: the decoders route
// extensions to Responses.Extra, so no parsed document puts one in Codes.
func TestExtensionKeyInCodesIsReportedAsInvalid(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Info:       &parser.Info{Title: "T", Version: "1.0.0"},
		Paths: parser.Paths{
			"/a": {
				Get: &parser.Operation{
					OperationID: "a",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200":     {Description: "OK"},
							"x-note":  {Description: "an extension, misfiled"},
							"default": {Description: "misfiled too"},
						},
					},
				},
			},
		},
	}

	v := New()
	v.IncludeWarnings = true
	result, err := v.ValidateParsed(parser.ParseResult{
		Document:   doc,
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
	})
	require.NoError(t, err)

	messages := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		messages = append(messages, e.Path+": "+e.Message)
	}
	joined := strings.Join(messages, "\n")

	assert.Contains(t, joined, "Invalid HTTP status code: x-note")
	assert.Contains(t, joined, "Invalid HTTP status code: default")
	assert.NotContains(t, joined, "Invalid HTTP status code: 200")
}

// TestMalformedCodeFromAParsedDocument is the parsed counterpart to
// TestMalformedCodeDoesNotSatisfyTheSuccessCheck. decodeFromMap keeps a key
// that is not a status code rather than discarding it, so a real document can
// put one in front of the validator, and this asserts it arrives that way
// rather than only when a test builds the value by hand.
//
// ResolveRefs selects that decode path. The YAML and JSON decoders reject the
// key outright, so neither can deliver this input.
func TestMalformedCodeFromAParsedDocument(t *testing.T) {
	const spec = `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths:
  /a:
    get:
      operationId: a
      summary: only a malformed code
      responses:
        "2foo": {description: not a status code}
`

	p := parser.New()
	p.ResolveRefs = true
	p.ValidateStructure = false
	parsed, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	doc, ok := parsed.Document.(*parser.OAS3Document)
	require.True(t, ok)
	require.Contains(t, doc.Paths["/a"].Get.Responses.Codes, "2foo",
		"the decoder must keep the key, or the validator never sees it")

	v := New()
	v.IncludeWarnings = true
	v.StrictMode = true
	result, err := v.ValidateParsed(*parsed)
	require.NoError(t, err)

	errs := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		errs = append(errs, e.Message)
	}
	warnings := make([]string, 0, len(result.Warnings))
	for _, w := range result.Warnings {
		warnings = append(warnings, w.Message)
	}

	assert.Contains(t, strings.Join(errs, "\n"), "Invalid HTTP status code: 2foo")
	assert.Contains(t, strings.Join(warnings, "\n"),
		"Operation should define at least one successful response")
}
