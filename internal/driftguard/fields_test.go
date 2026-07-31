package driftguard

import (
	"reflect"
	"strings"
)

// field describes one exported field of a struct that a guard can populate.
type field struct {
	// name is the Go field name, e.g. "DefaultMapping".
	name string
	// jsonKey is the field's json tag name, empty when tagged json:"-".
	jsonKey string
	// index is the field's index within the struct, for reflect.Value.Field.
	index int
}

// fieldsOf lists the exported fields of the struct type T, skipping the Extra
// extension map that every parser type carries and no guard is about.
func fieldsOf[T any]() []field {
	var zero T
	t := reflect.TypeOf(zero)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	fields := make([]field, 0, t.NumField())
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() || f.Name == "Extra" {
			continue
		}
		key, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if key == "-" {
			key = ""
		}
		fields = append(fields, field{name: f.Name, jsonKey: key, index: i})
	}
	return fields
}

// populate sets one field of target to a distinctive non-zero value and reports
// whether it managed to. target must be a non-nil pointer to a struct.
//
// This is what lets a guard ask the question a source-scanning check cannot
// answer honestly: with this field set and nothing else, does the behavior under
// test actually observe it? Whether the field appears in some switch statement is
// a proxy; whether it changes the output is the thing itself.
//
// false means the field's type has no obvious distinctive value, so the caller
// skips it rather than assuming. Reporting that separately keeps a type the
// helper does not understand from passing as though it were covered.
func populate(target any, f field) bool {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return false
	}
	fv := v.Elem().Field(f.index)
	if !fv.CanSet() {
		return false
	}

	switch fv.Kind() {
	case reflect.String:
		fv.SetString(marker)
		return true
	case reflect.Bool:
		fv.SetBool(true)
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fv.SetInt(7)
		return true
	case reflect.Float32, reflect.Float64:
		fv.SetFloat(7)
		return true
	case reflect.Pointer:
		// The pointer type, not its element: newValue allocates for a pointer and
		// returns a value for a struct, and the two are not interchangeable.
		fv.Set(newValue(fv.Type()))
		return true
	case reflect.Slice:
		fv.Set(reflect.Append(reflect.MakeSlice(fv.Type(), 0, 1), newValue(fv.Type().Elem())))
		return true
	case reflect.Map:
		key := reflect.New(fv.Type().Key()).Elem()
		if key.Kind() != reflect.String {
			return false
		}
		key.SetString(markerKey)
		m := reflect.MakeMap(fv.Type())
		m.SetMapIndex(key, newValue(fv.Type().Elem()))
		fv.Set(m)
		return true
	case reflect.Interface:
		// The schema-or-bool fields and the example and default values are declared
		// `any`, and a string inhabits every one of them. A named interface would
		// not accept one, and a panic here aborts the whole guard rather than
		// failing a case, so report it as uncovered instead.
		if !reflect.TypeOf(marker).AssignableTo(fv.Type()) {
			return false
		}
		fv.Set(reflect.ValueOf(marker))
		return true
	default:
		return false
	}
}

const (
	marker    = "drift-guard"
	markerKey = "driftKey"
)

// newValue builds a non-zero value of t, one level deep. A nested struct is left
// zero apart from its first string field, which is enough for the value to
// serialize to something a guard can see.
func newValue(t reflect.Type) reflect.Value {
	switch t.Kind() {
	case reflect.Pointer:
		p := reflect.New(t.Elem())
		fillFirstString(p.Elem())
		return p
	case reflect.String:
		return reflect.ValueOf(marker).Convert(t)
	case reflect.Bool:
		return reflect.ValueOf(true).Convert(t)
	case reflect.Interface:
		return reflect.ValueOf(marker)
	case reflect.Struct:
		v := reflect.New(t).Elem()
		fillFirstString(v)
		return v
	case reflect.Slice:
		return reflect.Append(reflect.MakeSlice(t, 0, 1), newValue(t.Elem()))
	case reflect.Map:
		m := reflect.MakeMap(t)
		key := reflect.New(t.Key()).Elem()
		if key.Kind() == reflect.String {
			key.SetString(markerKey)
			m.SetMapIndex(key, newValue(t.Elem()))
		}
		return m
	default:
		return reflect.New(t).Elem()
	}
}

// fillFirstString sets the first settable string field of a struct, skipping Ref
// so a nested value does not become a bare $ref that callers treat as an alias.
func fillFirstString(v reflect.Value) {
	if v.Kind() != reflect.Struct {
		return
	}
	for i := range v.NumField() {
		f := v.Field(i)
		if f.Kind() == reflect.String && f.CanSet() && v.Type().Field(i).Name != "Ref" {
			f.SetString(marker)
			return
		}
	}
}

// reflectExtraField returns the Extra map of a parser value, or the zero Value
// when it has none. Found by reflection rather than a type switch so adding a
// type to a guard's subject list needs no second edit here.
func reflectExtraField(value any) reflect.Value {
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return reflect.Value{}
	}
	return v.Elem().FieldByName("Extra")
}

// newExtension builds the specification extension that forces a type down its
// slow MarshalJSON path. Any non-empty Extra does; the key is an x- one so it is
// what ExtractExtensions would have produced.
func newExtension(t reflect.Type) reflect.Value {
	m := reflect.MakeMap(t)
	m.SetMapIndex(
		reflect.ValueOf("x-drift-guard").Convert(t.Key()),
		reflect.ValueOf(any("forces the slow path")),
	)
	return m
}
