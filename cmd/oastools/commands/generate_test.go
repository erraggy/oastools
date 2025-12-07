package commands

import (
	"testing"
)

func TestSetupGenerateFlags(t *testing.T) {
	fs, flags := SetupGenerateFlags()

	t.Run("default values", func(t *testing.T) {
		if flags.Output != "" {
			t.Errorf("expected Output to be empty by default, got '%s'", flags.Output)
		}
		if flags.PackageName != "api" {
			t.Errorf("expected PackageName 'api' by default, got '%s'", flags.PackageName)
		}
		if flags.Client {
			t.Error("expected Client to be false by default")
		}
		if flags.Server {
			t.Error("expected Server to be false by default")
		}
		if !flags.Types {
			t.Error("expected Types to be true by default")
		}
		if flags.NoPointers {
			t.Error("expected NoPointers to be false by default")
		}
		if flags.NoValidation {
			t.Error("expected NoValidation to be false by default")
		}
		if flags.Strict {
			t.Error("expected Strict to be false by default")
		}
		if flags.NoWarnings {
			t.Error("expected NoWarnings to be false by default")
		}
	})

	t.Run("parse flags", func(t *testing.T) {
		args := []string{"-o", "./output", "-p", "myapi", "--client", "--server", "--no-pointers", "--strict", "spec.yaml"}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		if flags.Output != "./output" {
			t.Errorf("expected Output './output', got '%s'", flags.Output)
		}
		if flags.PackageName != "myapi" {
			t.Errorf("expected PackageName 'myapi', got '%s'", flags.PackageName)
		}
		if !flags.Client {
			t.Error("expected Client to be true")
		}
		if !flags.Server {
			t.Error("expected Server to be true")
		}
		if !flags.NoPointers {
			t.Error("expected NoPointers to be true")
		}
		if !flags.Strict {
			t.Error("expected Strict to be true")
		}
		if fs.Arg(0) != "spec.yaml" {
			t.Errorf("expected file arg 'spec.yaml', got '%s'", fs.Arg(0))
		}
	})
}

func TestHandleGenerate_NoArgs(t *testing.T) {
	err := HandleGenerate([]string{})
	if err == nil {
		t.Error("expected error when no file provided")
	}
}

func TestHandleGenerate_Help(t *testing.T) {
	err := HandleGenerate([]string{"--help"})
	if err != nil {
		t.Errorf("unexpected error for help: %v", err)
	}
}

func TestHandleGenerate_NoOutput(t *testing.T) {
	err := HandleGenerate([]string{"spec.yaml"})
	if err == nil {
		t.Error("expected error when no output directory provided")
	}
}

func TestHandleGenerate_NoGenerationMode(t *testing.T) {
	err := HandleGenerate([]string{"-o", "./out", "--types=false", "spec.yaml"})
	if err == nil {
		t.Error("expected error when no generation mode enabled")
	}
}
