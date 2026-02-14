package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupValidateFlags(t *testing.T) {
	fs, flags := SetupValidateFlags()

	t.Run("default values", func(t *testing.T) {
		assert.False(t, flags.Strict, "expected Strict to be false by default")
		assert.True(t, flags.ValidateStructure, "expected ValidateStructure to be true by default")
		assert.False(t, flags.NoWarnings, "expected NoWarnings to be false by default")
		assert.False(t, flags.Quiet, "expected Quiet to be false by default")
		assert.Equal(t, FormatText, flags.Format)
		assert.False(t, flags.IncludeDocument, "expected IncludeDocument to be false by default")
	})

	t.Run("parse flags", func(t *testing.T) {
		args := []string{"--strict", "--no-warnings", "-q", "--format", "json", "test.yaml"}
		require.NoError(t, fs.Parse(args))

		assert.True(t, flags.Strict, "expected Strict to be true")
		assert.True(t, flags.NoWarnings, "expected NoWarnings to be true")
		assert.True(t, flags.Quiet, "expected Quiet to be true")
		assert.Equal(t, "json", flags.Format)
		assert.Equal(t, "test.yaml", fs.Arg(0))
	})

	t.Run("validate-structure flag", func(t *testing.T) {
		// Create fresh flagset to test validate-structure flag
		fs2, flags2 := SetupValidateFlags()
		args := []string{"--validate-structure=false", "test.yaml"}
		require.NoError(t, fs2.Parse(args))

		assert.False(t, flags2.ValidateStructure, "expected ValidateStructure to be false when --validate-structure=false")
	})
}

func TestHandleValidate_NoArgs(t *testing.T) {
	err := HandleValidate([]string{})
	assert.Error(t, err)
}

func TestHandleValidate_Help(t *testing.T) {
	err := HandleValidate([]string{"--help"})
	assert.NoError(t, err)
}

func TestHandleValidate_InvalidFormat(t *testing.T) {
	err := HandleValidate([]string{"--format", "invalid", "test.yaml"})
	assert.Error(t, err)
}
