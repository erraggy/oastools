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

	t.Run("nil typed slice keeps its nilness", func(t *testing.T) {
		var source []string
		copied, ok := deepCopyValue(source).([]string)
		require.True(t, ok)
		assert.Nil(t, copied)
	})
}
