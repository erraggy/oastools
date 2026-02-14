package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupConvertFlags(t *testing.T) {
	fs, flags := SetupConvertFlags()

	t.Run("default values", func(t *testing.T) {
		assert.Equal(t, "", flags.Target)
		assert.Equal(t, "", flags.Output)
		assert.False(t, flags.Strict, "expected Strict to be false by default")
		assert.False(t, flags.NoWarnings, "expected NoWarnings to be false by default")
		assert.False(t, flags.Quiet, "expected Quiet to be false by default")
	})

	t.Run("parse flags", func(t *testing.T) {
		args := []string{"-t", "3.0.3", "-o", "output.yaml", "--strict", "--no-warnings", "-q", "input.yaml"}
		require.NoError(t, fs.Parse(args))

		assert.Equal(t, "3.0.3", flags.Target)
		assert.Equal(t, "output.yaml", flags.Output)
		assert.True(t, flags.Strict, "expected Strict to be true")
		assert.True(t, flags.NoWarnings, "expected NoWarnings to be true")
		assert.True(t, flags.Quiet, "expected Quiet to be true")
		assert.Equal(t, "input.yaml", fs.Arg(0))
	})

	t.Run("long flags", func(t *testing.T) {
		fs2, flags2 := SetupConvertFlags()
		args := []string{"--target", "2.0", "--output", "out.yaml", "in.yaml"}
		require.NoError(t, fs2.Parse(args))

		assert.Equal(t, "2.0", flags2.Target)
		assert.Equal(t, "out.yaml", flags2.Output)
	})
}

func TestHandleConvert_NoArgs(t *testing.T) {
	err := HandleConvert([]string{})
	assert.Error(t, err)
}

func TestHandleConvert_Help(t *testing.T) {
	err := HandleConvert([]string{"--help"})
	assert.NoError(t, err)
}

func TestHandleConvert_NoTarget(t *testing.T) {
	err := HandleConvert([]string{"input.yaml"})
	assert.Error(t, err)
}
