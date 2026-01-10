package fixer_test

import (
	"fmt"
	"log"

	"github.com/erraggy/oastools/fixer"
	"github.com/erraggy/oastools/parser"
)

// Example demonstrates basic usage of the fixer package.
func Example() {
	// Parse a spec with missing path parameters
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	if err != nil {
		log.Fatal(err)
	}

	// Fix the specification
	f := fixer.New()
	result, err := f.FixParsed(*parseResult)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Applied %d fix(es)\n", result.FixCount)
	for _, fix := range result.Fixes {
		fmt.Printf("  %s: %s\n", fix.Type, fix.Description)
	}

	// Output:
	// Applied 1 fix(es)
	//   missing-path-parameter: Added missing path parameter 'userId' (type: string)
}

// ExampleFixWithOptions demonstrates using functional options.
func ExampleFixWithOptions() {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /projects/{projectId}:
    get:
      operationId: getProject
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	if err != nil {
		log.Fatal(err)
	}

	// Fix using functional options with type inference
	result, err := fixer.FixWithOptions(
		fixer.WithParsed(*parseResult),
		fixer.WithInferTypes(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Applied %d fix(es)\n", result.FixCount)
	for _, fix := range result.Fixes {
		fmt.Printf("  %s: %s\n", fix.Type, fix.Description)
	}

	// Output:
	// Applied 1 fix(es)
	//   missing-path-parameter: Added missing path parameter 'projectId' (type: integer)
}

// ExampleFixer_InferTypes demonstrates type inference from naming conventions.
func ExampleFixer_InferTypes() {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}/docs/{documentUuid}:
    get:
      operationId: getDocument
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	if err != nil {
		log.Fatal(err)
	}

	// Create fixer with type inference enabled
	f := fixer.New()
	f.InferTypes = true

	result, err := f.FixParsed(*parseResult)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Fixes:\n")
	for _, fix := range result.Fixes {
		fmt.Printf("  %s\n", fix.Description)
	}

	// Output:
	// Fixes:
	//   Added missing path parameter 'documentUuid' (type: string, format: uuid)
	//   Added missing path parameter 'userId' (type: integer)
}

// Example_swagger20 demonstrates fixing an OAS 2.0 (Swagger) specification.
func Example_swagger20() {
	spec := `
swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /pets/{petId}:
    get:
      operationId: getPet
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	if err != nil {
		log.Fatal(err)
	}

	f := fixer.New()
	result, err := f.FixParsed(*parseResult)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("OAS Version: %s\n", result.SourceVersion)
	fmt.Printf("Fixes: %d\n", result.FixCount)

	// Output:
	// OAS Version: 2.0
	// Fixes: 1
}

// ExampleFixResult_HasFixes demonstrates checking if fixes were applied.
func ExampleFixResult_HasFixes() {
	// A spec with no issues needs no fixes
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	if err != nil {
		log.Fatal(err)
	}

	f := fixer.New()
	result, err := f.FixParsed(*parseResult)
	if err != nil {
		log.Fatal(err)
	}

	if result.HasFixes() {
		fmt.Printf("Applied %d fixes\n", result.FixCount)
	} else {
		fmt.Println("No fixes needed")
	}

	// Output:
	// No fixes needed
}

// ExampleFixWithOptions_genericNaming demonstrates fixing schemas with invalid names
// like generic type parameters (e.g., Response[User]).
func ExampleFixWithOptions_genericNaming() {
	// A spec with generic-style schema names that need fixing
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Response[User]'
components:
  schemas:
    Response[User]:
      type: object
      properties:
        data:
          $ref: '#/components/schemas/User'
    User:
      type: object
      properties:
        id:
          type: integer
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	if err != nil {
		log.Fatal(err)
	}

	// Fix using the "Of" naming strategy: Response[User] -> ResponseOfUser
	result, err := fixer.FixWithOptions(
		fixer.WithParsed(*parseResult),
		fixer.WithEnabledFixes(fixer.FixTypeRenamedGenericSchema),
		fixer.WithGenericNaming(fixer.GenericNamingOf),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Applied %d fix(es)\n", result.FixCount)
	for _, fix := range result.Fixes {
		if fix.Type == fixer.FixTypeRenamedGenericSchema {
			fmt.Printf("  %s -> %s\n", fix.Before, fix.After)
		}
	}

	// Output:
	// Applied 1 fix(es)
	//   Response[User] -> ResponseOfUser
}

// ExampleWithGenericNamingConfig demonstrates fine-grained control over
// generic type name transformations.
func ExampleWithGenericNamingConfig() {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      operationId: listItems
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Map[string,Item]'
components:
  schemas:
    Map[string,Item]:
      type: object
    Item:
      type: object
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	if err != nil {
		log.Fatal(err)
	}

	// Configure custom naming with underscore strategy
	config := fixer.GenericNamingConfig{
		Strategy:       fixer.GenericNamingUnderscore,
		Separator:      "_",
		ParamSeparator: "_",
		PreserveCasing: false, // Convert to PascalCase
	}

	result, err := fixer.FixWithOptions(
		fixer.WithParsed(*parseResult),
		fixer.WithEnabledFixes(fixer.FixTypeRenamedGenericSchema),
		fixer.WithGenericNamingConfig(config),
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, fix := range result.Fixes {
		if fix.Type == fixer.FixTypeRenamedGenericSchema {
			fmt.Printf("Renamed: %s -> %s\n", fix.Before, fix.After)
		}
	}

	// Output:
	// Renamed: Map[string,Item] -> Map_String_Item_
}

// ExampleWithEnabledFixes demonstrates selectively enabling specific fix types.
func ExampleWithEnabledFixes() {
	// A spec with both missing parameters and invalid schema names
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Response[User]'
components:
  schemas:
    Response[User]:
      type: object
    User:
      type: object
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	if err != nil {
		log.Fatal(err)
	}

	// Only fix missing path parameters, ignore invalid schema names
	result, err := fixer.FixWithOptions(
		fixer.WithParsed(*parseResult),
		fixer.WithEnabledFixes(fixer.FixTypeMissingPathParameter),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Applied %d fix(es)\n", result.FixCount)
	for _, fix := range result.Fixes {
		fmt.Printf("  Type: %s\n", fix.Type)
	}

	// Output:
	// Applied 1 fix(es)
	//   Type: missing-path-parameter
}

// ExampleWithDryRun demonstrates previewing fixes without applying them.
func ExampleWithDryRun() {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	if err != nil {
		log.Fatal(err)
	}

	// Preview what fixes would be applied
	result, err := fixer.FixWithOptions(
		fixer.WithParsed(*parseResult),
		fixer.WithDryRun(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Would apply %d fix(es):\n", result.FixCount)
	for _, fix := range result.Fixes {
		fmt.Printf("  %s: %s\n", fix.Type, fix.Description)
	}

	// Output:
	// Would apply 1 fix(es):
	//   missing-path-parameter: Added missing path parameter 'userId' (type: string)
}

// Example_pruneUnusedSchemas demonstrates removing schemas that are not referenced.
func Example_pruneUnusedSchemas() {
	// A spec with an orphaned schema (UnusedModel is never referenced)
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
    UnusedModel:
      type: object
      properties:
        name:
          type: string
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	if err != nil {
		log.Fatal(err)
	}

	// Enable only the pruning fix
	result, err := fixer.FixWithOptions(
		fixer.WithParsed(*parseResult),
		fixer.WithEnabledFixes(fixer.FixTypePrunedUnusedSchema),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Pruned %d unused schema(s)\n", result.FixCount)
	for _, fix := range result.Fixes {
		fmt.Printf("  Removed: %s\n", fix.Before)
	}

	// Output:
	// Pruned 1 unused schema(s)
	//   Removed: UnusedModel
}

// Example_pruneEmptyPaths demonstrates removing path items with no operations.
func Example_pruneEmptyPaths() {
	// A spec with an empty path item (no HTTP methods defined)
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        '200':
          description: Success
  /empty:
    parameters:
      - name: version
        in: query
        schema:
          type: string
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	if err != nil {
		log.Fatal(err)
	}

	// Enable only empty path pruning
	result, err := fixer.FixWithOptions(
		fixer.WithParsed(*parseResult),
		fixer.WithEnabledFixes(fixer.FixTypePrunedEmptyPath),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Pruned %d empty path(s)\n", result.FixCount)
	for _, fix := range result.Fixes {
		fmt.Printf("  Removed: %s\n", fix.Path)
	}

	// Output:
	// Pruned 1 empty path(s)
	//   Removed: paths./empty
}

// ExampleParseGenericNamingStrategy demonstrates parsing strategy names from strings.
func ExampleParseGenericNamingStrategy() {
	strategies := []string{"of", "for", "underscore", "flattened", "dot"}

	for _, s := range strategies {
		strategy, err := fixer.ParseGenericNamingStrategy(s)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s -> %s\n", s, strategy)
	}

	// Output:
	// of -> of
	// for -> for
	// underscore -> underscore
	// flattened -> flattened
	// dot -> dot
}

// ExampleGenericNamingStrategy demonstrates the available naming strategies.
func ExampleGenericNamingStrategy() {
	// Show all available strategies
	strategies := []fixer.GenericNamingStrategy{
		fixer.GenericNamingUnderscore,
		fixer.GenericNamingOf,
		fixer.GenericNamingFor,
		fixer.GenericNamingFlattened,
		fixer.GenericNamingDot,
	}

	fmt.Println("Available strategies:")
	for _, s := range strategies {
		fmt.Printf("  %s\n", s)
	}

	// Output:
	// Available strategies:
	//   underscore
	//   of
	//   for
	//   flattened
	//   dot
}

// Example_csvEnumExpansion demonstrates fixing CSV-formatted enum values.
// Some tools incorrectly represent integer/number enums as comma-separated strings
// (e.g., "1,2,3" instead of [1, 2, 3]). This fix expands them to proper arrays.
func Example_csvEnumExpansion() {
	// A spec with CSV enum values (common mistake in generated specs)
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /items:
    get:
      operationId: getItems
      parameters:
        - name: status
          in: query
          schema:
            type: integer
            enum:
              - "1,2,3"
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	if err != nil {
		log.Fatal(err)
	}

	// Enable the CSV enum expansion fix (not enabled by default)
	result, err := fixer.FixWithOptions(
		fixer.WithParsed(*parseResult),
		fixer.WithEnabledFixes(fixer.FixTypeEnumCSVExpanded),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Applied %d fix(es)\n", result.FixCount)
	for _, fix := range result.Fixes {
		fmt.Printf("  %s: %s\n", fix.Type, fix.Description)
	}

	// Output:
	// Applied 1 fix(es)
	//   enum-csv-expanded: expanded CSV enum string to 3 individual values
}

// Example_toParseResult demonstrates using ToParseResult() to chain fixer
// output with other packages like validator, converter, or differ.
func Example_toParseResult() {
	// Fix a spec with missing path parameters
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	if err != nil {
		log.Fatal(err)
	}

	fixResult, err := fixer.FixWithOptions(
		fixer.WithParsed(*parseResult),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Convert to ParseResult for use with validator, converter, differ, etc.
	result := fixResult.ToParseResult()

	// The ParseResult can now be used with other packages:
	// - validator.ValidateParsed(*result)
	// - converter.ConvertParsed(*result, "3.1.0")
	// - differ.DiffParsed(*baseResult, *result)

	// SourcePath comes from the original parse (defaults to "ParseBytes.yaml" for bytes)
	fmt.Printf("Has source: %v\n", result.SourcePath != "")
	fmt.Printf("Version: %s\n", result.Version)
	fmt.Printf("Has document: %v\n", result.Document != nil)
	fmt.Printf("Fixes applied: %d\n", fixResult.FixCount)
	// Output:
	// Has source: true
	// Version: 3.0.0
	// Has document: true
	// Fixes applied: 1
}
