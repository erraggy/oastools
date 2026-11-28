package stringutil

import (
	"net"
	"net/url"
	"regexp"
	"strings"
)

// spdxParenSpacer makes strings.Fields yield each parenthesis as its own token.
// spdxIDPattern matches a license or exception id, an optional "+" suffix on a
// license, or a user-defined LicenseRef with an optional DocumentRef prefix.
var spdxIDPattern = regexp.MustCompile(`^(?:[A-Za-z0-9.\-]+\+?|(?:DocumentRef-[A-Za-z0-9.\-]+:)?LicenseRef-[A-Za-z0-9.\-]+)$`)

// spdxExceptionPattern matches an exception id, or a user-defined AdditionRef
// with an optional DocumentRef prefix. Unlike a license, an exception takes no
// "+" suffix.
var spdxExceptionPattern = regexp.MustCompile(`^(?:[A-Za-z0-9.\-]+|(?:DocumentRef-[A-Za-z0-9.\-]+:)?AdditionRef-[A-Za-z0-9.\-]+)$`)

var spdxParenSpacer = strings.NewReplacer("(", " ( ", ")", " ) ")

// IsValidEmail checks if s is a valid email address as defined by the "Mailbox"
// ABNF rule in RFC 5321, section 4.1.2.
func IsValidEmail(s string) bool {
	if len(s) == 0 || len(s) > 254 {
		return false
	}

	var localPart, domainPart string

	if s[0] == '"' {
		// Quoted-string local part (RFC 5321 Section 4.1.2)
		i := 1
		closed := false
		for i < len(s) {
			c := s[i]
			if c == '\\' {
				i++
				if i >= len(s) || s[i] < 32 || s[i] > 126 {
					return false
				}
				i++
			} else if c == '"' {
				closed = true
				i++
				break
			} else if (c >= 32 && c <= 33) || (c >= 35 && c <= 91) || (c >= 93 && c <= 126) {
				i++
			} else {
				return false
			}
		}
		if !closed || i >= len(s) || s[i] != '@' {
			return false
		}
		localPart = s[:i]
		domainPart = s[i+1:]
	} else {
		at := strings.IndexByte(s, '@')
		if at < 0 {
			return false
		}
		localPart = s[:at]
		domainPart = s[at+1:]
		if !isValidDotString(localPart) {
			return false
		}
	}

	if len(localPart) == 0 || len(localPart) > 64 {
		return false
	}

	if len(domainPart) == 0 || len(domainPart) > 255 {
		return false
	}

	if strings.HasPrefix(domainPart, "[") && strings.HasSuffix(domainPart, "]") {
		return isValidAddressLiteral(domainPart[1 : len(domainPart)-1])
	}

	return isValidDomain(domainPart)
}

func isValidDotString(s string) bool {
	if len(s) == 0 || s[0] == '.' || s[len(s)-1] == '.' {
		return false
	}
	atoms := strings.Split(s, ".")
	for _, atom := range atoms {
		if len(atom) == 0 {
			return false
		}
		for i := 0; i < len(atom); i++ {
			if !isAtext(atom[i]) {
				return false
			}
		}
	}
	return true
}

func isAtext(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	case strings.IndexByte("!#$%&'*+-/=?^_`{|}~", c) >= 0:
		return true
	default:
		return false
	}
}

func isValidDomain(s string) bool {
	if len(s) == 0 || s[0] == '.' || s[len(s)-1] == '.' {
		return false
	}
	labels := strings.Split(s, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if !isAlphaNum(label[0]) || !isAlphaNum(label[len(label)-1]) {
			return false
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			if !isAlphaNum(c) && c != '-' {
				return false
			}
		}
	}
	return true
}

func isAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func isValidAddressLiteral(s string) bool {
	if len(s) == 0 {
		return false
	}
	if strings.HasPrefix(strings.ToLower(s), "ipv6:") {
		ipv6 := s[5:]
		ip := net.ParseIP(ipv6)
		return ip != nil && strings.Contains(ipv6, ":")
	}
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 || len(p) > 3 {
			return false
		}
		if len(p) > 1 && p[0] == '0' {
			return false
		}
		val := 0
		for i := 0; i < len(p); i++ {
			if p[i] < '0' || p[i] > '9' {
				return false
			}
			val = val*10 + int(p[i]-'0')
		}
		if val > 255 {
			return false
		}
	}
	return true
}

// IsValidURL reports whether s is a URI reference as RFC 3986 defines it, which
// is the format the OpenAPI Specification's schemas require of its URL fields.
// A relative reference such as "./terms.html" is valid, and so is any scheme.
// An http or https URL must also have a host, which RFC 9110 requires. The
// empty string is not accepted.
func IsValidURL(s string) bool {
	if s == "" || !isURIReferenceText(s) {
		return false
	}

	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	if u.Scheme == "http" || u.Scheme == "https" {
		return u.Hostname() != ""
	}
	return true
}

// isURIReferenceText reports whether every character of s may appear in an
// RFC 3986 URI reference: unreserved and reserved characters, and "%" followed
// by two hex digits. Anything else, including spaces and non-ASCII characters,
// must be percent-encoded.
func isURIReferenceText(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case strings.IndexByte("-._~:/?#[]@!$&'()*+,;=", c) >= 0:
		case c == '%':
			if i+2 >= len(s) || !isHexDigit(s[i+1]) || !isHexDigit(s[i+2]) {
				return false
			}
			i += 2
		default:
			return false
		}
	}
	return true
}

func isHexDigit(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

// IsValidSPDXExpression reports whether s is a well-formed SPDX license
// expression: ids joined by the uppercase operators AND or OR, an id followed
// by WITH and an exception id, and parentheses for grouping. The ids are not
// checked against the SPDX license list.
func IsValidSPDXExpression(s string) bool {
	tokens := strings.Fields(spdxParenSpacer.Replace(s))
	p := spdxParser{tokens: tokens}
	return p.expression() && p.pos == len(tokens)
}

type spdxParser struct {
	tokens []string
	pos    int
}

func (p *spdxParser) peek() string {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return ""
}

func isSPDXKeyword(t string) bool {
	return t == "AND" || t == "OR" || t == "WITH"
}

// id consumes one license or exception id.
func (p *spdxParser) id(pattern *regexp.Regexp) bool {
	t := p.peek()
	if isSPDXKeyword(t) || !pattern.MatchString(t) {
		return false
	}
	p.pos++
	return true
}

// expression parses term { (AND | OR) term }.
func (p *spdxParser) expression() bool {
	if !p.term() {
		return false
	}
	for p.peek() == "AND" || p.peek() == "OR" {
		p.pos++
		if !p.term() {
			return false
		}
	}
	return true
}

// term parses "(" expression ")" or id [WITH id].
func (p *spdxParser) term() bool {
	if p.peek() == "(" {
		p.pos++
		if !p.expression() || p.peek() != ")" {
			return false
		}
		p.pos++
		return true
	}
	if !p.id(spdxIDPattern) {
		return false
	}
	if p.peek() == "WITH" {
		p.pos++
		return p.id(spdxExceptionPattern)
	}
	return true
}
