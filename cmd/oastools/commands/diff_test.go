package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupDiffFlags(t *testing.T) {
	fs, flags := SetupDiffFlags()

	t.Run("default values", func(t *testing.T) {
		assert.False(t, flags.Breaking, "expected Breaking to be false by default")
		assert.False(t, flags.NoInfo, "expected NoInfo to be false by default")
		assert.Equal(t, FormatText, flags.Format)
	})

	t.Run("parse flags", func(t *testing.T) {
		args := []string{"--breaking", "--no-info", "--format", "json", "v1.yaml", "v2.yaml"}
		require.NoError(t, fs.Parse(args))

		assert.True(t, flags.Breaking, "expected Breaking to be true")
		assert.True(t, flags.NoInfo, "expected NoInfo to be true")
		assert.Equal(t, "json", flags.Format)
		assert.Equal(t, 2, fs.NArg())
	})
}

func TestHandleDiff_NotEnoughArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{}},
		{"one arg", []string{"v1.yaml"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HandleDiff(tt.args)
			assert.Error(t, err)
		})
	}
}

func TestHandleDiff_Help(t *testing.T) {
	err := HandleDiff([]string{"--help"})
	assert.NoError(t, err)
}

func TestHandleDiff_InvalidFormat(t *testing.T) {
	err := HandleDiff([]string{"--format", "invalid", "v1.yaml", "v2.yaml"})
	assert.Error(t, err)
}
