package commands

import (
	"testing"
)

func TestSetupDiffFlags(t *testing.T) {
	fs, flags := SetupDiffFlags()

	t.Run("default values", func(t *testing.T) {
		if flags.Breaking {
			t.Error("expected Breaking to be false by default")
		}
		if flags.NoInfo {
			t.Error("expected NoInfo to be false by default")
		}
		if flags.Format != FormatText {
			t.Errorf("expected Format to be '%s' by default, got '%s'", FormatText, flags.Format)
		}
	})

	t.Run("parse flags", func(t *testing.T) {
		args := []string{"--breaking", "--no-info", "--format", "json", "v1.yaml", "v2.yaml"}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		if !flags.Breaking {
			t.Error("expected Breaking to be true")
		}
		if !flags.NoInfo {
			t.Error("expected NoInfo to be true")
		}
		if flags.Format != "json" {
			t.Errorf("expected Format 'json', got '%s'", flags.Format)
		}
		if fs.NArg() != 2 {
			t.Errorf("expected 2 file args, got %d", fs.NArg())
		}
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
			if err == nil {
				t.Error("expected error when not enough files provided")
			}
		})
	}
}

func TestHandleDiff_Help(t *testing.T) {
	err := HandleDiff([]string{"--help"})
	if err != nil {
		t.Errorf("unexpected error for help: %v", err)
	}
}

func TestHandleDiff_InvalidFormat(t *testing.T) {
	err := HandleDiff([]string{"--format", "invalid", "v1.yaml", "v2.yaml"})
	if err == nil {
		t.Error("expected error for invalid format")
	}
}
