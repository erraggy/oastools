package builder

import (
	"net/http"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebhookParamWithConstraints tests webhooks with parameter constraints.
func TestWebhookParamWithConstraints(t *testing.T) {
	b := New(parser.OASVersion310).
		SetTitle("Webhook API").
		SetVersion("1.0.0").
		AddWebhook("events", http.MethodPost,
			WithQueryParam("limit", int32(0),
				WithParamMinimum(1),
				WithParamMaximum(100),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	require.NotNil(t, doc.Webhooks)
	require.Contains(t, doc.Webhooks, "events")
	params := doc.Webhooks["events"].Post.Parameters
	require.Len(t, params, 1)
	param := params[0]
	require.NotNil(t, param.Schema)
	require.NotNil(t, param.Schema.Minimum)
	assert.Equal(t, 1.0, *param.Schema.Minimum)
	require.NotNil(t, param.Schema.Maximum)
	assert.Equal(t, 100.0, *param.Schema.Maximum)
}

// TestCombinedConstraints tests combining multiple constraint types.
func TestCombinedConstraints(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/search",
			WithQueryParam("q", "",
				WithParamDescription("Search query"),
				WithParamRequired(true),
				WithParamMinLength(1),
				WithParamMaxLength(100),
				WithParamPattern("^[a-zA-Z0-9\\s]+$"),
				WithParamExample("test query"),
			),
			WithQueryParam("page", int32(0),
				WithParamMinimum(1),
				WithParamDefault(1),
			),
			WithQueryParam("size", int32(0),
				WithParamMinimum(1),
				WithParamMaximum(100),
				WithParamMultipleOf(10),
				WithParamDefault(10),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	params := doc.Paths["/search"].Get.Parameters
	require.Len(t, params, 3)

	// Check "q" parameter
	qParam := params[0]
	assert.Equal(t, "q", qParam.Name)
	assert.True(t, qParam.Required)
	require.NotNil(t, qParam.Schema)
	require.NotNil(t, qParam.Schema.MinLength)
	assert.Equal(t, 1, *qParam.Schema.MinLength)
	require.NotNil(t, qParam.Schema.MaxLength)
	assert.Equal(t, 100, *qParam.Schema.MaxLength)
	assert.Equal(t, "^[a-zA-Z0-9\\s]+$", qParam.Schema.Pattern)

	// Check "page" parameter
	pageParam := params[1]
	assert.Equal(t, "page", pageParam.Name)
	require.NotNil(t, pageParam.Schema.Minimum)
	assert.Equal(t, 1.0, *pageParam.Schema.Minimum)
	assert.Equal(t, 1, pageParam.Schema.Default)

	// Check "size" parameter
	sizeParam := params[2]
	assert.Equal(t, "size", sizeParam.Name)
	require.NotNil(t, sizeParam.Schema.Minimum)
	assert.Equal(t, 1.0, *sizeParam.Schema.Minimum)
	require.NotNil(t, sizeParam.Schema.Maximum)
	assert.Equal(t, 100.0, *sizeParam.Schema.Maximum)
	require.NotNil(t, sizeParam.Schema.MultipleOf)
	assert.Equal(t, 10.0, *sizeParam.Schema.MultipleOf)
	assert.Equal(t, 10, sizeParam.Schema.Default)
}

// TestWebhookWithTypeFormatOverride tests webhooks with type/format overrides.
func TestWebhookWithTypeFormatOverride(t *testing.T) {
	b := New(parser.OASVersion310).
		SetTitle("Webhook API").
		SetVersion("1.0.0").
		AddWebhook("user-created", http.MethodPost,
			WithQueryParam("user_id", "",
				WithParamFormat("uuid"),
				WithParamRequired(true),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	require.NotNil(t, doc.Webhooks)
	require.Contains(t, doc.Webhooks, "user-created")
	params := doc.Webhooks["user-created"].Post.Parameters
	require.Len(t, params, 1)
	param := params[0]
	require.NotNil(t, param.Schema)
	assert.Equal(t, "string", param.Schema.Type)
	assert.Equal(t, "uuid", param.Schema.Format)
}
