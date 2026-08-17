// tuple_types.go generates Go code for the OAS 2.0 tuple form of `items`, where
// an array schema holds one schema per position rather than one for every
// element.
package generator

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// tupleItemSchemas returns the tuple form of an array schema's items, one schema
// per position. It returns nil for the single-schema and boolean forms, and for
// an empty tuple, which names no position and so generates as a plain slice.
//
// The slice is returned rather than iterated with schemautil.SchemaOrBoolSchemas
// because a generated field name carries the position, and a nil element means
// the position is unconstrained rather than absent.
func tupleItemSchemas(schema *parser.Schema) []*parser.Schema {
	if schema == nil {
		return nil
	}
	tuple, _ := schema.Items.([]*parser.Schema)
	if len(tuple) == 0 {
		return nil
	}
	return tuple
}

// isEmptyTuple reports whether items holds a tuple with no positions in it.
// The OAS 2.0 schema requires at least one, so this is a document the parser
// tolerates rather than one the version allows.
func isEmptyTuple(schema *parser.Schema) bool {
	if schema == nil {
		return false
	}
	tuple, ok := schema.Items.([]*parser.Schema)
	return ok && len(tuple) == 0
}

// addTupleImports records the imports the generated tuple methods need. Every
// tuple type marshals through encoding/json and wraps decode errors with fmt.
func addTupleImports(schema *parser.Schema, imports map[string]bool) {
	if tupleItemSchemas(schema) == nil {
		return
	}
	imports["encoding/json"] = true
	imports["fmt"] = true
}

// tupleRestType returns the element type for the field holding the positions
// past the end of a tuple, and whether that field is generated at all.
// additionalItems false forbids those positions, so no field holds them.
func tupleRestType(schema *parser.Schema, toGoType func(*parser.Schema, bool) string) (string, bool) {
	switch additional := schema.AdditionalItems.(type) {
	case bool:
		if !additional {
			return "", false
		}
	case *parser.Schema:
		return toGoType(additional, true), true
	}
	// Absent, true, or an array, none of which name a schema for those
	// positions, so they hold anything.
	return "any", true
}

// tupleFieldType returns the Go type for one tuple position. Every position is
// optional, since an array shorter than the tuple is valid, so the type is
// always one that can be nil.
func tupleFieldType(elem *parser.Schema, toGoType func(*parser.Schema, bool) string) string {
	if elem == nil {
		return "any"
	}
	// required is true so the mapper does not add a pointer of its own: whether
	// this field needs one is decided here, not by the UsePointers option.
	goType := toGoType(elem, true)
	if goType == "any" || strings.HasPrefix(goType, "*") ||
		strings.HasPrefix(goType, "[]") || strings.HasPrefix(goType, "map[") {
		return goType
	}
	return "*" + goType
}

// tupleFieldName returns the field name for the tuple position at index i.
func tupleFieldName(i int) string {
	return fmt.Sprintf("Item%d", i)
}

// writeTupleType writes the struct and the JSON methods for a tuple schema. The
// struct holds one field per position and, unless additionalItems forbids them,
// a Rest field for the positions past the end.
//
// The type marshals as a JSON array rather than an object, so it round-trips as
// the array the schema describes.
func writeTupleType(buf *bytes.Buffer, typeName string, tuple []*parser.Schema, schema *parser.Schema, toGoType func(*parser.Schema, bool) string) {
	restType, hasRest := tupleRestType(schema, toGoType)

	buf.WriteString("//\n")
	buf.WriteString("// The schema is a tuple: each position has its own schema. An array shorter\n")
	buf.WriteString("// than the tuple is valid, so every position is optional.\n")
	fmt.Fprintf(buf, "type %s struct {\n", typeName)
	for i, elem := range tuple {
		fmt.Fprintf(buf, "\t%s %s\n", tupleFieldName(i), tupleFieldType(elem, toGoType))
	}
	if hasRest {
		fmt.Fprintf(buf, "\tRest []%s // the positions past the end of the tuple\n", restType)
	}
	buf.WriteString("\n")
	buf.WriteString("\t// itemCount is how many positions the decoded array held. It keeps a\n")
	buf.WriteString("\t// position that was present and null from being written back as absent.\n")
	buf.WriteString("\t// A value built in Go leaves it zero and writes positions up to the last\n")
	buf.WriteString("\t// one that is set.\n")
	buf.WriteString("\titemCount int\n")
	buf.WriteString("}\n\n")

	writeTupleMarshalJSON(buf, typeName, tuple, hasRest)
	buf.WriteString("\n")
	writeTupleUnmarshalJSON(buf, typeName, tuple, restType, hasRest)
}

// writeTupleMarshalJSON writes the MarshalJSON method for a tuple type.
func writeTupleMarshalJSON(buf *bytes.Buffer, typeName string, tuple []*parser.Schema, hasRest bool) {
	fmt.Fprintf(buf, "// MarshalJSON encodes %s as the JSON array its schema describes.\n", typeName)
	buf.WriteString("// A shorter array is valid, so only the positions that carry something are\n")
	buf.WriteString("// written, but a gap in the middle is not, so an unset position before one\n")
	buf.WriteString("// that is set is written as null.\n")
	fmt.Fprintf(buf, "func (t %s) MarshalJSON() ([]byte, error) {\n", typeName)

	buf.WriteString("\t// n is how many positions to write: the count the decoded array held, or\n")
	buf.WriteString("\t// the last position that is set, whichever reaches further.\n")
	buf.WriteString("\tn := t.itemCount\n")

	// A single position needs no switch, which linters flag when it has one
	// case, so the two arities are written differently.
	if len(tuple) == 1 {
		fmt.Fprintf(buf, "\tif t.%s != nil && n < 1 {\n", tupleFieldName(0))
		buf.WriteString("\t\tn = 1\n")
		buf.WriteString("\t}\n")
	} else {
		buf.WriteString("\tswitch {\n")
		for i := len(tuple) - 1; i >= 0; i-- {
			fmt.Fprintf(buf, "\tcase t.%s != nil:\n", tupleFieldName(i))
			fmt.Fprintf(buf, "\t\tif n < %d {\n", i+1)
			fmt.Fprintf(buf, "\t\t\tn = %d\n", i+1)
			buf.WriteString("\t\t}\n")
		}
		buf.WriteString("\t}\n")
	}

	if hasRest {
		// Rest holds the positions after the tuple, so every tuple position has
		// to be written before them. Otherwise a set position followed by an
		// unset one and a non-empty Rest writes the Rest values into the unset
		// position.
		fmt.Fprintf(buf, "\tif len(t.Rest) > 0 && n < %d {\n", len(tuple))
		fmt.Fprintf(buf, "\t\tn = %d\n", len(tuple))
		buf.WriteString("\t}\n")
	}

	names := make([]string, 0, len(tuple))
	for i := range tuple {
		names = append(names, "t."+tupleFieldName(i))
	}
	fmt.Fprintf(buf, "\titems := make([]any, 0, %d", len(tuple))
	if hasRest {
		buf.WriteString("+len(t.Rest)")
	}
	buf.WriteString(")\n")
	fmt.Fprintf(buf, "\titems = append(items, []any{%s}[:n]...)\n", strings.Join(names, ", "))

	if hasRest {
		buf.WriteString("\tfor _, v := range t.Rest {\n")
		buf.WriteString("\t\titems = append(items, v)\n")
		buf.WriteString("\t}\n")
	}

	buf.WriteString("\treturn json.Marshal(items)\n")
	buf.WriteString("}\n")
}

// writeTupleUnmarshalJSON writes the UnmarshalJSON method for a tuple type.
func writeTupleUnmarshalJSON(buf *bytes.Buffer, typeName string, tuple []*parser.Schema, restType string, hasRest bool) {
	fmt.Fprintf(buf, "// UnmarshalJSON decodes the JSON array %s describes, assigning each position\n", typeName)
	buf.WriteString("// to its own field.\n")
	fmt.Fprintf(buf, "func (t *%s) UnmarshalJSON(data []byte) error {\n", typeName)
	buf.WriteString("\tvar raw []json.RawMessage\n")
	buf.WriteString("\tif err := json.Unmarshal(data, &raw); err != nil {\n")
	buf.WriteString("\t\treturn err\n")
	buf.WriteString("\t}\n")
	fmt.Fprintf(buf, "\t*t = %s{}\n", typeName)

	// Record how many positions were present, so one holding null is written
	// back rather than dropped as absent.
	buf.WriteString("\tt.itemCount = len(raw)\n")
	fmt.Fprintf(buf, "\tif t.itemCount > %d {\n", len(tuple))
	fmt.Fprintf(buf, "\t\tt.itemCount = %d\n", len(tuple))
	buf.WriteString("\t}\n")

	for i := range tuple {
		fmt.Fprintf(buf, "\tif len(raw) > %d {\n", i)
		fmt.Fprintf(buf, "\t\tif err := json.Unmarshal(raw[%d], &t.%s); err != nil {\n", i, tupleFieldName(i))
		fmt.Fprintf(buf, "\t\t\treturn fmt.Errorf(\"%s item %d: %%w\", err)\n", typeName, i)
		buf.WriteString("\t\t}\n")
		buf.WriteString("\t}\n")
	}

	fmt.Fprintf(buf, "\tif len(raw) > %d {\n", len(tuple))
	if hasRest {
		fmt.Fprintf(buf, "\t\tt.Rest = make([]%s, len(raw)-%d)\n", restType, len(tuple))
		fmt.Fprintf(buf, "\t\tfor i, r := range raw[%d:] {\n", len(tuple))
		buf.WriteString("\t\t\tif err := json.Unmarshal(r, &t.Rest[i]); err != nil {\n")
		fmt.Fprintf(buf, "\t\t\t\treturn fmt.Errorf(\"%s item %%d: %%w\", i+%d, err)\n", typeName, len(tuple))
		buf.WriteString("\t\t\t}\n")
		buf.WriteString("\t\t}\n")
	} else {
		fmt.Fprintf(buf, "\t\treturn fmt.Errorf(\"%s: got %%d items, additionalItems is false so at most %d are allowed\", len(raw))\n", typeName, len(tuple))
	}
	buf.WriteString("\t}\n")

	buf.WriteString("\treturn nil\n")
	buf.WriteString("}\n")
}
