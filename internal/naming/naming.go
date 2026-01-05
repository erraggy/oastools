// Package naming provides shared string case conversion utilities.
package naming

import (
	"strings"
	"unicode"
)

// ToPascalCase converts a string to PascalCase.
// Separators (underscore, hyphen, dot, slash) trigger capitalization of the next letter.
// Example: "user_profile" -> "UserProfile"
// Example: "api-client" -> "ApiClient"
func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	capitalizeNext := true

	for _, r := range s {
		if r == '_' || r == '-' || r == '.' || r == '/' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			result.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// ToCamelCase converts a string to camelCase.
// Like PascalCase but with the first letter lowercase.
// Example: "user_profile" -> "userProfile"
// Example: "UserProfile" -> "userProfile"
func ToCamelCase(s string) string {
	pascal := ToPascalCase(s)
	if pascal == "" {
		return ""
	}
	runes := []rune(pascal)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// ToSnakeCase converts a string to snake_case.
// Uppercase letters are prefixed with underscore and lowercased.
// Existing separators (hyphen, dot, slash) are converted to underscores.
// Example: "UserProfile" -> "user_profile"
// Example: "APIClient" -> "api_client"
func ToSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else if r == '-' || r == '.' || r == '/' {
			result.WriteRune('_')
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// ToKebabCase converts a string to kebab-case.
// Like snake_case but with hyphens instead of underscores.
// Example: "UserProfile" -> "user-profile"
func ToKebabCase(s string) string {
	return strings.ReplaceAll(ToSnakeCase(s), "_", "-")
}

// ToTitleCase converts the first letter to uppercase.
// Example: "hello" -> "Hello"
func ToTitleCase(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
