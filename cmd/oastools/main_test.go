package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			assert.Equal(t, tt.expected, got)
		})
	}
}
