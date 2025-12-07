package commands

import (
	"testing"
)

func TestSetupFixFlags(t *testing.T) {
	fs, flags := SetupFixFlags()

	t.Run("default values", func(t *testing.T) {
		if flags.Output != "" {
			t.Errorf("expected Output to be empty by default, got '%s'", flags.Output)
		}
		if flags.Infer {
			t.Error("expected Infer to be false by default")
		}
		if flags.Quiet {
			t.Error("expected Quiet to be false by default")
		}
	})

	t.Run("parse flags", func(t *testing.T) {
		args := []string{"-o", "fixed.yaml", "--infer", "-q", "input.yaml"}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		if flags.Output != "fixed.yaml" {
			t.Errorf("expected Output 'fixed.yaml', got '%s'", flags.Output)
		}
		if !flags.Infer {
			t.Error("expected Infer to be true")
		}
		if !flags.Quiet {
			t.Error("expected Quiet to be true")
		}
		if fs.Arg(0) != "input.yaml" {
			t.Errorf("expected file arg 'input.yaml', got '%s'", fs.Arg(0))
		}
	})

	t.Run("long flags", func(t *testing.T) {
		fs2, flags2 := SetupFixFlags()
		args := []string{"--output", "out.yaml", "--quiet", "in.yaml"}
		if err := fs2.Parse(args); err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		if flags2.Output != "out.yaml" {
			t.Errorf("expected Output 'out.yaml', got '%s'", flags2.Output)
		}
		if !flags2.Quiet {
			t.Error("expected Quiet to be true")
		}
	})
}

func TestHandleFix_NoArgs(t *testing.T) {
	err := HandleFix([]string{})
	if err == nil {
		t.Error("expected error when no file provided")
	}
}

func TestHandleFix_Help(t *testing.T) {
	err := HandleFix([]string{"--help"})
	if err != nil {
		t.Errorf("unexpected error for help: %v", err)
	}
}
