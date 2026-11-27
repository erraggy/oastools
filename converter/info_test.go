package converter

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const oas31InfoFieldsSource = `openapi: 3.1.0
info:
  title: T
  version: "1.0.0"
  summary: A short summary
  license:
    name: MIT
    identifier: MIT
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`

// convertInfoSource parses source, converts it to target, and returns the
// result along with the source document's Info as the parser built it.
func convertInfoSource(t *testing.T, source, target string) (*ConversionResult, *parser.Info) {
	t.Helper()

	parsed, err := parser.New().ParseBytes([]byte(source))
	require.NoError(t, err)

	var srcInfo *parser.Info
	switch doc := parsed.Document.(type) {
	case *parser.OAS3Document:
		srcInfo = doc.Info
	case *parser.OAS2Document:
		srcInfo = doc.Info
	}
	require.NotNil(t, srcInfo)

	result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion(target))
	require.NoError(t, err)
	return result, srcInfo
}

func outputInfo(t *testing.T, result *ConversionResult) *parser.Info {
	t.Helper()
	switch doc := result.Document.(type) {
	case *parser.OAS3Document:
		return doc.Info
	case *parser.OAS2Document:
		return doc.Info
	}
	require.Failf(t, "unexpected document type", "%T", result.Document)
	return nil
}

func requireValidOutput(t *testing.T, result *ConversionResult) {
	t.Helper()
	vr, err := validator.ValidateWithOptions(validator.WithParsed(*result.ToParseResult()))
	require.NoError(t, err)
	assert.True(t, vr.Valid, "converted output must validate: %+v", vr.Errors)
}

func issueAt(result *ConversionResult, path string) *ConversionIssue {
	for i := range result.Issues {
		if result.Issues[i].Path == path {
			return &result.Issues[i]
		}
	}
	return nil
}

// TestOAS31InfoFieldsDroppedBelow31 pins that a target below 3.1 loses the 3.1
// Info and License fields from the output only, reports each, and validates.
func TestOAS31InfoFieldsDroppedBelow31(t *testing.T) {
	for _, target := range []string{"3.0.3", "2.0"} {
		t.Run(target, func(t *testing.T) {
			result, srcInfo := convertInfoSource(t, oas31InfoFieldsSource, target)

			info := outputInfo(t, result)
			require.NotNil(t, info)
			require.NotNil(t, info.License)
			assert.Empty(t, info.Summary)
			assert.Empty(t, info.License.Identifier)
			assert.Equal(t, "MIT", info.License.Name)
			assert.Empty(t, info.License.URL, "no url is synthesized from the identifier")

			for _, path := range []string{"info.summary", "info.license.identifier"} {
				issue := issueAt(result, path)
				require.NotNil(t, issue, "expected an issue at %s", path)
				assert.Equal(t, SeverityWarning, issue.Severity)
				assert.Contains(t, issue.Message, "OAS 3.1+ only")
				assert.Contains(t, issue.Message, target)
				assert.Contains(t, issue.Message, "dropped")
			}
			assert.True(t, result.Success)

			assert.Equal(t, "A short summary", srcInfo.Summary, "source must not be mutated")
			require.NotNil(t, srcInfo.License)
			assert.Equal(t, "MIT", srcInfo.License.Identifier, "source must not be mutated")

			requireValidOutput(t, result)
		})
	}
}

// TestOAS31InfoFieldsKeptFrom31 pins that a target of 3.1 or later keeps both
// fields and reports neither.
func TestOAS31InfoFieldsKeptFrom31(t *testing.T) {
	for _, target := range []string{"3.1.0", "3.2.0"} {
		t.Run(target, func(t *testing.T) {
			result, _ := convertInfoSource(t, oas31InfoFieldsSource, target)

			info := outputInfo(t, result)
			require.NotNil(t, info.License)
			assert.Equal(t, "A short summary", info.Summary)
			assert.Equal(t, "MIT", info.License.Identifier)
			assert.Nil(t, issueAt(result, "info.summary"))
			assert.Nil(t, issueAt(result, "info.license.identifier"))

			requireValidOutput(t, result)
		})
	}
}

// TestOAS31InfoFieldsDroppedWithoutLicense pins that an Info without a License
// loses its summary without tripping over the missing object.
func TestOAS31InfoFieldsDroppedWithoutLicense(t *testing.T) {
	source := `openapi: 3.1.0
info:
  title: T
  version: "1.0.0"
  summary: A short summary
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`
	result, _ := convertInfoSource(t, source, "3.0.3")
	info := outputInfo(t, result)
	assert.Empty(t, info.Summary)
	assert.Nil(t, info.License)
	assert.NotNil(t, issueAt(result, "info.summary"))
	assert.Nil(t, issueAt(result, "info.license.identifier"))
	requireValidOutput(t, result)
}

// TestOAS2TermsOfService pins that a free-text termsOfService, legal in OAS
// 2.0, is dropped and reported on the way to OAS 3.x, while a URL is kept.
func TestOAS2TermsOfService(t *testing.T) {
	const freeText = `swagger: "2.0"
info:
  title: T
  version: "1.0.0"
  termsOfService: Use at your own risk.
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`
	const asURL = `swagger: "2.0"
info:
  title: T
  version: "1.0.0"
  termsOfService: https://example.com/terms
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`
	for _, target := range []string{"3.0.3", "3.1.0"} {
		t.Run("free text to "+target, func(t *testing.T) {
			result, srcInfo := convertInfoSource(t, freeText, target)

			assert.Empty(t, outputInfo(t, result).TermsOfService)
			issue := issueAt(result, "info.termsOfService")
			require.NotNil(t, issue)
			assert.Equal(t, SeverityWarning, issue.Severity)
			assert.Contains(t, issue.Message, "dropped")
			assert.Contains(t, issue.Context, "Use at your own risk.")
			assert.Equal(t, "Use at your own risk.", srcInfo.TermsOfService, "source must not be mutated")

			requireValidOutput(t, result)
		})
		t.Run("url to "+target, func(t *testing.T) {
			result, _ := convertInfoSource(t, asURL, target)

			assert.Equal(t, "https://example.com/terms", outputInfo(t, result).TermsOfService)
			assert.Nil(t, issueAt(result, "info.termsOfService"))
			requireValidOutput(t, result)
		})
	}
}
