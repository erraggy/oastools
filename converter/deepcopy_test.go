package converter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeepCopyValueCopiesTypedContainers covers the values a document built in
// Go can put in Default or Enum. Both are declared as any, so a parsed document
// yields []any and map[string]any while a programmatic one can hold a typed
// slice or map, and returning those unchanged shares the source's storage.
func TestDeepCopyValueCopiesTypedContainers(t *testing.T) {
	t.Run("typed slice", func(t *testing.T) {
		source := []string{"a", "b"}
		copied, ok := deepCopyValue(source).([]string)
		require.True(t, ok, "the concrete type must survive the copy")
		copied[0] = "mutated"
		assert.Equal(t, []string{"a", "b"}, source)
	})

	t.Run("typed map", func(t *testing.T) {
		source := map[string]string{"k": "v"}
		copied, ok := deepCopyValue(source).(map[string]string)
		require.True(t, ok)
		copied["k"] = "mutated"
		assert.Equal(t, map[string]string{"k": "v"}, source)
	})

	t.Run("nested typed slice", func(t *testing.T) {
		source := [][]string{{"a"}}
		copied, ok := deepCopyValue(source).([][]string)
		require.True(t, ok)
		copied[0][0] = "mutated"
		assert.Equal(t, [][]string{{"a"}}, source, "the inner slice must be copied too")
	})

	t.Run("any slice still works", func(t *testing.T) {
		source := []any{"a", map[string]any{"k": "v"}}
		copied, ok := deepCopyValue(source).([]any)
		require.True(t, ok)
		copied[1].(map[string]any)["k"] = "mutated"
		assert.Equal(t, map[string]any{"k": "v"}, source[1])
	})

	t.Run("nil and primitives pass through", func(t *testing.T) {
		assert.Nil(t, deepCopyValue(nil))
		assert.Equal(t, "s", deepCopyValue("s"))
		assert.Equal(t, 3.5, deepCopyValue(3.5))
		assert.Equal(t, true, deepCopyValue(true))
		assert.Equal(t, 7, deepCopyValue(7), "an int is immutable and needs no copy")
	})

	t.Run("array holding a slice", func(t *testing.T) {
		source := [1][]string{{"a"}}
		copied, ok := deepCopyValue(source).([1][]string)
		require.True(t, ok)
		copied[0][0] = "mutated"
		assert.Equal(t, [1][]string{{"a"}}, source, "the slice inside the array must be copied")
	})

	t.Run("pointer to a map", func(t *testing.T) {
		inner := map[string]string{"k": "v"}
		copied, ok := deepCopyValue(&inner).(*map[string]string)
		require.True(t, ok)
		require.NotSame(t, &inner, copied)
		(*copied)["k"] = "mutated"
		assert.Equal(t, map[string]string{"k": "v"}, inner)
	})

	t.Run("nil pointer stays nil", func(t *testing.T) {
		var p *map[string]string
		copied, ok := deepCopyValue(p).(*map[string]string)
		require.True(t, ok)
		assert.Nil(t, copied)
	})

	t.Run("struct with exported mutable fields", func(t *testing.T) {
		type payload struct {
			Tags   []string
			Labels map[string]string
		}
		source := payload{Tags: []string{"a"}, Labels: map[string]string{"k": "v"}}
		copied, ok := deepCopyValue(source).(payload)
		require.True(t, ok, "the concrete type must survive the copy")

		copied.Tags[0] = "mutated"
		copied.Labels["k"] = "mutated"

		assert.Equal(t, []string{"a"}, source.Tags)
		assert.Equal(t, map[string]string{"k": "v"}, source.Labels)
	})

	t.Run("struct with an unexported field keeps its value", func(t *testing.T) {
		source := withHidden{Tags: []string{"a"}, hidden: "kept"}
		copied, ok := deepCopyValue(source).(withHidden)
		require.True(t, ok)

		// The exported field is copied.
		copied.Tags[0] = "mutated"
		assert.Equal(t, []string{"a"}, source.Tags)

		// The unexported field survives the copy rather than being zeroed,
		// which is what a field-by-field copy would have done to it.
		assert.Equal(t, "kept", copied.hidden)
	})

	t.Run("pointer to a struct", func(t *testing.T) {
		type payload struct{ Tags []string }
		source := &payload{Tags: []string{"a"}}
		copied, ok := deepCopyValue(source).(*payload)
		require.True(t, ok)
		require.NotSame(t, source, copied)
		copied.Tags[0] = "mutated"
		assert.Equal(t, []string{"a"}, source.Tags)
	})

	t.Run("nil typed slice keeps its nilness", func(t *testing.T) {
		var source []string
		copied, ok := deepCopyValue(source).([]string)
		require.True(t, ok)
		assert.Nil(t, copied)
	})
}

// withHidden has an unexported field, which reflection cannot set. It is
// declared at package scope because a type literal inside a subtest cannot
// carry one that the test can also read.
type withHidden struct {
	Tags   []string
	hidden string
}
