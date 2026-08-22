package maputil

import (
	"maps"
	"slices"
)

// SortedKeys returns the keys of a map sorted alphabetically.
//
// An empty or nil map yields an empty slice rather than a nil one, so a caller
// that serializes the result writes an empty list and not a null.
//
// Not slices.Sorted(maps.Keys(m)): that grows without a size hint, measured at
// 3 to 15 allocations against 1 here, and this has callers on the walk and
// generate paths.
func SortedKeys[V any](m map[string]V) []string {
	keys := slices.AppendSeq(make([]string, 0, len(m)), maps.Keys(m))
	slices.Sort(keys)
	return keys
}
