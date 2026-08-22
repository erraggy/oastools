// headers.go converts Header Objects between the OAS 2.0 and OAS 3.x spellings.
//
// parser.Header is a union of both dialects. Schema, Style, Explode and Content
// are OAS 3.0+; Type, Format, Items and CollectionFormat are OAS 2.0. Each
// direction must therefore clear the fields it does not own, not only translate
// the ones it does.

package converter

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
)

// componentHeadersPrefix is the only $ref target an OAS 3.x Header Object can
// name, and the one OAS 2.0 has no section for.
const componentHeadersPrefix = "#/components/headers/"

// oas2TypedValue is the OAS 2.0 "typed value" field set that a Parameter, a
// Header and an Items Object all spell identically, and that OAS 3.x replaces
// with a Schema Object. Collected into one shape so the promotion is written
// once for the three of them.
type oas2TypedValue struct {
	Type             string
	Format           string
	Default          any
	Enum             []any
	Maximum          *float64
	ExclusiveMaximum bool
	Minimum          *float64
	ExclusiveMinimum bool
	MaxLength        *int
	MinLength        *int
	Pattern          string
	MaxItems         *int
	MinItems         *int
	UniqueItems      bool
	MultipleOf       *float64
	Items            *parser.Items
	CollectionFormat string
}

func oas2TypedValueOfHeader(h *parser.Header) oas2TypedValue {
	return oas2TypedValue{
		Type: h.Type, Format: h.Format, Default: h.Default, Enum: h.Enum,
		Maximum: h.Maximum, ExclusiveMaximum: h.ExclusiveMaximum,
		Minimum: h.Minimum, ExclusiveMinimum: h.ExclusiveMinimum,
		MaxLength: h.MaxLength, MinLength: h.MinLength, Pattern: h.Pattern,
		MaxItems: h.MaxItems, MinItems: h.MinItems, UniqueItems: h.UniqueItems,
		MultipleOf: h.MultipleOf, Items: h.Items, CollectionFormat: h.CollectionFormat,
	}
}

func oas2TypedValueOfParameter(p *parser.Parameter) oas2TypedValue {
	return oas2TypedValue{
		Type: p.Type, Format: p.Format, Default: p.Default, Enum: p.Enum,
		Maximum: p.Maximum, ExclusiveMaximum: p.ExclusiveMaximum,
		Minimum: p.Minimum, ExclusiveMinimum: p.ExclusiveMinimum,
		MaxLength: p.MaxLength, MinLength: p.MinLength, Pattern: p.Pattern,
		MaxItems: p.MaxItems, MinItems: p.MinItems, UniqueItems: p.UniqueItems,
		MultipleOf: p.MultipleOf, Items: p.Items, CollectionFormat: p.CollectionFormat,
	}
}

// oas2TypedValueToSchema builds the OAS 3.x Schema Object that an OAS 2.0 typed
// value describes.
//
// subject names the object in diagnostics ("Parameter", "Header", "Items"), the
// only respect in which the callers differ. The exclusive bound handling keys on
// the target: OAS 3.1 moved exclusiveMaximum and exclusiveMinimum from a boolean
// qualifying maximum/minimum to a number standing on its own, so a true with no
// bound beside it has nothing to become and is reported rather than dropped
// silently.
func (c *Converter) oas2TypedValueToSchema(v oas2TypedValue, subject string, result *ConversionResult, path string) *parser.Schema {
	s := &parser.Schema{
		Type:        v.Type,
		Format:      v.Format,
		Default:     v.Default,
		Enum:        v.Enum,
		Maximum:     v.Maximum,
		Minimum:     v.Minimum,
		MaxLength:   v.MaxLength,
		MinLength:   v.MinLength,
		Pattern:     v.Pattern,
		MaxItems:    v.MaxItems,
		MinItems:    v.MinItems,
		UniqueItems: v.UniqueItems,
		MultipleOf:  v.MultipleOf,
	}

	if v.ExclusiveMaximum {
		switch {
		case !c.isOAS31OrLater(result.TargetOASVersion):
			s.ExclusiveMaximum = true
		case v.Maximum != nil:
			s.ExclusiveMaximum = *v.Maximum
			s.Maximum = nil
		default:
			c.addIssueWithContext(result, path,
				fmt.Sprintf("%s has 'exclusiveMaximum: true' but no 'maximum' value; constraint dropped in OAS 3.1 conversion", subject),
				"Add a 'maximum' value to preserve this exclusive boundary in OAS 3.1")
		}
	}
	if v.ExclusiveMinimum {
		switch {
		case !c.isOAS31OrLater(result.TargetOASVersion):
			s.ExclusiveMinimum = true
		case v.Minimum != nil:
			s.ExclusiveMinimum = *v.Minimum
			s.Minimum = nil
		default:
			c.addIssueWithContext(result, path,
				fmt.Sprintf("%s has 'exclusiveMinimum: true' but no 'minimum' value; constraint dropped in OAS 3.1 conversion", subject),
				"Add a 'minimum' value to preserve this exclusive boundary in OAS 3.1")
		}
	}

	if v.Items != nil {
		s.Items = convertOAS2ItemsToSchema(c, v.Items, result, path+".items")
		reportCollectionFormat(c, v.Items.CollectionFormat, subject+" items", result, path)
	}
	reportCollectionFormat(c, v.CollectionFormat, subject, result, path)

	return s
}

// reportCollectionFormat notes a collectionFormat OAS 3.x cannot spell. csv is
// the OAS 2.0 default and maps onto the OAS 3.x defaults, so it costs nothing
// and is not reported.
func reportCollectionFormat(c *Converter, format, subject string, result *ConversionResult, path string) {
	if format == "" || format == "csv" {
		return
	}
	c.addIssueWithContext(result, path,
		fmt.Sprintf("%s uses collectionFormat '%s'", subject, format),
		"OAS 3.x uses 'style' and 'explode' instead; 'csv' format maps to style=form")
}

// convertOAS2HeaderToOAS3 promotes an OAS 2.0 header's type declaration into the
// Schema Object OAS 3.x uses, leaving no OAS 2.0 field on the result.
func (c *Converter) convertOAS2HeaderToOAS3(header *parser.Header, result *ConversionResult, path string) *parser.Header {
	if header == nil {
		return nil
	}

	converted := &parser.Header{
		Ref:         header.Ref,
		Description: header.Description,
		Required:    header.Required,
		Extra:       parser.DeepCopyExtensions(header.Extra),
	}

	switch {
	case header.Schema != nil:
		// Already the OAS 3.x spelling, and still subject to the per-schema passes.
		converted.Schema = c.convertOAS2SchemaToOAS3(header.Schema, result.TargetOASVersion, result, path+".schema")
	case header.Type != "":
		converted.Schema = c.oas2TypedValueToSchema(oas2TypedValueOfHeader(header), "Header", result, path)
	}

	return converted
}

// convertOAS3HeaderToOAS2 demotes an OAS 3.x header's Schema Object to the type
// declaration OAS 2.0 uses, and reports the OAS 3.x-only fields that have no
// OAS 2.0 spelling.
//
// The schema is run through the OAS 2.0 schema conversion before anything is
// read off it, so the per-schema passes report against this position.
func (c *Converter) convertOAS3HeaderToOAS2(header *parser.Header, result *ConversionResult, path string) *parser.Header {
	if header == nil {
		return nil
	}

	converted := &parser.Header{
		Ref:         header.Ref,
		Description: header.Description,
		Extra:       parser.DeepCopyExtensions(header.Extra),
	}

	if header.Schema != nil {
		schema := c.convertOAS3SchemaToOAS2(header.Schema, result, path+".schema")
		if schema != nil {
			c.oas2TypedValueFromSchema(schema, "Header", result, path).applyToHeader(converted)
			if converted.Type == "" {
				c.addIssueWithContext(result, path,
					"Header schema has no type OAS 2.0 can name; defaulted to 'string'",
					"An OAS 2.0 Header Object requires 'type'. Give the header schema an explicit primitive type")
				converted.Type = "string"
			}
		}
	}

	if len(header.Content) > 0 {
		c.addIssueWithContext(result, path,
			"Header uses 'content', which OAS 2.0 does not define; dropped",
			"An OAS 2.0 Header Object describes its value with 'type' and cannot carry a media type. Describe the header with a primitive schema instead")
	}
	if header.Style != "" || header.Explode != nil {
		c.addIssueWithContext(result, path,
			"Header uses 'style' or 'explode', which OAS 2.0 does not define; dropped",
			"OAS 2.0 spells array serialization with 'collectionFormat'")
	}
	if header.Required {
		c.addIssueWithContext(result, path,
			"Header uses 'required', which an OAS 2.0 Header Object does not define; dropped",
			"OAS 2.0 has no way to mark a response header required")
	}
	if header.Deprecated {
		c.addIssueWithContext(result, path,
			"Header uses 'deprecated', which an OAS 2.0 Header Object does not define; dropped",
			"OAS 2.0 has no way to mark a response header deprecated")
	}

	return converted
}

// oas2ExclusiveBound splits an OAS 3.1 numeric exclusive bound back into the
// draft 4 pair OAS 2.0 uses: a boolean flag beside maximum or minimum.
//
// A boolean is already the draft 4 spelling and passes through with the bound it
// qualified. A number replaces that bound, since draft 4 states the value in
// maximum/minimum and uses the flag only to make it exclusive.
func oas2ExclusiveBound(v any, bound *float64) (bool, *float64) {
	if b, isBool := v.(bool); isBool {
		return b, bound
	}
	if e, ok := numericBound(v); ok {
		return true, &e
	}
	return false, bound
}

// oas2ItemsFromSchema builds the Items Object OAS 2.0 uses for an array header
// from the schema of its elements. Only the keywords an Items Object defines
// come across; anything else the element schema carries has no OAS 2.0 spelling
// in this position.
//
// An Items Object describes every element with one schema, so a tuple cannot
// come across: the first position is kept and the rest are reported. A tuple
// reaches here as the draft 4 array form, which convertOAS3SchemaToOAS2 has
// already rewritten 2020-12's prefixItems into.
func (c *Converter) oas2ItemsFromSchema(schema *parser.Schema, subject string, result *ConversionResult, path string) *parser.Items {
	var elem *parser.Schema
	count := 0
	for _, s := range schemautil.SchemaOrBoolSchemas(schema.Items) {
		if count == 0 {
			elem = s
		}
		count++
	}
	if count > 1 {
		c.addIssueWithContext(result, path,
			fmt.Sprintf("%s describes a %d element tuple, which OAS 2.0 cannot express here; only the first position is kept", subject, count),
			"An OAS 2.0 'items' declaration applies one schema to every element. Describe the value with a single element schema, or keep the document at OAS 3.1 or later")
	}
	if elem == nil {
		return nil
	}

	items := &parser.Items{
		Type:        schemautil.GetPrimaryType(elem),
		Format:      elem.Format,
		Default:     elem.Default,
		Enum:        elem.Enum,
		Maximum:     elem.Maximum,
		Minimum:     elem.Minimum,
		MaxLength:   elem.MaxLength,
		MinLength:   elem.MinLength,
		Pattern:     elem.Pattern,
		MaxItems:    elem.MaxItems,
		MinItems:    elem.MinItems,
		UniqueItems: elem.UniqueItems,
		MultipleOf:  elem.MultipleOf,
	}
	items.ExclusiveMaximum, items.Maximum = oas2ExclusiveBound(elem.ExclusiveMaximum, items.Maximum)
	items.ExclusiveMinimum, items.Minimum = oas2ExclusiveBound(elem.ExclusiveMinimum, items.Minimum)
	if items.Type == "array" {
		items.Items = c.oas2ItemsFromSchema(elem, subject, result, path+".items")
	}
	return items
}

// convertHeadersToOAS3 converts a response's header map into the OAS 3.x
// spelling, preserving a nil entry as nil.
func (c *Converter) convertHeadersToOAS3(headers map[string]*parser.Header, result *ConversionResult, path string) map[string]*parser.Header {
	if headers == nil {
		return nil
	}
	converted := make(map[string]*parser.Header, len(headers))
	for name, header := range headers {
		converted[name] = c.convertOAS2HeaderToOAS3(header, result, fmt.Sprintf("%s.headers.%s", path, name))
	}
	return converted
}

// convertHeadersToOAS2 converts a response's header map into the OAS 2.0
// spelling. A header naming a component is inlined first, because OAS 2.0 has no
// components.headers to point at.
func (c *Converter) convertHeadersToOAS2(headers map[string]*parser.Header, result *ConversionResult, path string) map[string]*parser.Header {
	if headers == nil {
		return nil
	}
	converted := make(map[string]*parser.Header, len(headers))
	for name, header := range headers {
		headerPath := fmt.Sprintf("%s.headers.%s", path, name)
		source := header
		inlined := false
		if header != nil && header.Ref != "" && c.sourceHeaders != nil {
			if resolved := c.resolveHeaderRef(header.Ref, result, headerPath); resolved != nil {
				source = resolved
				inlined = true
			}
		}
		out := c.convertOAS3HeaderToOAS2(source, result, headerPath)
		if !inlined && out != nil && strings.HasPrefix(out.Ref, componentHeadersPrefix) {
			// OAS 2.0 has no components.headers, so this reference names a
			// place the output document does not have. Carrying it through
			// would emit a document validate rejects.
			c.addIssueWithContext(result, headerPath,
				fmt.Sprintf("Header references %s, which OAS 2.0 cannot express; reference dropped", out.Ref),
				"OAS 2.0 has no components.headers section. Define the header inline, or add the missing component so it can be inlined")
			out.Ref = ""
			if out.Type == "" {
				out.Type = "string"
			}
		}
		converted[name] = out
	}
	return converted
}

// oas2TypedValueFromSchema reads the OAS 2.0 type declaration a Schema Object
// describes. It is the inverse of oas2TypedValueToSchema, and serves the
// positions OAS 2.0 spells with a type rather than a schema: Header Objects, and
// parameters outside the body.
func (c *Converter) oas2TypedValueFromSchema(schema *parser.Schema, subject string, result *ConversionResult, path string) oas2TypedValue {
	v := oas2TypedValue{
		Type:        schemautil.GetPrimaryType(schema),
		Format:      schema.Format,
		Default:     schema.Default,
		Enum:        schema.Enum,
		Maximum:     schema.Maximum,
		Minimum:     schema.Minimum,
		MaxLength:   schema.MaxLength,
		MinLength:   schema.MinLength,
		Pattern:     schema.Pattern,
		MaxItems:    schema.MaxItems,
		MinItems:    schema.MinItems,
		UniqueItems: schema.UniqueItems,
		MultipleOf:  schema.MultipleOf,
	}
	v.ExclusiveMaximum, v.Maximum = oas2ExclusiveBound(schema.ExclusiveMaximum, v.Maximum)
	v.ExclusiveMinimum, v.Minimum = oas2ExclusiveBound(schema.ExclusiveMinimum, v.Minimum)
	if v.Type == "array" {
		v.Items = c.oas2ItemsFromSchema(schema, subject, result, path)
	}
	return v
}

func (v oas2TypedValue) applyToHeader(h *parser.Header) {
	h.Type, h.Format, h.Default, h.Enum = v.Type, v.Format, v.Default, v.Enum
	h.Maximum, h.ExclusiveMaximum = v.Maximum, v.ExclusiveMaximum
	h.Minimum, h.ExclusiveMinimum = v.Minimum, v.ExclusiveMinimum
	h.MaxLength, h.MinLength, h.Pattern = v.MaxLength, v.MinLength, v.Pattern
	h.MaxItems, h.MinItems, h.UniqueItems = v.MaxItems, v.MinItems, v.UniqueItems
	h.MultipleOf, h.Items = v.MultipleOf, v.Items
}

func (v oas2TypedValue) applyToParameter(p *parser.Parameter) {
	p.Type, p.Format, p.Default, p.Enum = v.Type, v.Format, v.Default, v.Enum
	p.Maximum, p.ExclusiveMaximum = v.Maximum, v.ExclusiveMaximum
	p.Minimum, p.ExclusiveMinimum = v.Minimum, v.ExclusiveMinimum
	p.MaxLength, p.MinLength, p.Pattern = v.MaxLength, v.MinLength, v.Pattern
	p.MaxItems, p.MinItems, p.UniqueItems = v.MaxItems, v.MinItems, v.UniqueItems
	p.MultipleOf, p.Items = v.MultipleOf, v.Items
}
