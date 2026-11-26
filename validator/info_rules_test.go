package validator

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// infoErrorPaths validates spec and returns the path of every error under info.
func infoErrorPaths(t *testing.T, spec string) []string {
	t.Helper()
	parsed, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)
	result, err := ValidateWithOptions(WithParsed(*parsed))
	require.NoError(t, err)
	var paths []string
	for _, e := range result.Errors {
		if e.Path == "info" || strings.HasPrefix(e.Path, "info.") {
			paths = append(paths, e.Path)
		}
	}
	return paths
}

// TestValidateInfo_Issue558 is the reproduction from #558: a 3.0 document
// breaking all four rules used to validate clean.
func TestValidateInfo_Issue558(t *testing.T) {
	const spec = `openapi: 3.0.3
info:
  title: T
  version: "1.0.0"
  summary: A summary
  termsOfService: terms of service
  license:
    url: https://example.com/l
    identifier: MIT
paths: {}
`
	assert.Equal(t, []string{
		"info.summary", "info.termsOfService", "info.license.name", "info.license.identifier",
	}, infoErrorPaths(t, spec))
}

func TestValidateInfo_VersionScoped(t *testing.T) {
	doc := func(header, info string) string {
		return header + "\ninfo:\n  title: T\n  version: \"1\"\n" + info + "paths: {}\n"
	}
	const (
		v20 = "swagger: \"2.0\""
		v30 = "openapi: 3.0.4"
		v31 = "openapi: 3.1.1"
		v32 = "openapi: 3.2.0"
	)
	tests := []struct {
		name   string
		header string
		info   string
		want   []string
	}{
		{"license name required 2.0", v20, "  license:\n    url: https://e.com/l\n", []string{"info.license.name"}},
		{"license name required 3.2", v32, "  license: {}\n", []string{"info.license.name"}},

		{"tos free text 2.0", v20, "  termsOfService: see the website\n", nil},
		{"tos must be a URI 3.0", v30, "  termsOfService: see the website\n", []string{"info.termsOfService"}},
		{"tos must be a URI 3.1", v31, "  termsOfService: see the website\n", []string{"info.termsOfService"}},

		{"summary absent before 3.1 (2.0)", v20, "  summary: s\n", []string{"info.summary"}},
		{"summary absent before 3.1 (3.0)", v30, "  summary: s\n", []string{"info.summary"}},
		{"summary from 3.1", v31, "  summary: s\n", nil},

		{"identifier absent before 3.1", v30, "  license:\n    name: MIT\n    identifier: MIT\n", []string{"info.license.identifier"}},
		{"identifier from 3.1", v31, "  license:\n    name: MIT\n    identifier: MIT\n", nil},
		{"identifier SPDX shape from 3.1", v32, "  license:\n    name: MIT\n    identifier: not spdx\n", []string{"info.license.identifier"}},

		{"identifier and url exclusive from 3.1", v31,
			"  license:\n    name: MIT\n    identifier: MIT\n    url: https://e.com/l\n", []string{"info.license.identifier"}},
		{"url alone from 3.1", v31, "  license:\n    name: MIT\n    url: https://e.com/l\n", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, infoErrorPaths(t, doc(tt.header, tt.info)))
		})
	}
}

func TestValidateInfo_SpecRefNamesObject(t *testing.T) {
	parsed, err := parser.ParseWithOptions(parser.WithBytes([]byte(
		"openapi: 3.1.1\ninfo:\n  title: T\n  version: \"1\"\n  license: {}\npaths: {}\n")))
	require.NoError(t, err)
	result, err := ValidateWithOptions(WithParsed(*parsed))
	require.NoError(t, err)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "info.license.name", result.Errors[0].Path)
	assert.True(t, strings.HasSuffix(result.Errors[0].SpecRef, "#license-object"), result.Errors[0].SpecRef)
}
