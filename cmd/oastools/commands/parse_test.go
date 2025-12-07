package commands

import (
	"testing"
)

func TestSetupParseFlags(t *testing.T) {
	fs, flags := SetupParseFlags()

	t.Run("default values", func(t *testing.T) {
		if flags.ResolveRefs {
			t.Error("expected ResolveRefs to be false by default")
		}
		if flags.ResolveHTTPRefs {
			t.Error("expected ResolveHTTPRefs to be false by default")
		}
		if flags.Insecure {
			t.Error("expected Insecure to be false by default")
		}
		if flags.ValidateStructure {
			t.Error("expected ValidateStructure to be false by default")
		}
		if flags.Quiet {
			t.Error("expected Quiet to be false by default")
		}
	})

	t.Run("parse flags", func(t *testing.T) {
		args := []string{"--resolve-refs", "--resolve-http-refs", "--insecure", "--validate-structure", "-q", "test.yaml"}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		if !flags.ResolveRefs {
			t.Error("expected ResolveRefs to be true")
		}
		if !flags.ResolveHTTPRefs {
			t.Error("expected ResolveHTTPRefs to be true")
		}
		if !flags.Insecure {
			t.Error("expected Insecure to be true")
		}
		if !flags.ValidateStructure {
			t.Error("expected ValidateStructure to be true")
		}
		if !flags.Quiet {
			t.Error("expected Quiet to be true")
		}
		if fs.Arg(0) != "test.yaml" {
			t.Errorf("expected file arg 'test.yaml', got '%s'", fs.Arg(0))
		}
	})
}

func TestHandleParse_NoArgs(t *testing.T) {
	err := HandleParse([]string{})
	if err == nil {
		t.Error("expected error when no file provided")
	}
}

func TestHandleParse_Help(t *testing.T) {
	err := HandleParse([]string{"--help"})
	if err != nil {
		t.Errorf("unexpected error for help: %v", err)
	}
}
