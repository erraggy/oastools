package builder

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
)

// TestBuildResponsesFromMap tests the buildResponsesFromMap helper function
// that converts a map of status codes to responses into a parser.Responses object.
func TestBuildResponsesFromMap(t *testing.T) {
	t.Run("empty map returns nil", func(t *testing.T) {
		result := buildResponsesFromMap(map[string]*parser.Response{})
		assert.Nil(t, result, "empty map should return nil")
	})

	t.Run("nil map returns nil", func(t *testing.T) {
		result := buildResponsesFromMap(nil)
		assert.Nil(t, result, "nil map should return nil")
	})

	t.Run("default response goes to Default field", func(t *testing.T) {
		defaultResp := &parser.Response{Description: "Default error"}
		result := buildResponsesFromMap(map[string]*parser.Response{
			"default": defaultResp,
		})

		assert.NotNil(t, result)
		assert.Equal(t, defaultResp, result.Default)
		assert.Empty(t, result.Codes)
	})

	t.Run("status codes go to Codes map", func(t *testing.T) {
		resp200 := &parser.Response{Description: "Success"}
		resp404 := &parser.Response{Description: "Not found"}
		result := buildResponsesFromMap(map[string]*parser.Response{
			"200": resp200,
			"404": resp404,
		})

		assert.NotNil(t, result)
		assert.Nil(t, result.Default)
		assert.Len(t, result.Codes, 2)
		assert.Equal(t, resp200, result.Codes["200"])
		assert.Equal(t, resp404, result.Codes["404"])
	})

	t.Run("mixed default and codes", func(t *testing.T) {
		defaultResp := &parser.Response{Description: "Default error"}
		resp200 := &parser.Response{Description: "Success"}
		result := buildResponsesFromMap(map[string]*parser.Response{
			"default": defaultResp,
			"200":     resp200,
		})

		assert.NotNil(t, result)
		assert.Equal(t, defaultResp, result.Default)
		assert.Len(t, result.Codes, 1)
		assert.Equal(t, resp200, result.Codes["200"])
	})
}

// TestWithResponseContentType_Default tests that response content type defaults correctly.
// Note: Other response option tests are in operation_test.go.
func TestWithResponseContentType_Default(t *testing.T) {
	cfg := &responseConfig{}
	// Default should be empty until set
	assert.Empty(t, cfg.contentType)

	WithResponseContentType("text/plain")(cfg)
	assert.Equal(t, "text/plain", cfg.contentType)
}
