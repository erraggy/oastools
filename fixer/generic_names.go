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
	"unicode/utf8"

	"github.com/erraggy/oastools/internal/pathutil"
	"github.com/erraggy/oastools/parser"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
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
//
// This is the whole detection rule only under [charsetUnrestricted], where the
// spec imposes none of its own. OAS 3.x has an allowlist that already covers
// every character here, so [charsetComponentName] never consults this list —
// see [nameCharset.allowsInName].
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

// nameCharset is the set of characters a schema name may use. The OAS versions
// disagree, so both halves of a rename — deciding a name needs one, and building
// the replacement — are driven by this rather than by one hardcoded rule.
type nameCharset uint8

const (
	// charsetUnrestricted applies to OAS 2.0, which places no charset constraint
	// on the keys of the definitions object. A name there is renamed only for
	// the characters in [invalidSchemaNameChars], which is the fixer's own
	// judgement about what trips up tooling rather than anything the spec says.
	charsetUnrestricted nameCharset = iota

	// charsetComponentName applies to OAS 3.x, whose Components Object requires
	// every key to match ^[a-zA-Z0-9._-]+$. That rule is an allowlist, so a
	// denylist can never keep up with it: "pkg/Pet", "Pet@v1" and "Pét" are all
	// illegal without containing a single character worth enumerating.
	charsetComponentName
)

// charsetForVersion returns the charset a document's schema names must satisfy.
func charsetForVersion(version parser.OASVersion) nameCharset {
	if version == parser.OASVersion20 {
		return charsetUnrestricted
	}
	return charsetComponentName
}

// allowsInName reports whether r may appear in a name the fixer leaves alone.
//
// This is the detection rule, and it is deliberately not the same question as
// [nameCharset.allowsInReplacement]. Under OAS 2.0 a "/" is legal in a
// definitions key and resolves correctly once escaped, so renaming it would
// rewrite a valid document; the fixer only steps in for the characters it knows
// break tooling. Under OAS 3.x the spec itself settles it.
func (c nameCharset) allowsInName(r rune) bool {
	if c == charsetComponentName {
		return isComponentNameChar(r)
	}
	return !slices.Contains(invalidSchemaNameChars, r)
}

// allowsInReplacement reports whether r may appear in a name the fixer builds.
//
// Stricter than [nameCharset.allowsInName] under OAS 2.0: a generated name
// should not reintroduce a character that forces every $ref reaching it to be
// escaped, even where the spec would permit it.
func (c nameCharset) allowsInReplacement(r rune) bool {
	if c == charsetComponentName {
		return isComponentNameChar(r)
	}
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.'
}

// isComponentNameChar reports whether r is permitted by the OAS 3.x Components
// Object key pattern, ^[a-zA-Z0-9._-]+$.
//
// Spelled out rather than deferring to unicode.IsLetter: the pattern is ASCII
// only, so "é" and "宠" are letters that it nonetheless rejects.
func isComponentNameChar(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	default:
		return r == '.' || r == '-' || r == '_'
	}
}

// hasInvalidSchemaNameChars returns true if name contains characters that
// require the schema to be renamed under the given charset.
func hasInvalidSchemaNameChars(name string, charset nameCharset) bool {
	// Empty or whitespace-only names are invalid
	if strings.TrimSpace(name) == "" {
		return true
	}

	for _, c := range name {
		if !charset.allowsInName(c) {
			return true
		}
	}
	return false
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

	var (
		params  []string
		current strings.Builder
		depth   = 0
	)
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
func transformSchemaName(name string, config GenericNamingConfig, charset nameCharset) string {
	// Handle empty or whitespace-only names
	if strings.TrimSpace(name) == "" {
		return "UnnamedSchema"
	}

	base, params, _ := parseGenericName(name)

	// If no type parameters, the whole name is one fragment. It gets the same
	// treatment a base or a type parameter would, so a path-qualified name
	// reduces the same way whether or not it happens to be generic:
	// "example.com/pkg/Pet" and "Resp[example.com/pkg/Pet]" both keep their
	// qualification as dots rather than losing it to underscores.
	if len(params) == 0 {
		fragment := toNameFragment(name, charset)
		if fragment == "" {
			return "UnnamedSchema"
		}
		return fragment
	}

	// The base is spliced into the result below, so it carries whatever the
	// original name had in front of the bracket — "Res ponse[User]" and
	// "pkg/v1.Response[Pet]" both reach here. Only the parameters were being
	// cleaned, so the base needs the same treatment or the new name is no more
	// legal than the old one.
	base = toNameFragment(base, charset)

	// Recursively transform nested generic types in parameters
	transformedParams := make([]string, len(params))
	for i, param := range params {
		transformedParams[i] = transformTypeParam(param, config, charset)
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
func transformTypeParam(param string, config GenericNamingConfig, charset nameCharset) string {
	// Strip leading pointer asterisks (Go pointer syntax leaking from code generators)
	param = strings.TrimLeft(param, "*")

	// If it's a package-qualified name (like common.Pet), preserve the
	// qualification to avoid corrupting the reference, cleaning only the
	// characters that cannot survive into a schema name
	if isPackageQualifiedName(param) {
		return toNameFragment(param, charset)
	}

	// For generic types or simple names, apply normal transformation
	transformed := transformSchemaName(param, config, charset)

	// Apply PascalCase if not preserving casing (for non-package names)
	if !config.PreserveCasing {
		transformed = toPascalCase(transformed)
	}

	return transformed
}

// foldToASCII rewrites accented letters as their base letter, so "Pét" becomes
// "Pet" rather than "P_t".
//
// It decomposes each rune into a base plus its combining marks, drops the marks,
// and recomposes. That covers the Latin-alphabet names code generators actually
// produce; a script with no ASCII base form, such as "宠物", is returned
// unchanged and the caller's sanitizer replaces it. Transliterating those would
// need a per-script romanization table, which is a different job.
//
// The transformer is built per call rather than shared, because it carries state
// and the fixer may run concurrently. The ASCII fast path keeps that off the
// common route: almost every name is already ASCII.
func foldToASCII(name string) string {
	if isASCII(name) {
		return name
	}

	folded, _, err := transform.String(
		transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC), name)
	if err != nil {
		return name
	}
	return folded
}

// isASCII reports whether name is entirely ASCII.
func isASCII(name string) bool {
	for i := range len(name) {
		if name[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// toNameFragment reduces one fragment of a generated schema name — a base name
// or a package-qualified type parameter — to the characters a schema name may
// contain.
//
// Every fragment spliced into a transformed name needs this. Cleaning only the
// parameters leaves the base free to carry a bracket-adjacent space, comma, or
// brace straight through, producing a name as illegal as the one it replaced.
//
// Path separators are folded to "." before the general sanitizer runs, so they
// do not become the "_" it would otherwise produce: a dot is what the fragment
// already uses to qualify its package, and it keeps every segment, so two types
// differing only by package still reduce to distinct names.
func toNameFragment(name string, charset nameCharset) string {
	return sanitizeSchemaName(flattenPathQualifier(name), charset)
}

// flattenPathQualifier rewrites the "/" separators of a path-qualified type
// name as dots: "example.com/petstore/pkg/store.Pet" becomes
// "example.com.petstore.pkg.store.Pet".
//
// Code-first generators emit these for package-qualified generic parameters, and
// a "/" cannot stay in the new name: OAS 3.x rejects a component name outright
// unless it matches ^[a-zA-Z0-9._-]+$, and while OAS 2.0 permits one in a
// definitions key, every $ref reaching it then needs escaping — exactly the
// encoding trouble this fix removes.
//
// Empty segments are dropped, so a doubled, leading, or trailing slash cannot
// leave a stray dot behind.
func flattenPathQualifier(name string) string {
	if !strings.Contains(name, "/") {
		return name
	}
	return strings.Join(strings.FieldsFunc(name, func(r rune) bool { return r == '/' }), ".")
}

// sanitizeSchemaName removes or replaces invalid characters with underscores.
// This is a fallback for names that aren't cleanly generic-style but still
// contain problematic characters.
func sanitizeSchemaName(name string, charset nameCharset) string {
	// Fold before the loop so an accented letter reaches it as its base letter
	// rather than as a rune the ASCII charset has no choice but to replace.
	if charset == charsetComponentName {
		name = foldToASCII(name)
	}

	var result strings.Builder
	result.Grow(len(name))

	for _, r := range name {
		if charset.allowsInReplacement(r) {
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

// splitSchemaRef splits a schema $ref into its prefix and its name token, for
// either OAS version. ok is false for a ref that names something other than a
// local schema, an external one included.
func splitSchemaRef(ref string) (prefix, name string, ok bool) {
	if n, found := strings.CutPrefix(ref, pathutil.RefPrefixSchemas); found {
		return pathutil.RefPrefixSchemas, n, true
	}
	if n, found := strings.CutPrefix(ref, pathutil.RefPrefixDefinitions); found {
		return pathutil.RefPrefixDefinitions, n, true
	}
	return "", "", false
}

// canonicalSchemaRefKey reduces a schema $ref to the decoded key
// [buildRefRenameMap] registers alongside the undecoded one: the prefix
// verbatim, followed by the name with both escaping conventions reversed.
//
// This is the fallback half of the lookup, not the whole of it. Decoding is
// lossy — see [pathutil.DecodeRefToken] — so a ref is matched against its exact
// spelling first and only reduced here when that misses. Reducing
// unconditionally would stop a component genuinely named "Foo%20Bar[Pet]" from
// matching a $ref that spells it the same way.
//
// A ref that names something other than a schema is returned unchanged, and so
// matches no key.
func canonicalSchemaRefKey(ref string) string {
	// Without an escape sequence the ref is already its own decoded form. This
	// runs for every $ref in the document, so the fast path is worth it.
	if !strings.ContainsAny(ref, "%~") {
		return ref
	}
	prefix, name, ok := splitSchemaRef(ref)
	if !ok {
		return ref
	}
	return prefix + pathutil.DecodeRefToken(name)
}

// lookupRenamedRef resolves ref against the rename map, trying its exact
// spelling before the decoded fallback.
func lookupRenamedRef(ref string, renames map[string]string) (string, bool) {
	if newRef, ok := renames[ref]; ok {
		return newRef, true
	}
	newRef, ok := renames[canonicalSchemaRefKey(ref)]
	return newRef, ok
}

// rewriteSchemaRefs recursively rewrites $ref values in a schema using the rename map.
// Look refs up with [lookupRenamedRef] rather than indexing the map directly,
// or a ref that spells the name in any encoding will miss.
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
		if newRef, ok := lookupRenamedRef(schema.Ref, renames); ok {
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
			if newValue, ok := lookupRenamedMappingValue(ref, renames); ok {
				schema.Discriminator.Mapping[key] = newValue
			}
		}
	}
}

// lookupRenamedMappingValue resolves one discriminator mapping value against the
// rename map, in the spelling the document used it: a full $ref stays a $ref, a
// bare schema name stays a bare name.
//
// A mapping may name its target either way — "#/components/schemas/Dog" or just
// "Dog" — so a bare name is completed into a ref before lookup rather than
// compared against the map's keys directly. That routes it through the same
// exact-then-decoded match every other ref gets, so a bare name carrying an
// encoding resolves too, and an exact spelling still wins over a decoded one.
// Comparing against extracted key names instead would have to iterate the map,
// and no iteration order can honor that precedence.
//
// Both prefixes are tried because a rename map is built for one OAS version, so
// the other simply matches nothing.
func lookupRenamedMappingValue(ref string, renames map[string]string) (string, bool) {
	if newRef, ok := lookupRenamedRef(ref, renames); ok {
		return newRef, true
	}

	for _, prefix := range [2]string{pathutil.RefPrefixSchemas, pathutil.RefPrefixDefinitions} {
		if newRef, ok := lookupRenamedRef(prefix+ref, renames); ok {
			// Values are escaped for a pointer, so the escaping is reversed
			// before the name goes back as a bare value.
			return pathutil.UnescapeRefToken(extractSchemaNameFromRefPath(newRef)), true
		}
	}

	return "", false
}

// extractSchemaNameFromRefPath extracts the schema name from a $ref path.
// Returns empty string if not a schema reference.
func extractSchemaNameFromRefPath(ref string) string {
	_, name, _ := splitSchemaRef(ref)
	return name
}
