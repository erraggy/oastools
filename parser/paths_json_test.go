package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLinkMarshalJSON tests Link.MarshalJSON with and without Extra fields.
func TestLinkMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		link     *Link
		expected map[string]any
	}{
		{
			name: "link without Extra fields",
			link: &Link{
				OperationRef: "#/paths/~1users~1{id}/get",
				Description:  "Get user by ID",
			},
			expected: map[string]any{
				"operationRef": "#/paths/~1users~1{id}/get",
				"description":  "Get user by ID",
			},
		},
		{
			name: "link with operationId",
			link: &Link{
				OperationID: "getUserById",
				Parameters: map[string]any{
					"id": "$response.body#/id",
				},
				Description: "Get user operation",
			},
			expected: map[string]any{
				"operationId": "getUserById",
				"parameters": map[string]any{
					"id": "$response.body#/id",
				},
				"description": "Get user operation",
			},
		},
		{
			name: "link with Extra fields",
			link: &Link{
				OperationID: "createUser",
				Extra: map[string]any{
					"x-custom":  "value",
					"x-version": "1.0",
				},
			},
			expected: map[string]any{
				"operationId": "createUser",
				"x-custom":    "value",
				"x-version":   "1.0",
			},
		},
		{
			name: "link with all fields",
			link: &Link{
				Ref:          "#/components/links/UserLink",
				OperationRef: "#/paths/~1users/get",
				OperationID:  "getUsers",
				Parameters: map[string]any{
					"page": "$request.query.page",
				},
				RequestBody: map[string]any{
					"name": "John",
				},
				Description: "Link to users",
				Server: &Server{
					URL: "https://api.example.com",
				},
				Extra: map[string]any{
					"x-internal": true,
				},
			},
			expected: map[string]any{
				"$ref":         "#/components/links/UserLink",
				"operationRef": "#/paths/~1users/get",
				"operationId":  "getUsers",
				"parameters": map[string]any{
					"page": "$request.query.page",
				},
				"requestBody": map[string]any{
					"name": "John",
				},
				"description": "Link to users",
				"server": map[string]any{
					"url": "https://api.example.com",
				},
				"x-internal": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.link)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			// Verify key fields are present
			for key := range tt.expected {
				assert.Contains(t, result, key, "Result should contain key: %s", key)
			}
		})
	}
}

// TestLinkUnmarshalJSON tests Link.UnmarshalJSON with and without extension fields.
func TestLinkUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Link
	}{
		{
			name:  "link without extensions",
			input: `{"operationRef":"#/paths/~1users~1{id}/get","description":"Get user"}`,
			expected: &Link{
				OperationRef: "#/paths/~1users~1{id}/get",
				Description:  "Get user",
			},
		},
		{
			name:  "link with x- extensions",
			input: `{"operationId":"getUser","x-custom":"value","x-internal":true}`,
			expected: &Link{
				OperationID: "getUser",
				Extra: map[string]any{
					"x-custom":   "value",
					"x-internal": true,
				},
			},
		},
		{
			name:  "link with parameters",
			input: `{"operationId":"createUser","parameters":{"id":"$response.body#/id"}}`,
			expected: &Link{
				OperationID: "createUser",
				Parameters: map[string]any{
					"id": "$response.body#/id",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var link Link
			err := json.Unmarshal([]byte(tt.input), &link)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.OperationRef, link.OperationRef, "OperationRef should match")
			assert.Equal(t, tt.expected.OperationID, link.OperationID, "OperationID should match")
			assert.Equal(t, tt.expected.Description, link.Description, "Description should match")
			assert.Equal(t, tt.expected.Extra, link.Extra, "Extra fields should match")
		})
	}
}

// TestMediaTypeMarshalJSON tests MediaType.MarshalJSON with and without Extra fields.
func TestMediaTypeMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		mt       *MediaType
		expected map[string]any
	}{
		{
			name: "mediaType without Extra fields",
			mt: &MediaType{
				Schema: &Schema{
					Type: "object",
				},
			},
			expected: map[string]any{
				"schema": map[string]any{
					"type": "object",
				},
			},
		},
		{
			name: "mediaType with example",
			mt: &MediaType{
				Schema: &Schema{
					Type: "string",
				},
				Example: "test value",
			},
			expected: map[string]any{
				"schema": map[string]any{
					"type": "string",
				},
				"example": "test value",
			},
		},
		{
			name: "mediaType with Extra fields",
			mt: &MediaType{
				Schema: &Schema{
					Type: "object",
				},
				Extra: map[string]any{
					"x-custom":  "value",
					"x-version": "2.0",
				},
			},
			expected: map[string]any{
				"schema": map[string]any{
					"type": "object",
				},
				"x-custom":  "value",
				"x-version": "2.0",
			},
		},
		{
			name: "mediaType with examples",
			mt: &MediaType{
				Schema: &Schema{
					Type: "string",
				},
				Examples: map[string]*Example{
					"example1": {
						Value:   "value1",
						Summary: "First example",
					},
				},
			},
			expected: map[string]any{
				"schema": map[string]any{
					"type": "string",
				},
				"examples": map[string]any{
					"example1": map[string]any{
						"value":   "value1",
						"summary": "First example",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.mt)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			// Verify key fields are present
			for key := range tt.expected {
				assert.Contains(t, result, key, "Result should contain key: %s", key)
			}
		})
	}
}

// TestMediaTypeUnmarshalJSON tests MediaType.UnmarshalJSON with and without extension fields.
func TestMediaTypeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *MediaType
	}{
		{
			name:  "mediaType without extensions",
			input: `{"schema":{"type":"string"}}`,
			expected: &MediaType{
				Schema: &Schema{
					Type: "string",
				},
			},
		},
		{
			name:  "mediaType with x- extensions",
			input: `{"schema":{"type":"object"},"x-custom":"value","x-parser":"custom"}`,
			expected: &MediaType{
				Schema: &Schema{
					Type: "object",
				},
				Extra: map[string]any{
					"x-custom": "value",
					"x-parser": "custom",
				},
			},
		},
		{
			name:  "mediaType with example",
			input: `{"schema":{"type":"string"},"example":"test"}`,
			expected: &MediaType{
				Schema: &Schema{
					Type: "string",
				},
				Example: "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mt MediaType
			err := json.Unmarshal([]byte(tt.input), &mt)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.Example, mt.Example, "Example should match")
			assert.Equal(t, tt.expected.Extra, mt.Extra, "Extra fields should match")
		})
	}
}

// TestExampleMarshalJSON tests Example.MarshalJSON with and without Extra fields.
func TestExampleMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		example  *Example
		expected map[string]any
	}{
		{
			name: "example without Extra fields",
			example: &Example{
				Summary: "Example summary",
				Value:   "example value",
			},
			expected: map[string]any{
				"summary": "Example summary",
				"value":   "example value",
			},
		},
		{
			name: "example with description",
			example: &Example{
				Summary:     "User example",
				Description: "An example user object",
				Value: map[string]any{
					"id":   1,
					"name": "John Doe",
				},
			},
			expected: map[string]any{
				"summary":     "User example",
				"description": "An example user object",
				"value": map[string]any{
					"id":   float64(1),
					"name": "John Doe",
				},
			},
		},
		{
			name: "example with Extra fields",
			example: &Example{
				Value: "test",
				Extra: map[string]any{
					"x-custom":  "value",
					"x-version": "1.0",
				},
			},
			expected: map[string]any{
				"value":     "test",
				"x-custom":  "value",
				"x-version": "1.0",
			},
		},
		{
			name: "example with externalValue",
			example: &Example{
				Summary:       "External example",
				ExternalValue: "https://example.com/example.json",
			},
			expected: map[string]any{
				"summary":       "External example",
				"externalValue": "https://example.com/example.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.example)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			// Verify key fields are present
			for key := range tt.expected {
				assert.Contains(t, result, key, "Result should contain key: %s", key)
			}
		})
	}
}

// TestExampleUnmarshalJSON tests Example.UnmarshalJSON with and without extension fields.
func TestExampleUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Example
	}{
		{
			name:  "example without extensions",
			input: `{"summary":"Test","value":"test value"}`,
			expected: &Example{
				Summary: "Test",
				Value:   "test value",
			},
		},
		{
			name:  "example with x- extensions",
			input: `{"value":"test","x-custom":"value","x-internal":true}`,
			expected: &Example{
				Value: "test",
				Extra: map[string]any{
					"x-custom":   "value",
					"x-internal": true,
				},
			},
		},
		{
			name:  "example with externalValue",
			input: `{"summary":"External","externalValue":"https://example.com/data.json"}`,
			expected: &Example{
				Summary:       "External",
				ExternalValue: "https://example.com/data.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var example Example
			err := json.Unmarshal([]byte(tt.input), &example)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.Summary, example.Summary, "Summary should match")
			assert.Equal(t, tt.expected.ExternalValue, example.ExternalValue, "ExternalValue should match")
			assert.Equal(t, tt.expected.Extra, example.Extra, "Extra fields should match")
		})
	}
}

// TestEncodingMarshalJSON tests Encoding.MarshalJSON with and without Extra fields.
func TestEncodingMarshalJSON(t *testing.T) {
	explodeTrue := true
	tests := []struct {
		name     string
		encoding *Encoding
		expected map[string]any
	}{
		{
			name: "encoding without Extra fields",
			encoding: &Encoding{
				ContentType: "application/json",
				Style:       "form",
			},
			expected: map[string]any{
				"contentType": "application/json",
				"style":       "form",
			},
		},
		{
			name: "encoding with headers",
			encoding: &Encoding{
				ContentType: "image/png",
				Headers: map[string]*Header{
					"X-Rate-Limit": {
						Description: "Rate limit header",
						Schema: &Schema{
							Type: "integer",
						},
					},
				},
			},
			expected: map[string]any{
				"contentType": "image/png",
				"headers": map[string]any{
					"X-Rate-Limit": map[string]any{
						"description": "Rate limit header",
						"schema": map[string]any{
							"type": "integer",
						},
					},
				},
			},
		},
		{
			name: "encoding with Extra fields",
			encoding: &Encoding{
				ContentType: "text/plain",
				Explode:     &explodeTrue,
				Extra: map[string]any{
					"x-custom":  "value",
					"x-version": "1.0",
				},
			},
			expected: map[string]any{
				"contentType": "text/plain",
				"explode":     true,
				"x-custom":    "value",
				"x-version":   "1.0",
			},
		},
		{
			name: "encoding with allowReserved",
			encoding: &Encoding{
				ContentType:   "application/x-www-form-urlencoded",
				AllowReserved: true,
			},
			expected: map[string]any{
				"contentType":   "application/x-www-form-urlencoded",
				"allowReserved": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.encoding)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			// Verify key fields are present
			for key := range tt.expected {
				assert.Contains(t, result, key, "Result should contain key: %s", key)
			}
		})
	}
}

// TestEncodingUnmarshalJSON tests Encoding.UnmarshalJSON with and without extension fields.
func TestEncodingUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Encoding
	}{
		{
			name:  "encoding without extensions",
			input: `{"contentType":"application/json","style":"form"}`,
			expected: &Encoding{
				ContentType: "application/json",
				Style:       "form",
			},
		},
		{
			name:  "encoding with x- extensions",
			input: `{"contentType":"text/plain","x-custom":"value","x-format":"special"}`,
			expected: &Encoding{
				ContentType: "text/plain",
				Extra: map[string]any{
					"x-custom": "value",
					"x-format": "special",
				},
			},
		},
		{
			name:  "encoding with explode",
			input: `{"contentType":"multipart/form-data","explode":true}`,
			expected: &Encoding{
				ContentType: "multipart/form-data",
				Explode:     func() *bool { b := true; return &b }(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var encoding Encoding
			err := json.Unmarshal([]byte(tt.input), &encoding)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.ContentType, encoding.ContentType, "ContentType should match")
			assert.Equal(t, tt.expected.Style, encoding.Style, "Style should match")
			assert.Equal(t, tt.expected.Extra, encoding.Extra, "Extra fields should match")
		})
	}
}
