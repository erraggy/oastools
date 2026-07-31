package pathutil

import (
	"net/url"
	"strconv"
	"strings"
)

// EscapeRefToken escapes a component name for use as a single JSON Pointer
// token, per RFC 6901: "~" becomes "~0" and "/" becomes "~1".
//
// Order matters — "~" must be escaped first, or the "~" introduced by escaping
// "/" would itself be escaped and produce "~01".
//
// OAS 2.0 places no charset constraint on the keys of the root-level
// parameters, definitions, and responses objects, so a name containing either
// character is legitimate and must be escaped for a document to reference it.
// Building a reference by raw concatenation instead produces a pointer that
// names a different location and therefore never matches.
func EscapeRefToken(name string) string {
	// Fast path: the overwhelming majority of component names need no escaping.
	if !strings.ContainsAny(name, "~/") {
		return name
	}
	name = strings.ReplaceAll(name, "~", "~0")
	return strings.ReplaceAll(name, "/", "~1")
}

// UnescapeRefToken reverses [EscapeRefToken], recovering a component name from
// a single JSON Pointer token.
//
// Use it anywhere a reference is split back into a name — renaming, pruning, or
// resolving a ref to its definition — or names containing "/" or "~" are
// corrupted. Order is the inverse of escaping: "~1" must be unescaped before
// "~0", or the "~" recovered from "~0" could combine with a following "1".
//
// Mirrors unescapeJSONPointer in the parser's resolver, which has always
// implemented this for reference resolution.
func UnescapeRefToken(token string) string {
	// Fast path: a token with no "~" cannot carry an escape sequence.
	if !strings.Contains(token, "~") {
		return token
	}
	token = strings.ReplaceAll(token, "~1", "/")
	return strings.ReplaceAll(token, "~0", "~")
}

// DecodeRefToken reverses both escaping conventions a reference token may
// carry, recovering the component name it denotes.
//
// Documents are inconsistent here. RFC 6901 asks for "~1" and "~0", but code
// generators commonly percent-encode instead — and often encode only what their
// URL escaper considers unsafe, so "[" becomes "%5B" while "/" is left raw. A
// token like "Paged%5Bexample.com/pkg.Pet%5D" is therefore in neither
// convention, and no single unescaper recovers it.
//
// Percent-decoding runs first so a percent-encoded tilde ("%7E1") is still
// recognized as a JSON Pointer escape afterwards. An invalid percent sequence
// leaves the token untouched rather than failing: a token that cannot be
// decoded is already the only form it has.
//
// Decoding is lossy, so prefer the undecoded token when it already matches:
// a component genuinely named "Foo%20Bar" decodes to "Foo Bar" and would
// otherwise stop matching itself. Callers should try the exact spelling first
// and fall back to this.
func DecodeRefToken(token string) string {
	if decoded, err := url.PathUnescape(token); err == nil {
		token = decoded
	}
	return UnescapeRefToken(token)
}

// CutRefPrefix strips a JSON Pointer prefix from a reference, tolerating a
// reference whose pointer separators are percent-encoded, and returns the
// remainder with its own escaping untouched.
//
// Generators reach that shape by running a URL escaper over a whole pointer
// rather than over each token, so "#/components/schemas/Paged%5BPet%5D" arrives
// as "#%2Fcomponents%2Fschemas%2FPaged%255BPet%255D". Rare, but a reference
// spelled that way denotes exactly the same component as the raw form and has
// to resolve to it.
//
// prefix is matched one decoded byte at a time rather than against a wholesale
// [url.PathUnescape] of ref, because a wholesale decode would also decode the
// token that follows. That token has to reach [DecodeRefToken] still escaped for
// the exact-then-decoded lookup order to mean anything: decoding twice turns a
// component genuinely named "Paged%5BPet%5D" into "Paged[Pet]" and stops it
// matching itself. Keeping the decode confined to the prefix leaves
// [DecodeRefToken] the single place a name's escaping is reversed.
//
// The prefix returned to the caller is the raw one it passed in, never the
// spelling ref used, so a decoded reference lands on the same key as a raw one.
func CutRefPrefix(ref, prefix string) (rest string, ok bool) {
	// Fast path: almost every reference spells its prefix raw.
	if rest, found := strings.CutPrefix(ref, prefix); found {
		return rest, true
	}
	// Only a percent sequence can make the prefix match after decoding; RFC 6901
	// escapes cannot, because "~0" and "~1" never produce "#" or "/".
	if !strings.Contains(ref, "%") {
		return "", false
	}

	i := 0
	for p := 0; p < len(prefix); p++ {
		switch {
		case i < len(ref) && ref[i] == prefix[p]:
			i++
		case i+3 <= len(ref) && ref[i] == '%':
			// ParseUint accepts either hex case, so "%2f" and "%2F" both match "/".
			b, err := strconv.ParseUint(ref[i+1:i+3], 16, 8)
			if err != nil || byte(b) != prefix[p] {
				return "", false
			}
			i += 3
		default:
			return "", false
		}
	}
	return ref[i:], true
}

// OAS 2.0 reference prefixes
const (
	RefPrefixDefinitions         = "#/definitions/"
	RefPrefixParameters          = "#/parameters/"
	RefPrefixResponses           = "#/responses/"
	RefPrefixSecurityDefinitions = "#/securityDefinitions/"
)

// OAS 3.x reference prefixes
const (
	RefPrefixSchemas         = "#/components/schemas/"
	RefPrefixParameters3     = "#/components/parameters/"
	RefPrefixResponses3      = "#/components/responses/"
	RefPrefixExamples        = "#/components/examples/"
	RefPrefixRequestBodies   = "#/components/requestBodies/"
	RefPrefixHeaders         = "#/components/headers/"
	RefPrefixSecuritySchemes = "#/components/securitySchemes/"
	RefPrefixLinks           = "#/components/links/"
	RefPrefixCallbacks       = "#/components/callbacks/"
	RefPrefixPathItems       = "#/components/pathItems/"
)

// SchemaRef builds "#/components/schemas/{name}" (OAS 3.x).
func SchemaRef(name string) string {
	return RefPrefixSchemas + EscapeRefToken(name)
}

// DefinitionRef builds "#/definitions/{name}" (OAS 2.0).
func DefinitionRef(name string) string {
	return RefPrefixDefinitions + EscapeRefToken(name)
}

// ParameterRef builds the appropriate parameter ref.
// If oas2 is true, returns "#/parameters/{name}", otherwise "#/components/parameters/{name}".
func ParameterRef(name string, oas2 bool) string {
	if oas2 {
		return RefPrefixParameters + EscapeRefToken(name)
	}
	return RefPrefixParameters3 + EscapeRefToken(name)
}

// ResponseRef builds the appropriate response ref.
// If oas2 is true, returns "#/responses/{name}", otherwise "#/components/responses/{name}".
func ResponseRef(name string, oas2 bool) string {
	if oas2 {
		return RefPrefixResponses + EscapeRefToken(name)
	}
	return RefPrefixResponses3 + EscapeRefToken(name)
}

// SecuritySchemeRef builds the appropriate security scheme ref.
// If oas2 is true, returns "#/securityDefinitions/{name}", otherwise "#/components/securitySchemes/{name}".
func SecuritySchemeRef(name string, oas2 bool) string {
	if oas2 {
		return RefPrefixSecurityDefinitions + EscapeRefToken(name)
	}
	return RefPrefixSecuritySchemes + EscapeRefToken(name)
}

// HeaderRef builds "#/components/headers/{name}" (OAS 3.x only).
func HeaderRef(name string) string {
	return RefPrefixHeaders + EscapeRefToken(name)
}

// RequestBodyRef builds "#/components/requestBodies/{name}" (OAS 3.x only).
func RequestBodyRef(name string) string {
	return RefPrefixRequestBodies + EscapeRefToken(name)
}

// ExampleRef builds "#/components/examples/{name}" (OAS 3.x only).
func ExampleRef(name string) string {
	return RefPrefixExamples + EscapeRefToken(name)
}

// LinkRef builds "#/components/links/{name}" (OAS 3.x only).
func LinkRef(name string) string {
	return RefPrefixLinks + EscapeRefToken(name)
}

// CallbackRef builds "#/components/callbacks/{name}" (OAS 3.x only).
func CallbackRef(name string) string {
	return RefPrefixCallbacks + EscapeRefToken(name)
}

// PathItemRef builds "#/components/pathItems/{name}" (OAS 3.1+ only).
func PathItemRef(name string) string {
	return RefPrefixPathItems + EscapeRefToken(name)
}
