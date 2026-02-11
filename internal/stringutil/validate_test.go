package stringutil

import "testing"

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
			got := IsValidEmail(tt.input)
			if got != tt.want {
				t.Errorf("IsValidEmail(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
