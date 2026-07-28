package converter

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A discriminator means the same thing in both dialects but is spelled
// differently: a bare string in OAS 2.0, an object in OAS 3.0+. Conversion has
// to re-spell it, or it produces a document that is invalid at the target
// version. See erraggy/oastools#394.

func TestConvertOAS2StringDiscriminatorToOAS3Object(t *testing.T) {
	result, err := New().Convert("../testdata/discriminator-2.0.yaml", "3.0.3")
	require.NoError(t, err)
	require.True(t, result.Success)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok)

	pet := doc.Components.Schemas["Pet"]
	require.NotNil(t, pet)
	require.NotNil(t, pet.Discriminator)
	assert.Equal(t, "petType", pet.Discriminator.PropertyName)
	assert.False(t, pet.Discriminator.StringForm,
		"a 2.0 string discriminator must become the 3.x object form")
}

func TestConvertOAS3DiscriminatorObjectToOAS2String(t *testing.T) {
	result, err := New().Convert("../testdata/join-discriminator-base-3.0.yaml", "2.0")
	require.NoError(t, err)
	require.True(t, result.Success)

	doc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok)

	pet := doc.Definitions["Pet"]
	require.NotNil(t, pet)
	require.NotNil(t, pet.Discriminator)
	assert.Equal(t, "petType", pet.Discriminator.PropertyName)
	assert.True(t, pet.Discriminator.StringForm,
		"a 3.x discriminator object must become the 2.0 string form")
	assert.Nil(t, pet.Discriminator.Mapping, "OAS 2.0 has no discriminator mapping")
}

func TestConvertOAS3ToOAS2ReportsDroppedDiscriminatorMapping(t *testing.T) {
	// The base fixture's Pet carries a mapping, which has no OAS 2.0
	// equivalent; silently dropping it would hide real information loss.
	result, err := New().Convert("../testdata/join-discriminator-base-3.0.yaml", "2.0")
	require.NoError(t, err)

	var found bool
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, "discriminator uses 'mapping'") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected an issue reporting the dropped discriminator mapping")
}

func TestConvertOAS3ToOAS2ReportsDroppedDiscriminatorExtensions(t *testing.T) {
	// OAS 2.0 spells the discriminator as a bare string, so there is no object
	// left to carry x- extensions. Dropping them is unavoidable; dropping them
	// silently is not.
	schema := &parser.Schema{
		Discriminator: &parser.Discriminator{
			PropertyName: "petType",
			Extra:        map[string]any{"x-vendor": true, "x-internal": "yes"},
		},
	}
	result := &ConversionResult{}

	discriminatorToStringForm(New(), schema, result, "definitions.Pet")

	require.Len(t, result.Issues, 1)
	// Keys are sorted so the message is stable across map iteration order.
	assert.Contains(t, result.Issues[0].Message, "x-internal, x-vendor")
	assert.Contains(t, result.Issues[0].Message, "extensions dropped")
	assert.Nil(t, schema.Discriminator.Extra)
	assert.True(t, schema.Discriminator.StringForm)
}

func TestConvertOAS3ToOAS2SilentWhenDiscriminatorHasNothingToDrop(t *testing.T) {
	// A discriminator with neither mapping nor extensions converts losslessly
	// and must not manufacture a warning.
	schema := &parser.Schema{
		Discriminator: &parser.Discriminator{PropertyName: "petType"},
	}
	result := &ConversionResult{}

	discriminatorToStringForm(New(), schema, result, "definitions.Pet")

	assert.Empty(t, result.Issues)
	assert.True(t, schema.Discriminator.StringForm)
}

func TestConvertOAS2DiscriminatorRoundTrip(t *testing.T) {
	// 2.0 -> 3.0 -> 2.0 must land back on the string form rather than
	// accumulating the dialect it passed through.
	toOAS3, err := New().Convert("../testdata/discriminator-2.0.yaml", "3.0.3")
	require.NoError(t, err)

	backToOAS2, err := New().ConvertParsed(parser.ParseResult{
		Document:   toOAS3.Document,
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Data:       make(map[string]any),
		SourcePath: "converted.yaml",
	}, "2.0")
	require.NoError(t, err)

	doc, ok := backToOAS2.Document.(*parser.OAS2Document)
	require.True(t, ok)

	pet := doc.Definitions["Pet"]
	require.NotNil(t, pet)
	require.NotNil(t, pet.Discriminator)
	assert.Equal(t, "petType", pet.Discriminator.PropertyName)
	assert.True(t, pet.Discriminator.StringForm)
}

func TestDiscriminatorFormNormalizationReachesNestedSchemas(t *testing.T) {
	// Normalization rides the shared schema walk, so it must reach a
	// discriminator nested below the schema root.
	nested := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"pet": {
				AllOf: []*parser.Schema{
					{Discriminator: &parser.Discriminator{PropertyName: "petType", StringForm: true}},
				},
			},
		},
	}

	discriminatorToObjectForm(nested)
	assert.False(t, nested.Properties["pet"].AllOf[0].Discriminator.StringForm)

	discriminatorToStringForm(New(), nested, &ConversionResult{}, "test")
	assert.True(t, nested.Properties["pet"].AllOf[0].Discriminator.StringForm)
}
