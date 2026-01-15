package builder

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddParameterWithConstraints_OAS3 tests AddParameter with constraints for OAS 3.x.
func TestAddParameterWithConstraints_OAS3(t *testing.T) {
	t.Run("numeric constraints", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("LimitParam", "query", "limit", int32(0),
				WithParamDescription("Max results"),
				WithParamMinimum(1),
				WithParamMaximum(100),
				WithParamDefault(20),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		require.NotNil(t, doc.Components.Parameters)
		param := doc.Components.Parameters["LimitParam"]
		require.NotNil(t, param)
		require.NotNil(t, param.Schema)

		// Constraints should be on schema for OAS 3.x
		require.NotNil(t, param.Schema.Minimum)
		assert.Equal(t, 1.0, *param.Schema.Minimum)
		require.NotNil(t, param.Schema.Maximum)
		assert.Equal(t, 100.0, *param.Schema.Maximum)
		assert.Equal(t, 20, param.Schema.Default)
	})

	t.Run("string constraints", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("NameParam", "query", "name", "",
				WithParamMinLength(1),
				WithParamMaxLength(50),
				WithParamPattern("^[a-zA-Z]+$"),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		param := doc.Components.Parameters["NameParam"]
		require.NotNil(t, param.Schema)
		require.NotNil(t, param.Schema.MinLength)
		assert.Equal(t, 1, *param.Schema.MinLength)
		require.NotNil(t, param.Schema.MaxLength)
		assert.Equal(t, 50, *param.Schema.MaxLength)
		assert.Equal(t, "^[a-zA-Z]+$", param.Schema.Pattern)
	})

	t.Run("enum constraint", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("StatusParam", "query", "status", "",
				WithParamEnum("available", "pending", "sold"),
				WithParamDefault("available"),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		param := doc.Components.Parameters["StatusParam"]
		require.NotNil(t, param.Schema)
		require.Len(t, param.Schema.Enum, 3)
		assert.Equal(t, "available", param.Schema.Enum[0])
		assert.Equal(t, "pending", param.Schema.Enum[1])
		assert.Equal(t, "sold", param.Schema.Enum[2])
		assert.Equal(t, "available", param.Schema.Default)
	})
}

// TestAddParameterWithConstraints_OAS2 tests AddParameter with constraints for OAS 2.0.
func TestAddParameterWithConstraints_OAS2(t *testing.T) {
	t.Run("numeric constraints", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("LimitParam", "query", "limit", int32(0),
				WithParamDescription("Max results"),
				WithParamMinimum(1),
				WithParamMaximum(100),
				WithParamDefault(20),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		require.NotNil(t, doc.Parameters)
		param := doc.Parameters["LimitParam"]
		require.NotNil(t, param)

		// Constraints should be directly on parameter for OAS 2.0
		require.NotNil(t, param.Minimum)
		assert.Equal(t, 1.0, *param.Minimum)
		require.NotNil(t, param.Maximum)
		assert.Equal(t, 100.0, *param.Maximum)
		assert.Equal(t, 20, param.Default)
	})

	t.Run("string constraints", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("NameParam", "query", "name", "",
				WithParamMinLength(1),
				WithParamMaxLength(50),
				WithParamPattern("^[a-zA-Z]+$"),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		param := doc.Parameters["NameParam"]
		require.NotNil(t, param.MinLength)
		assert.Equal(t, 1, *param.MinLength)
		require.NotNil(t, param.MaxLength)
		assert.Equal(t, 50, *param.MaxLength)
		assert.Equal(t, "^[a-zA-Z]+$", param.Pattern)
	})

	t.Run("enum constraint", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("StatusParam", "query", "status", "",
				WithParamEnum("available", "pending", "sold"),
				WithParamDefault("available"),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		param := doc.Parameters["StatusParam"]
		require.Len(t, param.Enum, 3)
		assert.Equal(t, "available", param.Enum[0])
		assert.Equal(t, "pending", param.Enum[1])
		assert.Equal(t, "sold", param.Enum[2])
		assert.Equal(t, "available", param.Default)
	})
}

// TestAddParameterWithInvalidConstraints tests that invalid constraints accumulate errors.
func TestAddParameterWithInvalidConstraints(t *testing.T) {
	t.Run("min greater than max", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("BadParam", "query", "bad", int32(0),
				WithParamMinimum(100),
				WithParamMaximum(1),
			)

		_, err := b.BuildOAS3()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minimum")
	})

	t.Run("invalid pattern", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("BadParam", "query", "bad", "",
				WithParamPattern("[invalid"),
			)

		_, err := b.BuildOAS3()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pattern")
	})
}

// TestSchemaCopying tests that schema copying works correctly to avoid mutations.
func TestSchemaCopying(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0")

	// Create two parameters with the same type but different constraints
	b.AddParameter("Param1", "query", "p1", int32(0),
		WithParamMinimum(1),
		WithParamMaximum(10),
	)
	b.AddParameter("Param2", "query", "p2", int32(0),
		WithParamMinimum(100),
		WithParamMaximum(1000),
	)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	param1 := doc.Components.Parameters["Param1"]
	param2 := doc.Components.Parameters["Param2"]

	// Verify constraints are independent
	require.NotNil(t, param1.Schema.Minimum)
	require.NotNil(t, param2.Schema.Minimum)
	assert.Equal(t, 1.0, *param1.Schema.Minimum)
	assert.Equal(t, 100.0, *param2.Schema.Minimum)
	assert.Equal(t, 10.0, *param1.Schema.Maximum)
	assert.Equal(t, 1000.0, *param2.Schema.Maximum)
}

// TestAddParameterWithTypeFormatOverride tests AddParameter with type/format overrides.
func TestAddParameterWithTypeFormatOverride(t *testing.T) {
	t.Run("OAS3 component parameter with format override", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("UserID", "path", "user_id", "",
				WithParamFormat("uuid"),
				WithParamDescription("User identifier"),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		require.NotNil(t, doc.Components.Parameters)
		param := doc.Components.Parameters["UserID"]
		require.NotNil(t, param)
		require.NotNil(t, param.Schema)
		assert.Equal(t, "string", param.Schema.Type)
		assert.Equal(t, "uuid", param.Schema.Format)
	})

	t.Run("OAS2 component parameter with format override", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("UserID", "path", "user_id", "",
				WithParamFormat("uuid"),
				WithParamDescription("User identifier"),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		require.NotNil(t, doc.Parameters)
		param := doc.Parameters["UserID"]
		require.NotNil(t, param)
		assert.Equal(t, "string", param.Type)
		assert.Equal(t, "uuid", param.Format)
	})
}
