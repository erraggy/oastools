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
