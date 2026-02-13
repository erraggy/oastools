package commands

import (
	"testing"

	"github.com/erraggy/oastools/joiner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupJoinFlags(t *testing.T) {
	fs, flags := SetupJoinFlags()

	t.Run("default values", func(t *testing.T) {
		assert.Equal(t, "", flags.Output)
		assert.Equal(t, "", flags.PathStrategy)
		assert.False(t, flags.NoMergeArrays, "expected NoMergeArrays to be false by default")
		assert.False(t, flags.NoDedupTags, "expected NoDedupTags to be false by default")
		assert.False(t, flags.Quiet, "expected Quiet to be false by default")
		// Operation context flags
		assert.False(t, flags.OperationContext, "expected OperationContext to be false by default")
		assert.Equal(t, "", flags.PrimaryOperationPolicy)
		// Overlay flags
		assert.Empty(t, flags.PreOverlays)
		assert.Equal(t, "", flags.PostOverlay)
	})

	t.Run("parse flags", func(t *testing.T) {
		args := []string{"-o", "output.yaml", "--path-strategy", "accept-left", "--no-merge-arrays", "-q", "file1.yaml", "file2.yaml"}
		require.NoError(t, fs.Parse(args))

		assert.Equal(t, "output.yaml", flags.Output)
		assert.Equal(t, "accept-left", flags.PathStrategy)
		assert.True(t, flags.NoMergeArrays, "expected NoMergeArrays to be true")
		assert.True(t, flags.Quiet, "expected Quiet to be true")
		assert.Equal(t, 2, fs.NArg())
	})
}

func TestSetupJoinFlags_Strategies(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected joiner.CollisionStrategy
	}{
		{"accept-left", []string{"--path-strategy", "accept-left", "f1.yaml", "f2.yaml"}, joiner.StrategyAcceptLeft},
		{"accept-right", []string{"--path-strategy", "accept-right", "f1.yaml", "f2.yaml"}, joiner.StrategyAcceptRight},
		{"fail", []string{"--path-strategy", "fail", "f1.yaml", "f2.yaml"}, joiner.StrategyFailOnCollision},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, flags := SetupJoinFlags()
			require.NoError(t, fs.Parse(tt.args))
			assert.Equal(t, tt.expected, joiner.CollisionStrategy(flags.PathStrategy))
		})
	}
}

func TestHandleJoin_NotEnoughFiles(t *testing.T) {
	err := HandleJoin([]string{"single.yaml"})
	assert.Error(t, err)
}

func TestHandleJoin_Help(t *testing.T) {
	err := HandleJoin([]string{"--help"})
	assert.NoError(t, err)
}

func TestHandleJoin_InvalidStrategy(t *testing.T) {
	err := HandleJoin([]string{"--path-strategy", "invalid", "f1.yaml", "f2.yaml"})
	assert.Error(t, err)
}

func TestSetupJoinFlags_NamespacePrefix(t *testing.T) {
	t.Run("parse single namespace prefix", func(t *testing.T) {
		fs, flags := SetupJoinFlags()
		args := []string{"--namespace-prefix", "api.yaml=Api", "f1.yaml", "f2.yaml"}
		require.NoError(t, fs.Parse(args))
		prefix, ok := flags.NamespacePrefix["api.yaml"]
		require.True(t, ok)
		assert.Equal(t, "Api", prefix)
	})

	t.Run("parse multiple namespace prefixes", func(t *testing.T) {
		fs, flags := SetupJoinFlags()
		args := []string{
			"--namespace-prefix", "users.yaml=Users",
			"--namespace-prefix", "billing.yaml=Billing",
			"f1.yaml", "f2.yaml",
		}
		require.NoError(t, fs.Parse(args))
		assert.Equal(t, "Users", flags.NamespacePrefix["users.yaml"])
		assert.Equal(t, "Billing", flags.NamespacePrefix["billing.yaml"])
	})

	t.Run("parse always-prefix flag", func(t *testing.T) {
		fs, flags := SetupJoinFlags()
		args := []string{"--always-prefix", "f1.yaml", "f2.yaml"}
		require.NoError(t, fs.Parse(args))
		assert.True(t, flags.AlwaysPrefix, "expected AlwaysPrefix to be true")
	})

	t.Run("parse namespace-prefix with always-prefix", func(t *testing.T) {
		fs, flags := SetupJoinFlags()
		args := []string{
			"--namespace-prefix", "api.yaml=Api",
			"--always-prefix",
			"f1.yaml", "f2.yaml",
		}
		require.NoError(t, fs.Parse(args))
		assert.Equal(t, "Api", flags.NamespacePrefix["api.yaml"])
		assert.True(t, flags.AlwaysPrefix, "expected AlwaysPrefix to be true")
	})

	t.Run("invalid namespace prefix format", func(t *testing.T) {
		fs, _ := SetupJoinFlags()
		args := []string{"--namespace-prefix", "invalid", "f1.yaml", "f2.yaml"}
		err := fs.Parse(args)
		assert.Error(t, err)
	})

	t.Run("empty source in namespace prefix", func(t *testing.T) {
		fs, _ := SetupJoinFlags()
		args := []string{"--namespace-prefix", "=Prefix", "f1.yaml", "f2.yaml"}
		err := fs.Parse(args)
		assert.Error(t, err)
	})

	t.Run("empty prefix in namespace prefix", func(t *testing.T) {
		fs, _ := SetupJoinFlags()
		args := []string{"--namespace-prefix", "api.yaml=", "f1.yaml", "f2.yaml"}
		err := fs.Parse(args)
		assert.Error(t, err)
	})
}

func TestNamespacePrefixFlag_String(t *testing.T) {
	npf := make(namespacePrefixFlag)
	npf["a.yaml"] = "A"

	str := npf.String()
	assert.Equal(t, "a.yaml=A", str)
}

func TestNamespacePrefixFlag_Set(t *testing.T) {
	t.Run("valid format", func(t *testing.T) {
		npf := make(namespacePrefixFlag)
		err := npf.Set("users.yaml=Users")
		require.NoError(t, err)
		assert.Equal(t, "Users", npf["users.yaml"])
	})

	t.Run("invalid format - no equals", func(t *testing.T) {
		npf := make(namespacePrefixFlag)
		err := npf.Set("invalid")
		assert.Error(t, err)
	})

	t.Run("handles spaces in value", func(t *testing.T) {
		npf := make(namespacePrefixFlag)
		err := npf.Set("  users.yaml  =  Users  ")
		require.NoError(t, err)
		assert.Equal(t, "Users", npf["users.yaml"])
	})
}

func TestJoinFlags_OperationContext(t *testing.T) {
	fs, flags := SetupJoinFlags()
	err := fs.Parse([]string{"--operation-context", "api1.yaml", "api2.yaml"})
	require.NoError(t, err)
	assert.True(t, flags.OperationContext, "expected OperationContext to be true")
}

func TestJoinFlags_PrimaryOperationPolicy(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		policy string
	}{
		{"first", []string{"--primary-operation-policy", "first", "a.yaml", "b.yaml"}, "first"},
		{"most-specific", []string{"--primary-operation-policy", "most-specific", "a.yaml", "b.yaml"}, "most-specific"},
		{"alphabetical", []string{"--primary-operation-policy", "alphabetical", "a.yaml", "b.yaml"}, "alphabetical"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, flags := SetupJoinFlags()
			err := fs.Parse(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.policy, flags.PrimaryOperationPolicy)
		})
	}
}

func TestJoinFlags_PreOverlay_Repeatable(t *testing.T) {
	fs, flags := SetupJoinFlags()
	err := fs.Parse([]string{
		"--pre-overlay", "overlay1.yaml",
		"--pre-overlay", "overlay2.yaml",
		"api1.yaml", "api2.yaml",
	})
	require.NoError(t, err)
	require.Len(t, flags.PreOverlays, 2)
	assert.Equal(t, "overlay1.yaml", flags.PreOverlays[0])
	assert.Equal(t, "overlay2.yaml", flags.PreOverlays[1])
}

func TestJoinFlags_PostOverlay(t *testing.T) {
	fs, flags := SetupJoinFlags()
	err := fs.Parse([]string{"--post-overlay", "final.yaml", "api1.yaml", "api2.yaml"})
	require.NoError(t, err)
	assert.Equal(t, "final.yaml", flags.PostOverlay)
}

func TestValidatePrimaryOperationPolicy(t *testing.T) {
	tests := []struct {
		policy  string
		wantErr bool
	}{
		{"", false},              // empty is valid (uses default)
		{"first", false},         // valid policy
		{"most-specific", false}, // valid policy
		{"alphabetical", false},  // valid policy
		{"invalid", true},        // invalid value
		{"FIRST", true},          // case sensitive
	}
	for _, tt := range tests {
		t.Run(tt.policy, func(t *testing.T) {
			err := ValidatePrimaryOperationPolicy(tt.policy)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMapPrimaryOperationPolicy(t *testing.T) {
	tests := []struct {
		policy string
		want   joiner.PrimaryOperationPolicy
	}{
		{"", joiner.PolicyFirstEncountered},      // empty defaults to first
		{"first", joiner.PolicyFirstEncountered}, // explicit first
		{"most-specific", joiner.PolicyMostSpecific},
		{"alphabetical", joiner.PolicyAlphabetical},
		{"unknown", joiner.PolicyFirstEncountered}, // unknown defaults to first
	}
	for _, tt := range tests {
		t.Run(tt.policy, func(t *testing.T) {
			got := MapPrimaryOperationPolicy(tt.policy)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStringSliceFlag(t *testing.T) {
	t.Run("empty string representation", func(t *testing.T) {
		var flag stringSliceFlag
		assert.Equal(t, "", flag.String())
	})

	t.Run("nil string representation", func(t *testing.T) {
		var flag *stringSliceFlag
		assert.Equal(t, "", flag.String())
	})

	t.Run("set and string", func(t *testing.T) {
		var flag stringSliceFlag
		require.NoError(t, flag.Set("value1"))
		require.NoError(t, flag.Set("value2"))
		assert.Len(t, flag, 2)
		assert.Equal(t, "value1,value2", flag.String())
	})
}

func TestHandleJoin_InvalidPrimaryOperationPolicy(t *testing.T) {
	err := HandleJoin([]string{"--primary-operation-policy", "invalid", "f1.yaml", "f2.yaml"})
	assert.Error(t, err)
}

func TestJoinFlags_CombinedNewFlags(t *testing.T) {
	fs, flags := SetupJoinFlags()
	err := fs.Parse([]string{
		"--operation-context",
		"--primary-operation-policy", "most-specific",
		"--pre-overlay", "pre1.yaml",
		"--pre-overlay", "pre2.yaml",
		"--post-overlay", "post.yaml",
		"api1.yaml", "api2.yaml",
	})
	require.NoError(t, err)
	assert.True(t, flags.OperationContext, "expected OperationContext to be true")
	assert.Equal(t, "most-specific", flags.PrimaryOperationPolicy)
	assert.Len(t, flags.PreOverlays, 2)
	assert.Equal(t, "post.yaml", flags.PostOverlay)
}

func TestHandleJoin_NonexistentPreOverlay(t *testing.T) {
	err := HandleJoin([]string{
		"--pre-overlay", "/nonexistent/path/overlay.yaml",
		"../../testdata/oas3/petstore.yaml",
		"../../testdata/oas3/petstore.yaml",
	})
	assert.Error(t, err)
}

func TestHandleJoin_NonexistentPostOverlay(t *testing.T) {
	err := HandleJoin([]string{
		"--post-overlay", "/nonexistent/path/overlay.yaml",
		"../../testdata/oas3/petstore.yaml",
		"../../testdata/oas3/petstore.yaml",
	})
	assert.Error(t, err)
}
