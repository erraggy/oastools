package main

import "testing"

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"validate", "valiate", 1},  // missing 'd'
		{"validate", "validaet", 2}, // transposition
		{"convert", "conert", 1},    // missing 'v'
		{"generate", "genrate", 1},  // missing 'e'
		{"diff", "dif", 1},          // missing 'f'
		{"parse", "prase", 2},       // transposition
		{"kitten", "sitting", 3},    // classic example
	}

	for _, tt := range tests {
		t.Run(tt.a+"->"+tt.b, func(t *testing.T) {
			got := levenshteinDistance(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestSuggestCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Typos within edit distance 2
		{"valiate", "validate"},
		{"validat", "validate"},
		{"vlidate", "validate"},
		{"conert", "convert"},
		{"convrt", "convert"},
		{"genrate", "generate"},
		{"generae", "generate"},
		{"dif", "diff"},
		{"prase", "parse"},
		{"parce", "parse"},
		{"joi", "join"},
		{"fixx", "fix"},
		{"overla", "overlay"},
		{"versio", "version"},
		{"hep", "help"},

		// Too far - no suggestion (distance > 2)
		{"xyz", ""},
		{"foobar", ""},
		{"validatation", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := suggestCommand(tt.input)
			if got != tt.expected {
				t.Errorf("suggestCommand(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
