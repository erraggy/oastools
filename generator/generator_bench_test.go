package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/erraggy/oastools/parser"
)

// Note on b.Fatalf usage in benchmarks:
// Using b.Fatalf for errors in benchmark setup or execution is an acceptable pattern.
// These operations (generate, parse) should never fail with valid test fixtures.
// If they do fail, it indicates a bug that should halt the benchmark immediately.

// Benchmark spec definitions
const (
	benchTypesSpec = `openapi: "3.0.0"
info:
  title: Benchmark API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
        tag:
          type: string
        category:
          $ref: '#/components/schemas/Category'
    Category:
      type: object
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
    Error:
      type: object
      properties:
        code:
          type: integer
        message:
          type: string
    Status:
      type: string
      enum:
        - available
        - pending
        - sold
`
	benchClientSpec = `openapi: "3.0.0"
info:
  title: Benchmark API
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        '200':
          description: A list of pets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
    post:
      operationId: createPet
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Pet'
      responses:
        '201':
          description: Created
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: A pet
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
    put:
      operationId: updatePet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Pet'
      responses:
        '200':
          description: Updated
    delete:
      operationId: deletePet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
      responses:
        '204':
          description: Deleted
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
`
	benchServerSpec = `openapi: "3.0.0"
info:
  title: Benchmark API
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        '200':
          description: A list of pets
    post:
      operationId: createPet
      responses:
        '201':
          description: Created
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: A pet
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
`
	benchFullSpec = `openapi: "3.0.0"
info:
  title: Benchmark API
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        '200':
          description: A list of pets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
    post:
      operationId: createPet
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Pet'
      responses:
        '201':
          description: Created
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: A pet
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
        category:
          $ref: '#/components/schemas/Category'
    Category:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
`
)

// parseSpecForBench parses a spec string and returns the parse result for benchmarking.
func parseSpecForBench(b *testing.B, spec, filename string) *parser.ParseResult {
	b.Helper()
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		b.Fatalf("failed to write temp file: %v", err)
	}

	p := parser.New()
	parseResult, err := p.Parse(tmpFile)
	if err != nil {
		b.Fatalf("failed to parse: %v", err)
	}
	return parseResult
}

// BenchmarkGenerate benchmarks code generation for different generation modes
func BenchmarkGenerate(b *testing.B) {
	b.Run("Types", func(b *testing.B) {
		parseResult := parseSpecForBench(b, benchTypesSpec, "bench-types.yaml")

		g := New()
		g.PackageName = "benchapi"
		g.GenerateTypes = true

		b.ReportAllocs()
		for b.Loop() {
			_, err := g.GenerateParsed(*parseResult)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Client", func(b *testing.B) {
		parseResult := parseSpecForBench(b, benchClientSpec, "bench-client.yaml")

		g := New()
		g.PackageName = "benchapi"
		g.GenerateClient = true

		b.ReportAllocs()
		for b.Loop() {
			_, err := g.GenerateParsed(*parseResult)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Server", func(b *testing.B) {
		parseResult := parseSpecForBench(b, benchServerSpec, "bench-server.yaml")

		g := New()
		g.PackageName = "benchapi"
		g.GenerateServer = true

		b.ReportAllocs()
		for b.Loop() {
			_, err := g.GenerateParsed(*parseResult)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("All", func(b *testing.B) {
		parseResult := parseSpecForBench(b, benchFullSpec, "bench-all.yaml")

		g := New()
		g.PackageName = "benchapi"
		g.GenerateTypes = true
		g.GenerateClient = true
		g.GenerateServer = true

		b.ReportAllocs()
		for b.Loop() {
			_, err := g.GenerateParsed(*parseResult)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
