package converter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/erraggy/oastools/overlay"
	"github.com/erraggy/oastools/parser"
)

func TestWithPreConversionOverlay(t *testing.T) {
	dir := t.TempDir()

	// Create a Swagger 2.0 spec
	spec := `swagger: "2.0"
info:
  title: Test API
  version: "1.0.0"
basePath: /api
paths:
  /users:
    get:
      produces:
        - application/json
      responses:
        "200":
          description: OK
`
	specPath := filepath.Join(dir, "swagger.yaml")
	if err := os.WriteFile(specPath, []byte(spec), 0600); err != nil {
		t.Fatal(err)
	}

	// Create overlay that adds an extension before conversion
	preOverlay := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Pre-conversion", Version: "1.0"},
		Actions: []overlay.Action{
			{
				Target: "$.info",
				Update: map[string]any{
					"x-pre-conversion": true,
				},
			},
		},
	}

	result, err := ConvertWithOptions(
		WithFilePath(specPath),
		WithTargetVersion("3.0.3"),
		WithPreConversionOverlay(preOverlay),
	)
	if err != nil {
		t.Fatalf("ConvertWithOptions failed: %v", err)
	}

	// Check that conversion succeeded
	if !result.Success {
		t.Errorf("Conversion should have succeeded")
	}

	// Check that the overlay was applied (x-pre-conversion should be in info)
	oas3Doc, ok := result.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Document is not *parser.OAS3Document")
	}
	if oas3Doc.Info == nil {
		t.Fatal("Info is nil")
	}
	if oas3Doc.Info.Extra == nil {
		t.Error("Pre-conversion overlay was not applied: Extra is nil")
	} else if _, exists := oas3Doc.Info.Extra["x-pre-conversion"]; !exists {
		t.Error("Pre-conversion overlay was not applied: x-pre-conversion not found")
	}
}

func TestWithPostConversionOverlay(t *testing.T) {
	dir := t.TempDir()

	// Create a Swagger 2.0 spec
	spec := `swagger: "2.0"
info:
  title: Test API
  version: "1.0.0"
basePath: /api
paths:
  /users:
    get:
      produces:
        - application/json
      responses:
        "200":
          description: OK
`
	specPath := filepath.Join(dir, "swagger.yaml")
	if err := os.WriteFile(specPath, []byte(spec), 0600); err != nil {
		t.Fatal(err)
	}

	// Create overlay that updates info after conversion
	postOverlay := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Post-conversion", Version: "1.0"},
		Actions: []overlay.Action{
			{
				Target: "$.info",
				Update: map[string]any{
					"description": "API converted with overlay enhancements",
				},
			},
		},
	}

	result, err := ConvertWithOptions(
		WithFilePath(specPath),
		WithTargetVersion("3.0.3"),
		WithPostConversionOverlay(postOverlay),
	)
	if err != nil {
		t.Fatalf("ConvertWithOptions failed: %v", err)
	}

	// The post-conversion overlay modifies the final result
	// Since the result Document may be typed or map, check for the description
	oas3Doc, ok := result.Document.(*parser.OAS3Document)
	if ok {
		if oas3Doc.Info != nil && oas3Doc.Info.Description == "API converted with overlay enhancements" {
			// Success - typed document was preserved
			return
		}
	}

	// Check if it's a map (overlay returns map[string]any)
	doc, ok := result.Document.(map[string]any)
	if !ok {
		t.Fatal("Document is neither *parser.OAS3Document nor map[string]any")
	}
	info, ok := doc["info"].(map[string]any)
	if !ok {
		t.Fatal("info is not a map")
	}
	if info["description"] != "API converted with overlay enhancements" {
		t.Errorf("Post-conversion overlay description not applied: got %v", info["description"])
	}
}

func TestWithOverlayFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a Swagger 2.0 spec
	spec := `swagger: "2.0"
info:
  title: Test API
  version: "1.0.0"
basePath: /api
paths:
  /users:
    get:
      produces:
        - application/json
      responses:
        "200":
          description: OK
`
	specPath := filepath.Join(dir, "swagger.yaml")
	if err := os.WriteFile(specPath, []byte(spec), 0600); err != nil {
		t.Fatal(err)
	}

	// Create overlay files
	preOverlayContent := `overlay: "1.0.0"
info:
  title: Pre-conversion Overlay
  version: "1.0"
actions:
  - target: $.info
    update:
      x-pre-file: true
`
	postOverlayContent := `overlay: "1.0.0"
info:
  title: Post-conversion Overlay
  version: "1.0"
actions:
  - target: $.info
    update:
      x-post-file: true
`

	preOverlayPath := filepath.Join(dir, "pre-overlay.yaml")
	postOverlayPath := filepath.Join(dir, "post-overlay.yaml")
	if err := os.WriteFile(preOverlayPath, []byte(preOverlayContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(postOverlayPath, []byte(postOverlayContent), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := ConvertWithOptions(
		WithFilePath(specPath),
		WithTargetVersion("3.0.3"),
		WithPreConversionOverlayFile(preOverlayPath),
		WithPostConversionOverlayFile(postOverlayPath),
	)
	if err != nil {
		t.Fatalf("ConvertWithOptions failed: %v", err)
	}

	// Check that the document was converted
	if result.TargetVersion != "3.0.3" {
		t.Errorf("Expected target version 3.0.3, got %s", result.TargetVersion)
	}

	// Check pre-overlay was applied (via typed document if preserved)
	oas3Doc, ok := result.Document.(*parser.OAS3Document)
	if ok && oas3Doc.Info != nil && oas3Doc.Info.Extra != nil {
		if _, exists := oas3Doc.Info.Extra["x-pre-file"]; exists {
			// Success
			return
		}
	}

	// Check via map
	doc, ok := result.Document.(map[string]any)
	if !ok {
		t.Fatal("Document is neither typed nor map")
	}
	info, ok := doc["info"].(map[string]any)
	if !ok {
		t.Fatal("info is not a map")
	}

	// At minimum the post-overlay should be visible
	if _, exists := info["x-post-file"]; !exists {
		t.Error("Post-conversion overlay file was not applied")
	}
}

func TestConversionWithoutOverlay(t *testing.T) {
	dir := t.TempDir()

	// Create a Swagger 2.0 spec
	spec := `swagger: "2.0"
info:
  title: Test API
  version: "1.0.0"
basePath: /api
paths:
  /users:
    get:
      produces:
        - application/json
      responses:
        "200":
          description: OK
`
	specPath := filepath.Join(dir, "swagger.yaml")
	if err := os.WriteFile(specPath, []byte(spec), 0600); err != nil {
		t.Fatal(err)
	}

	// Fast path: no overlays
	result, err := ConvertWithOptions(
		WithFilePath(specPath),
		WithTargetVersion("3.0.3"),
	)
	if err != nil {
		t.Fatalf("ConvertWithOptions failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Conversion should have succeeded")
	}
	if result.TargetVersion != "3.0.3" {
		t.Errorf("Expected target version 3.0.3, got %s", result.TargetVersion)
	}
}

func TestOverlayFileNotFound(t *testing.T) {
	dir := t.TempDir()

	spec := `swagger: "2.0"
info:
  title: Test API
  version: "1.0.0"
basePath: /api
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
`
	specPath := filepath.Join(dir, "swagger.yaml")
	if err := os.WriteFile(specPath, []byte(spec), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := ConvertWithOptions(
		WithFilePath(specPath),
		WithTargetVersion("3.0.3"),
		WithPreConversionOverlayFile("/nonexistent/overlay.yaml"),
	)
	if err == nil {
		t.Error("Expected error for nonexistent overlay file")
	}
}

func TestEmptyOverlayFilePath(t *testing.T) {
	dir := t.TempDir()

	spec := `swagger: "2.0"
info:
  title: Test API
  version: "1.0.0"
basePath: /api
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
`
	specPath := filepath.Join(dir, "swagger.yaml")
	if err := os.WriteFile(specPath, []byte(spec), 0600); err != nil {
		t.Fatal(err)
	}

	// Empty overlay file path should be ignored
	result, err := ConvertWithOptions(
		WithFilePath(specPath),
		WithTargetVersion("3.0.3"),
		WithPreConversionOverlayFile(""),
		WithPostConversionOverlayFile(""),
	)
	if err != nil {
		t.Fatalf("ConvertWithOptions should succeed with empty overlay paths: %v", err)
	}
	if result == nil {
		t.Fatal("Result should not be nil")
	}
}
