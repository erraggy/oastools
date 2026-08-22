// This file implements schema conversion between OAS 2.0 and OAS 3.x formats.

package converter

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
)

// The schema-or-bool field names, spelled once. parser models each of these as
// *Schema, []*Schema or bool, and the conversions name them in diagnostics, so
// the string appears in several places and has to agree in all of them.
const (
	fieldItems                 = "items"
	fieldAdditionalItems       = "additionalItems"
	fieldAdditionalProperties  = "additionalProperties"
	fieldUnevaluatedItems      = "unevaluatedItems"
	fieldUnevaluatedProperties = "unevaluatedProperties"
)

// convertOAS2SchemaToOAS3 converts an OAS 2.0 schema to OAS 3.x format
func (c *Converter) convertOAS2SchemaToOAS3(schema *parser.Schema, targetVersion parser.OASVersion, result *ConversionResult, path string) *parser.Schema {
	if schema == nil {
		return nil
	}

	// Deep copy to avoid mutations
	converted := schema.DeepCopy()

	// Rewrite all $ref paths from OAS 2.0 to OAS 3.x format
	rewriteSchemaRefsOAS2ToOAS3(converted)

	// Promote the OAS 2.0 bare-string discriminator to the OAS 3.x object form
	discriminatorToObjectForm(converted)

	dropArrayValuedSchemaOrBool(c, converted, result, path)

	// For OAS 3.1+, convert boolean exclusiveMaximum/exclusiveMinimum to numeric form
	if c.isOAS31OrLater(targetVersion) {
		fixSchemaExclusiveMinMaxForOAS31(c, converted, result, path, make(map[*parser.Schema]bool))
		tupleToPrefixItems(c, converted, result, path)
	} else {
		tupleForOAS30(c, converted, result, path, draft4Tuple)
	}

	return converted
}

// dropArrayValuedSchemaOrBool clears the schema-or-bool fields that no dialect
// lets hold an array, reporting each one.
//
// Two of the five are handled elsewhere and are not touched here. `items` takes
// an array in OAS 2.0, where it spells a tuple, and 3.1 moves those positions to
// prefixItems: see tupleToPrefixItems and tupleForOAS30. `additionalItems` is
// cleared by those same functions, which know whether a tuple beside it makes it
// live. The three that remain take a schema or a boolean in draft 4 and in
// 2020-12 alike, so an array says nothing in the source and would say nothing in
// the output. The parser
// still decodes one, because these fields are `any` and it is permissive, which
// is how such values reach a conversion at all.
func dropArrayValuedSchemaOrBool(c *Converter, schema *parser.Schema, result *ConversionResult, path string) {
	walkSchemas(schema, func(s *parser.Schema) {
		drop := func(field string, value any) (any, bool) {
			arr, ok := value.([]*parser.Schema)
			if !ok {
				return value, false
			}
			c.addIssueWithContext(result, path,
				fmt.Sprintf("Schema holds a %d element array in '%s', which no OAS version accepts there; dropped", len(arr), field),
				fmt.Sprintf("'%s' takes a schema or a boolean in every JSON Schema dialect the OAS versions use. Describe the constraint with a single schema", field))
			return nil, true
		}

		if v, dropped := drop(fieldAdditionalProperties, s.AdditionalProperties); dropped {
			s.AdditionalProperties = v
		}
		if v, dropped := drop(fieldUnevaluatedItems, s.UnevaluatedItems); dropped {
			s.UnevaluatedItems = v
		}
		if v, dropped := drop(fieldUnevaluatedProperties, s.UnevaluatedProperties); dropped {
			s.UnevaluatedProperties = v
		}
	})
}

// tupleToPrefixItems rewrites the OAS 2.0 tuple spelling of `items` into the one
// OAS 3.1 and later use. Both versions can express a tuple, so nothing is lost
// and nothing is reported: 2.0 takes `items` from JSON Schema draft 4, where an
// array of schemas constrains each position, and 3.1 inherits 2020-12, where
// `prefixItems` holds those positions and `items` constrains whatever follows
// them.
//
// draft 4 gives that trailing role to `additionalItems`, so it moves to `items`.
// Absent stays absent, which in both dialects means anything may follow.
func tupleToPrefixItems(c *Converter, schema *parser.Schema, result *ConversionResult, path string) {
	walkSchemas(schema, func(s *parser.Schema) {
		tuple, ok := s.Items.([]*parser.Schema)
		if !ok {
			// draft 4 ignores additionalItems unless items is an array, so
			// dropping it here says exactly what the source said, and no OAS 3.x
			// version has the keyword to carry it anyway.
			//
			// Silent even when it holds an array, which is reported below in the
			// tuple case. The difference is not the value but whether the field
			// was doing anything: beside a single-schema items it constrains
			// nothing, so there is no loss to announce.
			s.AdditionalItems = nil
			return
		}
		s.PrefixItems = tuple

		// additionalItems takes a schema or a bool in draft 4, never an array,
		// so a source holding one is malformed. It cannot move to `items`, which
		// 2020-12 also requires to be a schema, and carrying it across would put
		// an array back in the field this conversion exists to clear.
		if rest, isTuple := s.AdditionalItems.([]*parser.Schema); isTuple {
			c.addIssueWithContext(result, path,
				fmt.Sprintf("Schema holds a %d element array in 'additionalItems', which no OAS version accepts there; dropped", len(rest)),
				"JSON Schema draft 4 takes a schema or a boolean in 'additionalItems', and 2020-12 requires 'items' to be a schema, so there is nothing to convert this into. Describe what follows the tuple with a single schema")
			s.Items = nil
		} else {
			s.Items = s.AdditionalItems
		}
		s.AdditionalItems = nil
	})
}

// tupleForOAS30 removes tuple-form `items`, which OAS 3.0 forbids: "Value MUST
// be an object and not an array". 3.0 has no positional array validation at all,
// so unless the tuple collapses (see uniformTupleElement) the positions cannot
// come across and the loss is reported.
//
// What is left behind is weaker than the source rather than different from it:
// an array whose elements are unconstrained accepts everything the tuple
// accepted. minItems and maxItems are untouched, so the length constraints
// survive even when the positions do not.
// tupleSource names the keywords the source document used for a tuple and for
// the constraint on the elements past it, so a diagnostic quotes what the author
// wrote rather than the draft 4 spelling prefixItemsToTuple may have produced.
type tupleSource struct {
	tuple           string
	trailing        string
	tupleAdvice     string
	trailingAdvice  string
	malformedAdvice string
}

var (
	draft4Tuple = tupleSource{
		tuple:           fieldItems,
		trailing:        fieldAdditionalItems,
		tupleAdvice:     "OAS 3.0 requires 'items' to be a single schema, so it cannot say what belongs at each position. Convert to OAS 3.1 or later to keep the tuple as 'prefixItems', or describe the array with one schema that admits every position",
		trailingAdvice:  "draft 4 uses 'additionalItems' to constrain the elements past a tuple. OAS 3.0 has no tuple to qualify and no such keyword, so the constraint cannot come across. Convert to OAS 3.1 or later, where it becomes 'items' beside 'prefixItems'",
		malformedAdvice: "JSON Schema draft 4 takes a schema or a boolean in 'additionalItems', and OAS 3.0 has no such keyword at all. Describe what follows the tuple with a single schema, at OAS 3.1 or later where it becomes 'items' beside 'prefixItems'",
	}
	prefixItemsTuple = tupleSource{
		tuple:           "prefixItems",
		trailing:        fieldItems,
		tupleAdvice:     "OAS 3.0 requires 'items' to be a single schema and has no 'prefixItems', so it cannot say what belongs at each position. Describe the array with one schema that admits every position, or keep the document at OAS 3.1 or later",
		trailingAdvice:  "JSON Schema 2020-12 constrains the elements past 'prefixItems' with 'items'. OAS 3.0 has no tuple for it to qualify, so the constraint cannot come across. Describe the array with one schema that admits every position",
		malformedAdvice: "JSON Schema 2020-12 requires 'items' to be a schema. Describe what follows the listed positions with a single schema",
	}
)

func tupleForOAS30(c *Converter, schema *parser.Schema, result *ConversionResult, path string, src tupleSource) {
	walkSchemas(schema, func(s *parser.Schema) {
		tuple, ok := s.Items.([]*parser.Schema)
		if !ok {
			// Ignored by draft 4 without an array items, and not a field OAS 3.0
			// has: see tupleToPrefixItems.
			s.AdditionalItems = nil
			return
		}

		if uniform, ok := uniformTupleElement(s, tuple); ok {
			// Nothing is reported here, including an array in additionalItems,
			// which the branch below does report. The rule is the same in both:
			// report a malformed value only where it was doing something. A
			// collapse by maxItems means no element past the tuple can exist, so
			// additionalItems constrains nothing whatever it holds, and the other
			// two collapses only happen when it is absent, false, or the same
			// schema as the positions. Inert in every case.
			//
			// `additionalItems: false` capped the array at the tuple's length, and
			// collapsing keeps only the element schema. Without the cap the output
			// accepts any number of them, so it moves to maxItems, which OAS 3.0
			// does have. The other two collapse justifications need nothing: a
			// maxItems bound is already present, and an additionalItems equal to
			// the positions was never a bound at all.
			if b, isBool := s.AdditionalItems.(bool); isBool && !b && s.MaxItems == nil {
				bound := len(tuple)
				s.MaxItems = &bound
			}
			s.Items = uniform
			s.AdditionalItems = nil
			return
		}

		// additionalItems is live here, because a tuple sits beside it, and OAS
		// 3.0 has neither the tuple nor the keyword. Dropping it is a second
		// loss and is reported as one: the tuple message above speaks only for
		// the positions.
		if rest, isArray := s.AdditionalItems.([]*parser.Schema); isArray {
			// Malformed as well as unconvertible, and said so, matching how
			// tupleToPrefixItems reports the same value.
			c.addIssueWithContext(result, path,
				fmt.Sprintf("Schema holds a %d element array in '%s', which no OAS version accepts there; dropped", len(rest), src.trailing),
				src.malformedAdvice)
		} else if s.AdditionalItems != nil {
			c.addIssueWithContext(result, path,
				fmt.Sprintf("Schema uses '%s' to constrain the elements past a tuple, which OAS 3.0 cannot express; dropped along with the tuple it qualified", src.trailing),
				src.trailingAdvice)
		}

		// The empty schema, not a missing one: OAS 3.0 says "items MUST be
		// present if type is 'array'", so removing the keyword would trade a
		// forbidden tuple for a missing required field. An empty schema accepts
		// everything, which is what is left once the positions are gone.
		s.Items = &parser.Schema{}
		s.AdditionalItems = nil
		c.addIssueWithContext(result, path,
			fmt.Sprintf("Schema uses a %d element tuple in '%s', which OAS 3.0 forbids; positional element schemas dropped", len(tuple), src.tuple),
			src.tupleAdvice)
	})
}

// uniformTupleElement reports the single schema a tuple is equivalent to, when
// it has one. A tuple whose positions all carry the same schema says no more
// than that schema applied to every element, PROVIDED nothing longer than the
// tuple can slip past with a different shape.
//
// Three ways that is guaranteed: maxItems stops the array at or before the last
// tuple position, additionalItems forbids anything after it, or additionalItems
// repeats the same schema.
//
// A bare-boolean element is refused even when uniform, because OAS 3.0 has no
// boolean schema form and collapsing to one would trade an invalid tuple for an
// invalid `items`.
func uniformTupleElement(s *parser.Schema, tuple []*parser.Schema) (*parser.Schema, bool) {
	if len(tuple) == 0 {
		return nil, false
	}

	first := tuple[0]
	if first == nil {
		return nil, false
	}
	if _, isBool := first.IsBool(); isBool {
		return nil, false
	}
	for _, elem := range tuple[1:] {
		if !first.Equals(elem) {
			return nil, false
		}
	}

	switch {
	case s.MaxItems != nil && *s.MaxItems <= len(tuple):
		return first, true
	case s.AdditionalItems == nil:
		// draft 4 lets anything follow the tuple, so a single schema would say
		// more than the source did.
		return nil, false
	default:
		if b, ok := s.AdditionalItems.(bool); ok && !b {
			return first, true
		}
		if rest, ok := s.AdditionalItems.(*parser.Schema); ok && first.Equals(rest) {
			return first, true
		}
		return nil, false
	}
}

// discriminatorToObjectForm clears StringForm on every discriminator in the
// schema tree so they serialize as OAS 3.x objects. The property name carries
// over unchanged; only the spelling differs between dialects.
func discriminatorToObjectForm(schema *parser.Schema) {
	walkSchemas(schema, func(s *parser.Schema) {
		if s.Discriminator != nil {
			s.Discriminator.StringForm = false
		}
	})
}

// discriminatorToStringForm sets StringForm on every discriminator in the
// schema tree so they serialize as OAS 2.0 bare strings. OAS 2.0 spells the
// discriminator as a bare string with no object to hang anything off, so both
// the 3.x mapping and any specification extensions are dropped and reported.
func discriminatorToStringForm(c *Converter, schema *parser.Schema, result *ConversionResult, path string) {
	walkSchemas(schema, func(s *parser.Schema) {
		d := s.Discriminator
		if d == nil {
			return
		}
		if len(d.Mapping) > 0 {
			c.addIssueWithContext(result, path,
				"Schema discriminator uses 'mapping' which has no OAS 2.0 equivalent; mapping dropped",
				"OAS 2.0 resolves the discriminator by schema name only; rename the target definitions to match the discriminator values")
			d.Mapping = nil
		}
		// defaultMapping (OAS 3.2+) has no OAS 2.0 equivalent either, and unlike
		// mapping it names a single fallback schema, so losing it changes which
		// schema validates a payload with no discriminating value. Reported and
		// cleared here rather than left set: the string form drops it on
		// serialization regardless, and dropping it silently is the defect issue
		// #397 is about.
		if d.DefaultMapping != "" {
			c.addIssueWithContext(result, path,
				fmt.Sprintf("Schema discriminator uses 'defaultMapping' (%s) which has no OAS 2.0 equivalent; defaultMapping dropped", d.DefaultMapping),
				"OAS 2.0 has no fallback for a missing or unmapped discriminator value; describe the fallback schema explicitly in the enclosing definition")
			d.DefaultMapping = ""
		}
		if len(d.Extra) > 0 {
			c.addIssueWithContext(result, path,
				fmt.Sprintf("Schema discriminator carries extensions (%s) which have no OAS 2.0 equivalent; extensions dropped",
					strings.Join(slices.Sorted(maps.Keys(d.Extra)), ", ")),
				"OAS 2.0 spells the discriminator as a bare string, so it has no object to hold extensions; move them onto the enclosing Schema Object, which does accept them")
			d.Extra = nil
		}
		d.StringForm = true
	})
}

// fixSchemaExclusiveMinMaxForOAS31 recursively converts boolean exclusiveMaximum/exclusiveMinimum
// to OAS 3.1+ numeric semantics (number replaces boolean+maximum pair).
// Schemas with a $ref are skipped -- they are resolved separately.
// When result is non-nil, warnings are emitted for malformed constraints (true with no bound).
func fixSchemaExclusiveMinMaxForOAS31(c *Converter, schema *parser.Schema, result *ConversionResult, path string, visited map[*parser.Schema]bool) {
	if schema == nil || visited[schema] || schema.Ref != "" {
		return
	}
	visited[schema] = true

	if v, ok := schema.ExclusiveMaximum.(bool); ok {
		if v && schema.Maximum != nil {
			schema.ExclusiveMaximum = *schema.Maximum
			schema.Maximum = nil
		} else if v {
			if result != nil {
				c.addIssueWithContext(result, path,
					"Schema has 'exclusiveMaximum: true' but no 'maximum' value; constraint dropped in OAS 3.1 conversion",
					"Add a 'maximum' value to preserve this exclusive boundary in OAS 3.1")
			}
			schema.ExclusiveMaximum = nil
		} else {
			// false -- remove the no-op keyword
			schema.ExclusiveMaximum = nil
		}
	}
	if v, ok := schema.ExclusiveMinimum.(bool); ok {
		if v && schema.Minimum != nil {
			schema.ExclusiveMinimum = *schema.Minimum
			schema.Minimum = nil
		} else if v {
			if result != nil {
				c.addIssueWithContext(result, path,
					"Schema has 'exclusiveMinimum: true' but no 'minimum' value; constraint dropped in OAS 3.1 conversion",
					"Add a 'minimum' value to preserve this exclusive boundary in OAS 3.1")
			}
			schema.ExclusiveMinimum = nil
		} else {
			schema.ExclusiveMinimum = nil
		}
	}

	for name, s := range schema.Properties {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.properties.%s", path, name), visited)
	}
	for pattern, s := range schema.PatternProperties {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.patternProperties.%s", path, pattern), visited)
	}
	for i, s := range schemautil.SchemaOrBoolSchemas(schema.AdditionalProperties) {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, path+".additionalProperties"+schemautil.IndexSuffix(i), visited)
	}
	for i, s := range schemautil.SchemaOrBoolSchemas(schema.Items) {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, path+".items"+schemautil.IndexSuffix(i), visited)
	}
	for i, s := range schema.AllOf {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.allOf[%d]", path, i), visited)
	}
	for i, s := range schema.AnyOf {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.anyOf[%d]", path, i), visited)
	}
	for i, s := range schema.OneOf {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.oneOf[%d]", path, i), visited)
	}
	fixSchemaExclusiveMinMaxForOAS31(c, schema.Not, result, path+".not", visited)
	for i, s := range schemautil.SchemaOrBoolSchemas(schema.AdditionalItems) {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, path+".additionalItems"+schemautil.IndexSuffix(i), visited)
	}
	for i, s := range schema.PrefixItems {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.prefixItems[%d]", path, i), visited)
	}
	fixSchemaExclusiveMinMaxForOAS31(c, schema.Contains, result, path+".contains", visited)
	fixSchemaExclusiveMinMaxForOAS31(c, schema.PropertyNames, result, path+".propertyNames", visited)
	for name, s := range schema.DependentSchemas {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.dependentSchemas.%s", path, name), visited)
	}
	for i, s := range schemautil.SchemaOrBoolSchemas(schema.UnevaluatedProperties) {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, path+".unevaluatedProperties"+schemautil.IndexSuffix(i), visited)
	}
	for i, s := range schemautil.SchemaOrBoolSchemas(schema.UnevaluatedItems) {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, path+".unevaluatedItems"+schemautil.IndexSuffix(i), visited)
	}
	fixSchemaExclusiveMinMaxForOAS31(c, schema.ContentSchema, result, path+".contentSchema", visited)
	fixSchemaExclusiveMinMaxForOAS31(c, schema.If, result, path+".if", visited)
	fixSchemaExclusiveMinMaxForOAS31(c, schema.Then, result, path+".then", visited)
	fixSchemaExclusiveMinMaxForOAS31(c, schema.Else, result, path+".else", visited)
	for name, s := range schema.Defs {
		fixSchemaExclusiveMinMaxForOAS31(c, s, result, fmt.Sprintf("%s.$defs.%s", path, name), visited)
	}
}

// convertOAS3SchemaToOAS2 converts an OAS 3.x schema to OAS 2.0 format
func (c *Converter) convertOAS3SchemaToOAS2(schema *parser.Schema, result *ConversionResult, path string) *parser.Schema {
	if schema == nil {
		return nil
	}

	// Deep copy to avoid mutations on the returned schema
	converted := schema.DeepCopy()

	// Recursively detect OAS 3.x features in the original schema (read-only traversal)
	walkSchemaFeatures(c, schema, result, path, make(map[*parser.Schema]bool))

	// Rewrite all $ref paths from OAS 3.x to OAS 2.0 format on the deep copy
	rewriteSchemaRefsOAS3ToOAS2(converted)

	// Demote the OAS 3.x discriminator object to the OAS 2.0 bare-string form
	discriminatorToStringForm(c, converted, result, path)

	dropArrayValuedSchemaOrBool(c, converted, result, path)

	// Demote prefixItems to the OAS 2.0 tuple spelling of items
	prefixItemsToTuple(c, converted, result, path, oas20Spelling)

	return converted
}

// prefixItemsToTuple rewrites the 2020-12 tuple spelling into the draft 4 one
// OAS 2.0 uses: `prefixItems` becomes the array form of `items`, and `items`,
// which in 2020-12 constrains whatever follows the listed positions, becomes
// `additionalItems`, which is draft 4's name for that role. A bool is left
// alone there, since draft 4 accepts one in `additionalItems`.
//
// A bool in a POSITION is another matter, because draft 4 has no boolean schema
// form: every member of the items array must be an object. `true` accepts
// anything, which the empty schema also does, so it converts. `false` accepts
// nothing, and draft 4 cannot say that without `not`, which OAS 2.0 does not
// have, so the position becomes the empty schema and the loss is reported.
// tupleTarget names the version a tuple is being rewritten for, and why that
// version cannot hold a boolean schema at a position. OAS 2.0 has neither a
// boolean schema form nor 'not'; OAS 3.0 has 'not' and lacks only the form.
type tupleTarget struct {
	name   string
	noBool string
}

var (
	oas20Spelling = tupleTarget{
		name:   "OAS 2.0",
		noBool: "OAS 2.0 follows JSON Schema draft 4, which has no boolean schema form and no 'not' keyword to build one from.",
	}
	oas30Spelling = tupleTarget{
		name:   "OAS 3.0",
		noBool: "The OAS 3.0 Schema Object has no boolean schema form, so a position spelled 'false' has no OAS 3.0 equivalent.",
	}
)

func prefixItemsToTuple(c *Converter, schema *parser.Schema, result *ConversionResult, path string, target tupleTarget) {
	walkSchemas(schema, func(s *parser.Schema) {
		if len(s.PrefixItems) == 0 {
			// An array left in items is not touched here. OAS 2.0 spells a tuple
			// exactly that way, so it needs no conversion and loses nothing. The
			// array is only unconvertible when prefixItems already holds the
			// positions, which is the case handled below.
			//
			// An array in additionalItems is another matter: draft 4 takes a
			// schema or a boolean there, so it is invalid in the OAS 2.0 output
			// whatever the source meant by it. Reported only when a tuple in
			// items makes the field live, which matches how the other direction
			// treats the same value.
			if rest, isArray := s.AdditionalItems.([]*parser.Schema); isArray {
				if _, live := s.Items.([]*parser.Schema); live {
					c.addIssueWithContext(result, path,
						fmt.Sprintf("Schema holds a %d element array in '%s', which no OAS version accepts there; dropped", len(rest), fieldAdditionalItems),
						"JSON Schema draft 4 takes a schema or a boolean in 'additionalItems'. Describe what follows the tuple with a single schema")
				}
				s.AdditionalItems = nil
			}
			return
		}

		for i, elem := range s.PrefixItems {
			b, isBool := elem.IsBool()
			if !isBool {
				continue
			}
			s.PrefixItems[i] = &parser.Schema{}
			if !b {
				c.addIssueWithContext(result, fmt.Sprintf("%s.prefixItems[%d]", path, i),
					fmt.Sprintf("Schema uses the boolean schema 'false' at a tuple position, which %s cannot express; position now accepts any value", target.name),
					target.noBool+" Constrain the position with an explicit schema, or keep the document at OAS 3.1 or later")
			}
		}

		// additionalItems is about to be replaced by items, which is 2020-12's
		// field for the same role. An array already sitting there is malformed
		// in every dialect, so it is reported rather than quietly overwritten.
		if discarded, isArray := s.AdditionalItems.([]*parser.Schema); isArray {
			c.addIssueWithContext(result, path,
				fmt.Sprintf("Schema holds a %d element array in '%s', which no OAS version accepts there; dropped", len(discarded), fieldAdditionalItems),
				"JSON Schema 2020-12 has no 'additionalItems' and constrains the elements past a tuple with 'items', while draft 4 takes a schema or a boolean there. An array is neither")
		}

		// 2020-12 requires items to be a schema, so an array there is malformed,
		// and draft 4 would not accept one in additionalItems either.
		if rest, isTuple := s.Items.([]*parser.Schema); isTuple {
			c.addIssueWithContext(result, path,
				fmt.Sprintf("Schema holds a %d element array in 'items' beside 'prefixItems', which no OAS version accepts there; dropped", len(rest)),
				"JSON Schema 2020-12 uses 'items' for what follows the listed positions and requires it to be a schema. Describe the trailing elements with a single schema, or list them in 'prefixItems'")

			// Only the array reported just above is cleared. Anything else in
			// additionalItems stays: the output spells a draft 4 tuple in items,
			// which makes that field the live trailing constraint rather than the
			// inert one it was beside 2020-12's prefixItems.
			if _, wasArray := s.AdditionalItems.([]*parser.Schema); wasArray {
				s.AdditionalItems = nil
			}
		} else {
			s.AdditionalItems = s.Items
		}
		s.Items = s.PrefixItems
		s.PrefixItems = nil
	})
}

// detectOAS3SchemaFeatures checks a single schema for OAS 3.x-only features
// that are incompatible with OAS 2.0 and records issues in the conversion result.
func detectOAS3SchemaFeatures(c *Converter, schema *parser.Schema, result *ConversionResult, path string) {
	// Check for nullable (OAS 3.0+)
	if schema.Nullable {
		c.addIssueWithContext(result, path, "Schema uses 'nullable' which is OAS 3.0+",
			"Consider using 'x-nullable' extension for OAS 2.0 compatibility")
	}

	// Check for writeOnly (OAS 3.0+)
	if schema.WriteOnly {
		c.addIssueWithContext(result, path, "Schema uses 'writeOnly' which is OAS 3.0+",
			"Consider using 'x-writeOnly' extension for OAS 2.0 compatibility")
	}

	// Check for deprecated on schemas (OAS 3.0+)
	if schema.Deprecated {
		c.addIssueWithContext(result, path, "Schema uses 'deprecated' which is OAS 3.0+",
			"Consider using 'x-deprecated' extension for OAS 2.0 compatibility")
	}

	// Check for if/then/else (JSON Schema 2020-12, OAS 3.1+)
	if schema.If != nil {
		c.addIssueWithContext(result, path, "Schema uses 'if' which is OAS 3.1+ (JSON Schema 2020-12)",
			"Conditional schema composition has no OAS 2.0 equivalent")
	}
	if schema.Then != nil {
		c.addIssueWithContext(result, path, "Schema uses 'then' which is OAS 3.1+ (JSON Schema 2020-12)",
			"Conditional schema composition has no OAS 2.0 equivalent")
	}
	if schema.Else != nil {
		c.addIssueWithContext(result, path, "Schema uses 'else' which is OAS 3.1+ (JSON Schema 2020-12)",
			"Conditional schema composition has no OAS 2.0 equivalent")
	}

	// prefixItems is deliberately absent from this list. OAS 2.0 takes items from
	// JSON Schema draft 4, whose array form says the same thing, so the tuple
	// converts rather than being lost: see prefixItemsToTuple.

	// Check for contains (JSON Schema 2020-12, OAS 3.1+)
	if schema.Contains != nil {
		c.addIssueWithContext(result, path, "Schema uses 'contains' which is OAS 3.1+ (JSON Schema 2020-12)",
			"Array containment validation has no OAS 2.0 equivalent")
	}

	// Check for propertyNames (JSON Schema 2020-12, OAS 3.1+)
	if schema.PropertyNames != nil {
		c.addIssueWithContext(result, path, "Schema uses 'propertyNames' which is OAS 3.1+ (JSON Schema 2020-12)",
			"Property name validation has no OAS 2.0 equivalent")
	}
}

// walkSchemaFeatures recursively walks a schema and all nested schemas to detect
// OAS 3.x-only features that are incompatible with OAS 2.0. The visited map provides
// identity-based cycle detection using schema pointer identity. Schemas with a $ref
// set are skipped since the referenced definition will be checked separately at the
// top level.
func walkSchemaFeatures(c *Converter, schema *parser.Schema, result *ConversionResult, path string, visited map[*parser.Schema]bool) {
	if schema == nil || visited[schema] {
		return
	}
	visited[schema] = true

	// Skip schemas that have a $ref set — these point to definitions that will
	// be checked at the top level, so detecting features here would produce
	// duplicate warnings.
	if schema.Ref != "" {
		return
	}

	// Detect OAS 3.x features on the current schema
	detectOAS3SchemaFeatures(c, schema, result, path)

	// Recursively walk nested schemas in properties
	for name, propSchema := range schema.Properties {
		walkSchemaFeatures(c, propSchema, result, fmt.Sprintf("%s.properties.%s", path, name), visited)
	}

	for pattern, propSchema := range schema.PatternProperties {
		walkSchemaFeatures(c, propSchema, result, fmt.Sprintf("%s.patternProperties.%s", path, pattern), visited)
	}

	// Schema-or-bool fields
	for i, addProps := range schemautil.SchemaOrBoolSchemas(schema.AdditionalProperties) {
		walkSchemaFeatures(c, addProps, result, fmt.Sprintf("%s.additionalProperties%s", path, schemautil.IndexSuffix(i)), visited)
	}

	for i, items := range schemautil.SchemaOrBoolSchemas(schema.Items) {
		walkSchemaFeatures(c, items, result, fmt.Sprintf("%s.items%s", path, schemautil.IndexSuffix(i)), visited)
	}

	// Composition keywords
	for i, subSchema := range schema.AllOf {
		walkSchemaFeatures(c, subSchema, result, fmt.Sprintf("%s.allOf[%d]", path, i), visited)
	}

	for i, subSchema := range schema.AnyOf {
		walkSchemaFeatures(c, subSchema, result, fmt.Sprintf("%s.anyOf[%d]", path, i), visited)
	}

	for i, subSchema := range schema.OneOf {
		walkSchemaFeatures(c, subSchema, result, fmt.Sprintf("%s.oneOf[%d]", path, i), visited)
	}

	walkSchemaFeatures(c, schema.Not, result, fmt.Sprintf("%s.not", path), visited)

	// Array-related keywords
	for i, addItems := range schemautil.SchemaOrBoolSchemas(schema.AdditionalItems) {
		walkSchemaFeatures(c, addItems, result, fmt.Sprintf("%s.additionalItems%s", path, schemautil.IndexSuffix(i)), visited)
	}

	for i, prefixItem := range schema.PrefixItems {
		walkSchemaFeatures(c, prefixItem, result, fmt.Sprintf("%s.prefixItems[%d]", path, i), visited)
	}

	walkSchemaFeatures(c, schema.Contains, result, fmt.Sprintf("%s.contains", path), visited)

	// Object validation keywords
	walkSchemaFeatures(c, schema.PropertyNames, result, fmt.Sprintf("%s.propertyNames", path), visited)

	for name, depSchema := range schema.DependentSchemas {
		walkSchemaFeatures(c, depSchema, result, fmt.Sprintf("%s.dependentSchemas.%s", path, name), visited)
	}

	// JSON Schema 2020-12 unevaluated keywords
	for i, unevalProps := range schemautil.SchemaOrBoolSchemas(schema.UnevaluatedProperties) {
		walkSchemaFeatures(c, unevalProps, result, fmt.Sprintf("%s.unevaluatedProperties%s", path, schemautil.IndexSuffix(i)), visited)
	}

	for i, unevalItems := range schemautil.SchemaOrBoolSchemas(schema.UnevaluatedItems) {
		walkSchemaFeatures(c, unevalItems, result, fmt.Sprintf("%s.unevaluatedItems%s", path, schemautil.IndexSuffix(i)), visited)
	}

	// JSON Schema 2020-12 content keywords
	walkSchemaFeatures(c, schema.ContentSchema, result, fmt.Sprintf("%s.contentSchema", path), visited)

	// Conditional keywords
	walkSchemaFeatures(c, schema.If, result, fmt.Sprintf("%s.if", path), visited)
	walkSchemaFeatures(c, schema.Then, result, fmt.Sprintf("%s.then", path), visited)
	walkSchemaFeatures(c, schema.Else, result, fmt.Sprintf("%s.else", path), visited)

	// Schema definitions
	for name, defSchema := range schema.Defs {
		walkSchemaFeatures(c, defSchema, result, fmt.Sprintf("%s.$defs.%s", path, name), visited)
	}
}
