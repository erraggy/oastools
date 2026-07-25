package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// leakedSiblings are the keys that a $ref object used to emit alongside its
// $ref: name and in from Parameter, description from Response, and content from
// RequestBody. Each was a required field tagged without omitempty, so it was
// written even when empty, corrupting the object on every round trip.
var leakedSiblings = []string{"name", "in", "description", "content"}

// TestRefObjectsSerializeWithoutEmptySiblings pins that a $ref object round
// trips as $ref alone.
//
// Both encodings and both marshaling paths are covered deliberately. YAML goes
// through the struct tags while JSON has custom marshalers, and those marshalers
// take a fast path via the struct tags only when Extra is empty — so an x-
// extension on the object selects an entirely different code path. Before this
// was fixed the two paths disagreed, which meant the serialized shape of a $ref
// depended on whether it happened to carry an extension.
func TestRefObjectsSerializeWithoutEmptySiblings(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{
			name:  "parameter",
			value: &Parameter{Ref: "#/components/parameters/petIdParam"},
		},
		{
			name: "parameter with extension",
			value: &Parameter{
				Ref:   "#/components/parameters/petIdParam",
				Extra: map[string]any{"x-note": "hi"},
			},
		},
		{
			name:  "response",
			value: &Response{Ref: "#/components/responses/NotFound"},
		},
		{
			name: "response with extension",
			value: &Response{
				Ref:   "#/components/responses/NotFound",
				Extra: map[string]any{"x-note": "hi"},
			},
		},
		{
			name:  "request body",
			value: &RequestBody{Ref: "#/components/requestBodies/PetBody"},
		},
		{
			name: "request body with extension",
			value: &RequestBody{
				Ref:   "#/components/requestBodies/PetBody",
				Extra: map[string]any{"x-note": "hi"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/json", func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, json.Unmarshal(data, &got))
			assertOnlyRefAndExtensions(t, got, string(data))
		})

		t.Run(tt.name+"/yaml", func(t *testing.T) {
			data, err := yaml.Marshal(tt.value)
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, yaml.Unmarshal(data, &got))
			assertOnlyRefAndExtensions(t, got, string(data))
		})
	}
}

// assertOnlyRefAndExtensions checks that the serialized object carries its $ref
// and any x- extensions, and nothing else.
func assertOnlyRefAndExtensions(t *testing.T, got map[string]any, encoded string) {
	t.Helper()

	assert.Contains(t, got, "$ref", "encoded: %s", encoded)
	for _, key := range leakedSiblings {
		assert.NotContains(t, got, key, "$ref object leaked empty %q sibling; encoded: %s", key, encoded)
	}
}

// TestInlineObjectsStillSerializeRequiredFields guards the other direction: the
// omitempty tags added for $ref objects must not drop fields that an inline
// object genuinely sets.
func TestInlineObjectsStillSerializeRequiredFields(t *testing.T) {
	param := &Parameter{Name: "petId", In: ParamInPath, Required: true, Schema: &Schema{Type: "string"}}
	response := &Response{Description: "Not found"}
	requestBody := &RequestBody{Content: map[string]*MediaType{"application/json": {}}}

	tests := []struct {
		name  string
		value any
		want  []string
	}{
		{name: "parameter", value: param, want: []string{"name", "in", "required"}},
		{name: "parameter with extension", value: withExtra(param, "x-note"), want: []string{"name", "in", "required"}},
		{name: "response", value: response, want: []string{"description"}},
		{name: "response with extension", value: withExtra(response, "x-note"), want: []string{"description"}},
		{name: "request body", value: requestBody, want: []string{"content"}},
		{name: "request body with extension", value: withExtra(requestBody, "x-note"), want: []string{"content"}},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/json", func(t *testing.T) {
			data, err := json.Marshal(tt.value)
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, json.Unmarshal(data, &got))
			for _, key := range tt.want {
				assert.Contains(t, got, key, "encoded: %s", data)
			}
		})

		t.Run(tt.name+"/yaml", func(t *testing.T) {
			data, err := yaml.Marshal(tt.value)
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, yaml.Unmarshal(data, &got))
			for _, key := range tt.want {
				assert.Contains(t, got, key, "encoded: %s", data)
			}
		})
	}
}

// withExtra returns a deep copy of value carrying a single specification
// extension, so the JSON marshalers take their Extra-fields path.
func withExtra(value any, key string) any {
	switch v := value.(type) {
	case *Parameter:
		cp := v.DeepCopy()
		cp.Extra = map[string]any{key: "set"}
		return cp
	case *Response:
		cp := v.DeepCopy()
		cp.Extra = map[string]any{key: "set"}
		return cp
	case *RequestBody:
		cp := v.DeepCopy()
		cp.Extra = map[string]any{key: "set"}
		return cp
	}
	return value
}

// TestRefDocumentRoundTrip exercises the fix end to end: a document whose path
// item, request body, and response are all $ref objects must survive a parse and
// re-serialize unchanged, in both encodings.
func TestRefDocumentRoundTrip(t *testing.T) {
	const spec = `openapi: 3.0.3
info:
  title: Repro
  version: "1.0.0"
components:
  parameters:
    petIdParam: {name: petId, in: path, required: true, schema: {type: string}}
  responses:
    NotFound: {description: Not found}
  requestBodies:
    PetBody: {content: {application/json: {schema: {type: object}}}}
paths:
  /pets/{petId}:
    parameters:
      - $ref: '#/components/parameters/petIdParam'
    post:
      requestBody: {$ref: '#/components/requestBodies/PetBody'}
      responses:
        "404": {$ref: '#/components/responses/NotFound'}
`

	result, err := New().ParseBytes([]byte(spec))
	require.NoError(t, err)

	doc, ok := result.OAS3Document()
	require.True(t, ok)
	require.NotNil(t, doc)

	pathItem := doc.Paths["/pets/{petId}"]
	require.NotNil(t, pathItem)
	require.Len(t, pathItem.Parameters, 1)

	refObjects := map[string]any{
		"parameter":    pathItem.Parameters[0],
		"request body": pathItem.Post.RequestBody,
		"response":     pathItem.Post.Responses.Codes["404"],
	}

	for name, obj := range refObjects {
		t.Run(name+"/json", func(t *testing.T) {
			data, err := json.Marshal(obj)
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, json.Unmarshal(data, &got))
			assertOnlyRefAndExtensions(t, got, string(data))
		})

		t.Run(name+"/yaml", func(t *testing.T) {
			data, err := yaml.Marshal(obj)
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, yaml.Unmarshal(data, &got))
			assertOnlyRefAndExtensions(t, got, string(data))
		})
	}
}
