package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiscriminatorMarshalJSON tests Discriminator.MarshalJSON.
func TestDiscriminatorMarshalJSON(t *testing.T) {
	tests := []struct {
		name          string
		discriminator *Discriminator
		expected      map[string]any
	}{
		{
			name: "discriminator without Extra",
			discriminator: &Discriminator{
				PropertyName: "petType",
			},
			expected: map[string]any{
				"propertyName": "petType",
			},
		},
		{
			name: "discriminator with mapping",
			discriminator: &Discriminator{
				PropertyName: "objectType",
				Mapping: map[string]string{
					"dog": "#/components/schemas/Dog",
					"cat": "#/components/schemas/Cat",
				},
			},
			expected: map[string]any{
				"propertyName": "objectType",
				"mapping": map[string]any{
					"dog": "#/components/schemas/Dog",
					"cat": "#/components/schemas/Cat",
				},
			},
		},
		{
			name: "discriminator with Extra fields",
			discriminator: &Discriminator{
				PropertyName: "type",
				Extra: map[string]any{
					"x-custom": "value",
				},
			},
			expected: map[string]any{
				"propertyName": "type",
				"x-custom":     "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.discriminator)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			assert.Equal(t, tt.expected["propertyName"], result["propertyName"], "PropertyName should match")
		})
	}
}

// TestDiscriminatorUnmarshalJSON tests Discriminator.UnmarshalJSON.
func TestDiscriminatorUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *Discriminator
	}{
		{
			name:  "discriminator without extensions",
			input: `{"propertyName":"petType"}`,
			expected: &Discriminator{
				PropertyName: "petType",
			},
		},
		{
			name:  "discriminator with mapping",
			input: `{"propertyName":"objectType","mapping":{"dog":"#/components/schemas/Dog","cat":"#/components/schemas/Cat"}}`,
			expected: &Discriminator{
				PropertyName: "objectType",
				Mapping: map[string]string{
					"dog": "#/components/schemas/Dog",
					"cat": "#/components/schemas/Cat",
				},
			},
		},
		{
			name:  "discriminator with x- extensions",
			input: `{"propertyName":"type","x-custom":"value"}`,
			expected: &Discriminator{
				PropertyName: "type",
				Extra: map[string]any{
					"x-custom": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var discriminator Discriminator
			err := json.Unmarshal([]byte(tt.input), &discriminator)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.PropertyName, discriminator.PropertyName, "PropertyName should match")
			assert.Equal(t, tt.expected.Mapping, discriminator.Mapping, "Mapping should match")
			assert.Equal(t, tt.expected.Extra, discriminator.Extra, "Extra fields should match")
		})
	}
}

// TestXMLMarshalJSON tests XML.MarshalJSON.
func TestXMLMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		xml      *XML
		expected map[string]any
	}{
		{
			name: "XML without Extra",
			xml: &XML{
				Name: "animal",
			},
			expected: map[string]any{
				"name": "animal",
			},
		},
		{
			name: "XML with namespace",
			xml: &XML{
				Name:      "pet",
				Namespace: "http://example.com/schema/pet",
				Prefix:    "pet",
			},
			expected: map[string]any{
				"name":      "pet",
				"namespace": "http://example.com/schema/pet",
				"prefix":    "pet",
			},
		},
		{
			name: "XML with attribute flag",
			xml: &XML{
				Name:      "id",
				Attribute: true,
			},
			expected: map[string]any{
				"name":      "id",
				"attribute": true,
			},
		},
		{
			name: "XML with wrapped flag",
			xml: &XML{
				Name:    "pets",
				Wrapped: true,
			},
			expected: map[string]any{
				"name":    "pets",
				"wrapped": true,
			},
		},
		{
			name: "XML with Extra fields",
			xml: &XML{
				Name: "item",
				Extra: map[string]any{
					"x-custom": "value",
				},
			},
			expected: map[string]any{
				"name":     "item",
				"x-custom": "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.xml)
			require.NoError(t, err, "MarshalJSON should not error")

			var result map[string]any
			err = json.Unmarshal(data, &result)
			require.NoError(t, err, "Unmarshaling result should not error")

			assert.Equal(t, tt.expected["name"], result["name"], "Name should match")
		})
	}
}

// TestXMLUnmarshalJSON tests XML.UnmarshalJSON.
func TestXMLUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *XML
	}{
		{
			name:  "XML without extensions",
			input: `{"name":"animal"}`,
			expected: &XML{
				Name: "animal",
			},
		},
		{
			name:  "XML with namespace",
			input: `{"name":"pet","namespace":"http://example.com/schema/pet","prefix":"pet"}`,
			expected: &XML{
				Name:      "pet",
				Namespace: "http://example.com/schema/pet",
				Prefix:    "pet",
			},
		},
		{
			name:  "XML with attribute flag",
			input: `{"name":"id","attribute":true}`,
			expected: &XML{
				Name:      "id",
				Attribute: true,
			},
		},
		{
			name:  "XML with x- extensions",
			input: `{"name":"item","x-custom":"value"}`,
			expected: &XML{
				Name: "item",
				Extra: map[string]any{
					"x-custom": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var xml XML
			err := json.Unmarshal([]byte(tt.input), &xml)
			require.NoError(t, err, "UnmarshalJSON should not error")

			assert.Equal(t, tt.expected.Name, xml.Name, "Name should match")
			assert.Equal(t, tt.expected.Namespace, xml.Namespace, "Namespace should match")
			assert.Equal(t, tt.expected.Prefix, xml.Prefix, "Prefix should match")
			assert.Equal(t, tt.expected.Attribute, xml.Attribute, "Attribute should match")
			assert.Equal(t, tt.expected.Wrapped, xml.Wrapped, "Wrapped should match")
			assert.Equal(t, tt.expected.Extra, xml.Extra, "Extra fields should match")
		})
	}
}

// TestSchemaJSONRoundTrip tests that marshal/unmarshal round-trips preserve data.
func TestSchemaJSONRoundTrip(t *testing.T) {
	t.Run("Discriminator round-trip", func(t *testing.T) {
		original := &Discriminator{
			PropertyName: "petType",
			Mapping: map[string]string{
				"dog": "#/components/schemas/Dog",
			},
			Extra: map[string]any{
				"x-custom": "value",
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded Discriminator
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.PropertyName, decoded.PropertyName)
		assert.Equal(t, original.Mapping, decoded.Mapping)
		assert.Equal(t, original.Extra, decoded.Extra)
	})

	t.Run("XML round-trip", func(t *testing.T) {
		original := &XML{
			Name:      "pet",
			Namespace: "http://example.com/schema/pet",
			Prefix:    "pet",
			Attribute: false,
			Wrapped:   true,
			Extra: map[string]any{
				"x-example": "value",
			},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded XML
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Name, decoded.Name)
		assert.Equal(t, original.Namespace, decoded.Namespace)
		assert.Equal(t, original.Prefix, decoded.Prefix)
		assert.Equal(t, original.Attribute, decoded.Attribute)
		assert.Equal(t, original.Wrapped, decoded.Wrapped)
		assert.Equal(t, original.Extra, decoded.Extra)
	})
}
