package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLicenseMarshalJSON tests License.MarshalJSON with and without Extra fields.
func TestLicenseMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		license  *License
		expected map[string]any
	}{
		{
			name: "license without Extra fields",
			license: &License{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
			expected: map[string]any{
				"name": "MIT",
				"url":  "https://opensource.org/licenses/MIT",
			},
		},
		{
			name: "license with Extra fields",
			license: &License{
				Name: "Apache 2.0",
				URL:  "https://www.apache.org/licenses/LICENSE-2.0.html",
				Extra: map[string]any{
					"x-custom":  "value",
					"x-version": "2.0",
				},
			},
			expected: map[string]any{
				"name":      "Apache 2.0",
				"url":       "https://www.apache.org/licenses/LICENSE-2.0.html",
				"x-custom":  "value",
				"x-version": "2.0",
			},
		},
		{
			name: "license with identifier (OAS 3.1+)",
			license: &License{
				Name:       "MIT",
				Identifier: "MIT",
			},
			expected: map[string]any{
				"name":       "MIT",
				"identifier": "MIT",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.license)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			assert.Equal(t, tt.expected, result, "Marshaled JSON should match expected")
		})
	}
}

// TestLicenseUnmarshalJSON tests License.UnmarshalJSON with and without extension fields.
func TestLicenseUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *License
	}{
		{
			name:  "license without extensions",
			input: `{"name":"MIT","url":"https://opensource.org/licenses/MIT"}`,
			expected: &License{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
		},
		{
			name:  "license with x- extensions",
			input: `{"name":"Apache 2.0","url":"https://www.apache.org/licenses/LICENSE-2.0.html","x-custom":"value","x-version":"2.0"}`,
			expected: &License{
				Name: "Apache 2.0",
				URL:  "https://www.apache.org/licenses/LICENSE-2.0.html",
				Extra: map[string]any{
					"x-custom":  "value",
					"x-version": "2.0",
				},
			},
		},
		{
			name:  "license with identifier",
			input: `{"name":"MIT","identifier":"MIT"}`,
			expected: &License{
				Name:       "MIT",
				Identifier: "MIT",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var license License
			err := json.Unmarshal([]byte(tt.input), &license)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.Name, license.Name, "Name should match")
			assert.Equal(t, tt.expected.URL, license.URL, "URL should match")
			assert.Equal(t, tt.expected.Identifier, license.Identifier, "Identifier should match")
			assert.Equal(t, tt.expected.Extra, license.Extra, "Extra fields should match")
		})
	}
}

// TestExternalDocsMarshalJSON tests ExternalDocs.MarshalJSON.
func TestExternalDocsMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		docs     *ExternalDocs
		expected map[string]any
	}{
		{
			name: "external docs without Extra",
			docs: &ExternalDocs{
				URL:         "https://example.com/docs",
				Description: "API Documentation",
			},
			expected: map[string]any{
				"url":         "https://example.com/docs",
				"description": "API Documentation",
			},
		},
		{
			name: "external docs with Extra",
			docs: &ExternalDocs{
				URL: "https://example.com/docs",
				Extra: map[string]any{
					"x-internal": true,
					"x-team":     "platform",
				},
			},
			expected: map[string]any{
				"url":        "https://example.com/docs",
				"x-internal": true,
				"x-team":     "platform",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.docs)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			assert.Equal(t, tt.expected, result, "Marshaled JSON should match expected")
		})
	}
}

// TestExternalDocsUnmarshalJSON tests ExternalDocs.UnmarshalJSON.
func TestExternalDocsUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *ExternalDocs
	}{
		{
			name:  "external docs without extensions",
			input: `{"url":"https://example.com/docs","description":"API Documentation"}`,
			expected: &ExternalDocs{
				URL:         "https://example.com/docs",
				Description: "API Documentation",
			},
		},
		{
			name:  "external docs with x- extensions",
			input: `{"url":"https://example.com/docs","x-internal":true,"x-team":"platform"}`,
			expected: &ExternalDocs{
				URL: "https://example.com/docs",
				Extra: map[string]any{
					"x-internal": true,
					"x-team":     "platform",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var docs ExternalDocs
			err := json.Unmarshal([]byte(tt.input), &docs)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.URL, docs.URL, "URL should match")
			assert.Equal(t, tt.expected.Description, docs.Description, "Description should match")
			assert.Equal(t, tt.expected.Extra, docs.Extra, "Extra fields should match")
		})
	}
}

// TestTagMarshalJSON tests Tag.MarshalJSON.
func TestTagMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		tag      *Tag
		expected map[string]any
	}{
		{
			name: "tag without Extra",
			tag: &Tag{
				Name:        "pets",
				Description: "Pet operations",
			},
			expected: map[string]any{
				"name":        "pets",
				"description": "Pet operations",
			},
		},
		{
			name: "tag with ExternalDocs and Extra",
			tag: &Tag{
				Name: "users",
				ExternalDocs: &ExternalDocs{
					URL: "https://example.com/users",
				},
				Extra: map[string]any{
					"x-category": "core",
				},
			},
			expected: map[string]any{
				"name": "users",
				"externalDocs": map[string]any{
					"url": "https://example.com/users",
				},
				"x-category": "core",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.tag)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			// Compare name and description
			assert.Equal(t, tt.expected["name"], result["name"], "Name should match")
			if desc, ok := tt.expected["description"]; ok {
				assert.Equal(t, desc, result["description"], "Description should match")
			}

			// Compare x- fields
			for k, v := range tt.expected {
				if len(k) >= 2 && k[0] == 'x' && k[1] == '-' {
					assert.Equal(t, v, result[k], "Extension field %s should match", k)
				}
			}
		})
	}
}

// TestTagUnmarshalJSON tests Tag.UnmarshalJSON.
func TestTagUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Tag
	}{
		{
			name:  "tag without extensions",
			input: `{"name":"pets","description":"Pet operations"}`,
			expected: &Tag{
				Name:        "pets",
				Description: "Pet operations",
			},
		},
		{
			name:  "tag with x- extensions",
			input: `{"name":"users","x-category":"core","x-internal":true}`,
			expected: &Tag{
				Name: "users",
				Extra: map[string]any{
					"x-category": "core",
					"x-internal": true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tag Tag
			err := json.Unmarshal([]byte(tt.input), &tag)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.Name, tag.Name, "Name should match")
			assert.Equal(t, tt.expected.Description, tag.Description, "Description should match")
			assert.Equal(t, tt.expected.Extra, tag.Extra, "Extra fields should match")
		})
	}
}

// TestServerVariableMarshalJSON tests ServerVariable.MarshalJSON.
func TestServerVariableMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		variable *ServerVariable
		expected map[string]any
	}{
		{
			name: "server variable without Extra",
			variable: &ServerVariable{
				Default:     "v1",
				Enum:        []string{"v1", "v2"},
				Description: "API version",
			},
			expected: map[string]any{
				"default":     "v1",
				"enum":        []any{"v1", "v2"},
				"description": "API version",
			},
		},
		{
			name: "server variable with Extra",
			variable: &ServerVariable{
				Default: "https",
				Enum:    []string{"http", "https"},
				Extra: map[string]any{
					"x-deprecated": false,
				},
			},
			expected: map[string]any{
				"default":      "https",
				"enum":         []any{"http", "https"},
				"x-deprecated": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.variable)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			assert.Equal(t, tt.expected["default"], result["default"], "Default should match")
		})
	}
}

// TestServerVariableUnmarshalJSON tests ServerVariable.UnmarshalJSON.
func TestServerVariableUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *ServerVariable
	}{
		{
			name:  "server variable without extensions",
			input: `{"default":"v1","enum":["v1","v2"],"description":"API version"}`,
			expected: &ServerVariable{
				Default:     "v1",
				Enum:        []string{"v1", "v2"},
				Description: "API version",
			},
		},
		{
			name:  "server variable with x- extensions",
			input: `{"default":"https","enum":["http","https"],"x-deprecated":false}`,
			expected: &ServerVariable{
				Default: "https",
				Enum:    []string{"http", "https"},
				Extra: map[string]any{
					"x-deprecated": false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var variable ServerVariable
			err := json.Unmarshal([]byte(tt.input), &variable)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.Default, variable.Default, "Default should match")
			assert.Equal(t, tt.expected.Enum, variable.Enum, "Enum should match")
			assert.Equal(t, tt.expected.Description, variable.Description, "Description should match")
			assert.Equal(t, tt.expected.Extra, variable.Extra, "Extra fields should match")
		})
	}
}

// TestReferenceMarshalJSON tests Reference.MarshalJSON.
func TestReferenceMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		ref      *Reference
		expected map[string]any
	}{
		{
			name: "reference without Extra",
			ref: &Reference{
				Ref: "#/components/schemas/Pet",
			},
			expected: map[string]any{
				"$ref": "#/components/schemas/Pet",
			},
		},
		{
			name: "reference with summary and description",
			ref: &Reference{
				Ref:         "#/components/schemas/User",
				Summary:     "User reference",
				Description: "Reference to User schema",
			},
			expected: map[string]any{
				"$ref":        "#/components/schemas/User",
				"summary":     "User reference",
				"description": "Reference to User schema",
			},
		},
		{
			name: "reference with Extra",
			ref: &Reference{
				Ref: "#/components/parameters/UserId",
				Extra: map[string]any{
					"x-internal": true,
				},
			},
			expected: map[string]any{
				"$ref":       "#/components/parameters/UserId",
				"x-internal": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.ref)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			assert.Equal(t, tt.expected, result, "Marshaled JSON should match expected")
		})
	}
}

// TestReferenceUnmarshalJSON tests Reference.UnmarshalJSON.
func TestReferenceUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Reference
	}{
		{
			name:  "reference without extensions",
			input: `{"$ref":"#/components/schemas/Pet"}`,
			expected: &Reference{
				Ref: "#/components/schemas/Pet",
			},
		},
		{
			name:  "reference with summary and description",
			input: `{"$ref":"#/components/schemas/User","summary":"User reference","description":"Reference to User schema"}`,
			expected: &Reference{
				Ref:         "#/components/schemas/User",
				Summary:     "User reference",
				Description: "Reference to User schema",
			},
		},
		{
			name:  "reference with x- extensions",
			input: `{"$ref":"#/components/parameters/UserId","x-internal":true}`,
			expected: &Reference{
				Ref: "#/components/parameters/UserId",
				Extra: map[string]any{
					"x-internal": true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ref Reference
			err := json.Unmarshal([]byte(tt.input), &ref)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.Ref, ref.Ref, "Ref should match")
			assert.Equal(t, tt.expected.Summary, ref.Summary, "Summary should match")
			assert.Equal(t, tt.expected.Description, ref.Description, "Description should match")
			assert.Equal(t, tt.expected.Extra, ref.Extra, "Extra fields should match")
		})
	}
}

// TestJSONRoundTripCommonTypes tests that marshal/unmarshal round-trips preserve data.
func TestJSONRoundTripCommonTypes(t *testing.T) {
	t.Run("License round-trip", func(t *testing.T) {
		original := &License{
			Name:       "MIT",
			URL:        "https://opensource.org/licenses/MIT",
			Identifier: "MIT",
			Extra: map[string]any{
				"x-custom": "value",
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded License
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Name, decoded.Name)
		assert.Equal(t, original.URL, decoded.URL)
		assert.Equal(t, original.Identifier, decoded.Identifier)
		assert.Equal(t, original.Extra, decoded.Extra)
	})

	t.Run("ExternalDocs round-trip", func(t *testing.T) {
		original := &ExternalDocs{
			URL:         "https://example.com/docs",
			Description: "API Documentation",
			Extra: map[string]any{
				"x-internal": true,
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded ExternalDocs
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.URL, decoded.URL)
		assert.Equal(t, original.Description, decoded.Description)
		assert.Equal(t, original.Extra, decoded.Extra)
	})

	t.Run("Tag round-trip", func(t *testing.T) {
		original := &Tag{
			Name:        "pets",
			Description: "Pet operations",
			ExternalDocs: &ExternalDocs{
				URL: "https://example.com/pets",
			},
			Extra: map[string]any{
				"x-category": "core",
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded Tag
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Name, decoded.Name)
		assert.Equal(t, original.Description, decoded.Description)
		assert.Equal(t, original.Extra, decoded.Extra)
	})

	t.Run("Reference round-trip", func(t *testing.T) {
		original := &Reference{
			Ref:         "#/components/schemas/Pet",
			Summary:     "Pet reference",
			Description: "Reference to Pet schema",
			Extra: map[string]any{
				"x-internal": false,
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded Reference
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Ref, decoded.Ref)
		assert.Equal(t, original.Summary, decoded.Summary)
		assert.Equal(t, original.Description, decoded.Description)
		assert.Equal(t, original.Extra, decoded.Extra)
	})
}
