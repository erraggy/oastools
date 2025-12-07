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
