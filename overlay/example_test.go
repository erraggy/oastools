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

// Example_dryRun demonstrates previewing overlay changes without applying them.
func Example_dryRun() {
	// Create a document
	doc := map[string]any{
		"openapi": "3.0.3",
		"info":    map[string]any{"title": "API", "version": "1.0.0"},
		"paths": map[string]any{
			"/users": map[string]any{
				"get": map[string]any{"summary": "List users"},
			},
		},
	}

	// Create overlay with multiple actions
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Preview Changes", Version: "1.0.0"},
		Actions: []overlay.Action{
			{
				Target: "$.info",
				Update: map[string]any{"x-version": "v2"},
			},
			{
				Target: "$.paths.*",
				Update: map[string]any{"x-tested": true},
			},
		},
	}

	parseResult := &parser.ParseResult{
		Document:     doc,
		SourceFormat: parser.SourceFormatYAML,
	}

	// Preview changes without applying
	result, err := overlay.DryRunWithOptions(
		overlay.WithSpecParsed(*parseResult),
		overlay.WithOverlayParsed(o),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Would apply: %d actions\n", result.WouldApply)
	fmt.Printf("Would skip: %d actions\n", result.WouldSkip)
	for _, change := range result.Changes {
		fmt.Printf("- %s %d node(s) at %s\n", change.Operation, change.MatchCount, change.Target)
	}

	// Output:
	// Would apply: 2 actions
	// Would skip: 0 actions
	// - update 1 node(s) at $.info
	// - update 1 node(s) at $.paths.*
}

// Example_recursiveDescent demonstrates using $.. to find fields at any depth.
func Example_recursiveDescent() {
	// Create a document with descriptions at multiple levels
	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "API",
			"version":     "1.0.0",
			"description": "Top-level description",
		},
		"paths": map[string]any{
			"/users": map[string]any{
				"get": map[string]any{
					"summary":     "List users",
					"description": "Operation description",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Success response",
						},
					},
				},
			},
		},
	}

	// Update ALL descriptions at any depth using recursive descent
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Update Descriptions", Version: "1.0.0"},
		Actions: []overlay.Action{
			{
				Target: "$..description",
				Update: "Updated by overlay",
			},
		},
	}

	parseResult := &parser.ParseResult{
		Document:     doc,
		SourceFormat: parser.SourceFormatYAML,
	}

	// Use dry-run to see how many descriptions would be updated
	result, err := overlay.DryRunWithOptions(
		overlay.WithSpecParsed(*parseResult),
		overlay.WithOverlayParsed(o),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Recursive descent found: %d descriptions\n", result.Changes[0].MatchCount)

	// Output:
	// Recursive descent found: 3 descriptions
}

// Example_compoundFilter demonstrates using && and || in filter expressions.
func Example_compoundFilter() {
	// Create a document with various operation states
	doc := map[string]any{
		"openapi": "3.0.3",
		"info":    map[string]any{"title": "API", "version": "1.0.0"},
		"paths": map[string]any{
			"/deprecated-internal": map[string]any{
				"deprecated": true,
				"x-internal": true,
				"get":        map[string]any{"summary": "Old internal endpoint"},
			},
			"/deprecated-public": map[string]any{
				"deprecated": true,
				"x-internal": false,
				"get":        map[string]any{"summary": "Old public endpoint"},
			},
			"/active-internal": map[string]any{
				"deprecated": false,
				"x-internal": true,
				"get":        map[string]any{"summary": "Active internal endpoint"},
			},
			"/active-public": map[string]any{
				"deprecated": false,
				"x-internal": false,
				"get":        map[string]any{"summary": "Active public endpoint"},
			},
		},
	}

	// Use compound filter: deprecated AND internal
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info:    overlay.Info{Title: "Filter Test", Version: "1.0.0"},
		Actions: []overlay.Action{
			{
				Target: "$.paths[?@.deprecated==true && @.x-internal==true]",
				Update: map[string]any{"x-removal-scheduled": "2025-01-01"},
			},
		},
	}

	parseResult := &parser.ParseResult{
		Document:     doc,
		SourceFormat: parser.SourceFormatYAML,
	}

	result, err := overlay.DryRunWithOptions(
		overlay.WithSpecParsed(*parseResult),
		overlay.WithOverlayParsed(o),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Compound filter (deprecated && internal) matched: %d path(s)\n", result.Changes[0].MatchCount)

	// Output:
	// Compound filter (deprecated && internal) matched: 1 path(s)
}

// Example_toParseResult demonstrates converting an ApplyResult to a ParseResult
// for chaining with other oastools packages like validator.
func Example_toParseResult() {
	// Create a document as a raw map (simulating overlay's internal representation)
	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   "My API",
			"version": "1.0.0",
		},
		"paths": map[string]any{},
	}

	// Create an overlay
	o := &overlay.Overlay{
		Version: "1.0.0",
		Info: overlay.Info{
			Title:   "Add Environment Extension",
			Version: "1.0.0",
		},
		Actions: []overlay.Action{
			{
				Target: "$.info",
				Update: map[string]any{
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

	// Convert to ParseResult for chaining
	chainedResult := result.ToParseResult()

	// The chained result can be used with other packages
	// Note: Version info is empty because overlay works with raw maps.
	// For full version tracking, re-parse the result or use typed documents.
	fmt.Printf("Source: %s\n", chainedResult.SourcePath)
	fmt.Printf("Format: %s\n", chainedResult.SourceFormat)
	fmt.Printf("Has Document: %t\n", chainedResult.Document != nil)
	fmt.Printf("Actions applied: %d\n", result.ActionsApplied)

	// Output:
	// Source: overlay
	// Format: yaml
	// Has Document: true
	// Actions applied: 1
}
