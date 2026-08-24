package schemarefs

import (
	"reflect"
	"slices"
	"strconv"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A name can be interchangeable with the one a part settled on and still be
// held apart from another of its members, so a part is judged by every tree it
// has taken on rather than by the first name in it.
func TestDistinct_SplitWeighsEveryMemberOfAPart(t *testing.T) {
	// One tree references Location and Place. Address is referenced by no tree,
	// so nothing holds it apart from either of them.
	d := &Distinct{trees: map[string]map[string]struct{}{
		"Location": {"7": {}},
		"Place":    {"7": {}},
	}}

	assert.Equal(t, [][]string{{"Address", "Location"}, {"Place"}},
		d.Split([]string{"Address", "Location", "Place"}))
}

// The deduplicator hands over a group in whatever order it hashed the names in,
// so the partition has to settle the same way whatever that order was.
func TestDistinct_SplitDoesNotDependOnGroupOrder(t *testing.T) {
	d := &Distinct{trees: map[string]map[string]struct{}{
		"Location": {"7": {}},
		"Place":    {"7": {}},
	}}
	want := [][]string{{"Address", "Location"}, {"Place"}}

	for _, group := range [][]string{
		{"Address", "Location", "Place"},
		{"Place", "Location", "Address"},
		{"Location", "Place", "Address"},
		{"Place", "Address", "Location"},
	} {
		assert.Equal(t, want, d.Split(group), "group order %v", group)
	}
}

// EachRef walks the subschema keywords by hand, and both the reference graph
// and semantic deduplication depend on it: a keyword it does not read is one
// whose references go uncounted, which loses a reference from the graph and
// lets deduplication merge two names a document distinguishes under it. Reflection asks the question directly. With a
// reference sitting under one field and nothing else set, is it found?
//
// The typed schema fields are enumerated rather than listed, so a new one is
// covered the day it is added. The schema-or-bool fields are declared any and
// share that type with Default and Const, which hold no schema, so those are
// named.
func TestEachRefReadsEverySubschemaField(t *testing.T) {
	const target = "#/definitions/Target"
	child := func() *parser.Schema { return &parser.Schema{Ref: target} }

	found := func(t *testing.T, schema *parser.Schema) bool {
		t.Helper()
		seen := false
		EachRef(schema, "", func(name, _ string) {
			if name == "Target" {
				seen = true
			}
		})
		return seen
	}

	// The exact set the reflection reaches. A field whose type changes so the
	// switch below stops matching it would otherwise drop out of the guard
	// without failing anything, which is the guard quietly covering less.
	want := []string{
		"AllOf", "AnyOf", "Contains", "ContentSchema", "Defs", "DependentSchemas",
		"Else", "If", "Not", "OneOf", "PatternProperties", "PrefixItems",
		"Properties", "PropertyNames", "Then",
	}

	schemaType := reflect.TypeOf(parser.Schema{})
	var covered []string
	for i := range schemaType.NumField() {
		f := schemaType.Field(i)
		if !f.IsExported() {
			continue
		}

		var value reflect.Value
		switch f.Type {
		case reflect.TypeOf((*parser.Schema)(nil)):
			value = reflect.ValueOf(child())
		case reflect.TypeOf([]*parser.Schema{}):
			value = reflect.ValueOf([]*parser.Schema{child()})
		case reflect.TypeOf(map[string]*parser.Schema{}):
			value = reflect.ValueOf(map[string]*parser.Schema{"key": child()})
		default:
			continue
		}

		covered = append(covered, f.Name)
		t.Run(f.Name, func(t *testing.T) {
			schema := &parser.Schema{}
			reflect.ValueOf(schema).Elem().Field(i).Set(value)
			assert.True(t, found(t, schema),
				"a reference under Schema.%s is not read, so two names it "+
					"distinguishes can still be merged", f.Name)
		})
	}
	slices.Sort(covered)
	require.Equal(t, want, covered,
		"the set of schema-bearing fields changed; add the new one to EachRef and to want")

	// The schema-or-bool fields, which are any and so cannot be told from the
	// fields holding plain values by their type alone.
	for name, set := range map[string]func(*parser.Schema){
		"Items":                 func(s *parser.Schema) { s.Items = child() },
		"ItemsTuple":            func(s *parser.Schema) { s.Items = []*parser.Schema{child()} },
		"AdditionalProperties":  func(s *parser.Schema) { s.AdditionalProperties = child() },
		"AdditionalItems":       func(s *parser.Schema) { s.AdditionalItems = child() },
		"UnevaluatedProperties": func(s *parser.Schema) { s.UnevaluatedProperties = child() },
		"UnevaluatedItems":      func(s *parser.Schema) { s.UnevaluatedItems = child() },
	} {
		t.Run(name, func(t *testing.T) {
			schema := &parser.Schema{}
			set(schema)
			assert.True(t, found(t, schema),
				"a reference under Schema.%s is not read", name)
		})
	}
}

// The worked example in Split's doc comment, kept honest.
func TestDistinct_SplitMatchesItsDocumentedExample(t *testing.T) {
	d := &Distinct{trees: map[string]map[string]struct{}{
		"OriginAddress":      {"shipmentTree": {}},
		"DestinationAddress": {"shipmentTree": {}},
	}}

	assert.Equal(t,
		[][]string{{"BillingAddress", "DestinationAddress"}, {"OriginAddress"}},
		d.Split([]string{"OriginAddress", "DestinationAddress", "BillingAddress"}))
}

func TestJoinLocation(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		segment string
		want    string
	}{
		{name: "no base", base: "", segment: "properties.pet", want: "properties.pet"},
		{name: "base and segment", base: "properties.pet", segment: "items", want: "properties.pet.items"},
		{name: "empty segment", base: "allOf[0]", segment: "", want: "allOf[0]."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, joinLocation(tt.base, tt.segment))
		})
	}
}

// EachRef reports where each reference sits, which the reference graph uses to
// say what a collision is about.
func TestEachRefReportsLocations(t *testing.T) {
	schema := &parser.Schema{
		Properties: map[string]*parser.Schema{
			"pet": {Ref: "#/definitions/Pet"},
		},
		AllOf: []*parser.Schema{{Ref: "#/components/schemas/Base"}},
	}

	got := map[string]string{}
	EachRef(schema, "", func(name, at string) { got[name] = at })

	assert.Equal(t, map[string]string{
		"Pet":  "properties.pet",
		"Base": "allOf[0]",
	}, got)
}

// A schema that is itself a bare $ref has no location within itself.
func TestEachRefNamesABareRef(t *testing.T) {
	got := map[string]string{}
	EachRef(&parser.Schema{Ref: "#/definitions/Pet"}, "", func(name, at string) { got[name] = at })

	assert.Equal(t, map[string]string{"Pet": "$ref"}, got)
}

func TestSchemaName(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want string
	}{
		{name: "oas3 components", ref: "#/components/schemas/Pet", want: "Pet"},
		{name: "oas2 definitions", ref: "#/definitions/Pet", want: "Pet"},
		{name: "empty", ref: "", want: ""},
		{name: "points elsewhere", ref: "#/components/responses/NotFound", want: ""},
		{name: "external file", ref: "other.yaml#/definitions/Pet", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SchemaName(tt.ref))
		})
	}
}

// Split is the only SplitFunc the deduplicator is given, and the deduplicator
// takes its result at face value: a name it left out would be a schema
// dropped with no error, and a name in two parts would be one schema written
// under two names. Neither is possible by construction, since every name is
// placed exactly once, and this holds it that way over arbitrary groups and
// reference patterns.
func TestSplitReturnsEveryNameExactlyOnce(t *testing.T) {
	names := []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot"}

	// Every assignment of six names across three trees, so groups that are
	// wholly free, wholly conflicting, and every shape between are covered.
	for pattern := range 729 {
		d := &Distinct{trees: make(map[string]map[string]struct{})}
		remaining := pattern
		for _, name := range names {
			tree := remaining % 3
			remaining /= 3
			d.trees[name] = map[string]struct{}{strconv.Itoa(tree): {}}
		}

		counts := make(map[string]int, len(names))
		for _, part := range d.Split(names) {
			require.NotEmpty(t, part, "pattern %d produced an empty part", pattern)
			for _, name := range part {
				counts[name]++
			}
		}

		for _, name := range names {
			require.Equal(t, 1, counts[name],
				"pattern %d returned %s %d times", pattern, name, counts[name])
		}
	}
}

// Names sharing a tree never share a part, which is the property the whole
// partition exists for.
func TestSplitNeverPutsNamesSharingATreeInOnePart(t *testing.T) {
	names := []string{"Alpha", "Bravo", "Charlie", "Delta"}

	for pattern := range 256 {
		d := &Distinct{trees: make(map[string]map[string]struct{})}
		remaining := pattern
		for _, name := range names {
			tree := remaining % 4
			remaining /= 4
			d.trees[name] = map[string]struct{}{strconv.Itoa(tree): {}}
		}

		for _, part := range d.Split(names) {
			for i, left := range part {
				for _, right := range part[i+1:] {
					require.False(t, intersects(d.trees[left], d.trees[right]),
						"pattern %d put %s and %s in one part despite a shared tree",
						pattern, left, right)
				}
			}
		}
	}
}

// oas2Doc and oas3Doc are the same API in both spellings: an operation whose
// response is an inline schema referencing Origin, plus a component Shipment
// referencing Origin and Destination.
func oas2Doc() *parser.OAS2Document {
	return &parser.OAS2Document{
		Swagger: "2.0",
		Paths: parser.Paths{"/s": &parser.PathItem{Get: &parser.Operation{
			Responses: &parser.Responses{Codes: map[string]*parser.Response{
				"200": {Schema: &parser.Schema{Properties: map[string]*parser.Schema{
					"from": {Ref: "#/definitions/Origin"},
				}}},
			}},
		}}},
		Definitions: map[string]*parser.Schema{
			"Shipment": {Properties: map[string]*parser.Schema{
				"from": {Ref: "#/definitions/Origin"},
				"to":   {Ref: "#/definitions/Destination"},
			}},
			"Origin":      {Type: "object"},
			"Destination": {Type: "object"},
		},
		OASVersion: parser.OASVersion20,
	}
}

func oas3Doc() *parser.OAS3Document {
	return &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Paths: parser.Paths{"/s": &parser.PathItem{Get: &parser.Operation{
			Responses: &parser.Responses{Codes: map[string]*parser.Response{
				"200": {Content: map[string]*parser.MediaType{
					"application/json": {Schema: &parser.Schema{Properties: map[string]*parser.Schema{
						"from": {Ref: "#/components/schemas/Origin"},
					}}},
				}},
			}},
		}}},
		Components: &parser.Components{Schemas: map[string]*parser.Schema{
			"Shipment": {Properties: map[string]*parser.Schema{
				"from": {Ref: "#/components/schemas/Origin"},
				"to":   {Ref: "#/components/schemas/Destination"},
			}},
			"Origin":      {Type: "object"},
			"Destination": {Type: "object"},
		}},
		OASVersion: parser.OASVersion303,
	}
}

// Collect reads both document spellings, counts a reference nested under a
// property, and keeps the inline response tree apart from the component one.
func TestCollectReadsBothDocumentVersions(t *testing.T) {
	for name, doc := range map[string]any{"oas2": oas2Doc(), "oas3": oas3Doc()} {
		t.Run(name, func(t *testing.T) {
			d, err := Collect(doc)
			require.NoError(t, err)

			// Shipment names both, so they are held apart.
			require.Equal(t, [][]string{{"Destination"}, {"Origin"}},
				d.Split([]string{"Origin", "Destination"}))

			// Two trees reference Origin: Shipment and the inline response.
			assert.Len(t, d.trees["Origin"], 2)
			assert.Len(t, d.trees["Destination"], 1)
		})
	}
}

// Names in trees that never meet are free to consolidate, which is the case
// deduplication exists for.
func TestCollectLeavesIndependentTreesFree(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Components: &parser.Components{Schemas: map[string]*parser.Schema{
			"Left":  {Properties: map[string]*parser.Schema{"a": {Ref: "#/components/schemas/Alpha"}}},
			"Right": {Properties: map[string]*parser.Schema{"b": {Ref: "#/components/schemas/Bravo"}}},
			"Alpha": {Type: "object"},
			"Bravo": {Type: "object"},
		}},
		OASVersion: parser.OASVersion303,
	}

	d, err := Collect(doc)
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"Alpha", "Bravo"}}, d.Split([]string{"Alpha", "Bravo"}))
}

// A document Collect cannot walk is reported rather than read as one holding
// nothing apart, which would silently consolidate everything.
func TestCollectRefusesWhatItCannotWalk(t *testing.T) {
	for name, doc := range map[string]any{
		"nil document":      nil,
		"unsupported type":  struct{}{},
		"nil typed pointer": (*parser.OAS3Document)(nil),
	} {
		t.Run(name, func(t *testing.T) {
			d, err := Collect(doc)
			require.Error(t, err)
			assert.Nil(t, d)
		})
	}
}
