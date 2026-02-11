// This file implements name conversion from OpenAPI identifiers to valid Go identifiers,
// including reserved word escaping, PascalCase/camelCase conversion, and description formatting.

package generator

import (
	"strings"
	"unicode"

	"github.com/erraggy/oastools/parser"
)

// maxDescriptionLength is the maximum length for descriptions in Go comments
// before truncation. This keeps generated code readable and prevents excessively
// long comment lines.
const maxDescriptionLength = 200

// goReservedWords contains Go reserved keywords that cannot be used as identifiers.
// Note: We only include actual keywords, not predeclared identifiers like "error",
// because those can be shadowed and are commonly used as type names (e.g., "Error").
var goReservedWords = map[string]bool{
	// Keywords (these are truly reserved and cannot be used)
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// escapeReservedWord checks if a name is a Go reserved keyword and escapes it
// by appending an underscore if necessary. The check is case-insensitive because
// PascalCase names like "Range" or "Type" should still be escaped.
func escapeReservedWord(name string) string {
	if goReservedWords[strings.ToLower(name)] {
		return name + "_"
	}
	return name
}

// toTypeName converts an OpenAPI name to a valid Go type name (PascalCase).
// It handles special characters, ensures the name starts with a letter,
// and escapes Go reserved words.
func toTypeName(s string) string {
	if s == "" {
		return "Type"
	}

	// Split on non-alphanumeric and capitalize each part
	var result strings.Builder
	capitalizeNext := true

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if capitalizeNext {
				result.WriteRune(unicode.ToUpper(r))
				capitalizeNext = false
			} else {
				result.WriteRune(r)
			}
		} else {
			capitalizeNext = true
		}
	}

	name := result.String()

	// Ensure starts with a letter
	if len(name) > 0 && !unicode.IsLetter(rune(name[0])) {
		name = "T" + name
	}

	return escapeReservedWord(name)
}

// toFieldName converts an OpenAPI property name to a valid Go field name (PascalCase).
// It handles special characters, ensures the name starts with a letter,
// and escapes Go reserved words.
func toFieldName(s string) string {
	return toTypeName(s)
}

// toParamName converts an OpenAPI parameter name to a valid Go parameter name (camelCase).
// It handles special characters and escapes Go reserved words.
func toParamName(s string) string {
	name := toTypeName(s)
	if len(name) > 0 {
		// Convert first character to lowercase for camelCase
		name = strings.ToLower(name[:1]) + name[1:]
	} else {
		name = "param"
	}
	return escapeReservedWord(name)
}

// operationToMethodName generates a Go method name from an operation.
// It uses the operationId if available, otherwise generates from path and method.
func operationToMethodName(op *parser.Operation, path, method string) string {
	if op.OperationID != "" {
		return toTypeName(op.OperationID)
	}
	// Generate from path and method
	return generateMethodNameFromPathMethod(path, method)
}

// operationInfoToMethodName generates a Go method name from an OperationInfo.
// This is used by the file splitter to ensure consistent naming between
// split planning and code generation.
func operationInfoToMethodName(op *OperationInfo) string {
	if op.OperationID != "" {
		return toTypeName(op.OperationID)
	}
	// Generate from path and method
	return generateMethodNameFromPathMethod(op.Path, op.Method)
}

// generateMethodNameFromPathMethod generates a Go method name from path and HTTP method.
// This is the common implementation used when no operationId is provided.
func generateMethodNameFromPathMethod(path, method string) string {
	pathPart := path
	pathPart = strings.ReplaceAll(pathPart, "/", " ")
	pathPart = strings.ReplaceAll(pathPart, "{", "By ")
	pathPart = strings.ReplaceAll(pathPart, "}", "")
	return toTypeName(method + " " + pathPart)
}

// cleanDescription prepares an OpenAPI description for use in Go comments.
// It removes newlines, trims whitespace, and truncates long descriptions.
func cleanDescription(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) > maxDescriptionLength {
		// Truncate at rune boundary to avoid splitting multi-byte characters
		runes := []rune(s)
		if len(runes) > maxDescriptionLength-3 {
			s = string(runes[:maxDescriptionLength-3]) + "..."
		}
	}
	return s
}

// formatMultilineComment formats a description as multi-line Go comments.
// It handles newlines in the text and properly formats each line as a Go comment.
// The indent parameter specifies the indentation prefix (e.g., "" or "\t").
// The methodName is included as a prefix on the first line.
// If the text doesn't contain newlines, it's returned as a single-line comment.
func formatMultilineComment(text, methodName, indent string) string {
	if text == "" {
		return ""
	}

	var buf strings.Builder

	// Check if text contains newlines
	if !strings.Contains(text, "\n") {
		// Single line - simple case
		buf.WriteString(indent)
		buf.WriteString("// ")
		buf.WriteString(methodName)
		buf.WriteString(" ")
		buf.WriteString(text)
		buf.WriteString("\n")
		return buf.String()
	}

	// Multi-line case - split and format each line
	lines := strings.Split(text, "\n")

	// First line with method name prefix
	firstLine := strings.TrimSpace(lines[0])
	buf.WriteString(indent)
	buf.WriteString("// ")
	buf.WriteString(methodName)
	if firstLine != "" {
		buf.WriteString(" ")
		buf.WriteString(firstLine)
	}
	buf.WriteString("\n")

	// Remaining lines
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			buf.WriteString(indent)
			buf.WriteString("// ")
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}

	return buf.String()
}
