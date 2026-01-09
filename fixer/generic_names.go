// This file implements detection and transformation of invalid schema names.
// Third-party code generators often produce OpenAPI specs with schema names containing
// unencoded special characters (like Response[User] for generic types). This file provides
// detection, parsing, and transformation of such names into valid schema names.

package fixer

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/erraggy/oastools/parser"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// GenericNamingStrategy defines how generic type parameters are formatted
// in schema names when transforming invalid names to valid ones.
type GenericNamingStrategy int

const (
	// GenericNamingUnderscore replaces brackets with underscores.
	// Example: Response[User] -> Response_User_
	GenericNamingUnderscore GenericNamingStrategy = iota

	// GenericNamingOf uses "Of" separator between base type and parameters.
	// Example: Response[User] -> ResponseOfUser
	GenericNamingOf

	// GenericNamingFor uses "For" separator.
	// Example: Response[User] -> ResponseForUser
	GenericNamingFor

	// GenericNamingFlattened removes brackets entirely.
	// Example: Response[User] -> ResponseUser
	GenericNamingFlattened

	// GenericNamingDot uses dots as separator.
	// Example: Response[User] -> Response.User
	GenericNamingDot
)

// String returns the string representation of a GenericNamingStrategy.
func (s GenericNamingStrategy) String() string {
	switch s {
	case GenericNamingUnderscore:
		return "underscore"
	case GenericNamingOf:
		return "of"
	case GenericNamingFor:
		return "for"
	case GenericNamingFlattened:
		return "flattened"
	case GenericNamingDot:
		return "dot"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// ParseGenericNamingStrategy parses a string into a GenericNamingStrategy.
// Supported values: "underscore", "of", "for", "flattened", "dot" (case-insensitive).
func ParseGenericNamingStrategy(s string) (GenericNamingStrategy, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "underscore", "_":
		return GenericNamingUnderscore, nil
	case "of":
		return GenericNamingOf, nil
	case "for":
		return GenericNamingFor, nil
	case "flattened", "flat":
		return GenericNamingFlattened, nil
	case "dot", ".":
		return GenericNamingDot, nil
	default:
		return GenericNamingUnderscore, fmt.Errorf("fixer: unknown generic naming strategy: %q", s)
	}
}

// GenericNamingConfig provides fine-grained control over generic type naming.
type GenericNamingConfig struct {
	// Strategy is the primary generic naming approach.
	Strategy GenericNamingStrategy

	// Separator is used between base type and parameters for underscore strategy.
	// Default: "_"
	Separator string

	// ParamSeparator is used between multiple type parameters.
	// Example with ParamSeparator="_": Map[string,int] -> Map_string_int
	// Default: "_"
	ParamSeparator string

	// PreserveCasing when false converts type parameters to PascalCase.
	// When true, keeps original casing of type parameters.
	// Default: false (convert to PascalCase)
	PreserveCasing bool
}

// DefaultGenericNamingConfig returns the default generic naming configuration.
// This uses underscore strategy with "_" separators and converts params to PascalCase.
func DefaultGenericNamingConfig() GenericNamingConfig {
	return GenericNamingConfig{
		Strategy:       GenericNamingUnderscore,
		Separator:      "_",
		ParamSeparator: "_",
		PreserveCasing: false,
	}
}

// invalidSchemaNameChars contains characters that require URL encoding in $ref values.
// These characters cause issues when used in schema names because JSON Pointer
// references to them require percent-encoding.
var invalidSchemaNameChars = []rune{
	'[', ']', // square brackets (generics)
	'<', '>', // angle brackets (generics in some languages)
	',',      // comma (multiple type parameters)
	' ',      // space
	'{', '}', // curly braces
	'|',  // pipe
	'\\', // backslash
	'^',  // caret
	'`',  // backtick
}

// hasInvalidSchemaNameChars returns true if name contains characters that are
// problematic in schema names (require URL encoding in $ref values).
func hasInvalidSchemaNameChars(name string) bool {
	// Empty or whitespace-only names are invalid
	if strings.TrimSpace(name) == "" {
		return true
	}

	for _, c := range name {
		if slices.Contains(invalidSchemaNameChars, c) {
			return true
		}
	}
	return false
}

// isGenericStyleName returns true if name appears to be a generic type name
// (contains square or angle brackets indicating type parameters).
func isGenericStyleName(name string) bool {
	return strings.ContainsAny(name, "[]<>")
}

// isPackageQualifiedName returns true if name appears to be a package-qualified
// schema name (contains a dot but no brackets indicating generics).
// Examples: "common.Pet" → true, "Response[User]" → false, "Pet" → false
func isPackageQualifiedName(name string) bool {
	return strings.Contains(name, ".") && !strings.ContainsAny(name, "[]<>")
}

// parseGenericName extracts the base name and type parameters from a generic-style name.
// Returns the base name, list of type parameters, and the bracket style used.
// If the name is not generic-style, returns the name as base with empty params.
//
// Examples:
//
//	"Response[User]" -> ("Response", ["User"], '[')
//	"Map[string,int]" -> ("Map", ["string", "int"], '[')
//	"List<Item>" -> ("List", ["Item"], '<')
//	"PlainName" -> ("PlainName", nil, 0)
func parseGenericName(name string) (base string, params []string, bracketStyle rune) {
	// Try square brackets first
	if idx := strings.Index(name, "["); idx != -1 {
		endIdx := strings.LastIndex(name, "]")
		if endIdx > idx {
			base = name[:idx]
			paramStr := name[idx+1 : endIdx]
			params = splitTypeParams(paramStr)
			return base, params, '['
		}
	}

	// Try angle brackets
	if idx := strings.Index(name, "<"); idx != -1 {
		endIdx := strings.LastIndex(name, ">")
		if endIdx > idx {
			base = name[:idx]
			paramStr := name[idx+1 : endIdx]
			params = splitTypeParams(paramStr)
			return base, params, '<'
		}
	}

	// Not a generic name
	return name, nil, 0
}

// splitTypeParams splits a parameter string by commas, handling nested brackets.
// This correctly handles nested generic types like "User,List[Item],int".
//
// Examples:
//
//	"User" -> ["User"]
//	"string,int" -> ["string", "int"]
//	"User,List[Item],int" -> ["User", "List[Item]", "int"]
//	"Map[K,V],List[T]" -> ["Map[K,V]", "List[T]"]
func splitTypeParams(s string) []string {
	if s == "" {
		return nil
	}

	var params []string
	var current strings.Builder
	depth := 0

	for _, r := range s {
		switch r {
		case '[', '<':
			depth++
			current.WriteRune(r)
		case ']', '>':
			depth--
			current.WriteRune(r)
		case ',':
			if depth == 0 {
				// Top-level comma - end of parameter
				param := strings.TrimSpace(current.String())
				if param != "" {
					params = append(params, param)
				}
				current.Reset()
			} else {
				// Nested comma - part of parameter
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	// Add final parameter
	param := strings.TrimSpace(current.String())
	if param != "" {
		params = append(params, param)
	}

	return params
}

// transformSchemaName applies the naming strategy to generate a valid schema name
// from an invalid generic-style name.
//
// Examples with GenericNamingOf:
//
//	"Response[User]" -> "ResponseOfUser"
//	"Map[string,int]" -> "MapOfStringOfInt"
//	"Response[List[User]]" -> "ResponseOfListOfUser"
func transformSchemaName(name string, config GenericNamingConfig) string {
	// Handle empty or whitespace-only names
	if strings.TrimSpace(name) == "" {
		return "UnnamedSchema"
	}

	base, params, _ := parseGenericName(name)

	// If no type parameters, just sanitize and return
	if len(params) == 0 {
		sanitized := sanitizeSchemaName(name)
		if sanitized == "" {
			return "UnnamedSchema"
		}
		return sanitized
	}

	// Recursively transform nested generic types in parameters
	transformedParams := make([]string, len(params))
	for i, param := range params {
		transformedParams[i] = transformTypeParam(param, config)
	}

	// Apply strategy
	switch config.Strategy {
	case GenericNamingOf:
		return base + "Of" + strings.Join(transformedParams, config.ParamSeparator+"Of")

	case GenericNamingFor:
		return base + "For" + strings.Join(transformedParams, config.ParamSeparator+"For")

	case GenericNamingFlattened:
		return base + strings.Join(transformedParams, "")

	case GenericNamingDot:
		return base + "." + strings.Join(transformedParams, ".")

	default: // GenericNamingUnderscore
		sep := config.Separator
		if sep == "" {
			sep = "_"
		}
		paramSep := config.ParamSeparator
		if paramSep == "" {
			paramSep = "_"
		}
		return base + sep + strings.Join(transformedParams, paramSep) + sep
	}
}

// transformTypeParam transforms a type parameter while preserving package qualification.
// It strips leading pointer asterisks and preserves package-qualified names.
// Examples:
//
//	"*common.Pet" → "common.Pet" (pointer stripped, package preserved)
//	"common.Pet" → "common.Pet" (package preserved)
//	"*User" → "User" (pointer stripped, then PascalCased if configured)
//	"List[User]" → recursively transformed
func transformTypeParam(param string, config GenericNamingConfig) string {
	// Strip leading pointer asterisks (Go pointer syntax leaking from code generators)
	param = strings.TrimLeft(param, "*")

	// If it's a package-qualified name (like common.Pet), preserve it as-is
	// to avoid corrupting the reference
	if isPackageQualifiedName(param) {
		return param
	}

	// For generic types or simple names, apply normal transformation
	transformed := transformSchemaName(param, config)

	// Apply PascalCase if not preserving casing (for non-package names)
	if !config.PreserveCasing {
		transformed = toPascalCase(transformed)
	}

	return transformed
}

// sanitizeSchemaName removes or replaces invalid characters with underscores.
// This is a fallback for names that aren't cleanly generic-style but still
// contain problematic characters.
func sanitizeSchemaName(name string) string {
	var result strings.Builder
	result.Grow(len(name))

	for _, r := range name {
		if isValidSchemaNameChar(r) {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
	}

	// Clean up multiple consecutive underscores
	s := result.String()
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}

	// Remove leading/trailing underscores
	s = strings.Trim(s, "_")

	return s
}

// isValidSchemaNameChar returns true if the character is valid in schema names.
// Valid characters are: alphanumeric, underscore, hyphen, and dot.
func isValidSchemaNameChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.'
}

// toPascalCase converts a string to PascalCase.
// Separators (underscore, hyphen, dot, slash, space) trigger capitalization.
//
// Examples:
//
//	"user_data" -> "UserData"
//	"some-name" -> "SomeName"
//	"alreadyPascal" -> "AlreadyPascal"
func toPascalCase(s string) string {
	if s == "" {
		return ""
	}

	// Use golang.org/x/text/cases for proper Unicode title casing
	titleCaser := cases.Title(language.English, cases.NoLower)

	var result strings.Builder
	result.Grow(len(s))

	capitalizeNext := true

	for _, r := range s {
		if r == '_' || r == '-' || r == '.' || r == '/' || r == ' ' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext {
			// Use the title caser for proper Unicode handling
			result.WriteString(titleCaser.String(string(r)))
			capitalizeNext = false
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// rewriteSchemaRefs recursively rewrites $ref values in a schema using the rename map.
// The renames map contains old $ref -> new $ref mappings.
//
// Example:
//
//	renames := map[string]string{
//	    "#/components/schemas/Response[User]": "#/components/schemas/ResponseOfUser",
//	}
//
// This handles all schema locations where $ref can appear:
//   - Direct schema.Ref
//   - properties map
//   - additionalProperties
//   - items
//   - allOf, anyOf, oneOf arrays
//   - not schema
//   - prefixItems, contains, propertyNames
//   - dependentSchemas, if/then/else, $defs
//   - discriminator.mapping values
func rewriteSchemaRefs(schema *parser.Schema, renames map[string]string) {
	if schema == nil || len(renames) == 0 {
		return
	}

	// Track visited schemas to handle circular references
	visited := make(map[*parser.Schema]bool)
	rewriteSchemaRefsRecursive(schema, renames, visited)
}

// rewriteSchemaRefsRecursive is the internal recursive implementation.
func rewriteSchemaRefsRecursive(schema *parser.Schema, renames map[string]string, visited map[*parser.Schema]bool) {
	if schema == nil {
		return
	}

	// Circular reference protection
	if visited[schema] {
		return
	}
	visited[schema] = true
	defer delete(visited, schema)

	// Rewrite direct $ref
	if schema.Ref != "" {
		if newRef, ok := renames[schema.Ref]; ok {
			schema.Ref = newRef
		}
	}

	// Properties
	for _, propSchema := range schema.Properties {
		rewriteSchemaRefsRecursive(propSchema, renames, visited)
	}

	// AdditionalProperties (can be *Schema or bool)
	if schema.AdditionalProperties != nil {
		if addProps, ok := schema.AdditionalProperties.(*parser.Schema); ok {
			rewriteSchemaRefsRecursive(addProps, renames, visited)
		}
	}

	// Items (can be *Schema or bool in OAS 3.1+)
	if schema.Items != nil {
		if items, ok := schema.Items.(*parser.Schema); ok {
			rewriteSchemaRefsRecursive(items, renames, visited)
		}
	}

	// AdditionalItems (can be *Schema or bool)
	if schema.AdditionalItems != nil {
		if addItems, ok := schema.AdditionalItems.(*parser.Schema); ok {
			rewriteSchemaRefsRecursive(addItems, renames, visited)
		}
	}

	// Schema composition
	for _, s := range schema.AllOf {
		rewriteSchemaRefsRecursive(s, renames, visited)
	}
	for _, s := range schema.AnyOf {
		rewriteSchemaRefsRecursive(s, renames, visited)
	}
	for _, s := range schema.OneOf {
		rewriteSchemaRefsRecursive(s, renames, visited)
	}
	if schema.Not != nil {
		rewriteSchemaRefsRecursive(schema.Not, renames, visited)
	}

	// OAS 3.1+ / JSON Schema Draft 2020-12 fields
	for _, s := range schema.PrefixItems {
		rewriteSchemaRefsRecursive(s, renames, visited)
	}
	if schema.Contains != nil {
		rewriteSchemaRefsRecursive(schema.Contains, renames, visited)
	}
	if schema.PropertyNames != nil {
		rewriteSchemaRefsRecursive(schema.PropertyNames, renames, visited)
	}
	for _, depSchema := range schema.DependentSchemas {
		rewriteSchemaRefsRecursive(depSchema, renames, visited)
	}

	// Conditional schemas (OAS 3.1+)
	if schema.If != nil {
		rewriteSchemaRefsRecursive(schema.If, renames, visited)
	}
	if schema.Then != nil {
		rewriteSchemaRefsRecursive(schema.Then, renames, visited)
	}
	if schema.Else != nil {
		rewriteSchemaRefsRecursive(schema.Else, renames, visited)
	}

	// $defs (OAS 3.1+)
	for _, defSchema := range schema.Defs {
		rewriteSchemaRefsRecursive(defSchema, renames, visited)
	}

	// Pattern properties
	for _, propSchema := range schema.PatternProperties {
		rewriteSchemaRefsRecursive(propSchema, renames, visited)
	}

	// Discriminator mapping values
	if schema.Discriminator != nil && schema.Discriminator.Mapping != nil {
		for key, ref := range schema.Discriminator.Mapping {
			// Check if it's a full ref path
			if newRef, ok := renames[ref]; ok {
				schema.Discriminator.Mapping[key] = newRef
			} else {
				// Also check for bare names (discriminator mapping can use just the schema name)
				// e.g., "Dog" instead of "#/components/schemas/Dog"
				for oldRef, newRef := range renames {
					oldName := extractSchemaNameFromRefPath(oldRef)
					newName := extractSchemaNameFromRefPath(newRef)
					if ref == oldName {
						schema.Discriminator.Mapping[key] = newName
						break
					}
				}
			}
		}
	}
}

// extractSchemaNameFromRefPath extracts the schema name from a $ref path.
// Returns empty string if not a schema reference.
func extractSchemaNameFromRefPath(ref string) string {
	// OAS 3.x style
	if name, found := strings.CutPrefix(ref, "#/components/schemas/"); found {
		return name
	}
	// OAS 2.0 style
	if name, found := strings.CutPrefix(ref, "#/definitions/"); found {
		return name
	}
	return ""
}
