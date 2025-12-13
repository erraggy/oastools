// Package overlay provides support for OpenAPI Overlay Specification v1.0.0.
//
// The OpenAPI Overlay Specification provides a standardized mechanism for augmenting
// OpenAPI documents through targeted transformations. Overlays use JSONPath expressions
// to select specific locations in an OpenAPI document and apply updates or removals.
//
// # Quick Start
//
// Apply an overlay using functional options (recommended):
//
//	result, err := overlay.ApplyWithOptions(
//	    overlay.WithSpecFilePath("openapi.yaml"),
//	    overlay.WithOverlayFilePath("changes.yaml"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Applied %d changes\n", result.ActionsApplied)
//
// Or use a reusable Applier instance:
//
//	a := overlay.NewApplier()
//	a.StrictTargets = true
//	result, err := a.Apply("openapi.yaml", "changes.yaml")
//
// # Overlay Document Structure
//
// An overlay document contains:
//   - overlay: The specification version (must be "1.0.0")
//   - info: Metadata with title and version
//   - extends: Optional URI of the target document
//   - actions: Ordered list of transformation actions
//
// Example overlay document:
//
//	overlay: 1.0.0
//	info:
//	  title: Production Customizations
//	  version: 1.0.0
//	actions:
//	  - target: $.info
//	    update:
//	      title: Production API
//	      x-environment: production
//	  - target: $.paths[?@.x-internal==true]
//	    remove: true
//
// # Action Types
//
// Update actions merge content into matched nodes:
//   - For objects: Properties are recursively merged
//   - For arrays: The update value is appended
//   - Same-name properties are replaced, new properties are added
//
// Remove actions delete matched nodes from their parent container.
// When both update and remove are specified, remove takes precedence.
//
// # JSONPath Support
//
// This package includes a built-in JSONPath implementation supporting:
//   - Basic navigation: $.info, $.paths['/users']
//   - Wildcards: $.paths.*, $.paths.*.*
//   - Array indices: $.servers[0], $.servers[-1]
//   - Simple filters: $.paths[?@.x-internal==true]
//   - Compound filters: $.paths[?@.deprecated==true && @.x-internal==false]
//   - Recursive descent: $..description (find all descriptions at any depth)
//
// # Dry-Run Preview
//
// Preview overlay changes without modifying the document:
//
//	result, _ := overlay.DryRunWithOptions(
//	    overlay.WithSpecFilePath("openapi.yaml"),
//	    overlay.WithOverlayFilePath("changes.yaml"),
//	)
//	for _, change := range result.Changes {
//	    fmt.Printf("Would %s %d nodes at %s\n",
//	        change.Operation, change.MatchCount, change.Target)
//	}
//
// # Validation
//
// Overlays can be validated before application:
//
//	o, _ := overlay.ParseOverlayFile("changes.yaml")
//	if errs := overlay.Validate(o); len(errs) > 0 {
//	    for _, err := range errs {
//	        fmt.Println(err)
//	    }
//	}
//
// # Related Packages
//
// The overlay package integrates with other oastools packages:
//   - [github.com/erraggy/oastools/parser] - Parse OpenAPI specifications
//   - [github.com/erraggy/oastools/validator] - Validate specifications
//   - [github.com/erraggy/oastools/joiner] - Join multiple specifications
//   - [github.com/erraggy/oastools/converter] - Convert between OAS versions
package overlay
