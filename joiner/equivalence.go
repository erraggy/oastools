package joiner

import (
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/erraggy/oastools/internal/equalutil"
	"github.com/erraggy/oastools/parser"
)

// comparePath is a stack-based path builder that minimizes allocations.
// Instead of creating a new string on each recursive call, it appends
// segments to a slice and only builds the full string when needed.
type comparePath struct {
	segments []string
}

func (p *comparePath) push(segment string) {
	p.segments = append(p.segments, segment)
}

func (p *comparePath) pop() {
	if len(p.segments) > 0 {
		p.segments = p.segments[:len(p.segments)-1]
	}
}

func (p *comparePath) String() string {
	return strings.Join(p.segments, ".")
}

// EquivalenceMode defines how deeply to compare schemas
type EquivalenceMode string

const (
	// EquivalenceModeNone disables equivalence detection
	EquivalenceModeNone EquivalenceMode = "none"
	// EquivalenceModeShallow compares only top-level schema properties
	EquivalenceModeShallow EquivalenceMode = "shallow"
	// EquivalenceModeDeep recursively compares all nested schemas
	EquivalenceModeDeep EquivalenceMode = "deep"
)

// ValidEquivalenceModes returns all valid equivalence mode strings
func ValidEquivalenceModes() []string {
	return []string{
		string(EquivalenceModeNone),
		string(EquivalenceModeShallow),
		string(EquivalenceModeDeep),
	}
}

// IsValidEquivalenceMode checks if an equivalence mode string is valid
func IsValidEquivalenceMode(mode string) bool {
	switch EquivalenceMode(mode) {
	case EquivalenceModeNone, EquivalenceModeShallow, EquivalenceModeDeep:
		return true
	default:
		return false
	}
}

// EquivalenceDocs controls whether documentation-oriented metadata fields
// (title, description, example, examples) participate in schema equivalence.
//
// When EquivalenceDocsInclude (the default) is selected, two schemas that
// differ only in their documentation are treated as non-equivalent. This
// matters for semantic deduplication: merging schemas with divergent
// descriptions causes the canonical schema's docs to replace the other
// schemas' docs at every $ref site, which produces misleading API
// documentation (for example, a 403 response pointing at a schema whose
// description says "The request is invalid").
//
// When EquivalenceDocsIgnore is selected, title/description/example/examples
// are ignored during comparison. Use this when you intentionally want
// structurally identical schemas to be consolidated even if their prose
// differs, and the docs of the surviving canonical schema are acceptable for
// every reference site.
type EquivalenceDocs string

const (
	// EquivalenceDocsInclude makes title, description, example, and examples
	// part of the comparison. Default and recommended.
	EquivalenceDocsInclude EquivalenceDocs = "include"
	// EquivalenceDocsIgnore skips documentation metadata during comparison.
	EquivalenceDocsIgnore EquivalenceDocs = "ignore"
)

// ValidEquivalenceDocs returns all valid equivalence docs strings
func ValidEquivalenceDocs() []string {
	return []string{
		string(EquivalenceDocsInclude),
		string(EquivalenceDocsIgnore),
	}
}

// IsValidEquivalenceDocs checks if an equivalence docs string is valid
func IsValidEquivalenceDocs(mode string) bool {
	switch EquivalenceDocs(mode) {
	case EquivalenceDocsInclude, EquivalenceDocsIgnore:
		return true
	default:
		return false
	}
}

// CompareOptions configures schema equivalence comparison behavior.
//
// Mode controls structural comparison depth.
// Docs controls whether documentation metadata participates in comparison.
type CompareOptions struct {
	// Mode controls comparison depth: none, shallow, or deep.
	Mode EquivalenceMode
	// Docs controls whether title/description/example/examples participate.
	// Empty value is treated as EquivalenceDocsInclude (strict).
	Docs EquivalenceDocs
}

// defaultCompareOptions builds a CompareOptions with the provided mode and
// the strict (include) documentation policy.
func defaultCompareOptions(mode EquivalenceMode) CompareOptions {
	return CompareOptions{Mode: mode, Docs: EquivalenceDocsInclude}
}

// EquivalenceResult contains the outcome of schema comparison
type EquivalenceResult struct {
	Equivalent  bool
	Differences []SchemaDifference
}

// isEmptySchema reports whether a schema has no structural constraints.
// A schema is considered "empty" if it has no type, format, properties,
// validation rules, or composition keywords. Metadata fields (title,
// description, example, deprecated, extensions) are NOT considered constraints.
//
// Constraint fields checked:
//   - Basic: Type, Format, Enum, Const, Pattern, Required
//   - OAS-specific: Nullable, ReadOnly, WriteOnly, CollectionFormat
//   - Object: Properties, AdditionalProperties, MinProperties, MaxProperties,
//     PatternProperties, DependentRequired
//   - Array: Items, MinItems, MaxItems, UniqueItems, AdditionalItems,
//     MaxContains, MinContains
//   - Numeric: Minimum, Maximum, MultipleOf, ExclusiveMinimum, ExclusiveMaximum
//   - String: MinLength, MaxLength
//   - Composition: AllOf, AnyOf, OneOf, Not
//   - Conditional: If, Then, Else
//   - JSON Schema 2020-12: UnevaluatedProperties, UnevaluatedItems,
//     ContentEncoding, ContentMediaType, ContentSchema, PrefixItems,
//     Contains, PropertyNames, DependentSchemas
//
// Empty schemas are semantically distinct even when structurally identical,
// because they serve different purposes depending on context (placeholders,
// "any type" markers, context-specific wildcards). Returning false for nil
// schemas prevents nil-pointer panics in callers.
func isEmptySchema(s *parser.Schema) bool {
	if s == nil {
		return false
	}

	// A bare-boolean schema is the opposite of empty: `false` rejects every
	// instance and `true` accepts every instance, both deliberately. Without
	// this, CompareSchemasWithOptions took the empty-schema early return and
	// reported two identical `true` schemas as non-equivalent — with an empty
	// Differences slice, since nothing had actually been compared.
	if s.BoolForm != nil {
		return false
	}

	// Basic type constraints
	if s.Type != nil {
		return false
	}
	if s.Format != "" {
		return false
	}
	if len(s.Enum) > 0 {
		return false
	}
	if s.Const != nil {
		return false
	}
	if s.Pattern != "" {
		return false
	}
	if len(s.Required) > 0 {
		return false
	}

	// OAS-specific constraints
	if s.Nullable {
		return false
	}
	if s.ReadOnly {
		return false
	}
	if s.WriteOnly {
		return false
	}
	if s.CollectionFormat != "" {
		return false
	}

	// Properties and object constraints
	if len(s.Properties) > 0 {
		return false
	}
	if s.AdditionalProperties != nil {
		return false
	}
	if s.MinProperties != nil {
		return false
	}
	if s.MaxProperties != nil {
		return false
	}
	if len(s.PatternProperties) > 0 {
		return false
	}
	if len(s.DependentRequired) > 0 {
		return false
	}

	// Array constraints
	if s.Items != nil {
		return false
	}
	if s.MinItems != nil {
		return false
	}
	if s.MaxItems != nil {
		return false
	}
	if s.UniqueItems {
		return false
	}
	if s.AdditionalItems != nil {
		return false
	}
	if s.MaxContains != nil {
		return false
	}
	if s.MinContains != nil {
		return false
	}

	// Numeric constraints
	if s.Minimum != nil {
		return false
	}
	if s.Maximum != nil {
		return false
	}
	if s.MultipleOf != nil {
		return false
	}
	if s.ExclusiveMinimum != nil {
		return false
	}
	if s.ExclusiveMaximum != nil {
		return false
	}

	// String constraints
	if s.MinLength != nil {
		return false
	}
	if s.MaxLength != nil {
		return false
	}

	// Composition
	if len(s.AllOf) > 0 {
		return false
	}
	if len(s.AnyOf) > 0 {
		return false
	}
	if len(s.OneOf) > 0 {
		return false
	}
	if s.Not != nil {
		return false
	}

	// Conditional composition
	if s.If != nil {
		return false
	}
	if s.Then != nil {
		return false
	}
	if s.Else != nil {
		return false
	}

	// JSON Schema 2020-12 fields
	if s.UnevaluatedProperties != nil {
		return false
	}
	if s.UnevaluatedItems != nil {
		return false
	}
	if s.ContentEncoding != "" {
		return false
	}
	if s.ContentMediaType != "" {
		return false
	}
	if s.ContentSchema != nil {
		return false
	}
	if len(s.PrefixItems) > 0 {
		return false
	}
	if s.Contains != nil {
		return false
	}
	if s.PropertyNames != nil {
		return false
	}
	if len(s.DependentSchemas) > 0 {
		return false
	}

	return true
}

// String returns a human-readable representation of the equivalence result.
// Special case: When Equivalent is false but Differences is non-nil and empty,
// this indicates empty schemas that are structurally identical but semantically distinct.
func (r EquivalenceResult) String() string {
	if r.Equivalent {
		return "Schemas are equivalent"
	}
	if r.Differences != nil && len(r.Differences) == 0 {
		return "Schemas are non-equivalent (empty schemas are semantically distinct)"
	}
	var b strings.Builder
	b.WriteString("Schemas differ:\n")
	for _, d := range r.Differences {
		fmt.Fprintf(&b, "  - %s: %s\n", d.Path, d.Description)
	}
	return b.String()
}

// CompareSchemas compares two schemas for structural equivalence using the
// strict (docs-included) comparison policy. Title, description, example, and
// examples all participate in the comparison by default; two schemas that
// differ only in those fields will be reported as non-equivalent.
//
// Callers that need the legacy behavior of ignoring documentation metadata
// can use CompareSchemasWithOptions with Docs set to EquivalenceDocsIgnore.
func CompareSchemas(left, right *parser.Schema, mode EquivalenceMode) EquivalenceResult {
	return CompareSchemasWithOptions(left, right, defaultCompareOptions(mode))
}

// CompareSchemasWithOptions compares two schemas using the provided options.
//
// When opts.Docs is empty, it is treated as EquivalenceDocsInclude (strict).
// See CompareOptions and EquivalenceDocs for details.
func CompareSchemasWithOptions(left, right *parser.Schema, opts CompareOptions) EquivalenceResult {
	return compareSchemas(left, right, opts, nil, nil)
}

// compareSchemas compares two schemas, reading each side's reference targets
// through its own view. Both views are nil for the exported entry points, whose
// operands already name what they resolve to; see refView for what a non-nil
// one is for.
func compareSchemas(left, right *parser.Schema, opts CompareOptions, leftView, rightView *refView) EquivalenceResult {
	if opts.Mode == EquivalenceModeNone {
		return EquivalenceResult{Equivalent: false}
	}
	if opts.Docs == "" {
		opts.Docs = EquivalenceDocsInclude
	}

	result := EquivalenceResult{
		Differences: make([]SchemaDifference, 0),
	}

	// Handle nil schemas
	if left == nil && right == nil {
		result.Equivalent = true
		return result
	}
	if left == nil || right == nil {
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        "",
			LeftValue:   left != nil,
			RightValue:  right != nil,
			Description: "schema presence mismatch (one is nil)",
		})
		result.Equivalent = false
		return result
	}

	// Empty schemas are semantically distinct - never equivalent.
	// They serve different purposes depending on context (placeholders,
	// "any type" markers, context-specific wildcards) and should not be
	// consolidated during deduplication.
	if isEmptySchema(left) || isEmptySchema(right) {
		return EquivalenceResult{
			Equivalent:  false,
			Differences: []SchemaDifference{},
		}
	}

	st := &compareState{
		result: &result,
		// Tracks visited pointers to handle circular references.
		visited: make(map[pointerPair]bool),
		docs:    opts.Docs == EquivalenceDocsInclude,
		left:    leftView,
		right:   rightView,
	}

	// Use stack-based path builder to minimize allocations
	path := &comparePath{segments: make([]string, 0, 8)}

	// A bare-boolean operand settles the comparison on its own. compareDeep
	// repeats this for nested positions; running it here too is what covers
	// shallow mode, which does not recurse.
	if !equalBoolForms(left, right, path, &result) {
		result.Equivalent = len(result.Differences) == 0
		return result
	}

	if opts.Mode == EquivalenceModeShallow {
		compareShallow(left, right, path, st)
	} else {
		compareDeep(left, right, path, st)
	}

	result.Equivalent = len(result.Differences) == 0
	return result
}

// pointerPair tracks schema pointer pairs to detect cycles
type pointerPair struct {
	left  uintptr
	right uintptr
}

// refView maps the schema names one source document spells to the names those
// schemas end up under in the joined document. A nil view is the identity: the
// names it is asked about are already final.
//
// It lets a comparison run while renames are known but not yet applied and
// still reach the verdict the rewritten documents would produce. Two schemas
// that both say `#/definitions/Category` are interchangeable only if that
// spelling resolves to the same schema for both, which it does not once one
// document's Category has been renamed out of the way. Comparing the spellings
// alone merges schemas whose meanings diverge further down (#487).
type refView struct {
	// refs maps a full $ref path, "#/components/schemas/Old" to
	// "#/components/schemas/New".
	refs map[string]string
	// names maps the bare form, "Old" to "New", which a discriminator entry
	// may use instead of a full path.
	names map[string]string
}

// ref returns the $ref path value resolves to once the renames are applied.
func (v *refView) ref(value string) string {
	if v == nil {
		return value
	}
	if mapped, ok := v.refs[value]; ok {
		return mapped
	}
	return value
}

// name returns what a discriminator entry resolves to.
func (v *refView) name(value string) string {
	if v == nil {
		return value
	}
	// A discriminator may name a schema either way. Full path first, bare name
	// second, matching the order SchemaRewriter resolves them in.
	if mapped, ok := v.refs[value]; ok {
		return mapped
	}
	if mapped, ok := v.names[value]; ok {
		return mapped
	}
	return value
}

// compareState carries what a comparison needs beyond the pair of schemas at
// hand.
type compareState struct {
	result  *EquivalenceResult
	visited map[pointerPair]bool
	// docs reports whether documentation metadata participates in the
	// comparison. See EquivalenceDocs.
	docs bool
	// left and right resolve each side's reference targets. Both are nil
	// unless the comparison is running ahead of a pending rewrite.
	left, right *refView
}

// compareDocFields compares documentation metadata fields (title, description,
// example, examples) and records any differences on the result.
//
// Only invoked when the caller has elected to include documentation in
// equivalence (Docs == EquivalenceDocsInclude).
func compareDocFields(left, right *parser.Schema, path *comparePath, result *EquivalenceResult) {
	if left.Title != right.Title {
		path.push("title")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Title,
			RightValue:  right.Title,
			Description: "title mismatch",
		})
		path.pop()
	}

	if left.Description != right.Description {
		path.push("description")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Description,
			RightValue:  right.Description,
			Description: "description mismatch",
		})
		path.pop()
	}

	if !reflect.DeepEqual(left.Example, right.Example) {
		path.push("example")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Example,
			RightValue:  right.Example,
			Description: "example mismatch",
		})
		path.pop()
	}

	if !reflect.DeepEqual(left.Examples, right.Examples) {
		path.push("examples")
		result.Differences = append(result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Examples,
			RightValue:  right.Examples,
			Description: "examples mismatch",
		})
		path.pop()
	}

	// $comment, externalDocs and deprecated are documentation and advisory rather
	// than structural, so they belong to this set: they count by default, and are
	// ignored under EquivalenceDocsIgnore along with the four above.
	if left.Comment != right.Comment {
		result.record(path, "$comment", left.Comment, right.Comment, "$comment mismatch")
	}
	if !equalExternalDocs(left.ExternalDocs, right.ExternalDocs) {
		result.record(path, "externalDocs", left.ExternalDocs, right.ExternalDocs, "externalDocs mismatch")
	}
	if left.Deprecated != right.Deprecated {
		result.record(path, "deprecated", left.Deprecated, right.Deprecated, "deprecated mismatch")
	}
}

// record appends one field difference at path. The existing comparisons above
// spell this out inline; new ones use this so the check is the line you read.
func (r *EquivalenceResult) record(path *comparePath, field string, left, right any, description string) {
	path.push(field)
	r.Differences = append(r.Differences, SchemaDifference{
		Path:        path.String(),
		LeftValue:   left,
		RightValue:  right,
		Description: description,
	})
	path.pop()
}

// compareSchemaMaps compares two name-keyed schema maps deeply.
func compareSchemaMaps(
	field string,
	left, right map[string]*parser.Schema,
	path *comparePath,
	st *compareState,
) {
	if !equalPropertyNames(left, right) {
		st.result.record(path, field, getPropertyNames(left), getPropertyNames(right), field+" names mismatch")
		return
	}
	if left == nil {
		return
	}
	path.push(field)
	for name, leftValue := range left {
		// A nil entry needs no guard here: compareDeep checks its own operands and
		// records at this path.
		path.push(name)
		compareDeep(leftValue, right[name], path, st)
		path.pop()
	}
	path.pop()
}

// compareStructuralSchemaFields compares the Schema fields that affect what a
// document means but are not reached by the comparisons written out above.
//
// They were all unchecked until issue #410: deep comparison read 38 of the 65
// fields, so schemas differing only in nullability, in which property
// discriminates a union, or in OAS 2.0 array serialization were reported
// equivalent and merged. internal/driftguard is what keeps this list complete.
func compareStructuralSchemaFields(
	left, right *parser.Schema,
	path *comparePath,
	st *compareState,
) {
	// JSON Schema identity and dialect: these decide what a $ref or $dynamicRef
	// resolves to and which vocabulary validates it.
	//
	// Each side goes through its own view, so what is compared is what the two
	// references resolve to rather than how they are spelled.
	if leftRef, rightRef := st.left.ref(left.Ref), st.right.ref(right.Ref); leftRef != rightRef {
		st.result.record(path, "$ref", leftRef, rightRef, "$ref target mismatch")
	}
	if left.Schema != right.Schema {
		st.result.record(path, "$schema", left.Schema, right.Schema, "$schema mismatch")
	}
	if left.ID != right.ID {
		st.result.record(path, "$id", left.ID, right.ID, "$id mismatch")
	}
	if left.Anchor != right.Anchor {
		st.result.record(path, "$anchor", left.Anchor, right.Anchor, "$anchor mismatch")
	}
	if left.DynamicRef != right.DynamicRef {
		st.result.record(path, "$dynamicRef", left.DynamicRef, right.DynamicRef, "$dynamicRef mismatch")
	}
	if left.DynamicAnchor != right.DynamicAnchor {
		st.result.record(path, "$dynamicAnchor", left.DynamicAnchor, right.DynamicAnchor, "$dynamicAnchor mismatch")
	}
	// maps.Equal, not reflect.DeepEqual: this package and parser both treat a nil
	// map and an empty one as equal, and DeepEqual splits them, which made a schema
	// declaring `$vocabulary: {}` differ from one declaring none.
	if !maps.Equal(left.Vocabulary, right.Vocabulary) {
		st.result.record(path, "$vocabulary", left.Vocabulary, right.Vocabulary, "$vocabulary mismatch")
	}

	// Value and serialization semantics.
	if !reflect.DeepEqual(left.Default, right.Default) {
		st.result.record(path, "default", left.Default, right.Default, "default mismatch")
	}
	if left.CollectionFormat != right.CollectionFormat {
		st.result.record(path, "collectionFormat", left.CollectionFormat, right.CollectionFormat,
			"collectionFormat mismatch")
	}

	// OAS flags. Merging across any of these changes what a payload may contain.
	if left.Nullable != right.Nullable {
		st.result.record(path, "nullable", left.Nullable, right.Nullable, "nullable mismatch")
	}
	if left.ReadOnly != right.ReadOnly {
		st.result.record(path, "readOnly", left.ReadOnly, right.ReadOnly, "readOnly mismatch")
	}
	if left.WriteOnly != right.WriteOnly {
		st.result.record(path, "writeOnly", left.WriteOnly, right.WriteOnly, "writeOnly mismatch")
	}

	// Numeric and array constraints.
	if !equalutil.EqualPtr(left.MultipleOf, right.MultipleOf) {
		st.result.record(path, "multipleOf", left.MultipleOf, right.MultipleOf, "multipleOf constraint mismatch")
	}
	if !reflect.DeepEqual(left.ExclusiveMaximum, right.ExclusiveMaximum) {
		st.result.record(path, "exclusiveMaximum", left.ExclusiveMaximum, right.ExclusiveMaximum,
			"exclusiveMaximum constraint mismatch")
	}
	if !reflect.DeepEqual(left.ExclusiveMinimum, right.ExclusiveMinimum) {
		st.result.record(path, "exclusiveMinimum", left.ExclusiveMinimum, right.ExclusiveMinimum,
			"exclusiveMinimum constraint mismatch")
	}
	if !equalutil.EqualPtr(left.MaxContains, right.MaxContains) {
		st.result.record(path, "maxContains", left.MaxContains, right.MaxContains, "maxContains constraint mismatch")
	}
	if !equalutil.EqualPtr(left.MinContains, right.MinContains) {
		st.result.record(path, "minContains", left.MinContains, right.MinContains, "minContains constraint mismatch")
	}
	if !equalStringSliceMaps(left.DependentRequired, right.DependentRequired) {
		st.result.record(path, "dependentRequired", left.DependentRequired, right.DependentRequired,
			"dependentRequired mismatch")
	}

	// Polymorphism. A discriminator names the property that selects a subschema,
	// so two schemas discriminating differently describe different payloads.
	if !equalDiscriminators(left.Discriminator, right.Discriminator, st) {
		st.result.record(path, "discriminator", left.Discriminator, right.Discriminator, "discriminator mismatch")
	}

	// Nested schemas.
	compareSchemaOrBool("additionalItems", left.AdditionalItems, right.AdditionalItems, path, st)
	compareSchemaMaps("patternProperties", left.PatternProperties, right.PatternProperties, path, st)
	compareSchemaMaps("$defs", left.Defs, right.Defs, path, st)
	for _, c := range []struct {
		name        string
		left, right *parser.Schema
	}{
		{"if", left.If, right.If},
		{"then", left.Then, right.Then},
		{"else", left.Else, right.Else},
	} {
		if (c.left == nil) != (c.right == nil) {
			st.result.record(path, c.name, c.left != nil, c.right != nil, c.name+" presence mismatch")
			continue
		}
		if c.left != nil {
			path.push(c.name)
			compareDeep(c.left, c.right, path, st)
			path.pop()
		}
	}
}

// equalDiscriminators compares two Discriminator Objects.
//
// StringForm is excluded for the same reason parser's equalDiscriminator excludes
// it: it records which dialect spelled the discriminator, not what it selects.
func equalDiscriminators(left, right *parser.Discriminator, st *compareState) bool {
	if left == nil || right == nil {
		return left == right
	}
	if left.PropertyName != right.PropertyName {
		return false
	}
	// mapping and defaultMapping name schemas, so they go through each side's
	// view for the same reason $ref does. SchemaRewriter rewrites both, and
	// comparing the spellings would call two discriminators equal that select
	// different subschemas once it has.
	if st.left.name(left.DefaultMapping) != st.right.name(right.DefaultMapping) {
		return false
	}
	if len(left.Mapping) != len(right.Mapping) {
		return false
	}
	for key, leftTarget := range left.Mapping {
		rightTarget, ok := right.Mapping[key]
		if !ok || st.left.name(leftTarget) != st.right.name(rightTarget) {
			return false
		}
	}
	return true
}

// equalExternalDocs compares two External Documentation Objects.
//
// Mirrors parser's equalExternalDocs field by field rather than reaching for
// reflect.DeepEqual, whose comparison of the Extra map would split a nil map from
// an empty one and disagree with parser about the same pair.
func equalExternalDocs(left, right *parser.ExternalDocs) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Description == right.Description &&
		left.URL == right.URL &&
		// EqualFunc with DeepEqual, not maps.Equal: Extra holds arbitrary JSON, and
		// == on a slice value panics. Same pairing parser's equalMapStringAny uses.
		maps.EqualFunc(left.Extra, right.Extra, reflect.DeepEqual)
}

// equalStringSliceMaps compares two name-keyed string-slice maps, order-independently.
func equalStringSliceMaps(left, right map[string][]string) bool {
	if len(left) != len(right) {
		return false
	}
	for k, lv := range left {
		rv, ok := right[k]
		if !ok || !equalStringSlices(lv, rv) {
			return false
		}
	}
	return true
}

// compareCommonFields compares schema fields common to both shallow and deep comparison.
// This helper eliminates duplication between compareShallow and compareDeep.
//
// When compareDocs is true, documentation metadata (title, description,
// example, examples) is also compared.
func compareCommonFields(left, right *parser.Schema, path *comparePath, st *compareState) {
	if st.docs {
		compareDocFields(left, right, path, st.result)
	}

	// Compare type
	if !equalTypes(left.Type, right.Type) {
		path.push("type")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Type,
			RightValue:  right.Type,
			Description: "type mismatch",
		})
		path.pop()
	}

	// Compare format
	if left.Format != right.Format {
		path.push("format")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Format,
			RightValue:  right.Format,
			Description: "format mismatch",
		})
		path.pop()
	}

	// Compare required arrays (order-independent)
	if !equalStringSlices(left.Required, right.Required) {
		path.push("required")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Required,
			RightValue:  right.Required,
			Description: "required fields mismatch",
		})
		path.pop()
	}

	// Compare enum (order matters for enum)
	if !reflect.DeepEqual(left.Enum, right.Enum) {
		path.push("enum")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Enum,
			RightValue:  right.Enum,
			Description: "enum values mismatch",
		})
		path.pop()
	}

	// Compare property names (shallow - don't compare nested schemas)
	if !equalPropertyNames(left.Properties, right.Properties) {
		path.push("properties")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   getPropertyNames(left.Properties),
			RightValue:  getPropertyNames(right.Properties),
			Description: "property names mismatch",
		})
		path.pop()
	}
}

// compareShallow compares only the top-level properties of schemas
func compareShallow(left, right *parser.Schema, path *comparePath, st *compareState) {
	compareCommonFields(left, right, path, st)
}

// equalBoolForms compares two schemas when either is the bare-boolean form,
// recording a difference when they disagree. It reports whether comparison
// should continue: false means one or both sides were boolean and the verdict
// is settled, since a boolean schema has no other fields to compare.
//
// `true` and `false` are opposite schemas — one accepts every instance, the
// other none — and neither is equivalent to an object schema.
func equalBoolForms(left, right *parser.Schema, path *comparePath, result *EquivalenceResult) bool {
	leftBool, leftIsBool := left.IsBool()
	rightBool, rightIsBool := right.IsBool()
	if !leftIsBool && !rightIsBool {
		return true
	}
	if leftIsBool && rightIsBool && leftBool == rightBool {
		return false
	}
	result.Differences = append(result.Differences, SchemaDifference{
		Path:        path.String(),
		LeftValue:   boolDifferenceValue(leftBool, leftIsBool),
		RightValue:  boolDifferenceValue(rightBool, rightIsBool),
		Description: "boolean schema form mismatch",
	})
	return false
}

// boolDifferenceValue renders one side of a boolean-form mismatch for a
// SchemaDifference. Callers format LeftValue and RightValue with %v, and
// neither of the values available here survives that: a *bool prints as a
// pointer address, and a *Schema prints as a full struct dump. Both hide the
// one thing the difference exists to report.
//
// nil marks the side that is not a boolean schema at all, matching how the
// other comparators leave a value absent rather than inventing one.
func boolDifferenceValue(value, ok bool) any {
	if !ok {
		return nil
	}
	return value
}

// compareBoolOperands handles the bare-boolean form for the any-typed
// schema-or-bool fields, recording a difference when the two sides disagree.
// It reports whether the comparison is settled, which it is as soon as either
// side is boolean: a boolean schema has no other fields.
//
// Shared by the three functions that compare these fields — compareSchemaOrBool,
// compareItemsSchemas and comparePolymorphicSchemas — so the check cannot be
// added to one and forgotten in the others. Pass an empty field when the caller
// has already pushed the path segment.
func compareBoolOperands(field string, left, right any, path *comparePath, result *EquivalenceResult) bool {
	leftBool, leftIsBool := boolSchemaOperand(left)
	rightBool, rightIsBool := boolSchemaOperand(right)
	if !leftIsBool && !rightIsBool {
		return false
	}
	if leftIsBool && rightIsBool && leftBool == rightBool {
		return true
	}

	description := "boolean value mismatch"
	if field != "" {
		path.push(field)
		defer path.pop()
		description = field + " " + description
	}
	result.Differences = append(result.Differences, SchemaDifference{
		Path:        path.String(),
		LeftValue:   boolDifferenceValue(leftBool, leftIsBool),
		RightValue:  boolDifferenceValue(rightBool, rightIsBool),
		Description: description,
	})
	return true
}

// boolSchemaOperand reports the boolean a schema-or-bool operand represents, in
// either of the two representations it can arrive in: a raw bool, which is what
// the decoders leave in these any-typed fields, or a *Schema with BoolForm set,
// which is what a caller building one programmatically produces. They mean the
// same thing and must compare equal.
func boolSchemaOperand(v any) (value bool, ok bool) {
	switch t := v.(type) {
	case bool:
		return t, true
	case *parser.Schema:
		return t.IsBool()
	default:
		return false, false
	}
}

// compareDeep recursively compares all schema properties
func compareDeep(left, right *parser.Schema, path *comparePath, st *compareState) {
	// Everything below dereferences both operands, and a nested schema can
	// legitimately be nil. Guarded here rather than at the fourteen call sites so a
	// new nested walk cannot miss it, and recorded at the caller's path, which
	// already names the field. Issue #417 has the reachable cases; #416 spot-fixed
	// two of them before this replaced that approach.
	if left == nil || right == nil {
		if left != right {
			st.result.Differences = append(st.result.Differences, SchemaDifference{
				Path:        path.String(),
				LeftValue:   left != nil,
				RightValue:  right != nil,
				Description: "schema presence mismatch",
			})
		}
		return
	}

	// The bare-boolean form has no keywords, so it is compared by value and none
	// of the field-by-field comparison below applies. Checked here rather than
	// only at the top-level entry point because every nested schema position
	// routes through this function: without it, `{p: true}` and `{p: false}`
	// compared equal, since compareCommonFields finds nothing set on either side.
	if !equalBoolForms(left, right, path, st.result) {
		return
	}

	// Check for circular references
	pair := pointerPair{
		left:  reflect.ValueOf(left).Pointer(),
		right: reflect.ValueOf(right).Pointer(),
	}
	if st.visited[pair] {
		return // Already compared this pair
	}
	st.visited[pair] = true

	// Compare common fields (type, format, required, enum, propertyNames, and
	// optionally documentation metadata).
	compareCommonFields(left, right, path, st)

	// Compare pattern (deep only)
	if left.Pattern != right.Pattern {
		path.push("pattern")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Pattern,
			RightValue:  right.Pattern,
			Description: "pattern mismatch",
		})
		path.pop()
	}

	// Compare xml. It decides the element name, namespace, and node kind a value
	// serializes to, so schemas whose XML differs describe different payloads.
	// Calling them equivalent let deduplication merge them and change the wire
	// format; the structural hash omitted xml too, so each gap hid the other.
	if !equalXMLObjects(left.XML, right.XML) {
		path.push("xml")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.XML,
			RightValue:  right.XML,
			Description: "xml metadata mismatch",
		})
		path.pop()
	}

	// Compare const
	if !reflect.DeepEqual(left.Const, right.Const) {
		path.push("const")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Const,
			RightValue:  right.Const,
			Description: "const value mismatch",
		})
		path.pop()
	}

	// Compare numeric constraints
	if !equalutil.EqualPtr(left.Minimum, right.Minimum) {
		path.push("minimum")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Minimum,
			RightValue:  right.Minimum,
			Description: "minimum constraint mismatch",
		})
		path.pop()
	}
	if !equalutil.EqualPtr(left.Maximum, right.Maximum) {
		path.push("maximum")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Maximum,
			RightValue:  right.Maximum,
			Description: "maximum constraint mismatch",
		})
		path.pop()
	}

	// Compare string constraints
	if !equalutil.EqualPtr(left.MinLength, right.MinLength) {
		path.push("minLength")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.MinLength,
			RightValue:  right.MinLength,
			Description: "minLength constraint mismatch",
		})
		path.pop()
	}
	if !equalutil.EqualPtr(left.MaxLength, right.MaxLength) {
		path.push("maxLength")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.MaxLength,
			RightValue:  right.MaxLength,
			Description: "maxLength constraint mismatch",
		})
		path.pop()
	}

	// Compare array constraints
	if !equalutil.EqualPtr(left.MinItems, right.MinItems) {
		path.push("minItems")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.MinItems,
			RightValue:  right.MinItems,
			Description: "minItems constraint mismatch",
		})
		path.pop()
	}
	if !equalutil.EqualPtr(left.MaxItems, right.MaxItems) {
		path.push("maxItems")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.MaxItems,
			RightValue:  right.MaxItems,
			Description: "maxItems constraint mismatch",
		})
		path.pop()
	}
	if left.UniqueItems != right.UniqueItems {
		path.push("uniqueItems")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.UniqueItems,
			RightValue:  right.UniqueItems,
			Description: "uniqueItems constraint mismatch",
		})
		path.pop()
	}

	// Compare object constraints
	if !equalutil.EqualPtr(left.MinProperties, right.MinProperties) {
		path.push("minProperties")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.MinProperties,
			RightValue:  right.MinProperties,
			Description: "minProperties constraint mismatch",
		})
		path.pop()
	}
	if !equalutil.EqualPtr(left.MaxProperties, right.MaxProperties) {
		path.push("maxProperties")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.MaxProperties,
			RightValue:  right.MaxProperties,
			Description: "maxProperties constraint mismatch",
		})
		path.pop()
	}

	// Compare properties recursively (property names already checked by compareCommonFields)
	if equalPropertyNames(left.Properties, right.Properties) && left.Properties != nil {
		path.push("properties")
		for name, leftProp := range left.Properties {
			rightProp := right.Properties[name]
			path.push(name)
			compareDeep(leftProp, rightProp, path, st)
			path.pop()
		}
		path.pop()
	}

	// Compare the structural fields that are not written out above (issue #410).
	compareStructuralSchemaFields(left, right, path, st)

	// Compare items (array item schema)
	compareItemsSchemas(left.Items, right.Items, path, st)

	// Compare additionalProperties
	compareAdditionalPropertiesSchemas(left.AdditionalProperties, right.AdditionalProperties, path, st)

	// Compare composition (allOf, anyOf, oneOf)
	path.push("allOf")
	compareSchemaArrays(left.AllOf, right.AllOf, path, st)
	path.pop()
	path.push("anyOf")
	compareSchemaArrays(left.AnyOf, right.AnyOf, path, st)
	path.pop()
	path.push("oneOf")
	compareSchemaArrays(left.OneOf, right.OneOf, path, st)
	path.pop()

	// Compare not
	if (left.Not == nil) != (right.Not == nil) {
		path.push("not")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Not != nil,
			RightValue:  right.Not != nil,
			Description: "not schema presence mismatch",
		})
		path.pop()
	} else if left.Not != nil && right.Not != nil {
		path.push("not")
		compareDeep(left.Not, right.Not, path, st)
		path.pop()
	}

	// JSON Schema 2020-12 fields

	// Compare unevaluatedProperties (can be bool or *Schema)
	path.push("unevaluatedProperties")
	comparePolymorphicSchemas(left.UnevaluatedProperties, right.UnevaluatedProperties, path, st)
	path.pop()

	// Compare unevaluatedItems (can be bool or *Schema)
	path.push("unevaluatedItems")
	comparePolymorphicSchemas(left.UnevaluatedItems, right.UnevaluatedItems, path, st)
	path.pop()

	// Compare contentEncoding
	if left.ContentEncoding != right.ContentEncoding {
		path.push("contentEncoding")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.ContentEncoding,
			RightValue:  right.ContentEncoding,
			Description: "contentEncoding mismatch",
		})
		path.pop()
	}

	// Compare contentMediaType
	if left.ContentMediaType != right.ContentMediaType {
		path.push("contentMediaType")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.ContentMediaType,
			RightValue:  right.ContentMediaType,
			Description: "contentMediaType mismatch",
		})
		path.pop()
	}

	// Compare contentSchema
	if (left.ContentSchema == nil) != (right.ContentSchema == nil) {
		path.push("contentSchema")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.ContentSchema != nil,
			RightValue:  right.ContentSchema != nil,
			Description: "contentSchema presence mismatch",
		})
		path.pop()
	} else if left.ContentSchema != nil && right.ContentSchema != nil {
		path.push("contentSchema")
		compareDeep(left.ContentSchema, right.ContentSchema, path, st)
		path.pop()
	}

	// Compare prefixItems
	path.push("prefixItems")
	compareSchemaArrays(left.PrefixItems, right.PrefixItems, path, st)
	path.pop()

	// Compare contains
	if (left.Contains == nil) != (right.Contains == nil) {
		path.push("contains")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.Contains != nil,
			RightValue:  right.Contains != nil,
			Description: "contains schema presence mismatch",
		})
		path.pop()
	} else if left.Contains != nil && right.Contains != nil {
		path.push("contains")
		compareDeep(left.Contains, right.Contains, path, st)
		path.pop()
	}

	// Compare propertyNames
	if (left.PropertyNames == nil) != (right.PropertyNames == nil) {
		path.push("propertyNames")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left.PropertyNames != nil,
			RightValue:  right.PropertyNames != nil,
			Description: "propertyNames schema presence mismatch",
		})
		path.pop()
	} else if left.PropertyNames != nil && right.PropertyNames != nil {
		path.push("propertyNames")
		compareDeep(left.PropertyNames, right.PropertyNames, path, st)
		path.pop()
	}

	// Compare dependentSchemas
	if !equalPropertyNames(left.DependentSchemas, right.DependentSchemas) {
		path.push("dependentSchemas")
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   getPropertyNames(left.DependentSchemas),
			RightValue:  getPropertyNames(right.DependentSchemas),
			Description: "dependentSchemas keys mismatch",
		})
		path.pop()
	} else if left.DependentSchemas != nil && right.DependentSchemas != nil {
		path.push("dependentSchemas")
		for name, leftSchema := range left.DependentSchemas {
			rightSchema := right.DependentSchemas[name]
			path.push(name)
			compareDeep(leftSchema, rightSchema, path, st)
			path.pop()
		}
		path.pop()
	}
}

// Helper functions

func equalTypes(left, right any) bool {
	// Handle nil cases
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}

	// Handle string type
	leftStr, leftIsStr := left.(string)
	rightStr, rightIsStr := right.(string)
	if leftIsStr && rightIsStr {
		return leftStr == rightStr
	}

	// Handle array type (OAS 3.1+)
	leftArr, leftIsArr := left.([]string)
	rightArr, rightIsArr := right.([]string)
	if leftIsArr && rightIsArr {
		return equalStringSlices(leftArr, rightArr)
	}

	// Handle any slice that might contain strings
	leftIface, leftIsIface := left.([]any)
	rightIface, rightIsIface := right.([]any)
	if leftIsIface && rightIsIface {
		if len(leftIface) != len(rightIface) {
			return false
		}
		leftStrings := make([]string, len(leftIface))
		rightStrings := make([]string, len(rightIface))
		for i, v := range leftIface {
			if s, ok := v.(string); ok {
				leftStrings[i] = s
			} else {
				return false
			}
		}
		for i, v := range rightIface {
			if s, ok := v.(string); ok {
				rightStrings[i] = s
			} else {
				return false
			}
		}
		return equalStringSlices(leftStrings, rightStrings)
	}

	// Different types
	return false
}

// equalXMLObjects compares two XML Objects field by field.
// https://spec.openapis.org/oas/v3.2.0.html#xml-object
//
// Absent XML is not equal to present XML: inferring everything is a different
// payload from naming an element. Attribute, Wrapped, and NodeType are compared
// independently, matching how [parser.XML] models them.
func equalXMLObjects(left, right *parser.XML) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Name == right.Name &&
		left.Namespace == right.Namespace &&
		left.Prefix == right.Prefix &&
		left.Attribute == right.Attribute &&
		left.Wrapped == right.Wrapped &&
		left.NodeType == right.NodeType
}

func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	// Create sorted copies for order-independent comparison
	leftCopy := make([]string, len(left))
	rightCopy := make([]string, len(right))
	copy(leftCopy, left)
	copy(rightCopy, right)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}

func equalPropertyNames(left, right map[string]*parser.Schema) bool {
	if len(left) != len(right) {
		return false
	}
	for name := range left {
		if _, exists := right[name]; !exists {
			return false
		}
	}
	return true
}

func getPropertyNames(properties map[string]*parser.Schema) []string {
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func compareItemsSchemas(left, right any, path *comparePath, st *compareState) {
	compareSchemaOrBool("items", left, right, path, st)
}

func compareAdditionalPropertiesSchemas(left, right any, path *comparePath, st *compareState) {
	compareSchemaOrBool("additionalProperties", left, right, path, st)
}

// compareSchemaOrBool compares one schema-or-bool field, recording differences
// under the keyword it belongs to.
//
// The keyword is a parameter because additionalProperties and additionalItems are
// the same shape but not the same field: reusing the additionalProperties path for
// additionalItems reported an item difference under an object keyword.
func compareSchemaOrBool(field string, left, right any, path *comparePath, st *compareState) {
	// Both nil
	if left == nil && right == nil {
		return
	}
	// One nil
	if left == nil || right == nil {
		path.push(field)
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left != nil,
			RightValue:  right != nil,
			Description: field + " presence mismatch",
		})
		path.pop()
		return
	}

	if compareBoolOperands(field, left, right, path, st.result) {
		return
	}

	// Both schemas
	leftSchema, leftIsSchema := left.(*parser.Schema)
	rightSchema, rightIsSchema := right.(*parser.Schema)
	if leftIsSchema && rightIsSchema {
		// A typed nil passes the interface nil checks above and asserts cleanly.
		// compareDeep catches it and records at this path.
		path.push(field)
		compareDeep(leftSchema, rightSchema, path, st)
		path.pop()
		return
	}

	// Both tuples (the OAS 2.0 list form). Position is part of the meaning, so
	// they compare element by element, as the composition keywords do.
	leftTuple, leftIsTuple := left.([]*parser.Schema)
	rightTuple, rightIsTuple := right.([]*parser.Schema)
	if leftIsTuple && rightIsTuple {
		path.push(field)
		compareSchemaArrays(leftTuple, rightTuple, path, st)
		path.pop()
		return
	}

	// Type mismatch
	path.push(field)
	st.result.Differences = append(st.result.Differences, SchemaDifference{
		Path:        path.String(),
		LeftValue:   fmt.Sprintf("%T", left),
		RightValue:  fmt.Sprintf("%T", right),
		Description: field + " type mismatch",
	})
	path.pop()
}

// comparePolymorphicSchemas compares schema fields that can be bool or *Schema (e.g., unevaluatedProperties, unevaluatedItems)
func comparePolymorphicSchemas(left, right any, path *comparePath, st *compareState) {
	// Both nil
	if left == nil && right == nil {
		return
	}
	// One nil
	if left == nil || right == nil {
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   left != nil,
			RightValue:  right != nil,
			Description: "schema presence mismatch",
		})
		return
	}

	if compareBoolOperands("", left, right, path, st.result) {
		return
	}

	// Both schemas
	leftSchema, leftIsSchema := left.(*parser.Schema)
	rightSchema, rightIsSchema := right.(*parser.Schema)
	if leftIsSchema && rightIsSchema {
		compareDeep(leftSchema, rightSchema, path, st)
		return
	}

	// Type mismatch
	st.result.Differences = append(st.result.Differences, SchemaDifference{
		Path:        path.String(),
		LeftValue:   fmt.Sprintf("%T", left),
		RightValue:  fmt.Sprintf("%T", right),
		Description: "type mismatch",
	})
}

func compareSchemaArrays(left, right []*parser.Schema, path *comparePath, st *compareState) {
	if len(left) != len(right) {
		st.result.Differences = append(st.result.Differences, SchemaDifference{
			Path:        path.String(),
			LeftValue:   len(left),
			RightValue:  len(right),
			Description: "schema array length mismatch",
		})
		return
	}

	for i := range left {
		// Use strconv.Itoa instead of fmt.Sprintf for better performance
		path.push("[" + strconv.Itoa(i) + "]")
		compareDeep(left[i], right[i], path, st)
		path.pop()
	}
}

// pathJoin is kept for backward compatibility but internal code uses comparePath.
// This function is still used by tests that call it directly.
func pathJoin(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}
