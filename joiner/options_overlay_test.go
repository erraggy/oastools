package joiner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/erraggy/oastools/overlay"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, os.WriteFile(spec1Path, []byte(spec1), 0600))
	require.NoError(t, os.WriteFile(spec2Path, []byte(spec2), 0600))

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
	require.NoError(t, err)

	// Check that the overlay was applied (x-transformed should be in info)
	// Result is a typed *parser.OAS3Document
	oas3Doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "Document is not *parser.OAS3Document")
	require.NotNil(t, oas3Doc.Info)
	if assert.NotNil(t, oas3Doc.Info.Extra, "Pre-join overlay was not applied: Extra is nil") {
		assert.Contains(t, oas3Doc.Info.Extra, "x-transformed", "Pre-join overlay was not applied: x-transformed not found in info")
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
	require.NoError(t, os.WriteFile(spec1Path, []byte(spec1), 0600))
	require.NoError(t, os.WriteFile(spec2Path, []byte(spec2), 0600))

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
	require.NoError(t, err)

	// Check that the post-join overlay was applied
	doc, ok := result.Document.(map[string]any)
	require.True(t, ok, "Document is not a map")
	info, ok := doc["info"].(map[string]any)
	require.True(t, ok, "info is not a map")
	assert.Equal(t, "Combined API", info["title"])
	assert.Equal(t, "APIs joined with overlay", info["description"])
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
	require.NoError(t, os.WriteFile(spec1Path, []byte(spec1), 0600))
	require.NoError(t, os.WriteFile(spec2Path, []byte(spec2), 0600))

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
	require.NoError(t, os.WriteFile(preOverlayPath, []byte(preOverlayContent), 0600))
	require.NoError(t, os.WriteFile(postOverlayPath, []byte(postOverlayContent), 0600))

	result, err := JoinWithOptions(
		WithFilePaths(spec1Path, spec2Path),
		WithPreJoinOverlayFile(preOverlayPath),
		WithPostJoinOverlayFile(postOverlayPath),
		WithPathStrategy(StrategyAcceptLeft),
	)
	require.NoError(t, err)

	doc, ok := result.Document.(map[string]any)
	require.True(t, ok, "Document is not a map")
	info, ok := doc["info"].(map[string]any)
	require.True(t, ok, "info is not a map")

	// Both overlays should be applied
	assert.Contains(t, info, "x-pre-join", "Pre-join overlay file was not applied")
	assert.Contains(t, info, "x-post-join", "Post-join overlay file was not applied")
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
	require.NoError(t, os.WriteFile(spec1Path, []byte(spec1), 0600))
	require.NoError(t, os.WriteFile(spec2Path, []byte(spec2), 0600))

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
	require.NoError(t, err)

	// Result is a typed *parser.OAS3Document
	oas3Doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "Document is not *parser.OAS3Document")

	// Check users endpoint has the extension
	usersPath, ok := oas3Doc.Paths["/users"]
	require.True(t, ok, "/users path not found")
	require.NotNil(t, usersPath.Get, "GET operation not found on /users")
	if assert.NotNil(t, usersPath.Get.Extra, "Spec-specific overlay was not applied: Extra is nil") {
		assert.Contains(t, usersPath.Get.Extra, "x-users-only", "Spec-specific overlay was not applied to users API")
	}

	// Orders endpoint should NOT have the extension
	ordersPath, ok := oas3Doc.Paths["/orders"]
	require.True(t, ok, "/orders path not found")
	require.NotNil(t, ordersPath.Get, "GET operation not found on /orders")
	if ordersPath.Get.Extra != nil {
		assert.NotContains(t, ordersPath.Get.Extra, "x-users-only", "Spec-specific overlay should NOT be applied to orders API")
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
	require.NoError(t, os.WriteFile(spec1Path, []byte(spec1), 0600))
	require.NoError(t, os.WriteFile(spec2Path, []byte(spec2), 0600))

	// Nil overlay should be ignored without error
	result, err := JoinWithOptions(
		WithFilePaths(spec1Path, spec2Path),
		WithPreJoinOverlay(nil),
		WithPathStrategy(StrategyAcceptLeft),
	)
	require.NoError(t, err)
	require.NotNil(t, result, "Result should not be nil")
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
	require.NoError(t, os.WriteFile(spec1Path, []byte(spec1), 0600))

	// Parse spec1 first
	p := parser.New()
	parsed1, err := p.Parse(spec1Path)
	require.NoError(t, err, "Failed to parse spec1")

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
	require.NoError(t, os.WriteFile(spec2Path, []byte(spec2), 0600))

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
	require.NoError(t, err)

	// Result is a typed *parser.OAS3Document
	oas3Doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "Document is not *parser.OAS3Document")
	require.NotNil(t, oas3Doc.Info)
	if assert.NotNil(t, oas3Doc.Info.Extra, "Spec overlay for parsed doc was not applied: Extra is nil") {
		assert.Contains(t, oas3Doc.Info.Extra, "x-from-parsed", "Spec overlay for parsed doc was not applied")
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
	require.NoError(t, os.WriteFile(spec1Path, []byte(spec1), 0600))
	require.NoError(t, os.WriteFile(spec2Path, []byte(spec2), 0600))

	_, err := JoinWithOptions(
		WithFilePaths(spec1Path, spec2Path),
		WithPreJoinOverlayFile("/nonexistent/overlay.yaml"),
		WithPathStrategy(StrategyAcceptLeft),
	)
	assert.Error(t, err, "Expected error for nonexistent overlay file")
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
	require.NoError(t, os.WriteFile(spec1Path, []byte(spec1), 0600))
	require.NoError(t, os.WriteFile(spec2Path, []byte(spec2), 0600))

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
	require.NoError(t, err)

	// Result is a typed *parser.OAS3Document
	oas3Doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "Document is not *parser.OAS3Document")
	require.NotNil(t, oas3Doc.Info)
	require.NotNil(t, oas3Doc.Info.Extra, "Extra is nil - overlays not applied")

	// Both overlays should be applied
	assert.Contains(t, oas3Doc.Info.Extra, "x-overlay-1", "First pre-join overlay was not applied")
	assert.Contains(t, oas3Doc.Info.Extra, "x-overlay-2", "Second pre-join overlay was not applied")
}
