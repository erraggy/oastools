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

// TestInvalidStatusCodeInCodesIsReported pins the format check itself. The
// value is built here rather than parsed so the check can be exercised on its
// own, without the parser reporting the same key first.
// TestMalformedCodeFromAParsedDocument covers it arriving through the parser.
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
// ResolveRefs selects decodeFromMap, which is the path under test here. The
// YAML and JSON decoders keep the key too, so the input is not exclusive to
// this path; pinning one of the three is what makes the assertion specific.
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

// TestValidatorReportsStatusCodeWhenStructureValidationIsOff completes the
// chain the parser package cannot assert on its own, since validator imports
// parser and not the other way round.
//
// WithValidateStructure(false) turns off the parse-time report of an invalid
// status code, which is what that flag is for. This asserts the finding is not
// lost with it: the key stays in Responses.Codes and the validator names it
// independently of how the document was parsed.
func TestValidatorReportsStatusCodeWhenStructureValidationIsOff(t *testing.T) {
	const spec = `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
paths:
  /a:
    get:
      operationId: a
      summary: carries an unusable status code
      responses:
        "999": {description: not a status code}
`

	p := parser.New()
	p.ValidateStructure = false
	parsed, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)
	require.Empty(t, parsed.Errors, "structure validation is off, so the parser reports nothing")

	doc, ok := parsed.Document.(*parser.OAS3Document)
	require.True(t, ok)
	require.Contains(t, doc.Paths["/a"].Get.Responses.Codes, "999",
		"the decoder must keep the key, or there is nothing left to report")

	v := New()
	result, err := v.ValidateParsed(*parsed)
	require.NoError(t, err)

	messages := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		messages = append(messages, e.Path+": "+e.Message)
	}
	assert.Contains(t, strings.Join(messages, "\n"), "Invalid HTTP status code: 999")
	assert.False(t, result.Valid, "a document with an unusable status code is not valid")
}

// TestWildcardResponseRangesPermitted pins the version boundary the wildcard
// range sits on. OAS 3.0 introduced it, so 2.0 is the only released version
// that does not define it, and an unrecognized version permits it the way the
// other version gates do.
func TestWildcardResponseRangesPermitted(t *testing.T) {
	tests := []struct {
		name     string
		version  parser.OASVersion
		expected bool
	}{
		{"2.0 predates the wildcard range", parser.OASVersion20, false},
		{"3.0.0 introduced it", parser.OASVersion300, true},
		{"3.0.4", parser.OASVersion304, true},
		{"3.1.0", parser.OASVersion310, true},
		{"3.2.0", parser.OASVersion320, true},
		{"an unrecognized version permits it", parser.Unknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, wildcardResponseRangesPermitted(tt.version))
		})
	}
}

// TestWildcardRangeIsReportedInOAS2 covers a 2.0 document keyed by a wildcard
// range. The 2.0 Responses Object admits one property per HTTP status code and
// describes no ranges, so "2XX" names a key its own version does not define
// (#467). The numeric key beside it must stay unreported, or the check would be
// rejecting the Responses Object rather than the range.
func TestWildcardRangeIsReportedInOAS2(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "T", Version: "1.0.0"},
		Paths: map[string]*parser.PathItem{
			"/a": {
				Get: &parser.Operation{
					OperationID: "a",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "OK"},
							"2XX": {Description: "a range 2.0 does not define"},
						},
					},
				},
			},
		},
	}

	v := New()
	result, err := v.ValidateParsed(parser.ParseResult{
		Document:   doc,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
	})
	require.NoError(t, err)

	messages := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		messages = append(messages, e.Path+": "+e.Message)
	}
	joined := strings.Join(messages, "\n")

	assert.Contains(t, joined, "Invalid HTTP status code: 2XX")
	assert.Contains(t, joined, "wildcard ranges were introduced in OAS 3.0",
		"the message must name the version that defines the key, since most tooling accepts it")
	assert.NotContains(t, joined, "Invalid HTTP status code: 200",
		"a numeric code is legal in every version")
	assert.False(t, result.Valid)
}

// TestWildcardRangeIsAcceptedInOAS3 is the counterpart. Without it, a change
// that rejected the wildcard range in every version would pass the test above.
func TestWildcardRangeIsAcceptedInOAS3(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.4",
		OASVersion: parser.OASVersion304,
		Info:       &parser.Info{Title: "T", Version: "1.0.0"},
		Paths: parser.Paths{
			"/a": {
				Get: &parser.Operation{
					OperationID: "a",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"2XX": {Description: "a range 3.0 defines"},
						},
					},
				},
			},
		},
	}

	v := New()
	result, err := v.ValidateParsed(parser.ParseResult{
		Document:   doc,
		Version:    "3.0.4",
		OASVersion: parser.OASVersion304,
	})
	require.NoError(t, err)

	messages := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		messages = append(messages, e.Message)
	}
	assert.NotContains(t, strings.Join(messages, "\n"), "Invalid HTTP status code: 2XX")

	// Asserting the whole verdict as well, so that reporting the range under
	// some other message would fail here too.
	assert.Empty(t, result.Errors)
	assert.True(t, result.Valid, "a 3.0 document may key its responses by a wildcard range")
}

// TestWildcardRangeInOAS2DoesNotSatisfyTheSuccessCheck applies the rule the
// malformed-key case already establishes: a key reported as unusable cannot
// also stand in as the operation's success response. "2XX" begins with a 2,
// which is how the success check recognises one, so a version that does not
// define the key must not count it either.
func TestWildcardRangeInOAS2DoesNotSatisfyTheSuccessCheck(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "T", Version: "1.0.0"},
		Paths: map[string]*parser.PathItem{
			"/a": {
				Get: &parser.Operation{
					OperationID: "a",
					Summary:     "only a range 2.0 does not define",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"2XX": {Description: "a range 2.0 does not define"},
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
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
	})
	require.NoError(t, err)

	warnings := make([]string, 0, len(result.Warnings))
	for _, w := range result.Warnings {
		warnings = append(warnings, w.Message)
	}
	joined := strings.Join(warnings, "\n")

	assert.Contains(t, joined, "Operation should define at least one successful response",
		"a key 2.0 does not define cannot stand in for a 2XX response")
	assert.NotContains(t, joined, "Non-standard HTTP status code: 2XX",
		"the key is already reported as an error, so saying it twice helps nobody")
}

// TestPermittedWildcardRangeDrawsNoNonStandardWarning covers strict mode's
// non-standard-code warning, which asks whether a code is one the HTTP RFCs
// register. Only a numeric code can be: a wildcard range names a class of codes,
// so the registry has nothing to say about it and a range the version permits is
// not irregular. Microsoft Graph keys every operation this way, so warning on it
// buries the findings that matter.
func TestPermittedWildcardRangeDrawsNoNonStandardWarning(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.4",
		OASVersion: parser.OASVersion304,
		Info:       &parser.Info{Title: "T", Version: "1.0.0"},
		Paths: parser.Paths{
			"/a": {
				Get: &parser.Operation{
					OperationID: "a",
					Summary:     "keyed by ranges",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"2XX": {Description: "ranged success"},
							"5XX": {Description: "ranged failure"},
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
		Version:    "3.0.4",
		OASVersion: parser.OASVersion304,
	})
	require.NoError(t, err)

	warnings := make([]string, 0, len(result.Warnings))
	for _, w := range result.Warnings {
		warnings = append(warnings, w.Message)
	}
	joined := strings.Join(warnings, "\n")

	assert.NotContains(t, joined, "Non-standard HTTP status code: 2XX")
	assert.NotContains(t, joined, "Non-standard HTTP status code: 5XX")

	// 2XX is a success response, so the missing-success warning must not fire
	// either: the range covers every code the check is looking for.
	assert.NotContains(t, joined, "Operation should define at least one successful response")
}

// TestNonStandardNumericCodeStillWarns is the counterpart. Without it, a change
// that deleted the non-standard-code warning outright would pass the test above.
func TestNonStandardNumericCodeStillWarns(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.4",
		OASVersion: parser.OASVersion304,
		Info:       &parser.Info{Title: "T", Version: "1.0.0"},
		Paths: parser.Paths{
			"/a": {
				Get: &parser.Operation{
					OperationID: "a",
					Summary:     "keyed by an unregistered code",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"299": {Description: "valid format, not a registered code"},
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
		Version:    "3.0.4",
		OASVersion: parser.OASVersion304,
	})
	require.NoError(t, err)

	warnings := make([]string, 0, len(result.Warnings))
	for _, w := range result.Warnings {
		warnings = append(warnings, w.Message)
	}
	assert.Contains(t, strings.Join(warnings, "\n"), "Non-standard HTTP status code: 299")
}
