package commands

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

func TestValidateOutputFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{"valid text", FormatText, false},
		{"valid json", FormatJSON, false},
		{"valid yaml", FormatYAML, false},
		{"invalid format", "xml", true},
		{"empty format", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutputFormat(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOutputFormat(%q) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCollisionStrategy(t *testing.T) {
	tests := []struct {
		name         string
		strategyName string
		value        string
		wantErr      bool
	}{
		{"empty value", "path-strategy", "", false},
		{"valid accept-left", "path-strategy", "accept-left", false},
		{"valid accept-right", "schema-strategy", "accept-right", false},
		{"valid fail", "component-strategy", "fail", false},
		{"invalid strategy", "path-strategy", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCollisionStrategy(tt.strategyName, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCollisionStrategy(%q, %q) error = %v, wantErr %v", tt.strategyName, tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestMarshalDocument(t *testing.T) {
	doc := map[string]string{"key": "value"}

	t.Run("json format", func(t *testing.T) {
		data, err := MarshalDocument(doc, parser.SourceFormatJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) == 0 {
			t.Error("expected non-empty output")
		}
	})

	t.Run("yaml format", func(t *testing.T) {
		data, err := MarshalDocument(doc, parser.SourceFormatYAML)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) == 0 {
			t.Error("expected non-empty output")
		}
	})
}

func TestOutputStructured(t *testing.T) {
	data := map[string]string{"test": "value"}

	t.Run("invalid format", func(t *testing.T) {
		err := OutputStructured(data, "invalid")
		if err == nil {
			t.Error("expected error for invalid format")
		}
	})
}
