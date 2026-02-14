package maputil

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
