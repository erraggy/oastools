package maputil

import (
	"slices"
	"testing"
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
			if len(got) != len(tt.expected) {
				t.Errorf("SortedKeys(%v) = %v, want %v", tt.input, got, tt.expected)
				return
			}
			if !slices.Equal(got, tt.expected) {
				t.Errorf("SortedKeys(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSortedKeys_StringValues(t *testing.T) {
	input := map[string]string{"c": "3", "a": "1", "b": "2"}
	got := SortedKeys(input)
	expected := []string{"a", "b", "c"}
	if !slices.Equal(got, expected) {
		t.Errorf("SortedKeys(%v) = %v, want %v", input, got, expected)
	}
}

func TestSortedKeys_PointerValues(t *testing.T) {
	type item struct{ name string }
	input := map[string]*item{"z": {name: "z"}, "a": {name: "a"}}
	got := SortedKeys(input)
	expected := []string{"a", "z"}
	if !slices.Equal(got, expected) {
		t.Errorf("SortedKeys(pointer map) = %v, want %v", got, expected)
	}
}
