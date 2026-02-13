package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupFixFlags(t *testing.T) {
	fs, flags := SetupFixFlags()

	t.Run("default values", func(t *testing.T) {
		assert.Equal(t, "", flags.Output)
		assert.False(t, flags.Infer, "expected Infer to be false by default")
		assert.False(t, flags.Quiet, "expected Quiet to be false by default")
	})

	t.Run("parse flags", func(t *testing.T) {
		args := []string{"-o", "fixed.yaml", "--infer", "-q", "input.yaml"}
		require.NoError(t, fs.Parse(args))

		assert.Equal(t, "fixed.yaml", flags.Output)
		assert.True(t, flags.Infer, "expected Infer to be true")
		assert.True(t, flags.Quiet, "expected Quiet to be true")
		assert.Equal(t, "input.yaml", fs.Arg(0))
	})

	t.Run("long flags", func(t *testing.T) {
		fs2, flags2 := SetupFixFlags()
		args := []string{"--output", "out.yaml", "--quiet", "in.yaml"}
		require.NoError(t, fs2.Parse(args))

		assert.Equal(t, "out.yaml", flags2.Output)
		assert.True(t, flags2.Quiet, "expected Quiet to be true")
	})
}

func TestHandleFix_NoArgs(t *testing.T) {
	err := HandleFix([]string{})
	assert.Error(t, err)
}

func TestHandleFix_Help(t *testing.T) {
	err := HandleFix([]string{"--help"})
	assert.NoError(t, err)
}
