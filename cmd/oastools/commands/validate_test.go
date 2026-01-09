package commands

import (
	"testing"
)

func TestSetupValidateFlags(t *testing.T) {
	fs, flags := SetupValidateFlags()

	t.Run("default values", func(t *testing.T) {
		if flags.Strict {
			t.Error("expected Strict to be false by default")
		}
		if !flags.ValidateStructure {
			t.Error("expected ValidateStructure to be true by default")
		}
		if flags.NoWarnings {
			t.Error("expected NoWarnings to be false by default")
		}
		if flags.Quiet {
			t.Error("expected Quiet to be false by default")
		}
		if flags.Format != FormatText {
			t.Errorf("expected Format to be '%s' by default, got '%s'", FormatText, flags.Format)
		}
	})

	t.Run("parse flags", func(t *testing.T) {
		args := []string{"--strict", "--no-warnings", "-q", "--format", "json", "test.yaml"}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("unexpected parse error: %v", err)
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
		if flags.Format != "json" {
			t.Errorf("expected Format 'json', got '%s'", flags.Format)
		}
		if fs.Arg(0) != "test.yaml" {
			t.Errorf("expected file arg 'test.yaml', got '%s'", fs.Arg(0))
		}
	})

	t.Run("validate-structure flag", func(t *testing.T) {
		// Create fresh flagset to test validate-structure flag
		fs2, flags2 := SetupValidateFlags()
		args := []string{"--validate-structure=false", "test.yaml"}
		if err := fs2.Parse(args); err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		if flags2.ValidateStructure {
			t.Error("expected ValidateStructure to be false when --validate-structure=false")
		}
	})
}

func TestHandleValidate_NoArgs(t *testing.T) {
	err := HandleValidate([]string{})
	if err == nil {
		t.Error("expected error when no file provided")
	}
}

func TestHandleValidate_Help(t *testing.T) {
	err := HandleValidate([]string{"--help"})
	if err != nil {
		t.Errorf("unexpected error for help: %v", err)
	}
}

func TestHandleValidate_InvalidFormat(t *testing.T) {
	err := HandleValidate([]string{"--format", "invalid", "test.yaml"})
	if err == nil {
		t.Error("expected error for invalid format")
	}
}
