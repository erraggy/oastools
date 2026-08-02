// mutual_exclusions.go holds the constraints of the form "these two fields must
// not both be present", as tables rather than as code.
//
// Tables because the specification spells this shape differently per object and
// per version (a JSON Schema `not` clause, a `dependentSchemas` clause, or prose
// with no schema behind it) while the check is the same every time. The spelling
// is the specification's business; each entry cites the object that states it and
// records the version it starts at.
//
//   - OAS 3.2: https://spec.openapis.org/oas/v3.2.0.html
//   - OAS 3.1: https://spec.openapis.org/oas/v3.1.1.html
//   - OAS 3.0: https://spec.openapis.org/oas/v3.0.4.html

package validator

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
)

// mutualExclusion is one "A and B must not both be present" constraint.
type mutualExclusion struct {
	// object names the object in the reported message, such as "Media Type".
	object string
	// first and second are the two fields, in the order the message names them.
	// first is also the field the error is attributed to.
	first, second string
	// since is the earliest version stating the rule. A document below it is
	// left alone; a version this build does not recognize counts as in scope,
	// matching the other version gates.
	since parser.OASVersion
	// anchor is the owning object's spec fragment, resolved against the
	// document's own version. Section anchors are used rather than the numbered
	// `#fixed-fields-N` form, which points at different objects in different
	// versions.
	anchor string
}

// Field names shared by the exclusion tables and the presence lists their call
// sites build.
//
// Named rather than spelled twice because a mismatch between the two is silent:
// [fieldIsPresent] treats a name it was not given as absent, so a rule whose
// field is misspelled in either place simply never reports.
const (
	fieldExample         = "example"
	fieldExamples        = "examples"
	fieldValue           = "value"
	fieldExternalValue   = "externalValue"
	fieldDataValue       = "dataValue"
	fieldSerializedValue = "serializedValue"
	fieldEncoding        = "encoding"
	fieldItemEncoding    = "itemEncoding"
	fieldPrefixEncoding  = "prefixEncoding"
)

// fieldPresence records whether one field was written in the source document.
//
// Presence, not value: every rule here is about whether the key appears. The
// caller supplies it because only the caller knows which zero value means
// absent for a given field's Go type, and that varies: a nil map for
// `encoding`, an empty string for `externalValue`, a nil interface for `value`.
type fieldPresence struct {
	name    string
	present bool
}

// exampleExclusions are the [Example Object]'s mutual exclusions.
//
// The versions state different subsets, which is why `since` is per entry rather
// than per table: only value/externalValue reaches back to 3.0, the others naming
// fields that 3.2 introduced.
//
// No entry pairs dataValue with serializedValue. That combination is permitted,
// so its absence here is deliberate rather than an omission.
//
// [Example Object]: https://spec.openapis.org/oas/v3.2.0.html#example-object
var exampleExclusions = []mutualExclusion{
	{object: "Example", first: fieldDataValue, second: fieldValue, since: parser.OASVersion320, anchor: "#example-object"},
	{object: "Example", first: fieldSerializedValue, second: fieldValue, since: parser.OASVersion320, anchor: "#example-object"},
	{object: "Example", first: fieldSerializedValue, second: fieldExternalValue, since: parser.OASVersion320, anchor: "#example-object"},
	{object: "Example", first: fieldValue, second: fieldExternalValue, since: parser.OASVersion300, anchor: "#example-object"},
}

// mediaTypeExclusions are the [Media Type Object]'s mutual exclusions.
//
// No entry pairs prefixEncoding with itemEncoding. Only `encoding` triggers the
// constraint, so the two sequential forms are permitted together.
//
// [Media Type Object]: https://spec.openapis.org/oas/v3.2.0.html#media-type-object
var mediaTypeExclusions = []mutualExclusion{
	{object: "Media Type", first: fieldExample, second: fieldExamples, since: parser.OASVersion300, anchor: "#media-type-object"},
	{object: "Media Type", first: fieldEncoding, second: fieldPrefixEncoding, since: parser.OASVersion320, anchor: "#media-type-object"},
	{object: "Media Type", first: fieldEncoding, second: fieldItemEncoding, since: parser.OASVersion320, anchor: "#media-type-object"},
}

// encodingExclusions restate [mediaTypeExclusions]' encoding pair on the
// [Encoding Object], which 3.2 lets nest inside itself. No example/examples
// entry: the Encoding Object defines neither field.
//
// [Encoding Object]: https://spec.openapis.org/oas/v3.2.0.html#encoding-object
var encodingExclusions = []mutualExclusion{
	{object: "Encoding", first: fieldEncoding, second: fieldPrefixEncoding, since: parser.OASVersion320, anchor: "#encoding-object"},
	{object: "Encoding", first: fieldEncoding, second: fieldItemEncoding, since: parser.OASVersion320, anchor: "#encoding-object"},
}

// parameterExclusions and headerExclusions carry the example/examples rule for
// the [Parameter Object] and [Header Object], the two objects stating it besides
// Media Type. One table each, because [Validator.reportExclusions] takes one
// table per call site.
//
// [Parameter Object]: https://spec.openapis.org/oas/v3.2.0.html#parameter-object
// [Header Object]: https://spec.openapis.org/oas/v3.2.0.html#header-object
var (
	parameterExclusions = []mutualExclusion{
		{object: "Parameter", first: fieldExample, second: fieldExamples, since: parser.OASVersion300, anchor: "#parameter-object"},
	}
	headerExclusions = []mutualExclusion{
		{object: "Header", first: fieldExample, second: fieldExamples, since: parser.OASVersion300, anchor: "#header-object"},
	}
)

// reportExclusions reports every rule in table whose two fields are both
// present. fields supplies presence for the names the table references; a name
// the caller did not supply counts as absent, so a rule referencing it never
// fires.
func (v *Validator) reportExclusions(
	table []mutualExclusion,
	fields []fieldPresence,
	path string,
	result *ValidationResult,
) {
	for _, rule := range table {
		if v.oasVersion.IsValid() && v.oasVersion < rule.since {
			continue
		}
		if !fieldIsPresent(fields, rule.first) || !fieldIsPresent(fields, rule.second) {
			continue
		}
		v.addError(result, path,
			fmt.Sprintf("%s must not have both %s and %s; %s requires %s to be absent",
				rule.object, rule.first, rule.second, rule.first, rule.second),
			withSpecRef(v.specRef(rule.anchor)),
			withField(rule.first),
		)
	}
}

// fieldIsPresent reports whether fields records name as written. A linear scan
// over at most five entries, which beats a map at this size.
func fieldIsPresent(fields []fieldPresence, name string) bool {
	for _, f := range fields {
		if f.name == name {
			return f.present
		}
	}
	return false
}
