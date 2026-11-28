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
		{name: "valid with apostrophe in local part", input: "o'brien@example.com", want: true},
		{name: "valid with exclamation in local part", input: "ops!team@example.com", want: true},
		{name: "valid with punycode domain", input: "team@example.xn--p1ai", want: true},
		{name: "valid with single-label domain", input: "user@localhost", want: true},
		{name: "valid single label domain example", input: "user@example", want: true},
		{name: "valid single char label domain", input: "user@example.c", want: true},
		{name: "valid quoted local part", input: `"john doe"@example.com`, want: true},
		{name: "valid quoted local part space", input: `" "@example.com`, want: true},
		{name: "valid quoted local part at symbol", input: `"john@doe"@example.com`, want: true},
		{name: "valid quoted local part escaped quote", input: `"\"quoted\""@example.com`, want: true},
		{name: "valid quoted local part escaped backslash", input: `"user\\name"@example.com`, want: true},
		{name: "valid address literal ipv4", input: "user@[192.168.1.1]", want: true},
		{name: "valid address literal ipv6", input: "user@[IPv6:2001:db8::1]", want: true},
		{name: "valid address literal ipv6 loopback", input: "user@[IPv6:::1]", want: true},
		{name: "valid atext characters in local part", input: "user#123@example.com", want: true},
		{name: "valid atext dollar in local part", input: "user$@example.com", want: true},
		{name: "valid atext slash in local part", input: "user/test@example.com", want: true},
		{name: "valid atext equals in local part", input: "user=test@example.com", want: true},
		{name: "valid atext tilde in local part", input: "user~test@example.com", want: true},
		{name: "missing at sign", input: "userexample.com", want: false},
		{name: "missing domain", input: "user@", want: false},
		{name: "missing local part", input: "@example.com", want: false},
		{name: "empty string", input: "", want: false},
		{name: "unquoted spaces in local part", input: "user @example.com", want: false},
		{name: "spaces in domain", input: "user@ example.com", want: false},
		{name: "domain with underscore", input: "user@example_com", want: false},
		{name: "domain with leading hyphen", input: "user@-example.com", want: false},
		{name: "domain with trailing hyphen", input: "user@example-.com", want: false},
		{name: "domain with consecutive dots", input: "user@example..com", want: false},
		{name: "domain with leading dot", input: "user@.example.com", want: false},
		{name: "domain with trailing dot", input: "user@example.com.", want: false},
		{name: "local part with consecutive dots", input: "user..name@example.com", want: false},
		{name: "local part with leading dot", input: ".user@example.com", want: false},
		{name: "local part with trailing dot", input: "user.@example.com", want: false},
		{name: "angle brackets", input: "<user@example.com>", want: false},
		{name: "display name", input: "John Doe <user@example.com>", want: false},
		{name: "unclosed quote in local part", input: `"unclosed@example.com`, want: false},
		{name: "quote followed by unquoted atom without dot", input: `"quote"extra@example.com`, want: false},
		{name: "address literal invalid ipv4", input: "user@[192.168.1.256]", want: false},
		{name: "address literal ipv4 leading zero", input: "user@[192.168.01.1]", want: false},
		{name: "address literal missing ipv6 prefix", input: "user@[2001:db8::1]", want: false},
		{name: "address literal invalid ipv6", input: "user@[IPv6:invalid]", want: false},
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
		{name: "host with port", input: "https://example.com:8443/path", want: true},
		{name: "root relative", input: "/support", want: true},
		{name: "dot relative", input: "./terms.html", want: true},
		{name: "parent relative", input: "../LICENSE", want: true},
		{name: "bare path", input: "terms.html", want: true},
		{name: "path with a dot in the first segment", input: "example.com/support", want: true},
		{name: "network path", input: "//example.com/x", want: true},
		{name: "query only", input: "?page=1", want: true},
		{name: "fragment only", input: "#terms", want: true},
		{name: "percent encoded", input: "/a%20b", want: true},
		{name: "mailto", input: "mailto:a@example.com", want: true},
		{name: "ftp", input: "ftp://example.com/x", want: true},
		{name: "urn", input: "urn:isbn:0451450523", want: true},
		{name: "empty string", input: "", want: false},
		{name: "space", input: "a b", want: false},
		{name: "space in path", input: "/a b", want: false},
		{name: "non ASCII", input: "https://example.com/caf\u00e9", want: false},
		{name: "bad percent escape", input: "/a%zz", want: false},
		{name: "truncated percent escape", input: "/a%2", want: false},
		{name: "angle bracket", input: "/a<b", want: false},
		{name: "malformed host", input: "http://[::1", want: false},
		{name: "first segment with a colon", input: "1:2", want: false},
		{name: "https without host", input: "https://", want: false},
		{name: "http without host", input: "http:///path", want: false},
		{name: "port without host", input: "https://:443/path", want: false},
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
		{"GPL-2.0+ WITH Classpath-exception-2.0", true},
		{"GPL-2.0-only WITH Classpath-exception-2.0+", false},
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
