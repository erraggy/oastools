package jsonhelpers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalWithExtras(t *testing.T) {
	t.Run("without extras", func(t *testing.T) {
		base := map[string]any{
			"name":  "test",
			"value": 42,
		}
		data, err := MarshalWithExtras(base, nil)
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "test", result["name"])
		assert.Equal(t, float64(42), result["value"])
		assert.Len(t, result, 2)
	})

	t.Run("with extras", func(t *testing.T) {
		base := map[string]any{
			"name": "test",
		}
		extras := map[string]any{
			"x-custom": "value",
			"x-count":  10,
		}
		data, err := MarshalWithExtras(base, extras)
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "test", result["name"])
		assert.Equal(t, "value", result["x-custom"])
		assert.Equal(t, float64(10), result["x-count"])
		assert.Len(t, result, 3)
	})

	t.Run("extras override base", func(t *testing.T) {
		base := map[string]any{
			"name": "original",
		}
		extras := map[string]any{
			"name": "overridden",
		}
		data, err := MarshalWithExtras(base, extras)
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "overridden", result["name"])
	})

	t.Run("empty base and extras", func(t *testing.T) {
		data, err := MarshalWithExtras(map[string]any{}, nil)
		require.NoError(t, err)
		assert.Equal(t, "{}", string(data))
	})
}

func TestUnmarshalExtras(t *testing.T) {
	t.Run("all known fields", func(t *testing.T) {
		data := map[string]any{
			"name":  "test",
			"value": 42,
		}
		knownFields := map[string]bool{
			"name":  true,
			"value": true,
		}
		extras := UnmarshalExtras(data, knownFields)
		assert.Nil(t, extras)
	})

	t.Run("with extension fields", func(t *testing.T) {
		data := map[string]any{
			"name":     "test",
			"x-custom": "value",
			"x-count":  10,
		}
		knownFields := map[string]bool{
			"name": true,
		}
		extras := UnmarshalExtras(data, knownFields)
		require.NotNil(t, extras)
		assert.Equal(t, "value", extras["x-custom"])
		assert.Equal(t, 10, extras["x-count"])
		assert.Len(t, extras, 2)
	})

	t.Run("empty data", func(t *testing.T) {
		data := map[string]any{}
		knownFields := map[string]bool{"name": true}
		extras := UnmarshalExtras(data, knownFields)
		assert.Nil(t, extras)
	})

	t.Run("no known fields", func(t *testing.T) {
		data := map[string]any{
			"x-custom": "value",
		}
		knownFields := map[string]bool{}
		extras := UnmarshalExtras(data, knownFields)
		require.NotNil(t, extras)
		assert.Equal(t, "value", extras["x-custom"])
	})
}

func TestGetString(t *testing.T) {
	t.Run("existing string field", func(t *testing.T) {
		m := map[string]any{
			"name": "test",
		}
		result := GetString(m, "name")
		assert.Equal(t, "test", result)
		assert.NotContains(t, m, "name", "field should be deleted")
	})

	t.Run("non-existent field", func(t *testing.T) {
		m := map[string]any{}
		result := GetString(m, "name")
		assert.Equal(t, "", result)
	})

	t.Run("wrong type field", func(t *testing.T) {
		m := map[string]any{
			"name": 42,
		}
		result := GetString(m, "name")
		assert.Equal(t, "", result)
		assert.NotContains(t, m, "name", "field should be deleted even if wrong type")
	})
}

func TestGetBool(t *testing.T) {
	t.Run("existing true field", func(t *testing.T) {
		m := map[string]any{
			"flag": true,
		}
		result := GetBool(m, "flag")
		assert.True(t, result)
		assert.NotContains(t, m, "flag")
	})

	t.Run("existing false field", func(t *testing.T) {
		m := map[string]any{
			"flag": false,
		}
		result := GetBool(m, "flag")
		assert.False(t, result)
		assert.NotContains(t, m, "flag")
	})

	t.Run("non-existent field", func(t *testing.T) {
		m := map[string]any{}
		result := GetBool(m, "flag")
		assert.False(t, result)
	})

	t.Run("wrong type field", func(t *testing.T) {
		m := map[string]any{
			"flag": "true",
		}
		result := GetBool(m, "flag")
		assert.False(t, result)
		assert.NotContains(t, m, "flag")
	})
}

func TestGetInt(t *testing.T) {
	t.Run("existing int field", func(t *testing.T) {
		m := map[string]any{
			"count": float64(42),
		}
		result := GetInt(m, "count")
		assert.Equal(t, 42, result)
		assert.NotContains(t, m, "count")
	})

	t.Run("non-existent field", func(t *testing.T) {
		m := map[string]any{}
		result := GetInt(m, "count")
		assert.Equal(t, 0, result)
	})

	t.Run("wrong type field", func(t *testing.T) {
		m := map[string]any{
			"count": "42",
		}
		result := GetInt(m, "count")
		assert.Equal(t, 0, result)
		assert.NotContains(t, m, "count")
	})

	t.Run("negative int", func(t *testing.T) {
		m := map[string]any{
			"count": float64(-10),
		}
		result := GetInt(m, "count")
		assert.Equal(t, -10, result)
	})
}

func TestGetFloat64(t *testing.T) {
	t.Run("existing float field", func(t *testing.T) {
		m := map[string]any{
			"value": 3.14,
		}
		result := GetFloat64(m, "value")
		assert.Equal(t, 3.14, result)
		assert.NotContains(t, m, "value")
	})

	t.Run("non-existent field", func(t *testing.T) {
		m := map[string]any{}
		result := GetFloat64(m, "value")
		assert.Equal(t, 0.0, result)
	})

	t.Run("integer as float", func(t *testing.T) {
		m := map[string]any{
			"value": float64(42),
		}
		result := GetFloat64(m, "value")
		assert.Equal(t, 42.0, result)
	})
}

func TestGetStringSlice(t *testing.T) {
	t.Run("existing slice field", func(t *testing.T) {
		m := map[string]any{
			"items": []any{"a", "b", "c"},
		}
		result := GetStringSlice(m, "items")
		assert.Equal(t, []string{"a", "b", "c"}, result)
		assert.NotContains(t, m, "items")
	})

	t.Run("non-existent field", func(t *testing.T) {
		m := map[string]any{}
		result := GetStringSlice(m, "items")
		assert.Nil(t, result)
	})

	t.Run("empty slice", func(t *testing.T) {
		m := map[string]any{
			"items": []any{},
		}
		result := GetStringSlice(m, "items")
		assert.Equal(t, []string{}, result)
	})

	t.Run("mixed types in slice", func(t *testing.T) {
		m := map[string]any{
			"items": []any{"a", 42, "b", true, "c"},
		}
		result := GetStringSlice(m, "items")
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("wrong type field", func(t *testing.T) {
		m := map[string]any{
			"items": "not a slice",
		}
		result := GetStringSlice(m, "items")
		assert.Nil(t, result)
	})
}

func TestGetStringMap(t *testing.T) {
	t.Run("existing map field", func(t *testing.T) {
		m := map[string]any{
			"props": map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
		}
		result := GetStringMap(m, "props")
		assert.Equal(t, map[string]string{
			"key1": "value1",
			"key2": "value2",
		}, result)
		assert.NotContains(t, m, "props")
	})

	t.Run("non-existent field", func(t *testing.T) {
		m := map[string]any{}
		result := GetStringMap(m, "props")
		assert.Nil(t, result)
	})

	t.Run("empty map", func(t *testing.T) {
		m := map[string]any{
			"props": map[string]any{},
		}
		result := GetStringMap(m, "props")
		assert.Equal(t, map[string]string{}, result)
	})

	t.Run("mixed types in map", func(t *testing.T) {
		m := map[string]any{
			"props": map[string]any{
				"key1": "value1",
				"key2": 42,
				"key3": "value3",
			},
		}
		result := GetStringMap(m, "props")
		assert.Equal(t, map[string]string{
			"key1": "value1",
			"key3": "value3",
		}, result)
	})

	t.Run("wrong type field", func(t *testing.T) {
		m := map[string]any{
			"props": "not a map",
		}
		result := GetStringMap(m, "props")
		assert.Nil(t, result)
	})
}

func TestGetAny(t *testing.T) {
	t.Run("string value", func(t *testing.T) {
		m := map[string]any{
			"field": "value",
		}
		result := GetAny(m, "field")
		assert.Equal(t, "value", result)
		assert.NotContains(t, m, "field")
	})

	t.Run("complex object", func(t *testing.T) {
		obj := map[string]any{
			"nested": "value",
		}
		m := map[string]any{
			"field": obj,
		}
		result := GetAny(m, "field")
		assert.Equal(t, obj, result)
	})

	t.Run("nil value", func(t *testing.T) {
		m := map[string]any{
			"field": nil,
		}
		result := GetAny(m, "field")
		assert.Nil(t, result)
	})

	t.Run("non-existent field", func(t *testing.T) {
		m := map[string]any{}
		result := GetAny(m, "field")
		assert.Nil(t, result)
	})
}

func TestSetIfNotEmpty(t *testing.T) {
	t.Run("non-empty string", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotEmpty(m, "name", "test")
		assert.Equal(t, "test", m["name"])
	})

	t.Run("empty string", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotEmpty(m, "name", "")
		assert.NotContains(t, m, "name")
	})
}

func TestSetIfNotNil(t *testing.T) {
	t.Run("non-nil value", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotNil(m, "field", "value")
		assert.Equal(t, "value", m["field"])
	})

	t.Run("nil value", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotNil(m, "field", nil)
		assert.NotContains(t, m, "field")
	})

	t.Run("zero value is not nil", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotNil(m, "field", 0)
		assert.Equal(t, 0, m["field"])
	})

	// Typed nil edge cases - these are wrapped in interface{} and are NOT equal to nil
	t.Run("typed nil pointer", func(t *testing.T) {
		m := map[string]any{}
		var nilPtr *int = nil
		SetIfNotNil(m, "ptr", nilPtr)
		assert.NotContains(t, m, "ptr", "typed nil pointer should not be added")
	})

	t.Run("typed nil slice", func(t *testing.T) {
		m := map[string]any{}
		var nilSlice []string = nil
		SetIfNotNil(m, "slice", nilSlice)
		assert.NotContains(t, m, "slice", "typed nil slice should not be added")
	})

	t.Run("typed nil map", func(t *testing.T) {
		m := map[string]any{}
		var nilMap map[string]int = nil
		SetIfNotNil(m, "map", nilMap)
		assert.NotContains(t, m, "map", "typed nil map should not be added")
	})

	t.Run("typed nil chan", func(t *testing.T) {
		m := map[string]any{}
		var nilChan chan int = nil
		SetIfNotNil(m, "chan", nilChan)
		assert.NotContains(t, m, "chan", "typed nil channel should not be added")
	})

	t.Run("typed nil func", func(t *testing.T) {
		m := map[string]any{}
		var nilFunc func() = nil
		SetIfNotNil(m, "func", nilFunc)
		assert.NotContains(t, m, "func", "typed nil function should not be added")
	})

	t.Run("typed nil interface", func(t *testing.T) {
		m := map[string]any{}
		var nilInterface error = nil
		SetIfNotNil(m, "iface", nilInterface)
		assert.NotContains(t, m, "iface", "typed nil interface should not be added")
	})

	// Non-nil typed values should still be added
	t.Run("non-nil pointer", func(t *testing.T) {
		m := map[string]any{}
		val := 42
		SetIfNotNil(m, "ptr", &val)
		assert.Contains(t, m, "ptr")
		assert.Equal(t, &val, m["ptr"])
	})

	t.Run("non-nil slice", func(t *testing.T) {
		m := map[string]any{}
		slice := []string{"a", "b"}
		SetIfNotNil(m, "slice", slice)
		assert.Contains(t, m, "slice")
		assert.Equal(t, []string{"a", "b"}, m["slice"])
	})

	t.Run("empty but non-nil slice", func(t *testing.T) {
		m := map[string]any{}
		slice := []string{}
		SetIfNotNil(m, "slice", slice)
		assert.Contains(t, m, "slice", "empty but allocated slice should be added")
	})
}

func TestSetIfNotZero(t *testing.T) {
	t.Run("non-zero value", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotZero(m, "count", 42)
		assert.Equal(t, 42, m["count"])
	})

	t.Run("zero value", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotZero(m, "count", 0)
		assert.NotContains(t, m, "count")
	})

	t.Run("negative value", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotZero(m, "count", -10)
		assert.Equal(t, -10, m["count"])
	})
}

func TestSetIfTrue(t *testing.T) {
	t.Run("true value", func(t *testing.T) {
		m := map[string]any{}
		SetIfTrue(m, "flag", true)
		assert.Equal(t, true, m["flag"])
	})

	t.Run("false value", func(t *testing.T) {
		m := map[string]any{}
		SetIfTrue(m, "flag", false)
		assert.NotContains(t, m, "flag")
	})
}

func TestExtractExtensions(t *testing.T) {
	t.Run("no extensions", func(t *testing.T) {
		data := []byte(`{"name": "test", "value": 42}`)
		result := ExtractExtensions(data)
		assert.Nil(t, result)
	})

	t.Run("with extensions", func(t *testing.T) {
		data := []byte(`{"name": "test", "x-custom": "value", "x-count": 10}`)
		result := ExtractExtensions(data)
		require.NotNil(t, result)
		assert.Equal(t, "value", result["x-custom"])
		assert.Equal(t, float64(10), result["x-count"])
		assert.Len(t, result, 2)
	})

	t.Run("only extensions", func(t *testing.T) {
		data := []byte(`{"x-one": 1, "x-two": "two"}`)
		result := ExtractExtensions(data)
		require.NotNil(t, result)
		assert.Len(t, result, 2)
	})

	t.Run("empty object", func(t *testing.T) {
		data := []byte(`{}`)
		result := ExtractExtensions(data)
		assert.Nil(t, result)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		data := []byte(`{invalid`)
		result := ExtractExtensions(data)
		assert.Nil(t, result)
	})

	// Edge cases for streaming scan optimization
	t.Run("x- in string value not a key", func(t *testing.T) {
		// Pattern "x- appears in value, not as a key
		data := []byte(`{"desc": "Use x-api-key header"}`)
		result := ExtractExtensions(data)
		assert.Nil(t, result)
	})

	t.Run("x- in nested object key", func(t *testing.T) {
		// Extension is nested, not at top level
		data := []byte(`{"nested": {"x-custom": true}}`)
		result := ExtractExtensions(data)
		assert.Nil(t, result)
	})

	t.Run("x- in array element", func(t *testing.T) {
		data := []byte(`{"tags": ["x-custom-tag", "other"]}`)
		result := ExtractExtensions(data)
		assert.Nil(t, result)
	})

	t.Run("mixed extensions and nested x-", func(t *testing.T) {
		// Top-level extension should be found, nested should be ignored
		data := []byte(`{"x-top": true, "nested": {"x-nested": false}}`)
		result := ExtractExtensions(data)
		require.NotNil(t, result)
		assert.Equal(t, true, result["x-top"])
		assert.NotContains(t, result, "x-nested")
		assert.Len(t, result, 1)
	})

	t.Run("minimum extension name x-", func(t *testing.T) {
		data := []byte(`{"x-": "empty extension name"}`)
		result := ExtractExtensions(data)
		require.NotNil(t, result)
		assert.Equal(t, "empty extension name", result["x-"])
	})

	t.Run("complex extension value", func(t *testing.T) {
		data := []byte(`{"x-config": {"nested": {"deep": true}, "array": [1, 2, 3]}}`)
		result := ExtractExtensions(data)
		require.NotNil(t, result)
		config, ok := result["x-config"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, config, "nested")
		assert.Contains(t, config, "array")
	})

	// Unicode-escaped extension keys (JSON allows \uXXXX escapes)
	t.Run("unicode escaped x in key", func(t *testing.T) {
		// \u0078 = 'x'
		data := []byte(`{"\u0078-custom": "escaped x"}`)
		result := ExtractExtensions(data)
		require.NotNil(t, result)
		assert.Equal(t, "escaped x", result["x-custom"])
	})

	t.Run("unicode escaped dash in key", func(t *testing.T) {
		// \u002d = '-'
		data := []byte(`{"x\u002dcustom": "escaped dash"}`)
		result := ExtractExtensions(data)
		require.NotNil(t, result)
		assert.Equal(t, "escaped dash", result["x-custom"])
	})

	t.Run("unicode escaped x and dash in key", func(t *testing.T) {
		// Both x and dash escaped
		data := []byte(`{"\u0078\u002dcustom": "both escaped"}`)
		result := ExtractExtensions(data)
		require.NotNil(t, result)
		assert.Equal(t, "both escaped", result["x-custom"])
	})

	t.Run("unicode escape in value not key", func(t *testing.T) {
		// Unicode escape appears in value, not as a key - should return nil
		data := []byte(`{"desc": "\u0078-api-key is required"}`)
		result := ExtractExtensions(data)
		assert.Nil(t, result)
	})
}

// Integration test: Round-trip through marshal/unmarshal
func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	type TestStruct struct {
		Name  string
		Count int
		Extra map[string]any
	}

	original := TestStruct{
		Name:  "test",
		Count: 42,
		Extra: map[string]any{
			"x-custom": "value",
			"x-count":  10,
		},
	}

	// Marshal
	base := map[string]any{
		"name":  original.Name,
		"count": original.Count,
	}
	data, err := MarshalWithExtras(base, original.Extra)
	require.NoError(t, err)

	// Unmarshal
	var temp map[string]any
	err = json.Unmarshal(data, &temp)
	require.NoError(t, err)

	result := TestStruct{
		Name:  GetString(temp, "name"),
		Count: GetInt(temp, "count"),
	}
	result.Extra = UnmarshalExtras(temp, map[string]bool{
		"name":  true,
		"count": true,
	})

	// Verify
	assert.Equal(t, original.Name, result.Name)
	assert.Equal(t, original.Count, result.Count)
	// Note: Extra values that were ints become float64 after JSON round-trip
	assert.Equal(t, "value", result.Extra["x-custom"])
	assert.Equal(t, float64(10), result.Extra["x-count"])
}

// Benchmarks for ExtractExtensions
// These demonstrate the performance benefit of the streaming scan optimization.

// BenchmarkExtractExtensions_NoExtensions benchmarks the common case where
// objects have no extensions. The streaming scan optimization should make
// this significantly faster by avoiding JSON parsing entirely.
func BenchmarkExtractExtensions_NoExtensions(b *testing.B) {
	// Typical OpenAPI Operation object without extensions
	data := []byte(`{
		"operationId": "createPaymentIntent",
		"summary": "Creates a PaymentIntent object",
		"description": "After the PaymentIntent is created, attach a payment method.",
		"tags": ["Payment Intents"],
		"parameters": [
			{"name": "amount", "in": "body", "required": true},
			{"name": "currency", "in": "body", "required": true}
		],
		"responses": {
			"200": {"description": "Returns the PaymentIntent object"},
			"400": {"description": "Bad Request"}
		}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractExtensions(data)
	}
}

// BenchmarkExtractExtensions_WithExtensions benchmarks objects that have
// extensions. The streaming scan detects the pattern and falls through to
// full JSON parsing, so performance should be similar to the old implementation.
func BenchmarkExtractExtensions_WithExtensions(b *testing.B) {
	data := []byte(`{
		"operationId": "createPaymentIntent",
		"summary": "Creates a PaymentIntent object",
		"x-stripe-resource": "payment_intent",
		"x-expandable-fields": ["customer", "invoice"],
		"x-rate-limit": 100,
		"parameters": [{"name": "amount", "in": "body"}],
		"responses": {"200": {"description": "Success"}}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractExtensions(data)
	}
}

// BenchmarkExtractExtensions_FalsePositive benchmarks the edge case where
// "x- appears in a string value (not as a key). The streaming scan will
// trigger a full parse, but still return nil extensions correctly.
func BenchmarkExtractExtensions_FalsePositive(b *testing.B) {
	data := []byte(`{
		"operationId": "getUsers",
		"description": "Use the x-api-key header for authentication",
		"responses": {"200": {"description": "Success"}}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractExtensions(data)
	}
}
