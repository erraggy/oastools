package maputil

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSortedKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]bool
		expected []string
	}{
		{
			name:     "sorted keys",
			input:    map[string]bool{"zebra": true, "apple": true, "mango": true},
			expected: []string{"apple", "mango", "zebra"},
		},
		{
			name:     "single key",
			input:    map[string]bool{"only": true},
			expected: []string{"only"},
		},
		{
			name:     "empty map",
			input:    map[string]bool{},
			expected: []string{},
		},
		{
			name:     "nil map",
			input:    nil,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SortedKeys(tt.input)
			assert.Equal(t, tt.expected, got, "SortedKeys(%v)", tt.input)
		})
	}
}

func TestSortedKeys_StringValues(t *testing.T) {
	input := map[string]string{"c": "3", "a": "1", "b": "2"}
	got := SortedKeys(input)
	expected := []string{"a", "b", "c"}
	assert.Equal(t, expected, got, "SortedKeys(%v)", input)
}

func TestSortedKeys_PointerValues(t *testing.T) {
	type item struct{ name string }
	input := map[string]*item{"z": {name: "z"}, "a": {name: "a"}}
	got := SortedKeys(input)
	expected := []string{"a", "z"}
	assert.Equal(t, expected, got, "SortedKeys(pointer map)")
}

// TestSortedKeysReturnsEmptyNotNil pins the empty result. slices.Sorted over
// maps.Keys would return nil here, which serializes as null rather than as an
// empty list, so the difference is visible to any caller that marshals the
// result and is worth holding still.
func TestSortedKeysReturnsEmptyNotNil(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input map[string]bool
	}{
		{"nil map", nil},
		{"empty map", map[string]bool{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := SortedKeys(tc.input)
			assert.NotNil(t, got, "an empty result should still be an allocated slice")
			assert.Empty(t, got)

			marshaled, err := json.Marshal(got)
			require.NoError(t, err)
			assert.JSONEq(t, `[]`, string(marshaled))
		})
	}
}
