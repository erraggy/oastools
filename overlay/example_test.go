package overlay_test

import (
	"fmt"
	"log"

	"github.com/erraggy/oastools/overlay"
	"github.com/erraggy/oastools/parser"
)

// Example demonstrates applying an overlay using functional options.
func Example() {
	// Create a simple OpenAPI document
	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   "My API",
			"version": "1.0.0",
		},
		"paths": map[string]any{},
	}

	// Create a simple overlay
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info: overlay.Info{
			Title:   "Update API Title",
			Version: "1.0.0",
		},
		Actions: []overlay.Action{
			{
				Target: "$.info",
				Update: map[string]any{
					"title":         "Production API",
					"x-environment": "production",
				},
			},
		},
	}

	// Create a mock parse result
	parseResult := &parser.ParseResult{
		Document:     doc,
		SourceFormat: parser.SourceFormatYAML,
	}

	// Apply the overlay
	result, err := overlay.ApplyWithOptions(
		overlay.WithSpecParsed(*parseResult),
		overlay.WithOverlayParsed(o),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Check results
	fmt.Printf("Actions applied: %d\n", result.ActionsApplied)
	fmt.Printf("Actions skipped: %d\n", result.ActionsSkipped)

	// Access the modified document
	resultDoc := result.Document.(map[string]any)
	info := resultDoc["info"].(map[string]any)
	fmt.Printf("New title: %s\n", info["title"])

	// Output:
	// Actions applied: 1
	// Actions skipped: 0
	// New title: Production API
}

// Example_validate demonstrates validating an overlay document.
func Example_validate() {
	// Create an invalid overlay (missing required fields)
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info: overlay.Info{
			Title: "Test Overlay",
			// Missing Version
		},
		Actions: []overlay.Action{}, // Empty actions
	}

	// Validate the overlay
	errs := overlay.Validate(o)

	fmt.Printf("Validation errors: %d\n", len(errs))
	for _, err := range errs {
		fmt.Println(err.Message)
	}

	// Output:
	// Validation errors: 2
	// version is required
	// at least one action is required
}

// Example_parseOverlay demonstrates parsing an overlay from YAML.
func Example_parseOverlay() {
	yamlData := []byte(`
overlay: 1.0.0
info:
  title: My Overlay
  version: 1.0.0
actions:
  - target: $.info.title
    update: Updated Title
`)

	o, err := overlay.ParseOverlay(yamlData)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Overlay version: %s\n", o.Version)
	fmt.Printf("Overlay title: %s\n", o.Info.Title)
	fmt.Printf("Number of actions: %d\n", len(o.Actions))

	// Output:
	// Overlay version: 1.0.0
	// Overlay title: My Overlay
	// Number of actions: 1
}

// Example_removeAction demonstrates using remove actions.
func Example_removeAction() {
	// Create a document with internal paths
	doc := map[string]any{
		"openapi": "3.0.3",
		"info":    map[string]any{"title": "API", "version": "1.0.0"},
		"paths": map[string]any{
			"/public": map[string]any{
				"x-internal": false,
				"get":        map[string]any{"summary": "Public endpoint"},
			},
			"/internal": map[string]any{
				"x-internal": true,
				"get":        map[string]any{"summary": "Internal endpoint"},
			},
		},
	}

	// Create overlay to remove internal paths
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Remove Internal", Version: "1.0.0"},
		Actions: []overlay.Action{
			{
				Target: "$.paths[?@.x-internal==true]",
				Remove: true,
			},
		},
	}

	parseResult := &parser.ParseResult{
		Document:     doc,
		SourceFormat: parser.SourceFormatYAML,
	}

	result, err := overlay.ApplyWithOptions(
		overlay.WithSpecParsed(*parseResult),
		overlay.WithOverlayParsed(o),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Check remaining paths
	resultDoc := result.Document.(map[string]any)
	paths := resultDoc["paths"].(map[string]any)

	fmt.Printf("Remaining paths: %d\n", len(paths))
	for path := range paths {
		fmt.Printf("- %s\n", path)
	}

	// Output:
	// Remaining paths: 1
	// - /public
}
