package joiner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/erraggy/oastools/overlay"
	"github.com/erraggy/oastools/parser"
)

func TestWithPreJoinOverlay(t *testing.T) {
	// Create test fixtures
	dir := t.TempDir()

	// Create two simple specs to join
	spec1 := `openapi: "3.0.3"
info:
  title: API 1
  version: "1.0.0"
paths:
  /users:
    get:
      summary: List users
      responses:
        "200":
          description: OK
`
	spec2 := `openapi: "3.0.3"
info:
  title: API 2
  version: "1.0.0"
paths:
  /orders:
    get:
      summary: List orders
      responses:
        "200":
          description: OK
`

	spec1Path := filepath.Join(dir, "spec1.yaml")
	spec2Path := filepath.Join(dir, "spec2.yaml")
	if err := os.WriteFile(spec1Path, []byte(spec1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spec2Path, []byte(spec2), 0600); err != nil {
		t.Fatal(err)
	}

	// Create overlay that adds a tag to info
	preOverlay := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Pre-join", Version: "1.0"},
		Actions: []overlay.Action{
			{
				Target: "$.info",
				Update: map[string]any{
					"x-transformed": true,
				},
			},
		},
	}

	result, err := JoinWithOptions(
		WithFilePaths(spec1Path, spec2Path),
		WithPreJoinOverlay(preOverlay),
		WithPathStrategy(StrategyAcceptLeft),
	)
	if err != nil {
		t.Fatalf("JoinWithOptions failed: %v", err)
	}

	// Check that the overlay was applied (x-transformed should be in info)
	// Result is a typed *parser.OAS3Document
	oas3Doc, ok := result.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Document is not *parser.OAS3Document")
	}
	if oas3Doc.Info == nil {
		t.Fatal("Info is nil")
	}
	if oas3Doc.Info.Extra == nil {
		t.Error("Pre-join overlay was not applied: Extra is nil")
	} else if _, exists := oas3Doc.Info.Extra["x-transformed"]; !exists {
		t.Error("Pre-join overlay was not applied: x-transformed not found in info")
	}
}

func TestWithPostJoinOverlay(t *testing.T) {
	dir := t.TempDir()

	spec1 := `openapi: "3.0.3"
info:
  title: API 1
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
`
	spec2 := `openapi: "3.0.3"
info:
  title: API 2
  version: "1.0.0"
paths:
  /orders:
    get:
      responses:
        "200":
          description: OK
`

	spec1Path := filepath.Join(dir, "spec1.yaml")
	spec2Path := filepath.Join(dir, "spec2.yaml")
	if err := os.WriteFile(spec1Path, []byte(spec1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spec2Path, []byte(spec2), 0600); err != nil {
		t.Fatal(err)
	}

	// Create overlay that updates info after join
	postOverlay := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Post-join", Version: "1.0"},
		Actions: []overlay.Action{
			{
				Target: "$.info",
				Update: map[string]any{
					"title":       "Combined API",
					"description": "APIs joined with overlay",
				},
			},
		},
	}

	result, err := JoinWithOptions(
		WithFilePaths(spec1Path, spec2Path),
		WithPostJoinOverlay(postOverlay),
		WithPathStrategy(StrategyAcceptLeft),
	)
	if err != nil {
		t.Fatalf("JoinWithOptions failed: %v", err)
	}

	// Check that the post-join overlay was applied
	doc, ok := result.Document.(map[string]any)
	if !ok {
		t.Fatal("Document is not a map")
	}
	info, ok := doc["info"].(map[string]any)
	if !ok {
		t.Fatal("info is not a map")
	}
	if info["title"] != "Combined API" {
		t.Errorf("Post-join overlay title not applied: got %v", info["title"])
	}
	if info["description"] != "APIs joined with overlay" {
		t.Errorf("Post-join overlay description not applied: got %v", info["description"])
	}
}

func TestWithOverlayFiles(t *testing.T) {
	dir := t.TempDir()

	// Create specs
	spec1 := `openapi: "3.0.3"
info:
  title: API 1
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
`
	spec2 := `openapi: "3.0.3"
info:
  title: API 2
  version: "1.0.0"
paths:
  /orders:
    get:
      responses:
        "200":
          description: OK
`

	spec1Path := filepath.Join(dir, "spec1.yaml")
	spec2Path := filepath.Join(dir, "spec2.yaml")
	if err := os.WriteFile(spec1Path, []byte(spec1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spec2Path, []byte(spec2), 0600); err != nil {
		t.Fatal(err)
	}

	// Create overlay files
	preOverlayContent := `overlay: "1.0.0"
info:
  title: Pre-join Overlay
  version: "1.0"
actions:
  - target: $.info
    update:
      x-pre-join: true
`
	postOverlayContent := `overlay: "1.0.0"
info:
  title: Post-join Overlay
  version: "1.0"
actions:
  - target: $.info
    update:
      x-post-join: true
`

	preOverlayPath := filepath.Join(dir, "pre-overlay.yaml")
	postOverlayPath := filepath.Join(dir, "post-overlay.yaml")
	if err := os.WriteFile(preOverlayPath, []byte(preOverlayContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(postOverlayPath, []byte(postOverlayContent), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := JoinWithOptions(
		WithFilePaths(spec1Path, spec2Path),
		WithPreJoinOverlayFile(preOverlayPath),
		WithPostJoinOverlayFile(postOverlayPath),
		WithPathStrategy(StrategyAcceptLeft),
	)
	if err != nil {
		t.Fatalf("JoinWithOptions failed: %v", err)
	}

	doc, ok := result.Document.(map[string]any)
	if !ok {
		t.Fatal("Document is not a map")
	}
	info, ok := doc["info"].(map[string]any)
	if !ok {
		t.Fatal("info is not a map")
	}

	// Both overlays should be applied
	if _, exists := info["x-pre-join"]; !exists {
		t.Error("Pre-join overlay file was not applied")
	}
	if _, exists := info["x-post-join"]; !exists {
		t.Error("Post-join overlay file was not applied")
	}
}

func TestWithSpecOverlay(t *testing.T) {
	dir := t.TempDir()

	// Create specs with different content
	spec1 := `openapi: "3.0.3"
info:
  title: Users API
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
`
	spec2 := `openapi: "3.0.3"
info:
  title: Orders API
  version: "1.0.0"
paths:
  /orders:
    get:
      responses:
        "200":
          description: OK
`

	spec1Path := filepath.Join(dir, "users-api.yaml")
	spec2Path := filepath.Join(dir, "orders-api.yaml")
	if err := os.WriteFile(spec1Path, []byte(spec1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spec2Path, []byte(spec2), 0600); err != nil {
		t.Fatal(err)
	}

	// Create spec-specific overlay for users API only
	// Use bracket notation for paths with slashes
	usersOverlay := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Users Overlay", Version: "1.0"},
		Actions: []overlay.Action{
			{
				Target: "$.paths['/users'].get",
				Update: map[string]any{
					"x-users-only": true,
				},
			},
		},
	}

	result, err := JoinWithOptions(
		WithFilePaths(spec1Path, spec2Path),
		WithSpecOverlay(spec1Path, usersOverlay),
		WithPathStrategy(StrategyAcceptLeft),
	)
	if err != nil {
		t.Fatalf("JoinWithOptions failed: %v", err)
	}

	// Result is a typed *parser.OAS3Document
	oas3Doc, ok := result.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Document is not *parser.OAS3Document")
	}

	// Check users endpoint has the extension
	usersPath, ok := oas3Doc.Paths["/users"]
	if !ok {
		t.Fatal("/users path not found")
	}
	if usersPath.Get == nil {
		t.Fatal("GET operation not found on /users")
	}
	if usersPath.Get.Extra == nil {
		t.Error("Spec-specific overlay was not applied: Extra is nil")
	} else if _, exists := usersPath.Get.Extra["x-users-only"]; !exists {
		t.Error("Spec-specific overlay was not applied to users API")
	}

	// Orders endpoint should NOT have the extension
	ordersPath, ok := oas3Doc.Paths["/orders"]
	if !ok {
		t.Fatal("/orders path not found")
	}
	if ordersPath.Get == nil {
		t.Fatal("GET operation not found on /orders")
	}
	if ordersPath.Get.Extra != nil {
		if _, exists := ordersPath.Get.Extra["x-users-only"]; exists {
			t.Error("Spec-specific overlay should NOT be applied to orders API")
		}
	}
}

func TestWithNilPreJoinOverlay(t *testing.T) {
	dir := t.TempDir()

	spec1 := `openapi: "3.0.3"
info:
  title: API 1
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
`
	spec2 := `openapi: "3.0.3"
info:
  title: API 2
  version: "1.0.0"
paths:
  /orders:
    get:
      responses:
        "200":
          description: OK
`

	spec1Path := filepath.Join(dir, "spec1.yaml")
	spec2Path := filepath.Join(dir, "spec2.yaml")
	if err := os.WriteFile(spec1Path, []byte(spec1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spec2Path, []byte(spec2), 0600); err != nil {
		t.Fatal(err)
	}

	// Nil overlay should be ignored without error
	result, err := JoinWithOptions(
		WithFilePaths(spec1Path, spec2Path),
		WithPreJoinOverlay(nil),
		WithPathStrategy(StrategyAcceptLeft),
	)
	if err != nil {
		t.Fatalf("JoinWithOptions with nil overlay failed: %v", err)
	}
	if result == nil {
		t.Fatal("Result should not be nil")
	}
}

func TestJoinWithParsedAndOverlay(t *testing.T) {
	dir := t.TempDir()

	spec1 := `openapi: "3.0.3"
info:
  title: API 1
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
`
	spec1Path := filepath.Join(dir, "spec1.yaml")
	if err := os.WriteFile(spec1Path, []byte(spec1), 0600); err != nil {
		t.Fatal(err)
	}

	// Parse spec1 first
	p := parser.New()
	parsed1, err := p.Parse(spec1Path)
	if err != nil {
		t.Fatalf("Failed to parse spec1: %v", err)
	}

	// Spec2 as file path
	spec2 := `openapi: "3.0.3"
info:
  title: API 2
  version: "1.0.0"
paths:
  /orders:
    get:
      responses:
        "200":
          description: OK
`
	spec2Path := filepath.Join(dir, "spec2.yaml")
	if err := os.WriteFile(spec2Path, []byte(spec2), 0600); err != nil {
		t.Fatal(err)
	}

	// Overlay for the parsed doc (using index "0")
	parsedOverlay := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Parsed Overlay", Version: "1.0"},
		Actions: []overlay.Action{
			{
				Target: "$.info",
				Update: map[string]any{
					"x-from-parsed": true,
				},
			},
		},
	}

	result, err := JoinWithOptions(
		WithParsed(*parsed1),
		WithFilePaths(spec2Path),
		WithSpecOverlay("0", parsedOverlay), // Index 0 for pre-parsed doc
		WithPathStrategy(StrategyAcceptLeft),
	)
	if err != nil {
		t.Fatalf("JoinWithOptions failed: %v", err)
	}

	// Result is a typed *parser.OAS3Document
	oas3Doc, ok := result.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Document is not *parser.OAS3Document")
	}
	if oas3Doc.Info == nil {
		t.Fatal("Info is nil")
	}
	if oas3Doc.Info.Extra == nil {
		t.Error("Spec overlay for parsed doc was not applied: Extra is nil")
	} else if _, exists := oas3Doc.Info.Extra["x-from-parsed"]; !exists {
		t.Error("Spec overlay for parsed doc was not applied")
	}
}

func TestOverlayFileNotFound(t *testing.T) {
	dir := t.TempDir()

	spec1 := `openapi: "3.0.3"
info:
  title: API 1
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
`
	spec2 := `openapi: "3.0.3"
info:
  title: API 2
  version: "1.0.0"
paths:
  /orders:
    get:
      responses:
        "200":
          description: OK
`

	spec1Path := filepath.Join(dir, "spec1.yaml")
	spec2Path := filepath.Join(dir, "spec2.yaml")
	if err := os.WriteFile(spec1Path, []byte(spec1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spec2Path, []byte(spec2), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := JoinWithOptions(
		WithFilePaths(spec1Path, spec2Path),
		WithPreJoinOverlayFile("/nonexistent/overlay.yaml"),
		WithPathStrategy(StrategyAcceptLeft),
	)
	if err == nil {
		t.Error("Expected error for nonexistent overlay file")
	}
}

func TestMultiplePreJoinOverlays(t *testing.T) {
	dir := t.TempDir()

	spec1 := `openapi: "3.0.3"
info:
  title: API 1
  version: "1.0.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: OK
`
	spec2 := `openapi: "3.0.3"
info:
  title: API 2
  version: "1.0.0"
paths:
  /orders:
    get:
      responses:
        "200":
          description: OK
`

	spec1Path := filepath.Join(dir, "spec1.yaml")
	spec2Path := filepath.Join(dir, "spec2.yaml")
	if err := os.WriteFile(spec1Path, []byte(spec1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spec2Path, []byte(spec2), 0600); err != nil {
		t.Fatal(err)
	}

	// Create two pre-join overlays
	overlay1 := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Overlay 1", Version: "1.0"},
		Actions: []overlay.Action{
			{
				Target: "$.info",
				Update: map[string]any{
					"x-overlay-1": true,
				},
			},
		},
	}
	overlay2 := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Overlay 2", Version: "1.0"},
		Actions: []overlay.Action{
			{
				Target: "$.info",
				Update: map[string]any{
					"x-overlay-2": true,
				},
			},
		},
	}

	result, err := JoinWithOptions(
		WithFilePaths(spec1Path, spec2Path),
		WithPreJoinOverlay(overlay1),
		WithPreJoinOverlay(overlay2),
		WithPathStrategy(StrategyAcceptLeft),
	)
	if err != nil {
		t.Fatalf("JoinWithOptions failed: %v", err)
	}

	// Result is a typed *parser.OAS3Document
	oas3Doc, ok := result.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Document is not *parser.OAS3Document")
	}
	if oas3Doc.Info == nil {
		t.Fatal("Info is nil")
	}
	if oas3Doc.Info.Extra == nil {
		t.Fatal("Extra is nil - overlays not applied")
	}

	// Both overlays should be applied
	if _, exists := oas3Doc.Info.Extra["x-overlay-1"]; !exists {
		t.Error("First pre-join overlay was not applied")
	}
	if _, exists := oas3Doc.Info.Extra["x-overlay-2"]; !exists {
		t.Error("Second pre-join overlay was not applied")
	}
}
