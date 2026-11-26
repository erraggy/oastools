package stringutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "valid simple email", input: "user@example.com", want: true},
		{name: "valid with dots", input: "first.last@example.com", want: true},
		{name: "valid with plus", input: "user+tag@example.com", want: true},
		{name: "valid with subdomain", input: "user@sub.example.com", want: true},
		{name: "valid with percent", input: "user%name@example.com", want: true},
		{name: "valid with hyphen in domain", input: "user@my-domain.com", want: true},
		{name: "missing at sign", input: "userexample.com", want: false},
		{name: "missing domain", input: "user@", want: false},
		{name: "missing local part", input: "@example.com", want: false},
		{name: "missing TLD", input: "user@example", want: false},
		{name: "single char TLD", input: "user@example.c", want: false},
		{name: "empty string", input: "", want: false},
		{name: "spaces", input: "user @example.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidEmail(tt.input))
		})
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "https", input: "https://example.com/support", want: true},
		{name: "http", input: "http://example.com", want: true},
		{name: "relative path", input: "/support", want: true},
		{name: "empty string", input: "", want: false},
		{name: "bare word", input: "support", want: false},
		{name: "no scheme host", input: "example.com/support", want: false},
		{name: "mailto", input: "mailto:a@example.com", want: false},
		{name: "ftp", input: "ftp://example.com", want: false},
		{name: "malformed", input: "http://[::1", want: false},
		{name: "https without host", input: "https://", want: false},
		{name: "http without host", input: "http:///path", want: false},
		{name: "port without host", input: "https://:443/path", want: false},
		{name: "host with port", input: "https://example.com:8443/path", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidURL(tt.input))
		})
	}
}

func TestIsValidSPDXExpression(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"MIT", true},
		{"Apache-2.0", true},
		{"GPL-2.0+", true},
		{"MIT OR Apache-2.0", true},
		{"MIT AND BSD-3-Clause", true},
		{"GPL-2.0-only WITH Classpath-exception-2.0", true},
		{"MIT OR (Apache-2.0 AND BSD-3-Clause)", true},
		{"(MIT)", true},
		{"mit OR apache-2.0", true},
		{"DocumentRef-spdx-tool:LicenseRef-Custom", true},
		{"mit or apache-2.0", false},
		{"MIT And Apache-2.0", false},
		{"MIT AND @@@", false},
		{"MIT/Apache-2.0", false},
		{"LicenseRef-Custom", true},
		{"", false},
		{"not spdx", false},
		{"MIT OR", false},
		{"OR MIT", false},
		{"MIT AND OR Apache-2.0", false},
		{"MIT WITH", false},
		{"(MIT", false},
		{"MIT)", false},
		{"()", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidSPDXExpression(tt.input))
		})
	}
}
