package converter

import (
	"maps"
	"reflect"
	"slices"

	"github.com/erraggy/oastools/parser"
)

// Deep copies for the fields a conversion carries across unchanged. The
// converted document must not share objects with the source it was built from,
// so a field needing no conversion still needs a copy.

// deepCopyTags deep copies a document-level tag list.
func deepCopyTags(tags []*parser.Tag) []*parser.Tag {
	if tags == nil {
		return nil
	}
	cp := make([]*parser.Tag, len(tags))
	for i, tag := range tags {
		cp[i] = tag.DeepCopy()
	}
	return cp
}

// deepCopyStrings copies a plain string slice, so appending to the converted
// document's slice cannot reallocate over the source's backing array.
func deepCopyStrings(v []string) []string {
	if v == nil {
		return nil
	}
	return slices.Clone(v)
}

// deepCopyScopes copies an OAuth flow's scope map. The values are strings, so
// cloning the map is the whole copy. A nil map copies to nil.
func deepCopyScopes(v map[string]string) map[string]string {
	return maps.Clone(v)
}

// deepCopyValue deep copies an arbitrary value (used for schema Default field).
// Handles strings, numbers, booleans, slices, and maps. Returns nil for nil input.
func deepCopyValue(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case string, float64, bool:
		// Primitives are immutable in Go semantics
		return v
	case []any:
		cp := make([]any, len(val))
		for i, elem := range val {
			cp[i] = deepCopyValue(elem)
		}
		return cp
	case map[string]any:
		cp := make(map[string]any, len(val))
		for k, elem := range val {
			cp[k] = deepCopyValue(elem)
		}
		return cp
	default:
		return deepCopyReflected(v)
	}
}

// deepCopyReflected copies the slice and map types the type switch above does
// not name. Default and Enum are declared as any, so a document built in Go
// rather than parsed can hold []string or map[string]string there, and
// returning those unchanged leaves the converted document sharing the source's
// backing array.
//
// Anything that is not a slice or a map is returned as-is: the remaining kinds
// are either immutable in Go semantics or a pointer whose identity the caller
// chose deliberately.
func deepCopyReflected(v any) any {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice:
		if rv.IsNil() {
			return v
		}
		cp := reflect.MakeSlice(rv.Type(), rv.Len(), rv.Len())
		for i := range rv.Len() {
			copyReflectedInto(cp.Index(i), rv.Index(i))
		}
		return cp.Interface()
	case reflect.Array:
		cp := reflect.New(rv.Type()).Elem()
		for i := range rv.Len() {
			copyReflectedInto(cp.Index(i), rv.Index(i))
		}
		return cp.Interface()
	case reflect.Map:
		if rv.IsNil() {
			return v
		}
		cp := reflect.MakeMapWithSize(rv.Type(), rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			elem := reflect.New(rv.Type().Elem()).Elem()
			copyReflectedInto(elem, iter.Value())
			cp.SetMapIndex(iter.Key(), elem)
		}
		return cp.Interface()
	default:
		return v
	}
}

// copyReflectedInto writes a copy of src into dst, recursing through
// deepCopyValue so a nested slice or map is copied too.
func copyReflectedInto(dst, src reflect.Value) {
	copied := deepCopyValue(src.Interface())
	if copied == nil {
		return
	}
	cv := reflect.ValueOf(copied)
	if cv.Type().AssignableTo(dst.Type()) {
		dst.Set(cv)
		return
	}
	dst.Set(src)
}

// deepCopyEnumValues deep copies an Enum slice ([]any values).
func deepCopyEnumValues(v []any) []any {
	if v == nil {
		return nil
	}
	cp := make([]any, len(v))
	for i, elem := range v {
		cp[i] = deepCopyValue(elem)
	}
	return cp
}
