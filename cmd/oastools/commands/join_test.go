package commands

import (
	"testing"

	"github.com/erraggy/oastools/joiner"
)

func TestSetupJoinFlags(t *testing.T) {
	fs, flags := SetupJoinFlags()

	t.Run("default values", func(t *testing.T) {
		if flags.Output != "" {
			t.Errorf("expected Output to be empty by default, got '%s'", flags.Output)
		}
		if flags.PathStrategy != "" {
			t.Errorf("expected PathStrategy to be empty by default, got '%s'", flags.PathStrategy)
		}
		if flags.NoMergeArrays {
			t.Error("expected NoMergeArrays to be false by default")
		}
		if flags.NoDedupTags {
			t.Error("expected NoDedupTags to be false by default")
		}
		if flags.Quiet {
			t.Error("expected Quiet to be false by default")
		}
		// Operation context flags
		if flags.OperationContext {
			t.Error("expected OperationContext to be false by default")
		}
		if flags.PrimaryOperationPolicy != "" {
			t.Errorf("expected PrimaryOperationPolicy to be empty by default, got '%s'", flags.PrimaryOperationPolicy)
		}
		// Overlay flags
		if len(flags.PreOverlays) != 0 {
			t.Errorf("expected PreOverlays to be empty by default, got %v", flags.PreOverlays)
		}
		if flags.PostOverlay != "" {
			t.Errorf("expected PostOverlay to be empty by default, got '%s'", flags.PostOverlay)
		}
	})

	t.Run("parse flags", func(t *testing.T) {
		args := []string{"-o", "output.yaml", "--path-strategy", "accept-left", "--no-merge-arrays", "-q", "file1.yaml", "file2.yaml"}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		if flags.Output != "output.yaml" {
			t.Errorf("expected Output 'output.yaml', got '%s'", flags.Output)
		}
		if flags.PathStrategy != "accept-left" {
			t.Errorf("expected PathStrategy 'accept-left', got '%s'", flags.PathStrategy)
		}
		if !flags.NoMergeArrays {
			t.Error("expected NoMergeArrays to be true")
		}
		if !flags.Quiet {
			t.Error("expected Quiet to be true")
		}
		if fs.NArg() != 2 {
			t.Errorf("expected 2 file args, got %d", fs.NArg())
		}
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
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if joiner.CollisionStrategy(flags.PathStrategy) != tt.expected {
				t.Errorf("expected strategy %s, got %s", tt.expected, flags.PathStrategy)
			}
		})
	}
}

func TestHandleJoin_NotEnoughFiles(t *testing.T) {
	err := HandleJoin([]string{"single.yaml"})
	if err == nil {
		t.Error("expected error when only one file provided")
	}
}

func TestHandleJoin_Help(t *testing.T) {
	err := HandleJoin([]string{"--help"})
	if err != nil {
		t.Errorf("unexpected error for help: %v", err)
	}
}

func TestHandleJoin_InvalidStrategy(t *testing.T) {
	err := HandleJoin([]string{"--path-strategy", "invalid", "f1.yaml", "f2.yaml"})
	if err == nil {
		t.Error("expected error for invalid strategy")
	}
}

func TestSetupJoinFlags_NamespacePrefix(t *testing.T) {
	t.Run("parse single namespace prefix", func(t *testing.T) {
		fs, flags := SetupJoinFlags()
		args := []string{"--namespace-prefix", "api.yaml=Api", "f1.yaml", "f2.yaml"}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}
		if prefix, ok := flags.NamespacePrefix["api.yaml"]; !ok || prefix != "Api" {
			t.Errorf("expected NamespacePrefix['api.yaml'] = 'Api', got '%v'", flags.NamespacePrefix)
		}
	})

	t.Run("parse multiple namespace prefixes", func(t *testing.T) {
		fs, flags := SetupJoinFlags()
		args := []string{
			"--namespace-prefix", "users.yaml=Users",
			"--namespace-prefix", "billing.yaml=Billing",
			"f1.yaml", "f2.yaml",
		}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}
		if flags.NamespacePrefix["users.yaml"] != "Users" {
			t.Errorf("expected NamespacePrefix['users.yaml'] = 'Users', got '%s'", flags.NamespacePrefix["users.yaml"])
		}
		if flags.NamespacePrefix["billing.yaml"] != "Billing" {
			t.Errorf("expected NamespacePrefix['billing.yaml'] = 'Billing', got '%s'", flags.NamespacePrefix["billing.yaml"])
		}
	})

	t.Run("parse always-prefix flag", func(t *testing.T) {
		fs, flags := SetupJoinFlags()
		args := []string{"--always-prefix", "f1.yaml", "f2.yaml"}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}
		if !flags.AlwaysPrefix {
			t.Error("expected AlwaysPrefix to be true")
		}
	})

	t.Run("parse namespace-prefix with always-prefix", func(t *testing.T) {
		fs, flags := SetupJoinFlags()
		args := []string{
			"--namespace-prefix", "api.yaml=Api",
			"--always-prefix",
			"f1.yaml", "f2.yaml",
		}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}
		if flags.NamespacePrefix["api.yaml"] != "Api" {
			t.Errorf("expected NamespacePrefix['api.yaml'] = 'Api', got '%s'", flags.NamespacePrefix["api.yaml"])
		}
		if !flags.AlwaysPrefix {
			t.Error("expected AlwaysPrefix to be true")
		}
	})

	t.Run("invalid namespace prefix format", func(t *testing.T) {
		fs, _ := SetupJoinFlags()
		args := []string{"--namespace-prefix", "invalid", "f1.yaml", "f2.yaml"}
		err := fs.Parse(args)
		if err == nil {
			t.Error("expected error for invalid namespace prefix format")
		}
	})

	t.Run("empty source in namespace prefix", func(t *testing.T) {
		fs, _ := SetupJoinFlags()
		args := []string{"--namespace-prefix", "=Prefix", "f1.yaml", "f2.yaml"}
		err := fs.Parse(args)
		if err == nil {
			t.Error("expected error for empty source in namespace prefix")
		}
	})

	t.Run("empty prefix in namespace prefix", func(t *testing.T) {
		fs, _ := SetupJoinFlags()
		args := []string{"--namespace-prefix", "api.yaml=", "f1.yaml", "f2.yaml"}
		err := fs.Parse(args)
		if err == nil {
			t.Error("expected error for empty prefix in namespace prefix")
		}
	})
}

func TestNamespacePrefixFlag_String(t *testing.T) {
	npf := make(namespacePrefixFlag)
	npf["a.yaml"] = "A"

	str := npf.String()
	if str != "a.yaml=A" {
		t.Errorf("expected 'a.yaml=A', got '%s'", str)
	}
}

func TestNamespacePrefixFlag_Set(t *testing.T) {
	t.Run("valid format", func(t *testing.T) {
		npf := make(namespacePrefixFlag)
		err := npf.Set("users.yaml=Users")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if npf["users.yaml"] != "Users" {
			t.Errorf("expected 'Users', got '%s'", npf["users.yaml"])
		}
	})

	t.Run("invalid format - no equals", func(t *testing.T) {
		npf := make(namespacePrefixFlag)
		err := npf.Set("invalid")
		if err == nil {
			t.Error("expected error for invalid format")
		}
	})

	t.Run("handles spaces in value", func(t *testing.T) {
		npf := make(namespacePrefixFlag)
		err := npf.Set("  users.yaml  =  Users  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if npf["users.yaml"] != "Users" {
			t.Errorf("expected 'Users', got '%s'", npf["users.yaml"])
		}
	})
}

func TestJoinFlags_OperationContext(t *testing.T) {
	fs, flags := SetupJoinFlags()
	err := fs.Parse([]string{"--operation-context", "api1.yaml", "api2.yaml"})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if !flags.OperationContext {
		t.Error("expected OperationContext to be true")
	}
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
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if flags.PrimaryOperationPolicy != tt.policy {
				t.Errorf("expected policy %q, got %q", tt.policy, flags.PrimaryOperationPolicy)
			}
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
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(flags.PreOverlays) != 2 {
		t.Fatalf("expected 2 pre-overlays, got %d", len(flags.PreOverlays))
	}
	if flags.PreOverlays[0] != "overlay1.yaml" {
		t.Errorf("expected first overlay 'overlay1.yaml', got %q", flags.PreOverlays[0])
	}
	if flags.PreOverlays[1] != "overlay2.yaml" {
		t.Errorf("expected second overlay 'overlay2.yaml', got %q", flags.PreOverlays[1])
	}
}

func TestJoinFlags_PostOverlay(t *testing.T) {
	fs, flags := SetupJoinFlags()
	err := fs.Parse([]string{"--post-overlay", "final.yaml", "api1.yaml", "api2.yaml"})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if flags.PostOverlay != "final.yaml" {
		t.Errorf("expected PostOverlay 'final.yaml', got %q", flags.PostOverlay)
	}
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
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
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
			if got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestStringSliceFlag(t *testing.T) {
	t.Run("empty string representation", func(t *testing.T) {
		var flag stringSliceFlag
		if flag.String() != "" {
			t.Errorf("expected empty string, got %q", flag.String())
		}
	})

	t.Run("nil string representation", func(t *testing.T) {
		var flag *stringSliceFlag
		if flag.String() != "" {
			t.Errorf("expected empty string for nil, got %q", flag.String())
		}
	})

	t.Run("set and string", func(t *testing.T) {
		var flag stringSliceFlag
		if err := flag.Set("value1"); err != nil {
			t.Fatalf("unexpected error on first Set: %v", err)
		}
		if err := flag.Set("value2"); err != nil {
			t.Fatalf("unexpected error on second Set: %v", err)
		}
		if len(flag) != 2 {
			t.Errorf("expected length 2, got %d", len(flag))
		}
		expected := "value1,value2"
		if flag.String() != expected {
			t.Errorf("expected %q, got %q", expected, flag.String())
		}
	})
}

func TestHandleJoin_InvalidPrimaryOperationPolicy(t *testing.T) {
	err := HandleJoin([]string{"--primary-operation-policy", "invalid", "f1.yaml", "f2.yaml"})
	if err == nil {
		t.Error("expected error for invalid primary operation policy")
	}
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
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if !flags.OperationContext {
		t.Error("expected OperationContext to be true")
	}
	if flags.PrimaryOperationPolicy != "most-specific" {
		t.Errorf("expected policy 'most-specific', got %q", flags.PrimaryOperationPolicy)
	}
	if len(flags.PreOverlays) != 2 {
		t.Errorf("expected 2 pre-overlays, got %d", len(flags.PreOverlays))
	}
	if flags.PostOverlay != "post.yaml" {
		t.Errorf("expected PostOverlay 'post.yaml', got %q", flags.PostOverlay)
	}
}

func TestHandleJoin_NonexistentPreOverlay(t *testing.T) {
	err := HandleJoin([]string{
		"--pre-overlay", "/nonexistent/path/overlay.yaml",
		"../../testdata/oas3/petstore.yaml",
		"../../testdata/oas3/petstore.yaml",
	})
	if err == nil {
		t.Error("expected error for nonexistent pre-overlay file")
	}
}

func TestHandleJoin_NonexistentPostOverlay(t *testing.T) {
	err := HandleJoin([]string{
		"--post-overlay", "/nonexistent/path/overlay.yaml",
		"../../testdata/oas3/petstore.yaml",
		"../../testdata/oas3/petstore.yaml",
	})
	if err == nil {
		t.Error("expected error for nonexistent post-overlay file")
	}
}
