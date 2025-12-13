// Package jsonpath provides a minimal JSONPath implementation for OpenAPI Overlay support.
//
// This package implements a subset of RFC 9535 JSONPath sufficient for OpenAPI Overlay
// v1.0.0 specification requirements. It supports path navigation, wildcards, and simple
// filter expressions without external dependencies.
//
// Supported syntax:
//   - $ (root)
//   - .field or ['field'] (child access)
//   - .* (wildcard - all children)
//   - [0] (array index)
//   - [?@.field==value] (simple equality filter)
//
// Not supported (planned for future):
//   - .. (recursive descent)
//   - [start:end:step] (array slicing)
//   - && and || (complex boolean filters)
//   - Filter functions like length(), count()
package jsonpath

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Path represents a parsed JSONPath expression.
type Path struct {
	raw      string
	segments []Segment
}

// String returns the original JSONPath expression.
func (p *Path) String() string {
	return p.raw
}

// Segment represents a single segment in a JSONPath expression.
type Segment interface {
	// segmentType returns a string identifying the segment type for debugging.
	segmentType() string
}

// RootSegment represents the root selector ($).
type RootSegment struct{}

func (s RootSegment) segmentType() string { return "root" }

// ChildSegment represents a child property selector (.field or ['field']).
type ChildSegment struct {
	Key string
}

func (s ChildSegment) segmentType() string { return "child" }

// WildcardSegment represents a wildcard selector (.*).
type WildcardSegment struct{}

func (s WildcardSegment) segmentType() string { return "wildcard" }

// IndexSegment represents an array index selector ([n]).
type IndexSegment struct {
	Index int
}

func (s IndexSegment) segmentType() string { return "index" }

// FilterSegment represents a filter selector ([?expr]).
type FilterSegment struct {
	Expr *FilterExpr
}

func (s FilterSegment) segmentType() string { return "filter" }

// FilterExpr represents a simple filter expression.
type FilterExpr struct {
	Field    string // Field path after @ (e.g., "name" for @.name)
	Operator string // ==, !=, <, >, <=, >=
	Value    any    // The comparison value (string, number, bool, nil)
}

// Parse parses a JSONPath expression string into a Path.
//
// Examples:
//
//	Parse("$.info")                    // Navigate to info object
//	Parse("$.paths['/users'].get")    // Navigate to specific operation
//	Parse("$.paths.*.get")            // All GET operations
//	Parse("$.paths.*[?@.x-internal==true]")  // Filter by extension
func Parse(expr string) (*Path, error) {
	if expr == "" {
		return nil, fmt.Errorf("jsonpath: empty expression")
	}

	p := &parser{
		input: expr,
		pos:   0,
	}

	segments, err := p.parse()
	if err != nil {
		return nil, err
	}

	return &Path{
		raw:      expr,
		segments: segments,
	}, nil
}

// parser is the internal JSONPath parser.
type parser struct {
	input string
	pos   int
}

func (p *parser) parse() ([]Segment, error) {
	var segments []Segment

	// Must start with $
	if !p.consume('$') {
		return nil, fmt.Errorf("jsonpath: expression must start with '$'")
	}
	segments = append(segments, RootSegment{})

	// Parse remaining segments
	for p.pos < len(p.input) {
		ch := p.peek()

		switch ch {
		case '.':
			p.advance()
			seg, err := p.parseDotSegment()
			if err != nil {
				return nil, err
			}
			segments = append(segments, seg)

		case '[':
			p.advance()
			seg, err := p.parseBracketSegment()
			if err != nil {
				return nil, err
			}
			segments = append(segments, seg)

		default:
			return nil, fmt.Errorf("jsonpath: unexpected character %q at position %d", ch, p.pos)
		}
	}

	return segments, nil
}

func (p *parser) parseDotSegment() (Segment, error) {
	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("jsonpath: unexpected end after '.'")
	}

	// Check for wildcard
	if p.peek() == '*' {
		p.advance()
		return WildcardSegment{}, nil
	}

	// Parse identifier
	key := p.parseIdentifier()
	if key == "" {
		return nil, fmt.Errorf("jsonpath: expected identifier after '.' at position %d", p.pos)
	}

	return ChildSegment{Key: key}, nil
}

func (p *parser) parseBracketSegment() (Segment, error) {
	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("jsonpath: unexpected end after '['")
	}

	ch := p.peek()

	// Filter expression: [?...]
	if ch == '?' {
		p.advance()
		return p.parseFilterSegment()
	}

	// Wildcard: [*]
	if ch == '*' {
		p.advance()
		if !p.consume(']') {
			return nil, fmt.Errorf("jsonpath: expected ']' after '[*'")
		}
		return WildcardSegment{}, nil
	}

	// Quoted string: ['key'] or ["key"]
	if ch == '\'' || ch == '"' {
		quote := ch
		p.advance()
		key, err := p.parseQuotedString(quote)
		if err != nil {
			return nil, err
		}
		if !p.consume(']') {
			return nil, fmt.Errorf("jsonpath: expected ']' after quoted key")
		}
		return ChildSegment{Key: key}, nil
	}

	// Numeric index or unquoted key
	if unicode.IsDigit(rune(ch)) || ch == '-' {
		numStr := p.parseNumber()
		if !p.consume(']') {
			return nil, fmt.Errorf("jsonpath: expected ']' after index")
		}
		idx, err := strconv.Atoi(numStr)
		if err != nil {
			return nil, fmt.Errorf("jsonpath: invalid index %q: %w", numStr, err)
		}
		return IndexSegment{Index: idx}, nil
	}

	return nil, fmt.Errorf("jsonpath: unexpected character %q in bracket at position %d", ch, p.pos)
}

func (p *parser) parseFilterSegment() (Segment, error) {
	// Skip optional opening parenthesis (legacy syntax support)
	hadParen := p.consume('(')

	// Skip optional @ at start
	p.consume('@')

	// Must have . after @
	if !p.consume('.') {
		return nil, fmt.Errorf("jsonpath: expected '@.' in filter expression at position %d", p.pos)
	}

	// Parse field name
	field := p.parseIdentifier()
	if field == "" {
		return nil, fmt.Errorf("jsonpath: expected field name in filter at position %d", p.pos)
	}

	// Skip whitespace
	p.skipWhitespace()

	// Parse operator
	op := p.parseOperator()
	if op == "" {
		return nil, fmt.Errorf("jsonpath: expected operator in filter at position %d", p.pos)
	}

	// Skip whitespace
	p.skipWhitespace()

	// Parse value
	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	// Skip optional closing parenthesis
	if hadParen {
		p.skipWhitespace()
		p.consume(')')
	}

	// Must end with ]
	if !p.consume(']') {
		return nil, fmt.Errorf("jsonpath: expected ']' after filter expression")
	}

	return FilterSegment{
		Expr: &FilterExpr{
			Field:    field,
			Operator: op,
			Value:    value,
		},
	}, nil
}

func (p *parser) parseIdentifier() string {
	start := p.pos
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		// Allow alphanumeric, underscore, hyphen (for x-* extensions)
		if isIdentChar(ch) {
			p.pos++
		} else {
			break
		}
	}
	return p.input[start:p.pos]
}

func (p *parser) parseQuotedString(quote byte) (string, error) {
	var result strings.Builder
	for p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == quote {
			p.pos++
			return result.String(), nil
		}
		if ch == '\\' && p.pos+1 < len(p.input) {
			p.pos++
			escaped := p.input[p.pos]
			switch escaped {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case '\\':
				result.WriteByte('\\')
			case '\'':
				result.WriteByte('\'')
			case '"':
				result.WriteByte('"')
			default:
				result.WriteByte(escaped)
			}
			p.pos++
			continue
		}
		result.WriteByte(ch)
		p.pos++
	}
	return "", fmt.Errorf("jsonpath: unterminated string at position %d", p.pos)
}

func (p *parser) parseNumber() string {
	start := p.pos
	if p.pos < len(p.input) && p.input[p.pos] == '-' {
		p.pos++
	}
	for p.pos < len(p.input) && unicode.IsDigit(rune(p.input[p.pos])) {
		p.pos++
	}
	// Handle decimal part
	if p.pos < len(p.input) && p.input[p.pos] == '.' {
		p.pos++
		for p.pos < len(p.input) && unicode.IsDigit(rune(p.input[p.pos])) {
			p.pos++
		}
	}
	return p.input[start:p.pos]
}

func (p *parser) parseOperator() string {
	if p.pos+1 < len(p.input) {
		twoChar := p.input[p.pos : p.pos+2]
		switch twoChar {
		case "==", "!=", "<=", ">=":
			p.pos += 2
			return twoChar
		}
	}
	if p.pos < len(p.input) {
		ch := p.input[p.pos]
		if ch == '<' || ch == '>' {
			p.pos++
			return string(ch)
		}
	}
	return ""
}

func (p *parser) parseValue() (any, error) {
	p.skipWhitespace()

	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("jsonpath: expected value at position %d", p.pos)
	}

	ch := p.peek()

	// Quoted string
	if ch == '\'' || ch == '"' {
		quote := ch
		p.advance()
		return p.parseQuotedString(quote)
	}

	// Boolean or null
	if strings.HasPrefix(p.input[p.pos:], "true") {
		p.pos += 4
		return true, nil
	}
	if strings.HasPrefix(p.input[p.pos:], "false") {
		p.pos += 5
		return false, nil
	}
	if strings.HasPrefix(p.input[p.pos:], "null") {
		p.pos += 4
		return nil, nil
	}

	// Number
	if unicode.IsDigit(rune(ch)) || ch == '-' {
		numStr := p.parseNumber()
		if strings.Contains(numStr, ".") {
			f, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return nil, fmt.Errorf("jsonpath: invalid number %q: %w", numStr, err)
			}
			return f, nil
		}
		i, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("jsonpath: invalid number %q: %w", numStr, err)
		}
		return i, nil
	}

	return nil, fmt.Errorf("jsonpath: unexpected character %q when parsing value at position %d", ch, p.pos)
}

func (p *parser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func (p *parser) advance() {
	if p.pos < len(p.input) {
		p.pos++
	}
}

func (p *parser) consume(ch byte) bool {
	if p.peek() == ch {
		p.advance()
		return true
	}
	return false
}

func (p *parser) skipWhitespace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func isIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_' || ch == '-'
}
