package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParameterMarshalJSON tests Parameter.MarshalJSON.
func TestParameterMarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		param       *Parameter
		checkFields map[string]any // Fields to verify in output
	}{
		{
			name: "simple query parameter without Extra",
			param: &Parameter{
				Name:        "limit",
				In:          "query",
				Description: "Maximum number of results",
				Required:    false,
				Schema:      &Schema{Type: "integer"},
			},
			checkFields: map[string]any{
				"name":        "limit",
				"in":          "query",
				"description": "Maximum number of results",
			},
		},
		{
			name: "required path parameter",
			param: &Parameter{
				Name:     "id",
				In:       "path",
				Required: true,
				Schema:   &Schema{Type: "string"},
			},
			checkFields: map[string]any{
				"name":     "id",
				"in":       "path",
				"required": true,
			},
		},
		{
			name: "parameter with Extra fields",
			param: &Parameter{
				Name: "api_key",
				In:   "header",
				Extra: map[string]any{
					"x-example": "Bearer token",
				},
			},
			checkFields: map[string]any{
				"name":      "api_key",
				"in":        "header",
				"x-example": "Bearer token",
			},
		},
		{
			name: "deprecated parameter",
			param: &Parameter{
				Name:       "oldParam",
				In:         "query",
				Deprecated: true,
				Schema:     &Schema{Type: "string"},
			},
			checkFields: map[string]any{
				"name":       "oldParam",
				"in":         "query",
				"deprecated": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.param)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			for key, expected := range tt.checkFields {
				assert.Equal(t, expected, result[key], "Field %s should match", key)
			}
		})
	}
}

// TestParameterUnmarshalJSON tests Parameter.UnmarshalJSON.
func TestParameterUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Parameter
	}{
		{
			name:  "simple query parameter",
			input: `{"name":"limit","in":"query","description":"Maximum number of results","schema":{"type":"integer"}}`,
			expected: &Parameter{
				Name:        "limit",
				In:          "query",
				Description: "Maximum number of results",
				Schema:      &Schema{Type: "integer"},
			},
		},
		{
			name:  "parameter with x- extensions",
			input: `{"name":"api_key","in":"header","x-example":"Bearer token"}`,
			expected: &Parameter{
				Name: "api_key",
				In:   "header",
				Extra: map[string]any{
					"x-example": "Bearer token",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var param Parameter
			err := json.Unmarshal([]byte(tt.input), &param)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.Name, param.Name, "Name should match")
			assert.Equal(t, tt.expected.In, param.In, "In should match")
			assert.Equal(t, tt.expected.Description, param.Description, "Description should match")
			assert.Equal(t, tt.expected.Extra, param.Extra, "Extra fields should match")
		})
	}
}

// TestItemsMarshalJSON tests Items.MarshalJSON.
func TestItemsMarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		items       *Items
		checkFields map[string]any
	}{
		{
			name: "simple items without Extra",
			items: &Items{
				Type: "string",
			},
			checkFields: map[string]any{
				"type": "string",
			},
		},
		{
			name: "items with format",
			items: &Items{
				Type:   "string",
				Format: "date-time",
			},
			checkFields: map[string]any{
				"type":   "string",
				"format": "date-time",
			},
		},
		{
			name: "items with Extra fields",
			items: &Items{
				Type: "integer",
				Extra: map[string]any{
					"x-custom": "value",
				},
			},
			checkFields: map[string]any{
				"type":     "integer",
				"x-custom": "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.items)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			for key, expected := range tt.checkFields {
				assert.Equal(t, expected, result[key], "Field %s should match", key)
			}
		})
	}
}

// TestItemsUnmarshalJSON tests Items.UnmarshalJSON.
func TestItemsUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Items
	}{
		{
			name:  "simple items",
			input: `{"type":"string"}`,
			expected: &Items{
				Type: "string",
			},
		},
		{
			name:  "items with x- extensions",
			input: `{"type":"integer","x-custom":"value"}`,
			expected: &Items{
				Type: "integer",
				Extra: map[string]any{
					"x-custom": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var items Items
			err := json.Unmarshal([]byte(tt.input), &items)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.Type, items.Type, "Type should match")
			assert.Equal(t, tt.expected.Extra, items.Extra, "Extra fields should match")
		})
	}
}

// TestRequestBodyMarshalJSON tests RequestBody.MarshalJSON.
func TestRequestBodyMarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		requestBody *RequestBody
		checkFields map[string]any
	}{
		{
			name: "simple request body without Extra",
			requestBody: &RequestBody{
				Description: "User object",
				Required:    true,
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{Type: "object"},
					},
				},
			},
			checkFields: map[string]any{
				"description": "User object",
				"required":    true,
			},
		},
		{
			name: "request body with Extra fields",
			requestBody: &RequestBody{
				Description: "Pet object",
				Content: map[string]*MediaType{
					"application/json": {Schema: &Schema{Type: "object"}},
				},
				Extra: map[string]any{
					"x-example": "pet-example",
				},
			},
			checkFields: map[string]any{
				"description": "Pet object",
				"x-example":   "pet-example",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.requestBody)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			for key, expected := range tt.checkFields {
				assert.Equal(t, expected, result[key], "Field %s should match", key)
			}
		})
	}
}

// TestRequestBodyUnmarshalJSON tests RequestBody.UnmarshalJSON.
func TestRequestBodyUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *RequestBody
	}{
		{
			name:  "simple request body",
			input: `{"description":"User object","required":true}`,
			expected: &RequestBody{
				Description: "User object",
				Required:    true,
			},
		},
		{
			name:  "request body with x- extensions",
			input: `{"description":"Pet object","x-example":"pet-example"}`,
			expected: &RequestBody{
				Description: "Pet object",
				Extra: map[string]any{
					"x-example": "pet-example",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestBody RequestBody
			err := json.Unmarshal([]byte(tt.input), &requestBody)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.Description, requestBody.Description, "Description should match")
			assert.Equal(t, tt.expected.Required, requestBody.Required, "Required should match")
			assert.Equal(t, tt.expected.Extra, requestBody.Extra, "Extra fields should match")
		})
	}
}

// TestHeaderMarshalJSON tests Header.MarshalJSON.
func TestHeaderMarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		header      *Header
		checkFields map[string]any
	}{
		{
			name: "simple header without Extra",
			header: &Header{
				Description: "API version",
				Required:    true,
				Schema:      &Schema{Type: "string"},
			},
			checkFields: map[string]any{
				"description": "API version",
				"required":    true,
			},
		},
		{
			name: "header with Extra fields",
			header: &Header{
				Description: "Request ID",
				Schema:      &Schema{Type: "string"},
				Extra: map[string]any{
					"x-example": "req-12345",
				},
			},
			checkFields: map[string]any{
				"description": "Request ID",
				"x-example":   "req-12345",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.header)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			for key, expected := range tt.checkFields {
				assert.Equal(t, expected, result[key], "Field %s should match", key)
			}
		})
	}
}

// TestHeaderUnmarshalJSON tests Header.UnmarshalJSON.
func TestHeaderUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Header
	}{
		{
			name:  "simple header",
			input: `{"description":"API version","required":true}`,
			expected: &Header{
				Description: "API version",
				Required:    true,
			},
		},
		{
			name:  "header with x- extensions",
			input: `{"description":"Request ID","x-example":"req-12345"}`,
			expected: &Header{
				Description: "Request ID",
				Extra: map[string]any{
					"x-example": "req-12345",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var header Header
			err := json.Unmarshal([]byte(tt.input), &header)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.Description, header.Description, "Description should match")
			assert.Equal(t, tt.expected.Required, header.Required, "Required should match")
			assert.Equal(t, tt.expected.Extra, header.Extra, "Extra fields should match")
		})
	}
}

// TestParametersJSONRoundTrip tests that marshal/unmarshal round-trips preserve data.
func TestParametersJSONRoundTrip(t *testing.T) {
	t.Run("Parameter round-trip", func(t *testing.T) {
		original := &Parameter{
			Name:        "limit",
			In:          "query",
			Description: "Maximum number of results",
			Required:    false,
			Schema:      &Schema{Type: "integer"},
			Extra: map[string]any{
				"x-example": "100",
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded Parameter
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Name, decoded.Name)
		assert.Equal(t, original.In, decoded.In)
		assert.Equal(t, original.Description, decoded.Description)
		assert.Equal(t, original.Required, decoded.Required)
		assert.Equal(t, original.Extra, decoded.Extra)
	})

	t.Run("Items round-trip", func(t *testing.T) {
		original := &Items{
			Type:   "string",
			Format: "date-time",
			Extra: map[string]any{
				"x-custom": "value",
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded Items
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Type, decoded.Type)
		assert.Equal(t, original.Format, decoded.Format)
		assert.Equal(t, original.Extra, decoded.Extra)
	})

	t.Run("RequestBody round-trip", func(t *testing.T) {
		original := &RequestBody{
			Description: "User object",
			Required:    true,
			Extra: map[string]any{
				"x-example": "user-example",
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded RequestBody
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Description, decoded.Description)
		assert.Equal(t, original.Required, decoded.Required)
		assert.Equal(t, original.Extra, decoded.Extra)
	})

	t.Run("Header round-trip", func(t *testing.T) {
		original := &Header{
			Description: "API version",
			Required:    true,
			Schema:      &Schema{Type: "string"},
			Extra: map[string]any{
				"x-example": "v1",
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded Header
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Description, decoded.Description)
		assert.Equal(t, original.Required, decoded.Required)
		assert.Equal(t, original.Extra, decoded.Extra)
	})
}
