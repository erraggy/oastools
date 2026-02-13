package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupParseFlags(t *testing.T) {
	fs, flags := SetupParseFlags()

	t.Run("default values", func(t *testing.T) {
		assert.False(t, flags.ResolveRefs, "expected ResolveRefs to be false by default")
		assert.False(t, flags.ResolveHTTPRefs, "expected ResolveHTTPRefs to be false by default")
		assert.False(t, flags.Insecure, "expected Insecure to be false by default")
		assert.False(t, flags.ValidateStructure, "expected ValidateStructure to be false by default")
		assert.False(t, flags.Quiet, "expected Quiet to be false by default")
	})

	t.Run("parse flags", func(t *testing.T) {
		args := []string{"--resolve-refs", "--resolve-http-refs", "--insecure", "--validate-structure", "-q", "test.yaml"}
		require.NoError(t, fs.Parse(args))

		assert.True(t, flags.ResolveRefs, "expected ResolveRefs to be true")
		assert.True(t, flags.ResolveHTTPRefs, "expected ResolveHTTPRefs to be true")
		assert.True(t, flags.Insecure, "expected Insecure to be true")
		assert.True(t, flags.ValidateStructure, "expected ValidateStructure to be true")
		assert.True(t, flags.Quiet, "expected Quiet to be true")
		assert.Equal(t, "test.yaml", fs.Arg(0))
	})
}

func TestHandleParse_NoArgs(t *testing.T) {
	err := HandleParse([]string{})
	assert.Error(t, err)
}

func TestHandleParse_Help(t *testing.T) {
	err := HandleParse([]string{"--help"})
	assert.NoError(t, err)
}
