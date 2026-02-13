package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleWalk_NoArgs(t *testing.T) {
	err := HandleWalk([]string{})
	assert.Error(t, err)
}

func TestHandleWalk_InvalidSubcommand(t *testing.T) {
	err := HandleWalk([]string{"invalid"})
	assert.Error(t, err)
}

func TestHandleWalk_Help(t *testing.T) {
	err := HandleWalk([]string{"--help"})
	assert.NoError(t, err)
}

func TestHandleWalk_HelpShortFlag(t *testing.T) {
	err := HandleWalk([]string{"-h"})
	assert.NoError(t, err)
}

func TestHandleWalk_HelpSubcommand(t *testing.T) {
	err := HandleWalk([]string{"help"})
	assert.NoError(t, err)
}

func TestHandleWalk_ValidSubcommands(t *testing.T) {
	// All subcommands are implemented and return a "requires a spec file" error
	// when called with no arguments.
	subcommands := []string{"operations", "schemas", "parameters", "responses", "security", "paths"}

	for _, sub := range subcommands {
		t.Run(sub, func(t *testing.T) {
			err := HandleWalk([]string{sub})
			assert.Error(t, err)
		})
	}
}

func TestMatchPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{name: "empty pattern matches anything", path: "/pets", pattern: "", expected: true},
		{name: "exact match", path: "/pets", pattern: "/pets", expected: true},
		{name: "exact mismatch", path: "/pets", pattern: "/users", expected: false},
		{name: "glob single segment", path: "/pets/123", pattern: "/pets/*", expected: true},
		{name: "glob mismatch length", path: "/pets/123/details", pattern: "/pets/*", expected: false},
		{name: "glob first segment", path: "/v1/pets", pattern: "/*/pets", expected: true},
		{name: "no glob exact", path: "/pets/{petId}", pattern: "/pets/{petId}", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPath(tt.path, tt.pattern)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestMatchStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		pattern  string
		expected bool
	}{
		{name: "empty pattern matches anything", code: "200", pattern: "", expected: true},
		{name: "exact match", code: "200", pattern: "200", expected: true},
		{name: "exact mismatch", code: "200", pattern: "404", expected: false},
		{name: "wildcard 2xx matches 200", code: "200", pattern: "2xx", expected: true},
		{name: "wildcard 2xx matches 201", code: "201", pattern: "2xx", expected: true},
		{name: "wildcard 4xx matches 404", code: "404", pattern: "4xx", expected: true},
		{name: "wildcard 4xx does not match 200", code: "200", pattern: "4xx", expected: false},
		{name: "case insensitive wildcard", code: "200", pattern: "2XX", expected: true},
		{name: "default matches default", code: "default", pattern: "default", expected: true},
		{name: "wildcard 2xx does not match short code", code: "2", pattern: "2xx", expected: false},
		{name: "wildcard 2xx does not match default", code: "default", pattern: "2xx", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchStatusCode(tt.code, tt.pattern)
			assert.Equal(t, tt.expected, got)
		})
	}
}
