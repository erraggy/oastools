package commands

import (
	"testing"
)

func TestSetupConvertFlags(t *testing.T) {
	fs, flags := SetupConvertFlags()

	t.Run("default values", func(t *testing.T) {
		if flags.Target != "" {
			t.Errorf("expected Target to be empty by default, got '%s'", flags.Target)
		}
		if flags.Output != "" {
			t.Errorf("expected Output to be empty by default, got '%s'", flags.Output)
		}
		if flags.Strict {
			t.Error("expected Strict to be false by default")
		}
		if flags.NoWarnings {
			t.Error("expected NoWarnings to be false by default")
		}
		if flags.Quiet {
			t.Error("expected Quiet to be false by default")
		}
	})

	t.Run("parse flags", func(t *testing.T) {
		args := []string{"-t", "3.0.3", "-o", "output.yaml", "--strict", "--no-warnings", "-q", "input.yaml"}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		if flags.Target != "3.0.3" {
			t.Errorf("expected Target '3.0.3', got '%s'", flags.Target)
		}
		if flags.Output != "output.yaml" {
			t.Errorf("expected Output 'output.yaml', got '%s'", flags.Output)
		}
		if !flags.Strict {
			t.Error("expected Strict to be true")
		}
		if !flags.NoWarnings {
			t.Error("expected NoWarnings to be true")
		}
		if !flags.Quiet {
			t.Error("expected Quiet to be true")
		}
		if fs.Arg(0) != "input.yaml" {
			t.Errorf("expected file arg 'input.yaml', got '%s'", fs.Arg(0))
		}
	})

	t.Run("long flags", func(t *testing.T) {
		fs2, flags2 := SetupConvertFlags()
		args := []string{"--target", "2.0", "--output", "out.yaml", "in.yaml"}
		if err := fs2.Parse(args); err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		if flags2.Target != "2.0" {
			t.Errorf("expected Target '2.0', got '%s'", flags2.Target)
		}
		if flags2.Output != "out.yaml" {
			t.Errorf("expected Output 'out.yaml', got '%s'", flags2.Output)
		}
	})
}

func TestHandleConvert_NoArgs(t *testing.T) {
	err := HandleConvert([]string{})
	if err == nil {
		t.Error("expected error when no file provided")
	}
}

func TestHandleConvert_Help(t *testing.T) {
	err := HandleConvert([]string{"--help"})
	if err != nil {
		t.Errorf("unexpected error for help: %v", err)
	}
}

func TestHandleConvert_NoTarget(t *testing.T) {
	err := HandleConvert([]string{"input.yaml"})
	if err == nil {
		t.Error("expected error when no target version provided")
	}
}
