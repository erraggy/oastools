package overlay

import (
	"errors"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		require.NoError(t, err)

		assert.Equal(t, "1.0.0", o.Version)
		assert.Equal(t, "Test Overlay", o.Info.Title)
		assert.Len(t, o.Actions, 1)
	})

	t.Run("valid JSON overlay", func(t *testing.T) {
		data := []byte(`{
			"overlay": "1.0.0",
			"info": {"title": "JSON Overlay", "version": "1.0.0"},
			"actions": [{"target": "$.info", "update": {"x-test": true}}]
		}`)
		o, err := ParseOverlay(data)
		require.NoError(t, err)

		assert.Equal(t, "JSON Overlay", o.Info.Title)
	})

	t.Run("invalid YAML", func(t *testing.T) {
		data := []byte(`overlay: [invalid`)
		_, err := ParseOverlay(data)
		assert.Error(t, err)
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
		require.NoError(t, err)

		assert.Equal(t, "./openapi.yaml", o.Extends)
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
		assert.Empty(t, errs)
	})

	t.Run("missing version", func(t *testing.T) {
		o := &Overlay{
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{{Target: "$.info", Update: map[string]any{}}},
		}

		errs := Validate(o)
		assert.NotEmpty(t, errs)
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
		assert.True(t, found, "Expected error for unsupported version")
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
		assert.True(t, found, "Expected error for missing info.title")
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
		assert.True(t, found, "Expected error for missing info.version")
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
		assert.True(t, found, "Expected error for empty actions")
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
		assert.True(t, found, "Expected error for missing target")
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
		assert.True(t, found, "Expected error for invalid JSONPath")
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
		assert.True(t, found, "Expected error for action without update or remove")
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

	assert.True(t, IsValid(validOverlay), "Expected valid overlay to be valid")

	assert.False(t, IsValid(invalidOverlay), "Expected invalid overlay to be invalid")
}

// TestApplyResult tests ApplyResult helper methods.
func TestApplyResult(t *testing.T) {
	t.Run("HasChanges", func(t *testing.T) {
		r := &ApplyResult{ActionsApplied: 0}
		assert.False(t, r.HasChanges(), "Expected no changes")

		r.ActionsApplied = 1
		assert.True(t, r.HasChanges(), "Expected changes")
	})

	t.Run("HasWarnings", func(t *testing.T) {
		r := &ApplyResult{}
		assert.False(t, r.HasWarnings(), "Expected no warnings")

		r.Warnings = []string{"warning"}
		assert.True(t, r.HasWarnings(), "Expected warnings")
	})

	t.Run("WarningStrings prefers StructuredWarnings", func(t *testing.T) {
		r := &ApplyResult{
			Warnings: []string{"legacy warning"},
			StructuredWarnings: ApplyWarnings{
				NewNoMatchWarning(0, "$.test"),
			},
		}
		warnings := r.WarningStrings()
		assert.Len(t, warnings, 1)
		assert.NotEqual(t, "legacy warning", warnings[0], "Expected StructuredWarnings to take precedence")
	})

	t.Run("WarningStrings falls back to Warnings", func(t *testing.T) {
		r := &ApplyResult{
			Warnings: []string{"legacy warning"},
		}
		warnings := r.WarningStrings()
		assert.Len(t, warnings, 1)
		assert.Equal(t, "legacy warning", warnings[0])
	})

	t.Run("WarningStrings with empty result", func(t *testing.T) {
		r := &ApplyResult{}
		warnings := r.WarningStrings()
		assert.Nil(t, warnings, "Expected nil for empty result")
	})
}

// TestApplyResult_ToParseResult tests the ToParseResult method.
func TestApplyResult_ToParseResult(t *testing.T) {
	t.Run("OAS3 result converts correctly", func(t *testing.T) {
		applyResult := &ApplyResult{
			Document:       &parser.OAS3Document{OpenAPI: "3.0.3", OASVersion: parser.OASVersion303, Info: &parser.Info{Title: "Test API", Version: "1.0"}},
			SourceFormat:   parser.SourceFormatYAML,
			ActionsApplied: 2,
			Warnings:       []string{"warning1", "warning2"},
		}

		parseResult := applyResult.ToParseResult()

		assert.Equal(t, "overlay", parseResult.SourcePath)
		assert.Equal(t, parser.SourceFormatYAML, parseResult.SourceFormat)
		assert.Equal(t, "3.0.3", parseResult.Version)
		assert.Equal(t, parser.OASVersion303, parseResult.OASVersion)
		assert.NotNil(t, parseResult.Document)
		assert.Empty(t, parseResult.Errors)
		assert.Len(t, parseResult.Warnings, 2)

		// Verify Document type assertion works
		doc, ok := parseResult.Document.(*parser.OAS3Document)
		require.True(t, ok, "Document should be *parser.OAS3Document")
		assert.Equal(t, "Test API", doc.Info.Title)
	})

	t.Run("OAS2 result converts correctly", func(t *testing.T) {
		applyResult := &ApplyResult{
			Document:       &parser.OAS2Document{Swagger: "2.0", OASVersion: parser.OASVersion20, Info: &parser.Info{Title: "Swagger API", Version: "1.0"}},
			SourceFormat:   parser.SourceFormatJSON,
			ActionsApplied: 1,
		}

		parseResult := applyResult.ToParseResult()

		assert.Equal(t, "overlay", parseResult.SourcePath)
		assert.Equal(t, parser.SourceFormatJSON, parseResult.SourceFormat)
		assert.Equal(t, "2.0", parseResult.Version)
		assert.Equal(t, parser.OASVersion20, parseResult.OASVersion)

		// Verify Document type assertion works
		doc, ok := parseResult.Document.(*parser.OAS2Document)
		require.True(t, ok, "Document should be *parser.OAS2Document")
		assert.Equal(t, "Swagger API", doc.Info.Title)
	})

	t.Run("StructuredWarnings converted to strings", func(t *testing.T) {
		applyResult := &ApplyResult{
			Document:     &parser.OAS3Document{OpenAPI: "3.1.0", OASVersion: parser.OASVersion310},
			SourceFormat: parser.SourceFormatYAML,
			StructuredWarnings: ApplyWarnings{
				NewNoMatchWarning(0, "$.nonexistent"),
				NewNoMatchWarning(1, "$.missing"),
			},
		}

		parseResult := applyResult.ToParseResult()

		assert.Len(t, parseResult.Warnings, 2)
	})

	t.Run("empty warnings", func(t *testing.T) {
		applyResult := &ApplyResult{
			Document:     &parser.OAS3Document{OpenAPI: "3.0.0", OASVersion: parser.OASVersion300},
			SourceFormat: parser.SourceFormatYAML,
		}

		parseResult := applyResult.ToParseResult()

		assert.Empty(t, parseResult.Warnings)
	})

	t.Run("nil Document returns empty version with warning", func(t *testing.T) {
		applyResult := &ApplyResult{
			Document:     nil,
			SourceFormat: parser.SourceFormatYAML,
		}

		parseResult := applyResult.ToParseResult()

		assert.Equal(t, "", parseResult.Version)
		assert.Equal(t, parser.Unknown, parseResult.OASVersion)
		// Verify warning is added for nil document
		require.Len(t, parseResult.Warnings, 1)
		assert.Equal(t, "overlay: ToParseResult: Document is nil, downstream operations may fail", parseResult.Warnings[0])
	})

	t.Run("map[string]any document returns empty version with warning", func(t *testing.T) {
		// When overlay is applied to a raw map (not a typed document)
		applyResult := &ApplyResult{
			Document: map[string]any{
				"openapi": "3.0.3",
				"info":    map[string]any{"title": "Test"},
			},
			SourceFormat: parser.SourceFormatYAML,
		}

		parseResult := applyResult.ToParseResult()

		// Raw maps don't implement DocumentAccessor, so version is empty
		assert.Equal(t, "", parseResult.Version)
		assert.Equal(t, parser.Unknown, parseResult.OASVersion)
		// Verify warning is added for unrecognized document type
		require.Len(t, parseResult.Warnings, 1)
		assert.Contains(t, parseResult.Warnings[0], "overlay: ToParseResult: unrecognized document type")
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
		require.NoError(t, err)

		assert.Equal(t, 1, result.ActionsApplied)

		resultDoc := result.Document.(map[string]any)
		info := resultDoc["info"].(map[string]any)
		assert.Equal(t, "Updated", info["title"])
		assert.Equal(t, "added", info["x-extra"])
		// Original field should be preserved
		assert.Equal(t, "1.0.0", info["version"])
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
		require.NoError(t, err)

		assert.Equal(t, 1, result.ActionsApplied)

		resultDoc := result.Document.(map[string]any)
		paths := resultDoc["paths"].(map[string]any)
		assert.NotContains(t, paths, "/internal", "Internal path should have been removed")
		assert.Contains(t, paths, "/public", "Public path should still exist")
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
		require.NoError(t, err)

		assert.Equal(t, 3, result.ActionsApplied)

		resultDoc := result.Document.(map[string]any)
		info := resultDoc["info"].(map[string]any)
		assert.Equal(t, "Final", info["title"])
		assert.Equal(t, "2", info["x-step"])
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
		require.NoError(t, err)

		assert.Equal(t, 1, result.ActionsSkipped)
		assert.NotEmpty(t, result.Warnings, "Expected warning for no matches")
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
		assert.Error(t, err, "Expected error in strict mode for no matches")
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
		require.NoError(t, err)

		// Original should be unchanged
		originalInfo := doc["info"].(map[string]any)
		assert.Equal(t, "Original", originalInfo["title"], "Original document was modified")
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
		require.NoError(t, err)

		assert.Equal(t, 1, result.ActionsApplied)
	})

	t.Run("missing spec source", func(t *testing.T) {
		o := &Overlay{
			Version: "1.0.0",
			Info:    Info{Title: "Test", Version: "1.0.0"},
			Actions: []Action{{Target: "$.info", Update: map[string]any{}}},
		}

		_, err := ApplyWithOptions(WithOverlayParsed(o))
		assert.Error(t, err, "Expected error for missing spec source")
	})

	t.Run("missing overlay source", func(t *testing.T) {
		spec := parser.ParseResult{
			Document: map[string]any{},
		}

		_, err := ApplyWithOptions(WithSpecParsed(spec))
		assert.Error(t, err, "Expected error for missing overlay source")
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
		assert.Error(t, err, "Expected error with strict targets")
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
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestErrorTypes tests error type methods.
func TestErrorTypes(t *testing.T) {
	t.Run("ValidationError", func(t *testing.T) {
		e := ValidationError{Field: "test", Message: "error"}
		assert.NotEmpty(t, e.Error())

		e2 := ValidationError{Path: "actions[0]", Message: "error"}
		assert.NotEmpty(t, e2.Error())
	})

	t.Run("ApplyError", func(t *testing.T) {
		e := &ApplyError{ActionIndex: 0, Target: "$.info", Cause: nil}
		assert.NotEmpty(t, e.Error())
	})

	t.Run("ParseError", func(t *testing.T) {
		e := &ParseError{Path: "test.yaml", Cause: nil}
		assert.NotEmpty(t, e.Error())
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
	require.NoError(t, err)

	// Parse it back
	o2, err := ParseOverlay(data)
	require.NoError(t, err)

	assert.Equal(t, o.Version, o2.Version)
	assert.Equal(t, o.Info.Title, o2.Info.Title)
}

// TestParseOverlayFile tests file-based overlay parsing.
func TestParseOverlayFile(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		o, err := ParseOverlayFile("../testdata/overlay/valid/basic-update.yaml")
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", o.Version)
		assert.Len(t, o.Actions, 1)
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := ParseOverlayFile("nonexistent.yaml")
		assert.Error(t, err)
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
		require.NoError(t, err)

		assert.Equal(t, 3, result.ActionsApplied)
	})

	t.Run("missing spec file", func(t *testing.T) {
		a := NewApplier()
		_, err := a.Apply("nonexistent.yaml", "../testdata/overlay/valid/basic-update.yaml")
		assert.Error(t, err)
	})

	t.Run("missing overlay file", func(t *testing.T) {
		a := NewApplier()
		_, err := a.Apply("../testdata/overlay/fixtures/petstore-base.yaml", "nonexistent.yaml")
		assert.Error(t, err)
	})
}

// TestApplyWithOptionsFilePaths tests file path options.
func TestApplyWithOptionsFilePaths(t *testing.T) {
	t.Run("with file paths", func(t *testing.T) {
		result, err := ApplyWithOptions(
			WithSpecFilePath("../testdata/overlay/fixtures/petstore-base.yaml"),
			WithOverlayFilePath("../testdata/overlay/fixtures/petstore-overlay.yaml"),
		)
		require.NoError(t, err)

		assert.Equal(t, 3, result.ActionsApplied)
	})

	t.Run("empty spec path", func(t *testing.T) {
		_, err := ApplyWithOptions(
			WithSpecFilePath(""),
			WithOverlayFilePath("../testdata/overlay/valid/basic-update.yaml"),
		)
		assert.Error(t, err)
	})

	t.Run("empty overlay path", func(t *testing.T) {
		_, err := ApplyWithOptions(
			WithSpecFilePath("../testdata/overlay/fixtures/petstore-base.yaml"),
			WithOverlayFilePath(""),
		)
		assert.Error(t, err)
	})

	t.Run("nil overlay", func(t *testing.T) {
		_, err := ApplyWithOptions(
			WithSpecFilePath("../testdata/overlay/fixtures/petstore-base.yaml"),
			WithOverlayParsed(nil),
		)
		assert.Error(t, err)
	})
}

// TestErrorUnwrap tests error Unwrap methods.
func TestErrorUnwrap(t *testing.T) {
	t.Run("ApplyError Unwrap", func(t *testing.T) {
		cause := &ParseError{Path: "test"}
		e := &ApplyError{Cause: cause}
		unwrapped := e.Unwrap()
		assert.NotNil(t, unwrapped, "Unwrap should return non-nil cause")
		// Verify errors.As works with ApplyError
		var applyErr *ApplyError
		assert.True(t, errors.As(e, &applyErr), "errors.As should find ApplyError")
	})

	t.Run("ParseError Unwrap", func(t *testing.T) {
		cause := &ValidationError{Message: "test"}
		e := &ParseError{Cause: cause}
		unwrapped := e.Unwrap()
		assert.NotNil(t, unwrapped, "Unwrap should return non-nil cause")
		// Verify errors.As works with ParseError
		var parseErr *ParseError
		assert.True(t, errors.As(e, &parseErr), "errors.As should find ParseError")
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
	require.NoError(t, err)

	assert.Equal(t, 1, result.ActionsApplied)

	// Check the change record
	require.NotEmpty(t, result.Changes, "Expected change record")
	assert.Equal(t, "replace", result.Changes[0].Operation)
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
	require.NoError(t, err)

	// Check the change record
	require.NotEmpty(t, result.Changes, "Expected change record")
	assert.Equal(t, "append", result.Changes[0].Operation)

	// Verify array has two elements
	resultDoc := result.Document.(map[string]any)
	servers := resultDoc["servers"].([]any)
	assert.Len(t, servers, 2)
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
	assert.Error(t, err, "Expected validation error for invalid JSONPath")
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
	require.NoError(t, err)

	// Original should be unchanged
	originalNested := doc["nested"].(map[string]any)
	assert.NotContains(t, originalNested, "added", "Original document was modified")

	// Result should have the new field
	resultDoc := result.Document.(map[string]any)
	resultNested := resultDoc["nested"].(map[string]any)
	assert.Contains(t, resultNested, "added", "Result should have added field")
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
	require.NoError(t, err)

	// The field should be removed, not updated
	resultDoc := result.Document.(map[string]any)
	info := resultDoc["info"].(map[string]any)
	assert.NotContains(t, info, "x-temp", "x-temp should have been removed")

	assert.Equal(t, "remove", result.Changes[0].Operation)
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
		require.NoError(t, err)

		assert.Equal(t, 1, result.WouldApply)
		assert.Equal(t, 0, result.WouldSkip)
		require.Len(t, result.Changes, 1)
		assert.Equal(t, "update", result.Changes[0].Operation)
		assert.Equal(t, 1, result.Changes[0].MatchCount)
		assert.Equal(t, "Update info metadata", result.Changes[0].Description)

		// Verify original document was not modified
		origInfo := doc["info"].(map[string]any)
		assert.Equal(t, "Original", origInfo["title"], "Original document should not be modified by DryRun")
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
		require.NoError(t, err)

		assert.Equal(t, "remove", result.Changes[0].Operation)
		assert.Equal(t, 1, result.Changes[0].MatchCount)

		// Verify original document was not modified
		paths := doc["paths"].(map[string]any)
		assert.Contains(t, paths, "/internal", "Original document should not be modified by DryRun")
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
		require.NoError(t, err)

		assert.Equal(t, 0, result.WouldApply)
		assert.Equal(t, 1, result.WouldSkip)
		assert.Len(t, result.Warnings, 1)
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
		require.NoError(t, err)

		assert.Equal(t, "append", result.Changes[0].Operation)
	})
}

// TestDryRunWithOptions tests the functional options API for dry-run.
func TestDryRunWithOptions(t *testing.T) {
	result, err := DryRunWithOptions(
		WithSpecFilePath("../testdata/overlay/fixtures/petstore-base.yaml"),
		WithOverlayFilePath("../testdata/overlay/fixtures/petstore-overlay.yaml"),
	)
	require.NoError(t, err)

	assert.Equal(t, 3, result.WouldApply)
	assert.True(t, result.HasChanges(), "HasChanges() should return true")
}

// TestDryRunResultHelpers tests the helper methods on DryRunResult.
func TestDryRunResultHelpers(t *testing.T) {
	t.Run("HasChanges", func(t *testing.T) {
		r := &DryRunResult{WouldApply: 0}
		assert.False(t, r.HasChanges(), "HasChanges() should return false when WouldApply is 0")

		r.WouldApply = 1
		assert.True(t, r.HasChanges(), "HasChanges() should return true when WouldApply > 0")
	})

	t.Run("HasWarnings", func(t *testing.T) {
		r := &DryRunResult{}
		assert.False(t, r.HasWarnings(), "HasWarnings() should return false when Warnings is empty")

		r.Warnings = []string{"warning"}
		assert.True(t, r.HasWarnings(), "HasWarnings() should return true when Warnings is not empty")
	})
}
