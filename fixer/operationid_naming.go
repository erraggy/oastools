// This file implements configuration for operationId naming when fixing duplicates.
// The naming configuration supports template-based customization with placeholders
// for operationId, method, path, tags, and numeric suffixes.

package fixer

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/erraggy/oastools/internal/naming"
)

// OperationIdNamingConfig configures how duplicate operationId values are renamed.
// The Template field supports placeholders that are expanded with operation context.
type OperationIdNamingConfig struct {
	// Template is the naming template for duplicate operationIds.
	// Supported placeholders:
	//   {operationId} - Original operationId value
	//   {method}      - HTTP method (lowercase: get, post, etc.)
	//   {path}        - Sanitized path (e.g., /users/{id} -> users_id)
	//   {tag}         - First tag (empty string if no tags)
	//   {tags}        - All tags joined with TagSeparator
	//   {n}           - Numeric suffix (2, 3, 4, ...) for collision resolution
	//
	// Supported modifiers (appended with colon):
	//   :pascal - PascalCase (e.g., {operationId:pascal} -> "GetUser")
	//   :camel  - camelCase (e.g., {method:camel} -> "get")
	//   :snake  - snake_case (e.g., {path:snake} -> "users_id")
	//   :kebab  - kebab-case (e.g., {tag:kebab} -> "user-profile")
	//   :upper  - UPPERCASE (e.g., {method:upper} -> "GET")
	//   :lower  - lowercase (e.g., {operationId:lower} -> "getuser")
	//
	// Default: "{operationId}{n}" produces getUser, getUser2, getUser3, ...
	Template string

	// PathSeparator is used when expanding {path} placeholder.
	// Path segments and parameter names are joined with this separator.
	// Default: "_"
	PathSeparator string

	// TagSeparator is used when expanding {tags} placeholder.
	// Multiple tags are joined with this separator.
	// Default: "_"
	TagSeparator string
}

// DefaultOperationIdNamingConfig returns the default configuration.
// Uses "{operationId}{n}" template which produces: getUser, getUser2, getUser3, ...
func DefaultOperationIdNamingConfig() OperationIdNamingConfig {
	return OperationIdNamingConfig{
		Template:      "{operationId}{n}",
		PathSeparator: "_",
		TagSeparator:  "_",
	}
}

// OperationContext provides metadata about an operation for template expansion.
type OperationContext struct {
	// OperationId is the original operationId value
	OperationId string
	// Method is the HTTP method in lowercase (get, post, put, etc.)
	Method string
	// Path is the API path (e.g., /users/{id}/posts)
	Path string
	// Tags is the list of tags from the operation
	Tags []string
}

// validPlaceholders defines all valid placeholder names
var validPlaceholders = map[string]bool{
	"operationId": true,
	"method":      true,
	"path":        true,
	"tag":         true,
	"tags":        true,
	"n":           true,
}

// validModifiers defines all valid case modifier names
var validModifiers = map[string]bool{
	"pascal": true,
	"camel":  true,
	"snake":  true,
	"kebab":  true,
	"upper":  true,
	"lower":  true,
}

// placeholderRegex matches placeholders with optional modifiers: {name} or {name:modifier}
var placeholderRegex = regexp.MustCompile(`\{(\w+)(?::(\w+))?\}`)

// ParseOperationIdNamingTemplate validates that a template contains only valid placeholders and modifiers.
// Returns an error if the template is empty or contains unknown placeholders or modifiers.
func ParseOperationIdNamingTemplate(template string) error {
	if template == "" {
		return fmt.Errorf("fixer: operationId template cannot be empty")
	}

	// Find all placeholders with optional modifiers in the template
	matches := placeholderRegex.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		placeholder := match[1]
		modifier := match[2] // May be empty string if no modifier

		// Validate placeholder name
		if !validPlaceholders[placeholder] {
			return fmt.Errorf("fixer: unknown placeholder {%s} in operationId template; valid placeholders: {operationId}, {method}, {path}, {tag}, {tags}, {n}", placeholder)
		}

		// Validate modifier if present
		if modifier != "" && !validModifiers[modifier] {
			return fmt.Errorf("fixer: unknown modifier :%s in operationId template; valid modifiers: :pascal, :camel, :snake, :kebab, :upper, :lower", modifier)
		}
	}

	return nil
}

// applyModifier applies a case modifier to a value using the naming package utilities.
// If modifier is empty or unknown, returns the value unchanged.
func applyModifier(value, modifier string) string {
	switch modifier {
	case "pascal":
		return naming.ToPascalCase(value)
	case "camel":
		return naming.ToCamelCase(value)
	case "snake":
		return naming.ToSnakeCase(value)
	case "kebab":
		return naming.ToKebabCase(value)
	case "upper":
		return strings.ToUpper(value)
	case "lower":
		return strings.ToLower(value)
	default:
		return value // No modifier or unknown modifier
	}
}

// expandOperationIdTemplate expands the template with context values.
// The n parameter is the numeric suffix (2, 3, 4, ...) for collision resolution.
// When n is 0 or 1, {n} expands to empty string (first occurrence keeps original name).
// Supports modifiers like {placeholder:modifier} for case transformations.
func expandOperationIdTemplate(template string, ctx OperationContext, n int, config OperationIdNamingConfig) string {
	// Precompute values for placeholders
	sanitizedPath := sanitizePath(ctx.Path, config.PathSeparator)
	firstTag := ""
	if len(ctx.Tags) > 0 {
		firstTag = ctx.Tags[0]
	}
	tagSep := config.TagSeparator
	if tagSep == "" {
		tagSep = "_"
	}
	joinedTags := strings.Join(ctx.Tags, tagSep)
	nStr := ""
	if n > 1 {
		nStr = fmt.Sprintf("%d", n)
	}

	// Use regex replacement to handle modifiers
	result := placeholderRegex.ReplaceAllStringFunc(template, func(match string) string {
		// Parse the placeholder and optional modifier
		parts := placeholderRegex.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match // Should not happen with valid regex
		}

		placeholder := parts[1]
		modifier := ""
		if len(parts) >= 3 {
			modifier = parts[2]
		}

		// Get the raw value for the placeholder
		var value string
		switch placeholder {
		case "operationId":
			value = ctx.OperationId
		case "method":
			value = ctx.Method
		case "path":
			value = sanitizedPath
		case "tag":
			value = firstTag
		case "tags":
			value = joinedTags
		case "n":
			value = nStr
		default:
			return match // Unknown placeholder, leave as-is
		}

		// Apply modifier if present
		return applyModifier(value, modifier)
	})

	return result
}

// sanitizePath converts a path template to a safe identifier component.
// Removes leading/trailing slashes, replaces path separators and braces,
// and normalizes multiple separators.
//
// Examples:
//
//	"/users/{id}/posts" with separator "_" -> "users_id_posts"
//	"/api/v1/items" with separator "_" -> "api_v1_items"
func sanitizePath(path string, separator string) string {
	if separator == "" {
		separator = "_"
	}

	var result strings.Builder
	result.Grow(len(path))

	prevWasSep := true // Start true to skip leading separators

	for _, r := range path {
		switch {
		case r == '/' || r == '{' || r == '}':
			// Replace path-related characters with separator
			if !prevWasSep {
				result.WriteString(separator)
				prevWasSep = true
			}
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			// Keep alphanumeric characters
			result.WriteRune(r)
			prevWasSep = false
		case r == '_' || r == '-':
			// Keep common identifier characters
			result.WriteRune(r)
			prevWasSep = false
		default:
			// Replace other characters with separator
			if !prevWasSep {
				result.WriteString(separator)
				prevWasSep = true
			}
		}
	}

	// Trim trailing separator
	s := result.String()
	s = strings.TrimSuffix(s, separator)

	return s
}
