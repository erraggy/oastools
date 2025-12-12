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
