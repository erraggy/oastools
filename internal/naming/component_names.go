package naming

// ComponentNamePattern is the charset OAS 3.x requires of every fixed field of
// the Components Object: "All the fixed fields declared above are objects that
// MUST use keys that match the regular expression: ^[a-zA-Z0-9\.\-_]+$".
//
// See: https://spec.openapis.org/oas/v3.0.0.html#components-object
//
// The spec writes "." and "-" escaped inside the character class, where neither
// needs escaping; this is the same expression written plainly. It is a string
// rather than a compiled [regexp.Regexp] because nothing needs to run it —
// [IsValidComponentName] decides membership directly, and this exists so an
// error message can quote the rule it enforces.
//
// OAS 2.0 has no counterpart, deliberately. It places no charset constraint on
// the keys of the root-level parameters, definitions, and responses objects, so
// a name containing "/" or "~" is legitimate there and is escaped per RFC 6901
// when referenced. Applying this pattern to an OAS 2.0 document would reject
// valid specs.
const ComponentNamePattern = `^[a-zA-Z0-9._-]+$`

// IsComponentNameChar reports whether r may appear in an OAS 3.x component name.
//
// Spelled out rather than deferring to unicode.IsLetter or unicode.IsDigit:
// [ComponentNamePattern] is ASCII only, so "é" and "宠" are letters it
// nonetheless rejects, and "٣" is a digit it rejects.
func IsComponentNameChar(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	default:
		return r == '.' || r == '-' || r == '_'
	}
}

// IsValidComponentName reports whether name satisfies [ComponentNamePattern].
//
// This is the single implementation of that rule. Validation asks whether a
// document's names are legal and the fixer asks which characters it may keep
// when building a replacement, so both consume this rather than restating the
// charset; a hand-maintained copy on either side would be free to drift from
// the spec and from the other.
//
// The empty string is not valid: the pattern's "+" requires at least one
// character. Callers that want to report an empty name differently should test
// for it before calling.
func IsValidComponentName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !IsComponentNameChar(r) {
			return false
		}
	}
	return true
}
