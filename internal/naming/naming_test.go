package naming

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Empty and single characters
		{name: "empty string", input: "", want: ""},
		{name: "single lowercase letter", input: "a", want: "A"},
		{name: "single uppercase letter", input: "A", want: "A"},
		{name: "single digit", input: "1", want: "1"},

		// Underscore separators
		{name: "snake_case simple", input: "user_profile", want: "UserProfile"},
		{name: "snake_case three words", input: "get_user_by_id", want: "GetUserById"},
		{name: "leading underscore", input: "_private", want: "Private"},
		{name: "trailing underscore", input: "value_", want: "Value"},
		{name: "double underscore", input: "double__under", want: "DoubleUnder"},

		// Hyphen separators
		{name: "kebab-case simple", input: "api-client", want: "ApiClient"},
		{name: "kebab-case three words", input: "get-user-by-id", want: "GetUserById"},
		{name: "leading hyphen", input: "-private", want: "Private"},
		{name: "trailing hyphen", input: "value-", want: "Value"},

		// Dot separators
		{name: "dot separator", input: "com.example.api", want: "ComExampleApi"},
		{name: "leading dot", input: ".hidden", want: "Hidden"},

		// Slash separators
		{name: "slash separator", input: "users/profile", want: "UsersProfile"},
		{name: "path-like", input: "/api/v1/users", want: "ApiV1Users"},

		// Mixed separators
		{name: "mixed separators", input: "get_user-by.id/name", want: "GetUserByIdName"},
		{name: "consecutive mixed separators", input: "foo_-bar", want: "FooBar"},

		// Already cased
		{name: "already PascalCase", input: "UserProfile", want: "UserProfile"},
		{name: "all caps", input: "API", want: "API"},
		{name: "camelCase", input: "userProfile", want: "UserProfile"},

		// Unicode characters
		{name: "unicode lowercase", input: "über_user", want: "ÜberUser"},
		{name: "unicode uppercase", input: "Über_user", want: "ÜberUser"},
		{name: "japanese characters", input: "日本語_test", want: "日本語Test"},

		// Numbers
		{name: "with numbers", input: "api_v2_client", want: "ApiV2Client"},
		{name: "leading number", input: "123_abc", want: "123Abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToPascalCase(tt.input)
			assert.Equal(t, tt.want, got, "ToPascalCase(%q)", tt.input)
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Empty and single characters
		{name: "empty string", input: "", want: ""},
		{name: "single lowercase letter", input: "a", want: "a"},
		{name: "single uppercase letter", input: "A", want: "a"},
		{name: "single digit", input: "1", want: "1"},

		// Underscore separators
		{name: "snake_case simple", input: "user_profile", want: "userProfile"},
		{name: "snake_case three words", input: "get_user_by_id", want: "getUserById"},

		// Hyphen separators
		{name: "kebab-case simple", input: "api-client", want: "apiClient"},

		// Already cased
		{name: "already camelCase", input: "userProfile", want: "userProfile"},
		{name: "PascalCase", input: "UserProfile", want: "userProfile"},

		// Mixed separators
		{name: "mixed separators", input: "get_user-by.id", want: "getUserById"},

		// Unicode characters
		{name: "unicode lowercase", input: "über_user", want: "überUser"},
		{name: "unicode uppercase", input: "Über_user", want: "überUser"},

		// Numbers
		{name: "with numbers", input: "api_v2_client", want: "apiV2Client"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToCamelCase(tt.input)
			assert.Equal(t, tt.want, got, "ToCamelCase(%q)", tt.input)
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Empty and single characters
		{name: "empty string", input: "", want: ""},
		{name: "single lowercase letter", input: "a", want: "a"},
		{name: "single uppercase letter", input: "A", want: "a"},
		{name: "single digit", input: "1", want: "1"},

		// PascalCase
		{name: "PascalCase simple", input: "UserProfile", want: "user_profile"},
		{name: "PascalCase three words", input: "GetUserById", want: "get_user_by_id"},

		// camelCase
		{name: "camelCase simple", input: "userProfile", want: "user_profile"},

		// All caps
		{name: "all caps", input: "API", want: "a_p_i"},
		{name: "caps prefix", input: "APIClient", want: "a_p_i_client"},

		// Hyphen separators
		{name: "kebab-case", input: "api-client", want: "api_client"},
		{name: "leading hyphen", input: "-private", want: "_private"},

		// Dot separators
		{name: "dot separator", input: "com.example.api", want: "com_example_api"},

		// Slash separators
		{name: "slash separator", input: "users/profile", want: "users_profile"},

		// Already snake_case
		{name: "already snake_case", input: "user_profile", want: "user_profile"},

		// Unicode characters
		{name: "unicode", input: "ÜberUser", want: "über_user"},

		// Numbers
		{name: "with numbers", input: "ApiV2Client", want: "api_v2_client"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToSnakeCase(tt.input)
			assert.Equal(t, tt.want, got, "ToSnakeCase(%q)", tt.input)
		})
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Empty and single characters
		{name: "empty string", input: "", want: ""},
		{name: "single lowercase letter", input: "a", want: "a"},
		{name: "single uppercase letter", input: "A", want: "a"},

		// PascalCase
		{name: "PascalCase simple", input: "UserProfile", want: "user-profile"},
		{name: "PascalCase three words", input: "GetUserById", want: "get-user-by-id"},

		// camelCase
		{name: "camelCase simple", input: "userProfile", want: "user-profile"},

		// snake_case
		{name: "snake_case", input: "user_profile", want: "user-profile"},

		// Already kebab-case
		{name: "already kebab-case", input: "user-profile", want: "user-profile"},

		// Dot separators
		{name: "dot separator", input: "com.example.api", want: "com-example-api"},

		// Unicode
		{name: "unicode", input: "ÜberUser", want: "über-user"},

		// Numbers
		{name: "with numbers", input: "ApiV2Client", want: "api-v2-client"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToKebabCase(tt.input)
			assert.Equal(t, tt.want, got, "ToKebabCase(%q)", tt.input)
		})
	}
}

func TestToTitleCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Empty and single characters
		{name: "empty string", input: "", want: ""},
		{name: "single lowercase letter", input: "a", want: "A"},
		{name: "single uppercase letter", input: "A", want: "A"},
		{name: "single digit", input: "1", want: "1"},

		// Words
		{name: "lowercase word", input: "hello", want: "Hello"},
		{name: "uppercase word", input: "HELLO", want: "HELLO"},
		{name: "mixed case", input: "hELLO", want: "HELLO"},
		{name: "multiple words", input: "hello world", want: "Hello world"},
		{name: "already titled", input: "Hello", want: "Hello"},

		// Unicode
		{name: "unicode lowercase", input: "über", want: "Über"},
		{name: "unicode uppercase", input: "Über", want: "Über"},
		{name: "japanese", input: "日本語", want: "日本語"},

		// With separators (only first letter is affected)
		{name: "snake_case", input: "hello_world", want: "Hello_world"},
		{name: "kebab-case", input: "hello-world", want: "Hello-world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToTitleCase(tt.input)
			assert.Equal(t, tt.want, got, "ToTitleCase(%q)", tt.input)
		})
	}
}

// Edge case tests for additional coverage
func TestEdgeCases(t *testing.T) {
	t.Run("consecutive separators in PascalCase", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"a__b", "AB"},
			{"a---b", "AB"},
			{"a...b", "AB"},
			{"a///b", "AB"},
			{"_-._", ""},
		}
		for _, tt := range tests {
			got := ToPascalCase(tt.input)
			assert.Equal(t, tt.want, got, "ToPascalCase(%q)", tt.input)
		}
	})

	t.Run("only separators in camelCase", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"___", ""},
			{"---", ""},
			{"...", ""},
			{"///", ""},
		}
		for _, tt := range tests {
			got := ToCamelCase(tt.input)
			assert.Equal(t, tt.want, got, "ToCamelCase(%q)", tt.input)
		}
	})
}
