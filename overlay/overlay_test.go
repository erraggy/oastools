package overlay

import (
	"errors"
	"testing"

	"github.com/erraggy/oastools/parser"
)

// TestParseOverlay tests parsing overlay documents.
func TestParseOverlay(t *testing.T) {
	t.Run("valid YAML overlay", func(t *testing.T) {
		data := []byte(`
overlay: 1.0.0
info:
  title: Test Overlay
  version: 1.0.0
actions:
  - target: $.info
    update:
      title: New Title
`)
		o, err := ParseOverlay(data)
		if err != nil {
			t.Fatalf("ParseOverlay error: %v", err)
		}

		if o.Version != "1.0.0" {
			t.Errorf("Version = %q, want %q", o.Version, "1.0.0")
		}
		if o.Info.Title != "Test Overlay" {
			t.Errorf("Info.Title = %q, want %q", o.Info.Title, "Test Overlay")
		}
		if len(o.Actions) != 1 {
			t.Errorf("len(Actions) = %d, want 1", len(o.Actions))
		}
	})

	t.Run("valid JSON overlay", func(t *testing.T) {
		data := []byte(`{
			"overlay": "1.0.0",
			"info": {"title": "JSON Overlay", "version": "1.0.0"},
			"actions": [{"target": "$.info", "update": {"x-test": true}}]
		}`)
		o, err := ParseOverlay(data)
		if err != nil {
			t.Fatalf("ParseOverlay error: %v", err)
		}

		if o.Info.Title != "JSON Overlay" {
			t.Errorf("Info.Title = %q, want %q", o.Info.Title, "JSON Overlay")
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		data := []byte(`overlay: [invalid`)
		_, err := ParseOverlay(data)
		if err == nil {
			t.Error("Expected error for invalid YAML")
		}
	})

	t.Run("overlay with extends", func(t *testing.T) {
		data := []byte(`
overlay: 1.0.0
info:
  title: With Extends
  version: 1.0.0
extends: ./openapi.yaml
actions:
  - target: $.info
    remove: true
`)
		o, err := ParseOverlay(data)
		if err != nil {
			t.Fatalf("ParseOverlay error: %v", err)
		}

		if o.Extends != "./openapi.yaml" {
			t.Errorf("Extends = %q, want %q", o.Extends, "./openapi.yaml")
		}
	})
}

// TestValidate tests overlay validation.
func TestValidate(t *testing.T) {
	t.Run("valid overlay", func(t *testing.T) {
		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{{Target: "$.info", Update: map[string]any{"x": 1}}},
		}

		errs := Validate(o)
		if len(errs) != 0 {
			t.Errorf("Expected no errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("missing version", func(t *testing.T) {
		o := &Overlay{
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{{Target: "$.info", Update: map[string]any{}}},
		}

		errs := Validate(o)
		if len(errs) == 0 {
			t.Error("Expected error for missing version")
		}
	})

	t.Run("unsupported version", func(t *testing.T) {
		o := &Overlay{
			Version: "2.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{{Target: "$.info", Update: map[string]any{}}},
		}

		errs := Validate(o)
		found := false
		for _, e := range errs {
			if e.Field == "overlay" {
				found = true
			}
		}
		if !found {
			t.Error("Expected error for unsupported version")
		}
	})

	t.Run("missing info title", func(t *testing.T) {
		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Version: "1.0.0"},
			Actions: []Action{{Target: "$.info", Update: map[string]any{}}},
		}

		errs := Validate(o)
		found := false
		for _, e := range errs {
			if e.Field == "info.title" {
				found = true
			}
		}
		if !found {
			t.Error("Expected error for missing info.title")
		}
	})

	t.Run("missing info version", func(t *testing.T) {
		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test"},
			Actions: []Action{{Target: "$.info", Update: map[string]any{}}},
		}

		errs := Validate(o)
		found := false
		for _, e := range errs {
			if e.Field == "info.version" {
				found = true
			}
		}
		if !found {
			t.Error("Expected error for missing info.version")
		}
	})

	t.Run("empty actions", func(t *testing.T) {
		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{},
		}

		errs := Validate(o)
		found := false
		for _, e := range errs {
			if e.Field == "actions" {
				found = true
			}
		}
		if !found {
			t.Error("Expected error for empty actions")
		}
	})

	t.Run("action missing target", func(t *testing.T) {
		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{{Update: map[string]any{"x": 1}}},
		}

		errs := Validate(o)
		found := false
		for _, e := range errs {
			if e.Path == "actions[0].target" {
				found = true
			}
		}
		if !found {
			t.Error("Expected error for missing target")
		}
	})

	t.Run("action invalid JSONPath", func(t *testing.T) {
		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{{Target: "invalid[", Update: map[string]any{}}},
		}

		errs := Validate(o)
		found := false
		for _, e := range errs {
			if e.Path == "actions[0].target" {
				found = true
			}
		}
		if !found {
			t.Error("Expected error for invalid JSONPath")
		}
	})

	t.Run("action no update or remove", func(t *testing.T) {
		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{{Target: "$.info"}},
		}

		errs := Validate(o)
		found := false
		for _, e := range errs {
			if e.Path == "actions[0]" {
				found = true
			}
		}
		if !found {
			t.Error("Expected error for action without update or remove")
		}
	})
}

// TestIsValid tests the IsValid helper function.
func TestIsValid(t *testing.T) {
	validOverlay := &Overlay{
		Version: "1.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Actions: []Action{{Target: "$.info", Update: map[string]any{}}},
	}

	invalidOverlay := &Overlay{
		Version: "",
		Info:    Info{},
		Actions: []Action{},
	}

	if !IsValid(validOverlay) {
		t.Error("Expected valid overlay to be valid")
	}

	if IsValid(invalidOverlay) {
		t.Error("Expected invalid overlay to be invalid")
	}
}

// TestApplyResult tests ApplyResult helper methods.
func TestApplyResult(t *testing.T) {
	t.Run("HasChanges", func(t *testing.T) {
		r := &ApplyResult{ActionsApplied: 0}
		if r.HasChanges() {
			t.Error("Expected no changes")
		}

		r.ActionsApplied = 1
		if !r.HasChanges() {
			t.Error("Expected changes")
		}
	})

	t.Run("HasWarnings", func(t *testing.T) {
		r := &ApplyResult{}
		if r.HasWarnings() {
			t.Error("Expected no warnings")
		}

		r.Warnings = []string{"warning"}
		if !r.HasWarnings() {
			t.Error("Expected warnings")
		}
	})
}

// TestApplier tests the Applier.
func TestApplier(t *testing.T) {
	t.Run("apply update action", func(t *testing.T) {
		doc := map[string]any{
			"openapi": "3.0.3",
			"info": map[string]any{
				"title":   "Original",
				"version": "1.0.0",
			},
		}

		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{
				{
					Target: "$.info",
					Update: map[string]any{
						"title":   "Updated",
						"x-extra": "added",
					},
				},
			},
		}

		spec := &parser.ParseResult{
			Document:     doc,
			SourceFormat: parser.SourceFormatYAML,
		}

		a := NewApplier()
		result, err := a.ApplyParsed(spec, o)
		if err != nil {
			t.Fatalf("Apply error: %v", err)
		}

		if result.ActionsApplied != 1 {
			t.Errorf("ActionsApplied = %d, want 1", result.ActionsApplied)
		}

		resultDoc := result.Document.(map[string]any)
		info := resultDoc["info"].(map[string]any)
		if info["title"] != "Updated" {
			t.Errorf("title = %v, want Updated", info["title"])
		}
		if info["x-extra"] != "added" {
			t.Errorf("x-extra = %v, want added", info["x-extra"])
		}
		// Original field should be preserved
		if info["version"] != "1.0.0" {
			t.Errorf("version = %v, want 1.0.0", info["version"])
		}
	})

	t.Run("apply remove action", func(t *testing.T) {
		doc := map[string]any{
			"paths": map[string]any{
				"/public": map[string]any{
					"x-internal": false,
				},
				"/internal": map[string]any{
					"x-internal": true,
				},
			},
		}

		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{
				{
					Target: "$.paths[?@.x-internal==true]",
					Remove: true,
				},
			},
		}

		spec := &parser.ParseResult{
			Document:     doc,
			SourceFormat: parser.SourceFormatYAML,
		}

		a := NewApplier()
		result, err := a.ApplyParsed(spec, o)
		if err != nil {
			t.Fatalf("Apply error: %v", err)
		}

		if result.ActionsApplied != 1 {
			t.Errorf("ActionsApplied = %d, want 1", result.ActionsApplied)
		}

		resultDoc := result.Document.(map[string]any)
		paths := resultDoc["paths"].(map[string]any)
		if _, exists := paths["/internal"]; exists {
			t.Error("Internal path should have been removed")
		}
		if _, exists := paths["/public"]; !exists {
			t.Error("Public path should still exist")
		}
	})

	t.Run("sequential actions", func(t *testing.T) {
		doc := map[string]any{
			"info": map[string]any{
				"title":   "Original",
				"version": "1.0.0",
			},
		}

		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{
				{Target: "$.info", Update: map[string]any{"title": "Step1"}},
				{Target: "$.info", Update: map[string]any{"x-step": "2"}},
				{Target: "$.info", Update: map[string]any{"title": "Final"}},
			},
		}

		spec := &parser.ParseResult{
			Document:     doc,
			SourceFormat: parser.SourceFormatYAML,
		}

		a := NewApplier()
		result, err := a.ApplyParsed(spec, o)
		if err != nil {
			t.Fatalf("Apply error: %v", err)
		}

		if result.ActionsApplied != 3 {
			t.Errorf("ActionsApplied = %d, want 3", result.ActionsApplied)
		}

		resultDoc := result.Document.(map[string]any)
		info := resultDoc["info"].(map[string]any)
		if info["title"] != "Final" {
			t.Errorf("title = %v, want Final", info["title"])
		}
		if info["x-step"] != "2" {
			t.Errorf("x-step = %v, want 2", info["x-step"])
		}
	})

	t.Run("no match warning", func(t *testing.T) {
		doc := map[string]any{"info": map[string]any{}}

		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{
				{Target: "$.nonexistent", Update: map[string]any{"x": 1}},
			},
		}

		spec := &parser.ParseResult{
			Document:     doc,
			SourceFormat: parser.SourceFormatYAML,
		}

		a := NewApplier()
		a.StrictTargets = false

		result, err := a.ApplyParsed(spec, o)
		if err != nil {
			t.Fatalf("Apply error: %v", err)
		}

		if result.ActionsSkipped != 1 {
			t.Errorf("ActionsSkipped = %d, want 1", result.ActionsSkipped)
		}
		if len(result.Warnings) == 0 {
			t.Error("Expected warning for no matches")
		}
	})

	t.Run("strict mode error", func(t *testing.T) {
		doc := map[string]any{"info": map[string]any{}}

		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{
				{Target: "$.nonexistent", Update: map[string]any{"x": 1}},
			},
		}

		spec := &parser.ParseResult{
			Document:     doc,
			SourceFormat: parser.SourceFormatYAML,
		}

		a := NewApplier()
		a.StrictTargets = true

		_, err := a.ApplyParsed(spec, o)
		if err == nil {
			t.Error("Expected error in strict mode for no matches")
		}
	})

	t.Run("original document unchanged", func(t *testing.T) {
		doc := map[string]any{
			"info": map[string]any{
				"title": "Original",
			},
		}

		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{
				{Target: "$.info", Update: map[string]any{"title": "Changed"}},
			},
		}

		spec := &parser.ParseResult{
			Document:     doc,
			SourceFormat: parser.SourceFormatYAML,
		}

		a := NewApplier()
		_, err := a.ApplyParsed(spec, o)
		if err != nil {
			t.Fatalf("Apply error: %v", err)
		}

		// Original should be unchanged
		originalInfo := doc["info"].(map[string]any)
		if originalInfo["title"] != "Original" {
			t.Error("Original document was modified")
		}
	})
}

// TestApplyWithOptions tests the functional options API.
func TestApplyWithOptions(t *testing.T) {
	t.Run("with parsed inputs", func(t *testing.T) {
		doc := map[string]any{
			"info": map[string]any{"title": "Test"},
		}
		spec := parser.ParseResult{
			Document:     doc,
			SourceFormat: parser.SourceFormatYAML,
		}
		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{
				{Target: "$.info", Update: map[string]any{"x": 1}},
			},
		}

		result, err := ApplyWithOptions(
			WithSpecParsed(spec),
			WithOverlayParsed(o),
		)
		if err != nil {
			t.Fatalf("ApplyWithOptions error: %v", err)
		}

		if result.ActionsApplied != 1 {
			t.Errorf("ActionsApplied = %d, want 1", result.ActionsApplied)
		}
	})

	t.Run("missing spec source", func(t *testing.T) {
		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{{Target: "$.info", Update: map[string]any{}}},
		}

		_, err := ApplyWithOptions(WithOverlayParsed(o))
		if err == nil {
			t.Error("Expected error for missing spec source")
		}
	})

	t.Run("missing overlay source", func(t *testing.T) {
		spec := parser.ParseResult{
			Document: map[string]any{},
		}

		_, err := ApplyWithOptions(WithSpecParsed(spec))
		if err == nil {
			t.Error("Expected error for missing overlay source")
		}
	})

	t.Run("with strict targets", func(t *testing.T) {
		doc := map[string]any{"info": map[string]any{}}
		spec := parser.ParseResult{Document: doc}
		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{
				{Target: "$.nonexistent", Update: map[string]any{}},
			},
		}

		_, err := ApplyWithOptions(
			WithSpecParsed(spec),
			WithOverlayParsed(o),
			WithStrictTargets(true),
		)
		if err == nil {
			t.Error("Expected error with strict targets")
		}
	})
}

// TestIsOverlayDocument tests the document detection function.
func TestIsOverlayDocument(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"YAML overlay", "overlay: 1.0.0\ninfo:", true},
		{"JSON overlay", `{"overlay": "1.0.0"}`, true},
		{"OpenAPI spec", "openapi: 3.0.3\ninfo:", false},
		{"Swagger spec", "swagger: \"2.0\"\ninfo:", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsOverlayDocument([]byte(tt.data))
			if got != tt.want {
				t.Errorf("IsOverlayDocument() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestErrorTypes tests error type methods.
func TestErrorTypes(t *testing.T) {
	t.Run("ValidationError", func(t *testing.T) {
		e := ValidationError{Field: "test", Message: "error"}
		if e.Error() == "" {
			t.Error("Expected non-empty error message")
		}

		e2 := ValidationError{Path: "actions[0]", Message: "error"}
		if e2.Error() == "" {
			t.Error("Expected non-empty error message with path")
		}
	})

	t.Run("ApplyError", func(t *testing.T) {
		e := &ApplyError{ActionIndex: 0, Target: "$.info", Cause: nil}
		if e.Error() == "" {
			t.Error("Expected non-empty error message")
		}
	})

	t.Run("ParseError", func(t *testing.T) {
		e := &ParseError{Path: "test.yaml", Cause: nil}
		if e.Error() == "" {
			t.Error("Expected non-empty error message")
		}
	})
}

// TestMarshalOverlay tests overlay serialization.
func TestMarshalOverlay(t *testing.T) {
	o := &Overlay{
		Version: "1.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Actions: []Action{
			{Target: "$.info", Update: map[string]any{"x": 1}},
		},
	}

	data, err := MarshalOverlay(o)
	if err != nil {
		t.Fatalf("MarshalOverlay error: %v", err)
	}

	// Parse it back
	o2, err := ParseOverlay(data)
	if err != nil {
		t.Fatalf("ParseOverlay error: %v", err)
	}

	if o2.Version != o.Version {
		t.Errorf("Version mismatch: %v != %v", o2.Version, o.Version)
	}
	if o2.Info.Title != o.Info.Title {
		t.Errorf("Title mismatch: %v != %v", o2.Info.Title, o.Info.Title)
	}
}

// TestParseOverlayFile tests file-based overlay parsing.
func TestParseOverlayFile(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		o, err := ParseOverlayFile("../testdata/overlay/valid/basic-update.yaml")
		if err != nil {
			t.Fatalf("ParseOverlayFile error: %v", err)
		}
		if o.Version != "1.0.0" {
			t.Errorf("Version = %q, want 1.0.0", o.Version)
		}
		if len(o.Actions) != 1 {
			t.Errorf("len(Actions) = %d, want 1", len(o.Actions))
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := ParseOverlayFile("nonexistent.yaml")
		if err == nil {
			t.Error("Expected error for missing file")
		}
	})

	t.Run("invalid yaml file", func(t *testing.T) {
		_, err := ParseOverlayFile("../testdata/overlay/invalid/invalid-jsonpath.yaml")
		// File parses but validation would fail
		if err != nil {
			t.Logf("Got error (expected for invalid file): %v", err)
		}
	})
}

// TestApplierApply tests the file-based Apply method.
func TestApplierApply(t *testing.T) {
	t.Run("valid files", func(t *testing.T) {
		a := NewApplier()
		result, err := a.Apply(
			"../testdata/overlay/fixtures/petstore-base.yaml",
			"../testdata/overlay/fixtures/petstore-overlay.yaml",
		)
		if err != nil {
			t.Fatalf("Apply error: %v", err)
		}

		if result.ActionsApplied != 3 {
			t.Errorf("ActionsApplied = %d, want 3", result.ActionsApplied)
		}
	})

	t.Run("missing spec file", func(t *testing.T) {
		a := NewApplier()
		_, err := a.Apply("nonexistent.yaml", "../testdata/overlay/valid/basic-update.yaml")
		if err == nil {
			t.Error("Expected error for missing spec file")
		}
	})

	t.Run("missing overlay file", func(t *testing.T) {
		a := NewApplier()
		_, err := a.Apply("../testdata/overlay/fixtures/petstore-base.yaml", "nonexistent.yaml")
		if err == nil {
			t.Error("Expected error for missing overlay file")
		}
	})
}

// TestApplyWithOptionsFilePaths tests file path options.
func TestApplyWithOptionsFilePaths(t *testing.T) {
	t.Run("with file paths", func(t *testing.T) {
		result, err := ApplyWithOptions(
			WithSpecFilePath("../testdata/overlay/fixtures/petstore-base.yaml"),
			WithOverlayFilePath("../testdata/overlay/fixtures/petstore-overlay.yaml"),
		)
		if err != nil {
			t.Fatalf("ApplyWithOptions error: %v", err)
		}

		if result.ActionsApplied != 3 {
			t.Errorf("ActionsApplied = %d, want 3", result.ActionsApplied)
		}
	})

	t.Run("empty spec path", func(t *testing.T) {
		_, err := ApplyWithOptions(
			WithSpecFilePath(""),
			WithOverlayFilePath("../testdata/overlay/valid/basic-update.yaml"),
		)
		if err == nil {
			t.Error("Expected error for empty spec path")
		}
	})

	t.Run("empty overlay path", func(t *testing.T) {
		_, err := ApplyWithOptions(
			WithSpecFilePath("../testdata/overlay/fixtures/petstore-base.yaml"),
			WithOverlayFilePath(""),
		)
		if err == nil {
			t.Error("Expected error for empty overlay path")
		}
	})

	t.Run("nil overlay", func(t *testing.T) {
		_, err := ApplyWithOptions(
			WithSpecFilePath("../testdata/overlay/fixtures/petstore-base.yaml"),
			WithOverlayParsed(nil),
		)
		if err == nil {
			t.Error("Expected error for nil overlay")
		}
	})
}

// TestErrorUnwrap tests error Unwrap methods.
func TestErrorUnwrap(t *testing.T) {
	t.Run("ApplyError Unwrap", func(t *testing.T) {
		cause := &ParseError{Path: "test"}
		e := &ApplyError{Cause: cause}
		unwrapped := e.Unwrap()
		if unwrapped == nil {
			t.Error("Unwrap should return non-nil cause")
		}
		// Verify errors.As works with ApplyError
		var applyErr *ApplyError
		if !errors.As(e, &applyErr) {
			t.Error("errors.As should find ApplyError")
		}
	})

	t.Run("ParseError Unwrap", func(t *testing.T) {
		cause := &ValidationError{Message: "test"}
		e := &ParseError{Cause: cause}
		unwrapped := e.Unwrap()
		if unwrapped == nil {
			t.Error("Unwrap should return non-nil cause")
		}
		// Verify errors.As works with ParseError
		var parseErr *ParseError
		if !errors.As(e, &parseErr) {
			t.Error("errors.As should find ParseError")
		}
	})
}

// TestApplyScalarReplace tests replacing scalar values.
func TestApplyScalarReplace(t *testing.T) {
	doc := map[string]any{
		"info": map[string]any{
			"title": "Old Title",
		},
	}

	o := &Overlay{
		Version: "1.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Actions: []Action{
			{Target: "$.info.title", Update: "New Title"},
		},
	}

	spec := &parser.ParseResult{
		Document:     doc,
		SourceFormat: parser.SourceFormatYAML,
	}

	a := NewApplier()
	result, err := a.ApplyParsed(spec, o)
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}

	if result.ActionsApplied != 1 {
		t.Errorf("ActionsApplied = %d, want 1", result.ActionsApplied)
	}

	// Check the change record
	if len(result.Changes) == 0 {
		t.Fatal("Expected change record")
	}
	if result.Changes[0].Operation != "replace" {
		t.Errorf("Operation = %q, want replace", result.Changes[0].Operation)
	}
}

// TestApplyArrayAppend tests appending to arrays.
func TestApplyArrayAppend(t *testing.T) {
	doc := map[string]any{
		"servers": []any{
			map[string]any{"url": "https://api.example.com"},
		},
	}

	o := &Overlay{
		Version: "1.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Actions: []Action{
			{Target: "$.servers", Update: map[string]any{"url": "https://staging.example.com"}},
		},
	}

	spec := &parser.ParseResult{
		Document:     doc,
		SourceFormat: parser.SourceFormatYAML,
	}

	a := NewApplier()
	result, err := a.ApplyParsed(spec, o)
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}

	// Check the change record
	if len(result.Changes) == 0 {
		t.Fatal("Expected change record")
	}
	if result.Changes[0].Operation != "append" {
		t.Errorf("Operation = %q, want append", result.Changes[0].Operation)
	}

	// Verify array has two elements
	resultDoc := result.Document.(map[string]any)
	servers := resultDoc["servers"].([]any)
	if len(servers) != 2 {
		t.Errorf("len(servers) = %d, want 2", len(servers))
	}
}

// TestInvalidOverlayValidation tests validation errors for invalid overlays.
func TestInvalidOverlayValidation(t *testing.T) {
	doc := map[string]any{"info": map[string]any{}}

	o := &Overlay{
		Version: "1.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Actions: []Action{
			{Target: "invalid[[path", Update: map[string]any{}},
		},
	}

	spec := &parser.ParseResult{
		Document:     doc,
		SourceFormat: parser.SourceFormatYAML,
	}

	a := NewApplier()
	a.StrictTargets = false

	// ApplyParsed validates first, so invalid JSONPath should cause validation error
	_, err := a.ApplyParsed(spec, o)
	if err == nil {
		t.Error("Expected validation error for invalid JSONPath")
	}
}

// TestDeepCopyError tests deepCopy with non-serializable input.
func TestDeepCopyPreservesStructure(t *testing.T) {
	doc := map[string]any{
		"nested": map[string]any{
			"array": []any{1, 2, 3},
			"map":   map[string]any{"key": "value"},
		},
	}

	o := &Overlay{
		Version: "1.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Actions: []Action{
			{Target: "$.nested", Update: map[string]any{"added": true}},
		},
	}

	spec := &parser.ParseResult{
		Document:     doc,
		SourceFormat: parser.SourceFormatYAML,
	}

	a := NewApplier()
	result, err := a.ApplyParsed(spec, o)
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}

	// Original should be unchanged
	originalNested := doc["nested"].(map[string]any)
	if _, exists := originalNested["added"]; exists {
		t.Error("Original document was modified")
	}

	// Result should have the new field
	resultDoc := result.Document.(map[string]any)
	resultNested := resultDoc["nested"].(map[string]any)
	if _, exists := resultNested["added"]; !exists {
		t.Error("Result should have added field")
	}
}

// TestRemovePrecedence tests that remove takes precedence over update.
func TestRemovePrecedence(t *testing.T) {
	doc := map[string]any{
		"info": map[string]any{
			"title":   "Test",
			"x-temp":  "to-remove",
			"version": "1.0.0",
		},
	}

	o := &Overlay{
		Version: "1.0.0",
		Info:    Info{Title: "Test", Version: "1.0.0"},
		Actions: []Action{
			{
				Target: "$.info.x-temp",
				Update: map[string]any{"should": "not-apply"},
				Remove: true,
			},
		},
	}

	spec := &parser.ParseResult{
		Document:     doc,
		SourceFormat: parser.SourceFormatYAML,
	}

	a := NewApplier()
	result, err := a.ApplyParsed(spec, o)
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}

	// The field should be removed, not updated
	resultDoc := result.Document.(map[string]any)
	info := resultDoc["info"].(map[string]any)
	if _, exists := info["x-temp"]; exists {
		t.Error("x-temp should have been removed")
	}

	if result.Changes[0].Operation != "remove" {
		t.Errorf("Operation = %q, want remove", result.Changes[0].Operation)
	}
}

// TestDryRun tests the dry-run preview functionality.
func TestDryRun(t *testing.T) {
	t.Run("preview update action", func(t *testing.T) {
		doc := map[string]any{
			"info": map[string]any{
				"title":   "Original",
				"version": "1.0.0",
			},
		}

		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{
				{
					Target:      "$.info",
					Description: "Update info metadata",
					Update:      map[string]any{"title": "Updated"},
				},
			},
		}

		spec := &parser.ParseResult{
			Document:     doc,
			SourceFormat: parser.SourceFormatYAML,
		}

		a := NewApplier()
		result, err := a.DryRun(spec, o)
		if err != nil {
			t.Fatalf("DryRun error: %v", err)
		}

		if result.WouldApply != 1 {
			t.Errorf("WouldApply = %d, want 1", result.WouldApply)
		}
		if result.WouldSkip != 0 {
			t.Errorf("WouldSkip = %d, want 0", result.WouldSkip)
		}
		if len(result.Changes) != 1 {
			t.Fatalf("len(Changes) = %d, want 1", len(result.Changes))
		}
		if result.Changes[0].Operation != "update" {
			t.Errorf("Operation = %q, want update", result.Changes[0].Operation)
		}
		if result.Changes[0].MatchCount != 1 {
			t.Errorf("MatchCount = %d, want 1", result.Changes[0].MatchCount)
		}
		if result.Changes[0].Description != "Update info metadata" {
			t.Errorf("Description = %q, want 'Update info metadata'", result.Changes[0].Description)
		}

		// Verify original document was not modified
		origInfo := doc["info"].(map[string]any)
		if origInfo["title"] != "Original" {
			t.Error("Original document should not be modified by DryRun")
		}
	})

	t.Run("preview remove action", func(t *testing.T) {
		doc := map[string]any{
			"paths": map[string]any{
				"/internal": map[string]any{"x-internal": true},
				"/public":   map[string]any{"x-internal": false},
			},
		}

		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{
				{
					Target: "$.paths[?@.x-internal==true]",
					Remove: true,
				},
			},
		}

		spec := &parser.ParseResult{
			Document:     doc,
			SourceFormat: parser.SourceFormatYAML,
		}

		a := NewApplier()
		result, err := a.DryRun(spec, o)
		if err != nil {
			t.Fatalf("DryRun error: %v", err)
		}

		if result.Changes[0].Operation != "remove" {
			t.Errorf("Operation = %q, want remove", result.Changes[0].Operation)
		}
		if result.Changes[0].MatchCount != 1 {
			t.Errorf("MatchCount = %d, want 1", result.Changes[0].MatchCount)
		}

		// Verify original document was not modified
		paths := doc["paths"].(map[string]any)
		if _, exists := paths["/internal"]; !exists {
			t.Error("Original document should not be modified by DryRun")
		}
	})

	t.Run("preview no matches", func(t *testing.T) {
		doc := map[string]any{
			"info": map[string]any{"title": "Test"},
		}

		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{
				{
					Target: "$.nonexistent",
					Update: map[string]any{"foo": "bar"},
				},
			},
		}

		spec := &parser.ParseResult{
			Document:     doc,
			SourceFormat: parser.SourceFormatYAML,
		}

		a := NewApplier()
		result, err := a.DryRun(spec, o)
		if err != nil {
			t.Fatalf("DryRun error: %v", err)
		}

		if result.WouldApply != 0 {
			t.Errorf("WouldApply = %d, want 0", result.WouldApply)
		}
		if result.WouldSkip != 1 {
			t.Errorf("WouldSkip = %d, want 1", result.WouldSkip)
		}
		if len(result.Warnings) != 1 {
			t.Errorf("len(Warnings) = %d, want 1", len(result.Warnings))
		}
	})

	t.Run("preview append to array", func(t *testing.T) {
		doc := map[string]any{
			"servers": []any{
				map[string]any{"url": "https://api.example.com"},
			},
		}

		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{
				{
					Target: "$.servers",
					Update: map[string]any{"url": "https://staging.example.com"},
				},
			},
		}

		spec := &parser.ParseResult{
			Document:     doc,
			SourceFormat: parser.SourceFormatYAML,
		}

		a := NewApplier()
		result, err := a.DryRun(spec, o)
		if err != nil {
			t.Fatalf("DryRun error: %v", err)
		}

		if result.Changes[0].Operation != "append" {
			t.Errorf("Operation = %q, want append", result.Changes[0].Operation)
		}
	})
}

// TestDryRunWithOptions tests the functional options API for dry-run.
func TestDryRunWithOptions(t *testing.T) {
	result, err := DryRunWithOptions(
		WithSpecFilePath("../testdata/overlay/fixtures/petstore-base.yaml"),
		WithOverlayFilePath("../testdata/overlay/fixtures/petstore-overlay.yaml"),
	)
	if err != nil {
		t.Fatalf("DryRunWithOptions error: %v", err)
	}

	if result.WouldApply != 3 {
		t.Errorf("WouldApply = %d, want 3", result.WouldApply)
	}
	if !result.HasChanges() {
		t.Error("HasChanges() should return true")
	}
}

// TestDryRunResultHelpers tests the helper methods on DryRunResult.
func TestDryRunResultHelpers(t *testing.T) {
	t.Run("HasChanges", func(t *testing.T) {
		r := &DryRunResult{WouldApply: 0}
		if r.HasChanges() {
			t.Error("HasChanges() should return false when WouldApply is 0")
		}

		r.WouldApply = 1
		if !r.HasChanges() {
			t.Error("HasChanges() should return true when WouldApply > 0")
		}
	})

	t.Run("HasWarnings", func(t *testing.T) {
		r := &DryRunResult{}
		if r.HasWarnings() {
			t.Error("HasWarnings() should return false when Warnings is empty")
		}

		r.Warnings = []string{"warning"}
		if !r.HasWarnings() {
			t.Error("HasWarnings() should return true when Warnings is not empty")
		}
	})
}
